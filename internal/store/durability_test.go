//go:build linux

package store

import (
	"bufio"
	"bytes"
	"encoding/json"
	"errors"
	"fmt"
	"os"
	"os/exec"
	"os/signal"
	"path/filepath"
	"syscall"
	"testing"
	"time"
	"tunnelctl/internal/domain"
)

func durableStore(t *testing.T) *Store {
	t.Helper()
	root := t.TempDir()
	if e := os.Chmod(root, 0700); e != nil {
		t.Fatal(e)
	}
	s, e := New(root)
	if e != nil {
		t.Fatal(e)
	}
	if e = s.Update(func(*domain.Settings) error { return nil }); e != nil {
		t.Fatal(e)
	}
	return s
}

// Separate processes exercise kernel locks and real write/rename/syscall failures.
// No fault-injection option exists in the public binary.
func TestStorageWorker(t *testing.T) {
	mode := os.Getenv("TUNNELCTL_STORE_WORKER")
	if mode == "" {
		return
	}
	s, e := New(os.Getenv("TUNNELCTL_STORE_ROOT"))
	if e != nil {
		t.Fatal(e)
	}
	switch mode {
	case "limited-write":
		signal.Ignore(syscall.SIGXFSZ)
		var limit syscall.Rlimit
		if e = syscall.Getrlimit(syscall.RLIMIT_FSIZE, &limit); e != nil {
			t.Fatal(e)
		}
		limit.Cur = 1
		if e = syscall.Setrlimit(syscall.RLIMIT_FSIZE, &limit); e != nil {
			t.Fatal(e)
		}
		if e = s.Update(func(v *domain.Settings) error { v.Preferences.ListenerPort = 9999; return nil }); e == nil {
			t.Fatal("limited write falsely succeeded")
		}
	case "crash":
		lock, e := Acquire(filepath.Join(s.Paths.Config, "profiles.lock"))
		if e != nil {
			t.Fatal(e)
		}
		defer Release(lock)
		v, e := s.Load()
		if e != nil {
			t.Fatal(e)
		}
		v.Preferences.ListenerPort = 9999
		v.Revision++
		b, e := json.Marshal(v)
		if e != nil {
			t.Fatal(e)
		}
		e = atomicBytes(filepath.Join(s.Paths.Config, "profiles.json"), b, func(stage string) {
			if stage == os.Getenv("TUNNELCTL_STORE_STAGE") {
				fmt.Println("checkpoint")
				var one [1]byte
				os.Stdin.Read(one[:])
			}
		})
		if e != nil {
			t.Fatal(e)
		}
	case "concurrent":
		for n := 0; n < 25; n++ {
			deadline := time.Now().Add(10 * time.Second)
			for {
				e = s.Update(func(v *domain.Settings) error {
					id, e := domain.NewID()
					if e != nil {
						return e
					}
					v.Profiles = append(v.Profiles, domain.Profile{ID: id, Name: "synthetic", Protocol: "shadowsocks", Host: "127.0.0.1", Port: 8388, Method: "aes-256-gcm", Password: "synthetic"})
					return nil
				})
				if e == nil {
					break
				}
				var de *domain.Error
				if !errors.As(e, &de) || de.Code != 6 || time.Now().After(deadline) {
					t.Fatal("concurrent transaction failed")
				}
				time.Sleep(time.Millisecond)
			}
		}
	default:
		t.Fatal("unknown worker")
	}
}

func worker(s *Store, mode, stage string) *exec.Cmd {
	c := exec.Command(os.Args[0], "-test.run=^TestStorageWorker$", "-test.count=1")
	c.Env = append(os.Environ(), "TUNNELCTL_STORE_WORKER="+mode, "TUNNELCTL_STORE_ROOT="+filepath.Dir(s.Paths.Config), "TUNNELCTL_STORE_STAGE="+stage)
	return c
}

