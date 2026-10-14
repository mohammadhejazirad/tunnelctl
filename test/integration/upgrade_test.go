//go:build linux

package integration

import (
	"encoding/json"
	"fmt"
	"net/http"
	"net/http/httptest"
	"os"
	"path/filepath"
	"testing"
	"tunnelctl/internal/domain"
)

func TestUpgradeFromBaselineAndDowngradeRefusal(t *testing.T) {
	previous := os.Getenv("TUNNELCTL_PREVIOUS_BIN")
	if previous == "" {
		t.Skip("set baseline 0.1 development binary to verify actual executable upgrade")
	}
	f := newFixture(t, "aes-256-gcm")
	current := *f
	current.root = private(t)
	legacy := current
	legacy.bin = previous
	legacy.command("", 0, "init")
	legacy.command(fmt.Sprintf("ss://aes-256-gcm:synthetic-upstream-password@127.0.0.1:%d#baseline-profile", f.upstream), 0, "profile", "import", "--stdin")
	path := filepath.Join(current.root, "config", "profiles.json")
	before, e := os.ReadFile(path)
	if e != nil {
		t.Fatal(e)
	}
	var old domain.Settings
	if e = json.Unmarshal(before, &old); e != nil || old.SchemaVersion != 1 || len(old.Profiles) != 1 {
		t.Fatal("baseline did not create schema 1")
	}
	current.command("", 0, "profile", "list")
	current.command("https://example.invalid/synthetic-feed", 0, "subscription", "add", "--name", "new-version-source", "--url-stdin")
	upgraded, e := os.ReadFile(path)
	if e != nil {
		t.Fatal(e)
	}
	var next domain.Settings
	if e = json.Unmarshal(upgraded, &next); e != nil || next.SchemaVersion != 2 || next.Selected != old.Selected || next.Profiles[0].ID != old.Profiles[0].ID {
		t.Fatal("upgrade lost identity/selection")
	}
	backup, e := os.ReadFile(filepath.Join(current.root, "config", "profiles.schema1.backup.json"))
	if e != nil || string(backup) != string(before) {
		t.Fatal("schema upgrade backup missing")
	}
	// The historical schema-1 decoder rejects newly added fields before its
	// schema check, so this exact baseline reports invalid-storage code 2, not 3.
	// Both refusal and byte-for-byte preservation are required here.
	legacy.command("", 2, "profile", "list")
	after, e := os.ReadFile(path)
	if e != nil || string(after) != string(upgraded) {
		t.Fatal("old binary damaged newer schema")
	}
	target := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) { fmt.Fprint(w, "upgraded-proxy-response") }))
	defer target.Close()
	port := freePort(t)
	run := current.start(port, "--duration", "30s")
	current.ready(port)
	body, e := request(t, port, target.URL, false)
	if e != nil || body != "upgraded-proxy-response" {
		t.Fatal("upgraded executable cannot connect", e)
	}
	current.command("", 0, "stop")
	run.wait(t, 0)
}
