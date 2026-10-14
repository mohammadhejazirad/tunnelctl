//go:build linux

package platform

import (
	"os"
	"path/filepath"
	"testing"
)

func TestRejectSetIDExecutable(t *testing.T) {
	path := filepath.Join(t.TempDir(), "synthetic-executable")
	if e := os.WriteFile(path, []byte("not executed"), 0700); e != nil {
		t.Fatal(e)
	}
	f, e := os.Open(path)
	if e != nil {
		t.Fatal(e)
	}
	defer f.Close()
	if !UnprivilegedExecutable(f) {
		t.Fatal("ordinary executable refused")
	}
	if e = os.Chmod(path, os.ModeSetuid|0700); e != nil {
		t.Fatal(e)
	}
	if UnprivilegedExecutable(f) {
		t.Fatal("set-ID executable accepted for proxy")
	}
}
