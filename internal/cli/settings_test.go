//go:build linux

package cli

import (
	"bytes"
	"encoding/json"
	"github.com/spf13/cobra"
	"os"
	"path/filepath"
	"strings"
	"testing"
	"time"
	"tunnelctl/internal/app"
	"tunnelctl/internal/domain"
)

func TestDurableCLISettingsAndMetadata(t *testing.T) {
	root := t.TempDir()
	os.Chmod(root, 0700)
	call := func(code int, input string, args ...string) string {
		t.Helper()
		var out, diag bytes.Buffer
		actual := Execute(append([]string{"--data-dir", root, "--json"}, args...), strings.NewReader(input), &out, &diag)
		if actual != code || !json.Valid(out.Bytes()) || diag.Len() != 0 || strings.Contains(out.String(), "synthetic-secret") {
			t.Fatalf("unexpected CLI result: %d %s", actual, out.String())
		}
		return out.String()
	}
	call(0, "", "init")
	call(0, "ss://aes-256-gcm:synthetic-secret@127.0.0.1:8388#persisted", "profile", "import", "--stdin")
	call(0, "https://example.invalid/feed?token=synthetic-secret", "subscription", "add", "--name", "durable", "--url-stdin")
	call(0, "", "profile", "metadata", "persisted", "--tags", "work,private", "--favorite")
	call(0, "", "settings", "set", "listener-port", "9074")
	call(0, "", "settings", "set", "duration", "5m")
	call(0, "", "settings", "set", "color", "false")
	t.Chdir(t.TempDir())
	if out := call(0, "", "settings", "show"); !strings.Contains(out, `"listener_port":9074`) {
		t.Fatal("settings lost after different CWD")
	}
	if out := call(0, "", "profile", "show", "persisted"); !strings.Contains(out, `"favorite":true`) || !strings.Contains(out, `"work"`) {
		t.Fatal("metadata not durable")
	}
	call(0, "", "init")
	s, e := app.New(root, "/missing/core", time.Second)
	if e != nil {
		t.Fatal(e)
	}
	v, e := s.Store.Load()
	if e != nil || len(v.Profiles) != 1 || v.Profiles[0].Password != "synthetic-secret" || v.Selected != v.Profiles[0].ID || len(v.Subscriptions) != 1 || v.Subscriptions[0].URL != "https://example.invalid/feed?token=synthetic-secret" || v.Preferences.ListenerPort != 9074 {
		t.Fatal("reinitialization/reload lost data")
	}
	before, _ := os.ReadFile(filepath.Join(s.Paths().Config, "profiles.json"))
	call(3, "", "settings", "set", "preferred-mode", "tun")
	call(2, "", "settings", "set", "probe-target", "https://user:synthetic-secret@example.invalid")
	after, _ := os.ReadFile(filepath.Join(s.Paths().Config, "profiles.json"))
	if !bytes.Equal(before, after) {
		t.Fatal("rejected settings changed snapshot")
	}
	call(0, "", "paths")
	if out := call(0, "", "status"); !strings.Contains(out, `"process_running":false`) {
		t.Fatal("saved selection caused auto-connect")
	}
	call(2, "", "settings", "reset")
	call(0, "", "settings", "reset", "--yes")
	v, e = s.Store.Load()
	if e != nil || len(v.Profiles) != 1 || len(v.Subscriptions) != 1 || v.Selected == "" || v.Preferences != domain.DefaultPreferences() {
		t.Fatal("preferences reset removed user data")
	}
	path := filepath.Join(s.Paths().Config, "profiles.json")
	os.Chmod(path, 0644)
	defer os.Chmod(path, 0600)
	call(4, "", "settings", "set", "listener-port", "9075")
	// Paths remain diagnosable even when the authoritative snapshot cannot be read.
	call(0, "", "paths")
	call(0, "", "status")
	call(0, "", "stop")
	os.Chmod(path, 0600)
	if e := os.WriteFile(path, []byte("broken synthetic snapshot"), 0600); e != nil {
		t.Fatal(e)
	}
	call(2, "", "settings", "show")
	call(0, "", "paths")
	call(0, "", "status")
	call(0, "", "stop")
}

func TestSavedDefaultsAndExplicitSessionOverrides(t *testing.T) {
	root := t.TempDir()
	os.Chmod(root, 0700)
	s, e := app.New(root, "/missing/core", time.Second)
	if e != nil {
		t.Fatal(e)
	}
	_, e = s.EditSettings(nil, func(p *domain.Preferences) error {
		p.ListenerPort = 9076
		p.Duration = "3m"
		p.AuthFile = "/private/auth.json"
		return nil
	})
	if e != nil {
		t.Fatal(e)
	}
	c := &cobra.Command{}
	c.Flags().Uint("listen-port", 2080, "")
	c.Flags().Duration("duration", 0, "")
	c.Flags().String("proxy-auth-file", "", "")
	c.Flags().String("mode", "proxy", "")
	port, d, auth, mode, e := sessionDefaults(c, s, 2080, 0, "", "proxy")
	if e != nil || port != 9076 || d != 3*time.Minute || auth != "/private/auth.json" || mode != "proxy" {
		t.Fatal("persisted defaults ignored")
	}
	c.Flags().Set("listen-port", "9080")
	c.Flags().Set("duration", "0s")
	c.Flags().Set("proxy-auth-file", "")
	port, d, auth, _, e = sessionDefaults(c, s, 9080, 0, "", "proxy")
	if e != nil || port != 9080 || d != 0 || auth != "" {
		t.Fatal("explicit overrides ignored")
	}
	saved, _ := s.Settings()
	if saved.Preferences.ListenerPort != 9076 || saved.Preferences.Duration != "3m" {
		t.Fatal("per-run overrides modified persistent settings")
	}
}
