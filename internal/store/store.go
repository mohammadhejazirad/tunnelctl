package store

import (
	"bytes"
	"encoding/json"
	"errors"
	"fmt"
	"io"
	"os"
	"path/filepath"
	"strings"
	"tunnelctl/internal/domain"
	"tunnelctl/internal/platform"
)

const MaxStoreBytes = 10 * 1024 * 1024

type Paths struct {
	Config  string `json:"config"`
	State   string `json:"state"`
	Cache   string `json:"cache"`
	Runtime string `json:"runtime"`
}
type Store struct{ Paths Paths }

func New(root string) (*Store, error) {
	if os.Getuid() != os.Geteuid() || os.Getenv("SUDO_USER") != "" || os.Getenv("SUDO_UID") != "" || os.Getenv("PKEXEC_UID") != "" {
		return nil, domain.Fail(4, "User storage must not be opened through elevation.", "Run tunnelctl as the original user without sudo or a privileged helper.")
	}
	if !platform.Supported() {
		return nil, domain.Fail(3, "Runtime support is currently limited to Linux.", "macOS and Windows native tests and lifecycle adapters are pending.")
	}
	var p Paths
	if root != "" {
		a, e := filepath.Abs(root)
		if e != nil {
			return nil, domain.Unexpected()
		}
		if e = PrivateDir(a); e != nil {
			return nil, e
		}
		p = Paths{filepath.Join(a, "config"), filepath.Join(a, "state"), filepath.Join(a, "cache"), filepath.Join(a, "runtime")}
	} else {
		home, e := os.UserHomeDir()
		if e != nil {
			return nil, domain.Unexpected()
		}
		xdg := func(key, defaultPath string) string {
			v := os.Getenv(key)
			if v == "" || !filepath.IsAbs(v) {
				v = defaultPath
			}
			return filepath.Join(v, "tunnelctl")
		}
		p.Config = xdg("XDG_CONFIG_HOME", filepath.Join(home, ".config"))
		p.State = xdg("XDG_STATE_HOME", filepath.Join(home, ".local", "state"))
		p.Cache = xdg("XDG_CACHE_HOME", filepath.Join(home, ".cache"))
		if v := os.Getenv("XDG_RUNTIME_DIR"); v != "" && filepath.IsAbs(v) {
			p.Runtime = filepath.Join(v, "tunnelctl")
		} else {
			p.Runtime = filepath.Join(p.State, "runtime")
		}
	}
	for _, d := range []string{p.Config, p.State, p.Cache, p.Runtime} {
		if e := PrivateDir(d); e != nil {
			return nil, e
		}
	}
	return &Store{p}, nil
}
func PrivateDir(path string) error {
	// Reject symlink ancestors as well as symlink leaf directories.
	for cur := filepath.Clean(path); cur != "."; cur = filepath.Dir(cur) {
		st, e := os.Lstat(cur)
		if e == nil && st.Mode()&os.ModeSymlink != 0 {
			return domain.Fail(4, "Storage paths must not contain symlinks.", "Choose a private directory with no symlink components.")
		}
		if e != nil && !os.IsNotExist(e) {
			return permission()
		}
		if cur == filepath.Dir(cur) {
			break
		}
	}
	if e := os.MkdirAll(path, 0700); e != nil {
		return permission()
	}
	st, e := os.Lstat(path)
	if e != nil || !st.IsDir() || st.Mode().Perm()&0077 != 0 || !platform.CheckOwner(st) {
		return permission()
	}
	return nil
}
func permission() error {
	return domain.Fail(4, "Storage must be owned by the current user and private.", "Use 0700 directories and 0600 files; choose --data-dir on a filesystem supporting Unix permissions.")
}
func checkFile(f *os.File) error {
	st, e := f.Stat()
	if e != nil || !st.Mode().IsRegular() || st.Mode().Perm()&0077 != 0 || !platform.CheckOwner(st) {
		return permission()
	}
	return nil
}
func OpenPrivate(path string, flags int) (*os.File, error) {
	f, e := platform.OpenPrivate(path, flags)
	if e != nil {
		return nil, e
	}
	if e = checkFile(f); e != nil {
		f.Close()
		return nil, e
	}
	return f, nil
}
func Acquire(path string) (*os.File, error) {
	f, e := OpenPrivate(path, os.O_CREATE|os.O_RDWR)
	if e != nil {
		return nil, permission()
	}
	if e = platform.Lock(f); e != nil {
		f.Close()
		return nil, e
	}
	return f, nil
}
func Release(f *os.File) error {
	a := platform.Unlock(f)
	b := f.Close()
	if a != nil || b != nil {
		return domain.Fail(7, "Cannot release the owned lock.", "Check storage and retry status.")
	}
	return nil
}
func AtomicJSON(path string, value any) error {
	b, e := json.MarshalIndent(value, "", "  ")
	if e != nil {
		return domain.Unexpected()
	}
	return AtomicBytes(path, b)
}
func AtomicBytes(path string, b []byte) (err error) {
	return atomicBytes(path, b, nil)
}

