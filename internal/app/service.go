package app

import (
	"context"
	"errors"
	"fmt"
	"io"
	"os"
	"path/filepath"
	"runtime"
	"strings"
	"time"
	"tunnelctl/internal/core"
	"tunnelctl/internal/core/singbox"
	"tunnelctl/internal/domain"
	"tunnelctl/internal/importer"
	"tunnelctl/internal/session"
	"tunnelctl/internal/store"
	"unicode"
)

type Service struct {
	Store    *store.Store
	CorePath string
	Timeout  time.Duration
}

func New(root, path string, timeout time.Duration) (*Service, error) {
	s, e := store.New(root)
	if e != nil {
		return nil, e
	}
	return &Service{Store: s, CorePath: path, Timeout: timeout}, nil
}
func (s *Service) Init() error { return s.Store.Update(func(*domain.Settings) error { return nil }) }
func (s *Service) Import(r io.Reader) (domain.PublicProfile, error) {
	p, e := importer.Parse(r)
	if e != nil {
		return domain.PublicProfile{}, e
	}
	selected := false
	e = s.Store.Update(func(v *domain.Settings) error {
		if len(v.Profiles) >= 10000 {
			return domain.Fail(2, "Profile count limit exceeded.", "Use a separate private profile store.")
		}
		for _, old := range v.Profiles {
			if old.Identity() == p.Identity() {
				return domain.Fail(6, "This connection profile is already stored.", "Use profile list to find the existing profile ID.")
			}
		}
		v.Profiles = append(v.Profiles, p)
		if v.Selected == "" {
			v.Selected = p.ID
			selected = true
		}
		return nil
	})
	return p.Public(selected), e
}
func (s *Service) List() ([]domain.PublicProfile, error) {
	v, e := s.Store.Load()
	if e != nil {
		return nil, e
	}
	out := make([]domain.PublicProfile, 0, len(v.Profiles))
	for _, p := range v.Profiles {
		out = append(out, p.Public(p.ID == v.Selected))
	}
	return out, nil
}
func (s *Service) Show(key string) (domain.PublicProfile, error) {
	v, e := s.Store.Load()
	if e != nil {
		return domain.PublicProfile{}, e
	}
	p, e := store.Resolve(v, key)
	return p.Public(p.ID == v.Selected), e
}
func (s *Service) Select(key string) (domain.PublicProfile, error) {
	var out domain.PublicProfile
	e := s.Store.Update(func(v *domain.Settings) error {
		p, e := store.Resolve(*v, key)
		if e != nil {
			return e
		}
		v.Selected = p.ID
		out = p.Public(true)
		return nil
	})
	return out, e
}
func (s *Service) supervisor() *session.Supervisor {
	return &session.Supervisor{Runtime: s.Store.Paths.Runtime, LogPath: filepath.Join(s.Store.Paths.State, "events.json")}
}
func (s *Service) Status() (domain.Session, error) { return s.supervisor().Status() }
func (s *Service) Stop() error                     { return s.supervisor().Stop() }
func (s *Service) StopSession(id string) error     { return s.supervisor().StopSession(id) }
func (s *Service) Run(ctx context.Context, key, mode string, l core.Listener, duration time.Duration, onReady func(domain.Session) error) (err error) {
	if e := checkSessionExecution(); e != nil {
		return e
	}
	if mode != "proxy" {
		return domain.Fail(3, "Only proxy mode is implemented.", "TUN and host network mutation are deferred.")
	}
	if duration < 0 {
		return domain.Fail(2, "Duration cannot be negative.", "Use --duration 10m or omit it for a session without a deadline.")
	}
	if runtime.GOARCH != "amd64" {
		return domain.Fail(3, "The core artifact is currently approved only for Linux amd64.", "Validate and record an arm64 artifact before adding runtime support.")
	}
	v, e := s.Store.Load()
	if e != nil {
		return e
	}
	p, e := store.Resolve(v, key)
	if e != nil {
		return e
	}
	a, e := singbox.Open(s.CorePath)
	if e != nil {
		return e
	}
	defer func() {
		if e := a.Close(); e != nil {
			err = errors.Join(err, domain.Fail(7, "Cannot close the verified core executable.", "Check operating system file resources."))
		}
	}()
	return s.supervisor().Run(ctx, a, session.Options{Profile: p, Listener: l, Duration: duration, Timeout: s.Timeout}, onReady)
}

type Doctor struct {
	Platform            string            `json:"platform"`
	CoreVersion         string            `json:"core_version"`
	Capabilities        core.Capabilities `json:"capabilities"`
	Storage             string            `json:"storage"`
	HostNetworkMutation bool              `json:"host_network_mutation"`
}

func (s *Service) Doctor(ctx context.Context) (Doctor, error) {
	if _, e := s.Store.Load(); e != nil {
		return Doctor{}, e
	}
	a, e := singbox.Open(s.CorePath)
	if e != nil {
		return Doctor{}, e
	}
	defer a.Close()
	ctx, cancel := context.WithTimeout(ctx, s.Timeout)
	defer cancel()
	v, e := a.Version(ctx)
	if e != nil {
		return Doctor{}, e
	}
	return Doctor{runtime.GOOS + "/" + runtime.GOARCH, v, a.Capabilities(), fmt.Sprintf("Private, schema version %d; plaintext at rest", domain.SchemaVersion), false}, nil
}
func ImportFile(path string) (*os.File, error) {
	f, e := store.OpenPrivate(path, os.O_RDONLY)
	if e != nil {
		return nil, domain.Fail(4, "Profile input must be a readable private regular file owned by the current user.", "Set its permissions to 0600, or provide the link through --stdin.")
	}
	return f, nil
}
func AuthFile(path string, l core.Listener) (core.Listener, error) {
	if path == "" {
		return l, nil
	}
	f, e := ImportFile(path)
	if e != nil {
		return l, e
	}
	defer f.Close()
	if st, e := f.Stat(); e != nil || st.Size() > 8192 {
		return l, authError()
	}
	var v struct {
		Username string `json:"username"`
		Password string `json:"password"`
	}
	if store.DecodeStrict(f, 8192, &v) != nil {
		return l, authError()
	}
	if len(v.Username) == 0 || len(v.Username) > 255 || len(v.Password) == 0 || len(v.Password) > 255 {
		return l, authError()
	}
	for _, s := range []string{v.Username, v.Password} {
		for _, c := range s {
			if unicode.IsControl(c) || unicode.IsSpace(c) || c == ':' || c > 127 {
				return l, authError()
			}
		}
	}
	l.Username = v.Username
	l.Password = v.Password
	return l, nil
}
func authError() error {
	return domain.Fail(2, "The local proxy authentication file is invalid.", "Use JSON with username and password: 1–255 printable ASCII bytes each, without whitespace or colon.")
}
func DefaultCore() string {
	p := os.Getenv("TUNNELCTL_CORE")
	if strings.TrimSpace(p) == "" {
		return "/usr/local/bin/sing-box"
	}
	return p
}
