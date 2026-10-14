//go:build linux

package integration

import (
	"bytes"
	"encoding/json"
	"fmt"
	"io"
	"net/http"
	"net/http/httptest"
	"os"
	"os/exec"
	"path/filepath"
	"strconv"
	"testing"
	"time"
	"tunnelctl/internal/app"
	"tunnelctl/internal/domain"
)

func TestDurableDefaultsReconnectAndActiveSessionIsolation(t *testing.T) {
	f := newFixture(t, "aes-256-gcm")
	port := freePort(t)
	for _, kv := range [][2]string{{"listener-port", strconv.Itoa(int(port))}, {"duration", "30s"}, {"core-path", f.core}, {"timeout", "3s"}} {
		f.command("", 0, "settings", "set", kv[0], kv[1])
	}
	f.command("https://example.invalid/feed?token=synthetic-source-secret", 0, "subscription", "add", "--name", "persistent-source", "--url-stdin")
	f.command("", 0, "profile", "metadata", "controlled", "--tags", "work", "--favorite")
	// Separate process, different CWD, no core/listener/duration flags: use stored defaults.
	r := &running{cmd: exec.Command(f.bin, "--data-dir", f.root, "--json", "run"), done: make(chan error, 1)}
	r.cmd.Dir = private(t)
	r.cmd.Stdout = &r.out
	r.cmd.Stderr = io.Discard
	if e := r.cmd.Start(); e != nil {
		t.Fatal(e)
	}
	go func() { r.done <- r.cmd.Wait() }()
	defer func() {
		f.command("", 0, "stop")
		if !r.consumed {
			r.cmd.Process.Kill()
			<-r.done
		}
	}()
	f.ready(port)
	initial := f.state()
	if initial.Deadline == nil || initial.ProfileID == "" {
		t.Fatal("saved defaults not applied")
	}
	f.command("", 0, "settings", "set", "listener-port", strconv.Itoa(int(freePort(t))))
	f.command("", 0, "settings", "set", "duration", "1s")
	f.command("", 0, "settings", "set", "core-path", "/missing/future-core")
	f.command("", 0, "init")
	active := f.state()
	if active.ID != initial.ID || active.Listener != initial.Listener || !active.ListenerReady || active.Deadline == nil || !active.Deadline.Equal(*initial.Deadline) {
		t.Fatal("saved defaults changed active session")
	}
	target := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) { fmt.Fprint(w, "persisted-profile-real-request") }))
	defer target.Close()
	body, e := request(t, port, target.URL, false)
	if e != nil || body != "persisted-profile-real-request" {
		t.Fatal("reloaded profile did not connect", e)
	}
	f.command("", 0, "stop")
	r.wait(t, 0)
	// Shutdown/reopen does not auto-connect, even with selection and mode persisted.
	s, e := app.New(f.root, f.core, time.Second)
	if e != nil {
		t.Fatal(e)
	}
	v, e := s.Store.Load()
	if e != nil || len(v.Subscriptions) != 1 || v.Subscriptions[0].URL != "https://example.invalid/feed?token=synthetic-source-secret" || len(v.Profiles) != 1 || !v.Profiles[0].Favorite || v.Selected != initial.ProfileID {
		t.Fatal("reopen lost private data", e)
	}
	if state := f.state(); state.ProcessRunning || state.ListenerReady {
		t.Fatal("reopen auto-connected")
	}
}

func TestUpgradeFromSchemaTwoExecutable(t *testing.T) {
	previous := os.Getenv("TUNNELCTL_SCHEMA2_BIN")
	if previous == "" {
		t.Skip("set TUNNELCTL_SCHEMA2_BIN to a previous schema-2 executable")
	}
	f := newFixture(t, "aes-256-gcm")
	current := *f
	current.root = private(t)
	legacy := current
	legacy.bin = previous
	legacy.command("", 0, "init")
	legacy.command(fmt.Sprintf("ss://aes-256-gcm:synthetic-upstream-password@127.0.0.1:%d#previous", f.upstream), 0, "profile", "import", "--stdin")
	legacy.command("https://example.invalid/feed?token=synthetic-source-secret", 0, "subscription", "add", "--name", "old-source", "--url-stdin")
	path := filepath.Join(current.root, "config", "profiles.json")
	before, e := os.ReadFile(path)
	if e != nil {
		t.Fatal(e)
	}
	var old domain.Settings
	if json.Unmarshal(before, &old) != nil || old.SchemaVersion != 2 {
		t.Fatal("previous executable did not create schema 2")
	}
	current.command("", 0, "settings", "set", "duration", "5m")
	after, e := os.ReadFile(path)
	if e != nil {
		t.Fatal(e)
	}
	var next domain.Settings
	if json.Unmarshal(after, &next) != nil || next.SchemaVersion != 3 || next.Selected != old.Selected || next.Profiles[0].Password != old.Profiles[0].Password || next.Subscriptions[0].URL != old.Subscriptions[0].URL {
		t.Fatal("actual upgrade lost private data")
	}
	backup, e := os.ReadFile(filepath.Join(current.root, "config", "profiles.schema2.backup.json"))
	if e != nil || !bytes.Equal(backup, before) {
		t.Fatal("actual upgrade backup changed")
	}
	legacy.command("", 2, "init")
	unchanged, _ := os.ReadFile(path)
	if !bytes.Equal(after, unchanged) {
		t.Fatal("old binary damaged newer data")
	}
}
