//go:build linux

package store

import (
	"os"
	"path/filepath"
	"testing"
	"tunnelctl/internal/domain"
)

func TestPrivateAtomicStorage(t *testing.T) {
	root := t.TempDir()
	os.Chmod(root, 0700)
	s, e := New(root)
	if e != nil {
		t.Fatal(e)
	}
	p := domain.Profile{ID: "01234567890123456789012345678901", Name: "test", Protocol: "shadowsocks", Host: "127.0.0.1", Port: 8388, Method: "aes-256-gcm", Password: "synthetic"}
	if e = s.Update(func(v *domain.Settings) error { v.Profiles = append(v.Profiles, p); v.Selected = p.ID; return nil }); e != nil {
		t.Fatal(e)
	}
	v, e := s.Load()
	if e != nil || len(v.Profiles) != 1 || v.Profiles[0].Password != "synthetic" {
		t.Fatal("storage round trip failed", e)
	}
	for _, path := range []string{s.Paths.Config, s.Paths.State, s.Paths.Runtime, s.Paths.Cache} {
		st, e := os.Stat(path)
		if e != nil || st.Mode().Perm() != 0700 {
			t.Fatal("directory permissions")
		}
	}
	path := filepath.Join(s.Paths.Config, "profiles.json")
	st, e := os.Stat(path)
	if e != nil || st.Mode().Perm() != 0600 {
		t.Fatal("file permissions")
	}
	before, _ := os.ReadFile(path)
	e = s.Update(func(v *domain.Settings) error { v.Profiles = nil; return domain.Fail(2, "synthetic failure", "") })
	after, _ := os.ReadFile(path)
	if e == nil || string(before) != string(after) {
		t.Fatal("failed transaction changed storage")
	}
	files, _ := os.ReadDir(s.Paths.Config)
	for _, f := range files {
		if f.Name() != "profiles.json" && f.Name() != "profiles.lock" {
			t.Fatal("temporary file leaked")
		}
	}
	lock, e := Acquire(filepath.Join(s.Paths.Config, "profiles.lock"))
	if e != nil {
		t.Fatal(e)
	}
	if e = s.Update(func(*domain.Settings) error { return nil }); e == nil {
		t.Fatal("concurrent lock accepted")
	}
	Release(lock)
	if e = os.Chmod(path, 0644); e != nil {
		t.Fatal(e)
	}
	if _, e = s.Load(); e == nil {
		t.Fatal("public secret file accepted")
	}
}
func TestSymlinksAndUnknownSchema(t *testing.T) {
	root := t.TempDir()
	os.Chmod(root, 0700)
	s, e := New(root)
	if e != nil {
		t.Fatal(e)
	}
	path := filepath.Join(s.Paths.Config, "profiles.json")
	if e = AtomicJSON(path, domain.Settings{SchemaVersion: 99}); e != nil {
		t.Fatal(e)
	}
	if _, e = s.Load(); e == nil {
		t.Fatal("unknown schema accepted")
	}
	link := filepath.Join(root, "link")
	os.Symlink(s.Paths.Config, link)
	if _, e = New(filepath.Join(link, "nested")); e == nil {
		t.Fatal("symlink ancestor accepted")
	}
	os.Remove(path)
	os.Symlink(filepath.Join(root, "target"), path)
	if _, e = s.Load(); e == nil {
		t.Fatal("symlink secret file accepted")
	}
}