func TestKilledWriterAtAtomicBoundaries(t *testing.T) {
	for _, stage := range []string{"created", "synced", "replaced", "durable"} {
		t.Run(stage, func(t *testing.T) {
			s := durableStore(t)
			c := worker(s, "crash", stage)
			output, e := c.StdoutPipe()
			if e != nil {
				t.Fatal(e)
			}
			input, e := c.StdinPipe()
			if e != nil {
				t.Fatal(e)
			}
			defer input.Close()
			if e = c.Start(); e != nil {
				t.Fatal(e)
			}
			defer func() { c.Process.Kill(); c.Wait() }()
			ready := make(chan bool, 1)
			go func() { scanner := bufio.NewScanner(output); ready <- scanner.Scan() && scanner.Text() == "checkpoint" }()
			select {
			case ok := <-ready:
				if !ok {
					t.Fatal("child did not reach boundary")
				}
			case <-time.After(15 * time.Second):
				t.Fatal("checkpoint timeout")
			}
			if e = c.Process.Kill(); e != nil {
				t.Fatal(e)
			}
			c.Wait()
			v, e := s.Load()
			if e != nil {
				t.Fatal("killed writer corrupted snapshot", e)
			}
			want := uint16(2080)
			if stage == "replaced" || stage == "durable" {
				want = 9999
			}
			if v.Preferences.ListenerPort != want {
				t.Fatal("unexpected transaction boundary value")
			}
			files, e := filepath.Glob(filepath.Join(s.Paths.Config, ".atomic-*"))
			if e != nil {
				t.Fatal(e)
			}
			for _, p := range files {
				st, e := os.Stat(p)
				if e != nil || st.Mode().Perm() != 0600 {
					t.Fatal("staging privacy")
				}
			}
			// SIGKILL releases the flock; abandoned private stages are never loaded.
			if e = s.Update(func(*domain.Settings) error { return nil }); e != nil {
				t.Fatal("lock not recovered", e)
			}
		})
	}
}

func TestFailedWritePreservesSnapshot(t *testing.T) {
	s := durableStore(t)
	path := filepath.Join(s.Paths.Config, "profiles.json")
	before, _ := os.ReadFile(path)
	if out, e := worker(s, "limited-write", "").CombinedOutput(); e != nil {
		t.Fatalf("write failure helper: %s", out)
	}
	after, _ := os.ReadFile(path)
	if !bytes.Equal(before, after) {
		t.Fatal("failed write replaced healthy snapshot")
	}
	stages, _ := filepath.Glob(filepath.Join(s.Paths.Config, ".atomic-*"))
	if len(stages) != 0 {
		t.Fatal("failed write leaked stage")
	}
	if e := os.Chmod(path, 0644); e != nil {
		t.Fatal(e)
	}
	if e := s.Update(func(*domain.Settings) error { return nil }); e == nil {
		t.Fatal("public file accepted for mutation")
	}
	after, _ = os.ReadFile(path)
	if !bytes.Equal(before, after) {
		t.Fatal("permission failure changed data")
	}
}

func TestConcurrentProcessesNoLostUpdate(t *testing.T) {
	s := durableStore(t)
	a, b := worker(s, "concurrent", ""), worker(s, "concurrent", "")
	var oa, ob bytes.Buffer
	a.Stdout = &oa
	a.Stderr = &oa
	b.Stdout = &ob
	b.Stderr = &ob
	if e := a.Start(); e != nil {
		t.Fatal(e)
	}
	if e := b.Start(); e != nil {
		a.Process.Kill()
		a.Wait()
		t.Fatal(e)
	}
	ea, eb := a.Wait(), b.Wait()
	if ea != nil || eb != nil {
		t.Fatalf("workers failed: %s %s", oa.String(), ob.String())
	}
	v, e := s.Load()
	if e != nil || len(v.Profiles) != 50 || v.Revision != 51 {
		t.Fatal("concurrent commits were lost", e)
	}
}

