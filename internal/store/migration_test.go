//go:build linux

package store

import (
	"os"
	"path/filepath"
	"testing"
	"tunnelctl/internal/domain"
)

func TestSchemaOneMigrationAndDowngradeBoundary(t *testing.T) {
	root := t.TempDir()
	os.Chmod(root, 0700)
	s, e := New(root)
	if e != nil {
		t.Fatal(e)
	}
	path := filepath.Join(s.Paths.Config, "profiles.json")
	if e = AtomicJSON(path, domain.Settings{SchemaVersion: 1, Profiles: []domain.Profile{}}); e != nil {
		t.Fatal(e)
	}
	before, _ := os.ReadFile(path)
	if e = s.Update(func(*domain.Settings) error { return domain.Fail(2, "synthetic failure", "") }); e == nil {
		t.Fatal("expected failed mutation")
	}
	after, _ := os.ReadFile(path)
	if string(before) != string(after) {
		t.Fatal("failed mutation migrated or changed data")
	}
	if e = s.Update(func(*domain.Settings) error { return nil }); e != nil {
		t.Fatal(e)
	}
	v, e := s.Load()
	if e != nil || v.SchemaVersion != 2 {
		t.Fatal("schema migration failed", e)
	}
	backup := filepath.Join(s.Paths.Config, "profiles.schema1.backup.json")
	b, e := os.ReadFile(backup)
	if e != nil || string(b) != string(before) {
		t.Fatal("original schema backup not preserved", e)
	}
	if e = AtomicJSON(path, domain.Settings{SchemaVersion: 99}); e != nil {
		t.Fatal(e)
	}
	before, _ = os.ReadFile(path)
	if e = s.Update(func(*domain.Settings) error { return nil }); e == nil {
		t.Fatal("unknown newer schema changed")
	}
	after, _ = os.ReadFile(path)
	if string(before) != string(after) {
		t.Fatal("incompatible downgrade damaged data")
	}
}