// checkpoint is an internal fault-injection seam, never exposed by the application.
func atomicBytes(path string, b []byte, checkpoint func(string)) (err error) {
	if len(b) > MaxStoreBytes {
		return domain.Fail(2, "Storage size limit exceeded.", "Reduce the number or size of stored profiles.")
	}
	f, e := os.CreateTemp(filepath.Dir(path), ".atomic-")
	if e != nil {
		return permission()
	}
	name := f.Name()
	defer func() {
		if e := os.Remove(name); e != nil && !os.IsNotExist(e) {
			err = errors.Join(err, domain.Fail(7, "Cannot remove a private staging file.", "Inspect private storage for .atomic-* files after stopping all writers; never publish their contents."))
		}
	}()
	if checkpoint != nil {
		checkpoint("created")
	}
	if _, e = f.Write(b); e == nil {
		e = f.Sync()
	}
	ce := f.Close()
	if e != nil || ce != nil {
		return domain.Unexpected()
	}
	if checkpoint != nil {
		checkpoint("synced")
	}
	if e = os.Rename(name, path); e != nil {
		return domain.Unexpected()
	}
	if checkpoint != nil {
		checkpoint("replaced")
	}
	d, e := os.Open(filepath.Dir(path))
	if e != nil {
		return domain.Fail(7, "Storage replacement completed but its directory cannot be synchronized.", "Reload private storage; replacement may be visible but durability is uncertain.")
	}
	defer func() {
		if d.Close() != nil {
			err = errors.Join(err, domain.Fail(7, "Cannot close the storage directory after replacement.", "Reload private storage and inspect filesystem health."))
		}
	}()
	if e = d.Sync(); e != nil {
		return domain.Fail(7, "Storage replacement completed but directory synchronization failed.", "Check filesystem durability before continuing.")
	}
	if checkpoint != nil {
		checkpoint("durable")
	}
	return nil
}
func (s *Store) Load() (domain.Settings, error) {
	v := domain.Settings{SchemaVersion: domain.SchemaVersion, Profiles: []domain.Profile{}, Preferences: domain.DefaultPreferences()}
	f, e := OpenPrivate(filepath.Join(s.Paths.Config, "profiles.json"), os.O_RDONLY)
	if os.IsNotExist(e) {
		return v, nil
	}
	if e != nil {
		return v, permission()
	}
	defer f.Close()
	if st, e := f.Stat(); e != nil || st.Size() > MaxStoreBytes {
		return v, corrupt()
	}
	v.Preferences = domain.Preferences{}
	if e = DecodeStrict(f, MaxStoreBytes, &v); e != nil {
		return v, corrupt()
	}
	if v.SchemaVersion != domain.SchemaVersion && v.SchemaVersion != 1 && v.SchemaVersion != 2 && v.SchemaVersion != 3 {
		return v, domain.Fail(3, "The storage schema is unsupported.", "Use a compatible application; no automatic destructive migration is performed.")
	}
	if v.SchemaVersion == 1 && len(v.Subscriptions) != 0 {
		return v, corrupt()
	}
	if v.SchemaVersion < 3 {
		if v.Preferences != (domain.Preferences{}) {
			return v, corrupt()
		}
		v.Preferences = domain.DefaultPreferences()
	}
	return v, validate(v)
}