func TestSchemaTwoMigrationPreservesPrivateData(t *testing.T) {
	s := durableStore(t)
	path := filepath.Join(s.Paths.Config, "profiles.json")
	const old = `{"schema_version":2,"revision":9,"selected":"11111111111111111111111111111111","profiles":[{"id":"11111111111111111111111111111111","name":"custom","protocol":"shadowsocks","host":"127.0.0.1","port":8388,"method":"aes-256-gcm","password":"synthetic-secret","subscription_id":"22222222222222222222222222222222","user_named":true}],"subscriptions":[{"id":"22222222222222222222222222222222","name":"source","url":"https://example.invalid/feed?token=synthetic","revision":1,"preview_key":"33333333333333333333333333333333"}]}`
	if e := AtomicBytes(path, []byte(old)); e != nil {
		t.Fatal(e)
	}
	v, e := s.Load()
	if e != nil || v.SchemaVersion != 2 || v.Preferences != domain.DefaultPreferences() {
		t.Fatal("read normalization", e)
	}
	before, _ := os.ReadFile(path)
	if string(before) != old {
		t.Fatal("load performed migration")
	}
	if e = s.Update(func(*domain.Settings) error { return nil }); e != nil {
		t.Fatal(e)
	}
	v, e = s.Load()
	if e != nil || v.SchemaVersion != domain.SchemaVersion || v.Revision != 10 || v.Selected != "11111111111111111111111111111111" || v.Profiles[0].Password != "synthetic-secret" || v.Subscriptions[0].URL != "https://example.invalid/feed?token=synthetic" {
		t.Fatal("migration lost private data", e)
	}
	backup := filepath.Join(s.Paths.Config, "profiles.schema2.backup.json")
	b, e := os.ReadFile(backup)
	if e != nil || string(b) != old {
		t.Fatal("backup differs from original")
	}
	st, e := os.Stat(backup)
	if e != nil || st.Mode().Perm() != 0600 {
		t.Fatal("backup not private")
	}
	// A conflicting preexisting backup is never overwritten.
	if e = AtomicBytes(path, []byte(old+" ")); e != nil {
		t.Fatal(e)
	}
	if e = s.Update(func(*domain.Settings) error { return nil }); e == nil {
		t.Fatal("conflicting backup ignored")
	}
	b, _ = os.ReadFile(backup)
	if string(b) != old {
		t.Fatal("backup overwritten")
	}
}

func TestCorruptAndNewerDataNeverReset(t *testing.T) {
	for _, data := range []string{`{"schema_version":3,`, `{"schema_version":999,"profiles":[]}`, `{"schema_version":3,"profiles":[]}`, `{"schema_version":3,"preferences":null}`} {
		s := durableStore(t)
		path := filepath.Join(s.Paths.Config, "profiles.json")
		if e := AtomicBytes(path, []byte(data)); e != nil {
			t.Fatal(e)
		}
		if e := s.Update(func(v *domain.Settings) error { v.Profiles = nil; return nil }); e == nil {
			t.Fatal("invalid snapshot reset")
		}
		b, _ := os.ReadFile(path)
		if string(b) != data {
			t.Fatal("invalid snapshot overwritten")
		}
	}
}

func TestXDGPathsAreUserScopedAndIndependentOfCWD(t *testing.T) {
	a, b := t.TempDir(), t.TempDir()
	os.Chmod(a, 0700)
	os.Chmod(b, 0700)
	for _, base := range []string{a, b} {
		t.Setenv("HOME", base)
		t.Setenv("XDG_CONFIG_HOME", filepath.Join(base, "cfg"))
		t.Setenv("XDG_STATE_HOME", filepath.Join(base, "state"))
		t.Setenv("XDG_CACHE_HOME", filepath.Join(base, "cache"))
		t.Setenv("XDG_RUNTIME_DIR", filepath.Join(base, "run"))
		s, e := New("")
		if e != nil {
			t.Fatal(e)
		}
		if s.Paths.Config != filepath.Join(base, "cfg", "tunnelctl") || s.Paths.Runtime != filepath.Join(base, "run", "tunnelctl") {
			t.Fatal("incorrect per-user paths")
		}
		initial, e := s.Load()
		if e != nil || initial.Preferences.ListenerPort != 2080 || len(initial.Profiles) != 0 {
			t.Fatal("different user roots shared data")
		}
		if e = s.Update(func(v *domain.Settings) error { v.Preferences.ListenerPort = 9998; return nil }); e != nil {
			t.Fatal(e)
		}
		t.Chdir(t.TempDir())
		again, e := New("")
		if e != nil {
			t.Fatal(e)
		}
		v, e := again.Load()
		if e != nil || v.Preferences.ListenerPort != 9998 {
			t.Fatal("data depended on CWD")
		}
	}
	t.Setenv("SUDO_UID", "1000")
	if _, e := New(a); e == nil {
		t.Fatal("elevated storage was opened")
	}
}
