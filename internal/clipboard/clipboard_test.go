//go:build linux

package clipboard

import (
	"context"
	"os"
	"path/filepath"
	"testing"
)

func TestReadUsesAvailableClipboardCommand(t *testing.T) {
	dir := t.TempDir()
	tool := filepath.Join(dir, "wl-paste")
	if e := os.WriteFile(tool, []byte("#!/bin/sh\nprintf '%s' 'vless://synthetic'\n"), 0700); e != nil {
		t.Fatal(e)
	}
	t.Setenv("PATH", dir)
	t.Setenv("WAYLAND_DISPLAY", "synthetic")
	value, e := Read(context.Background())
	if e != nil || value != "vless://synthetic" {
		t.Fatal("clipboard command was not read", e)
	}
}