func validate(v domain.Settings) error {
	if v.Preferences.Validate() != nil {
		return corrupt()
	}
	subs := map[string]bool{}
	names := map[string]bool{}
	if len(v.Subscriptions) > 1000 {
		return corrupt()
	}
	for _, sub := range v.Subscriptions {
		if len(sub.PreviewKey) != 32 || strings.Trim(sub.PreviewKey, "0123456789abcdef") != "" || sub.Revision == 0 {
			return corrupt()
		}
		if len(sub.ID) != 32 || strings.Trim(sub.ID, "0123456789abcdef") != "" || subs[sub.ID] || sub.Name == "" || len(sub.Name) > 256 || domain.SafeName(sub.Name) != sub.Name || names[sub.Name] || len(sub.URL) > 16384 {
			return corrupt()
		}
		subs[sub.ID] = true
		names[sub.Name] = true
	}
	if len(v.Profiles) > 10000 {
		return corrupt()
	}
	ids := map[string]bool{}
	found := v.Selected == ""
	for _, p := range v.Profiles {
		if len(p.ID) != 32 || strings.Trim(p.ID, "0123456789abcdef") != "" || ids[p.ID] || p.Validate() != nil || domain.SafeName(p.Name) != p.Name || len(p.Name) > 256 || domain.ValidateTags(p.Tags) != nil {
			return corrupt()
		}
		ids[p.ID] = true
		if p.SubscriptionID != "" && !subs[p.SubscriptionID] {
			return corrupt()
		}
		found = found || p.ID == v.Selected
	}
	if !found {
		return corrupt()
	}
	return nil
}
func corrupt() error {
	return domain.Fail(2, "Private profile storage is invalid.", "Restore a known good private backup; the existing file has not been overwritten.")
}
func (s *Store) Update(fn func(*domain.Settings) error) (err error) {
	lock, e := Acquire(filepath.Join(s.Paths.Config, "profiles.lock"))
	if e != nil {
		return e
	}
	defer func() { err = errors.Join(err, Release(lock)) }()
	v, e := s.Load()
	if e != nil {
		return e
	}
	oldSchema := v.SchemaVersion
	migrate := oldSchema < domain.SchemaVersion
	var old []byte
	if migrate {
		f, e := OpenPrivate(filepath.Join(s.Paths.Config, "profiles.json"), os.O_RDONLY)
		if e != nil {
			return permission()
		}
		old, e = io.ReadAll(io.LimitReader(f, MaxStoreBytes+1))
		ce := f.Close()
		if e != nil || ce != nil || len(old) > MaxStoreBytes {
			return corrupt()
		}
	}
	if e = fn(&v); e != nil {
		return e
	}
	v.SchemaVersion = domain.SchemaVersion
	if e = validate(v); e != nil {
		return e
	}
	if migrate {
		backup := filepath.Join(s.Paths.Config, fmt.Sprintf("profiles.schema%d.backup.json", oldSchema))
		if _, e := os.Lstat(backup); os.IsNotExist(e) {
			if e = AtomicBytes(backup, old); e != nil {
				return e
			}
		} else if e != nil {
			return permission()
		} else {
			f, e := OpenPrivate(backup, os.O_RDONLY)
			if e != nil {
				return permission()
			}
			b, re := io.ReadAll(io.LimitReader(f, MaxStoreBytes+1))
			ce := f.Close()
			if re != nil || ce != nil || !bytes.Equal(b, old) {
				return domain.Fail(6, "An existing migration backup differs from the current data.", "Preserve both private files and move the older backup to a separate private location before retrying.")
			}
		}
	}
	if v.Revision == ^uint64(0) {
		return domain.Fail(6, "Storage revision limit reached.", "Preserve your private data and contact the maintainer.")
	}
	v.Revision++
	return AtomicJSON(filepath.Join(s.Paths.Config, "profiles.json"), v)
}
func Resolve(v domain.Settings, key string) (domain.Profile, error) {
	if key == "" {
		key = v.Selected
	}
	for _, p := range v.Profiles {
		if p.ID == key {
			return p, nil
		}
	}
	var result domain.Profile
	count := 0
	for _, p := range v.Profiles {
		if p.Name == key {
			result = p
			count++
		}
	}
	if count == 1 {
		return result, nil
	}
	if count > 1 {
		return result, domain.Fail(6, "The profile name is ambiguous.", "Use the full profile ID.")
	}
	return result, domain.Fail(2, "Profile not found or no profile is selected.", "Import a profile and select its ID or pass --profile.")
}
