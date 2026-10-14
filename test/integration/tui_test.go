//go:build linux

package integration

import (
	"bytes"
	"errors"
	"fmt"
	"net/http"
	"net/http/httptest"
	"os"
	"os/exec"
	"strconv"
	"strings"
	"testing"
	"time"
	"tunnelctl/internal/app"
	"tunnelctl/internal/domain"
)

func TestTUIPseudoTerminal(t *testing.T) {
	for _, explicit := range []bool{true, false} {
		t.Run(fmt.Sprintf("explicit-%t", explicit), func(t *testing.T) { tuiSmoke(t, explicit) })
	}
}
func tuiSmoke(t *testing.T, explicit bool) {
	f := newFixture(t, "aes-256-gcm")
	script, e := exec.LookPath("script")
	if e != nil {
		t.Skip("util-linux script required for PTY smoke test")
	}
	// Only generated local fixture paths enter this test-owned shell command.
	command := fmt.Sprintf("%q --data-dir %q --core %q --no-color", f.bin, f.root, f.core)
	if explicit {
		command += " tui"
	}
	cmd := exec.Command(script, "-qefc", command, "/dev/null")
	cmd.Env = append(os.Environ(), "TERM=xterm-256color", "NO_COLOR=1")
	input, e := cmd.StdinPipe()
	if e != nil {
		t.Fatal(e)
	}
	var output bytes.Buffer
	cmd.Stdout = &output
	cmd.Stderr = &output
	if e = cmd.Start(); e != nil {
		t.Fatal(e)
	}
	done := make(chan error, 1)
	go func() { done <- cmd.Wait() }()
	time.Sleep(1500 * time.Millisecond)
	input.Write([]byte("2?1q"))
	input.Close()
	select {
	case e = <-done:
		if e != nil {
			t.Fatal("TUI PTY failed", e, output.String())
		}
	case <-time.After(8 * time.Second):
		cmd.Process.Kill()
		<-done
		t.Fatal("TUI quit timed out")
	}
	if !strings.Contains(output.String(), "tunnelctl") || strings.Contains(output.String(), "synthetic-upstream-password") {
		t.Fatal("TUI did not render safely")
	}
}

func TestTUIMenuConnectDetachedExitAndConfirmedStop(t *testing.T) {
	f := newFixture(t, "aes-256-gcm")
	script, e := exec.LookPath("script")
	if e != nil {
		t.Skip("util-linux script required")
	}
	s, e := app.New(f.root, f.core, 3*time.Second)
	if e != nil {
		t.Fatal(e)
	}
	command := fmt.Sprintf("%q --data-dir %q --core %q --no-color", f.bin, f.root, f.core)
	cmd := exec.Command(script, "-qefc", command, "/dev/null")
	cmd.Env = append(os.Environ(), "TERM=xterm-256color", "NO_COLOR=1")
	input, e := cmd.StdinPipe()
	if e != nil {
		t.Fatal(e)
	}
	var output bytes.Buffer
	cmd.Stdout = &output
	cmd.Stderr = &output
	if e = cmd.Start(); e != nil {
		t.Fatal(e)
	}
	done := make(chan error, 1)
	go func() { done <- cmd.Wait() }()
	finished := false
	defer func() {
		if !finished {
			cmd.Process.Kill()
			<-done
		}
		s.Stop()
		input.Close()
	}()
	time.Sleep(1200 * time.Millisecond)
	// Main menu -> random unprivileged port -> explicit start confirmation.
	port := freePort(t)
	input.Write([]byte("s\x1b[B\r"))
	time.Sleep(100 * time.Millisecond)
	input.Write([]byte("\x15" + strconv.Itoa(int(port)) + "\r"))
	time.Sleep(100 * time.Millisecond)
	input.Write([]byte("\x1b[B\x1b[B\x1b[B\r"))
	time.Sleep(150 * time.Millisecond)
	input.Write([]byte("\r"))
	waitFor(t, func() bool { state, e := s.Status(); return e == nil && state.ListenerReady }, "menu-started detached proxy")
	state, e := s.Status()
	if e != nil {
		t.Fatal(e)
	}
	target := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) { fmt.Fprint(w, "menu-real-proxy") }))
	defer target.Close()
	body, e := request(t, port, target.URL, false)
	if e != nil || body != "menu-real-proxy" {
		t.Fatal("menu proxy request failed", e)
	}
	input.Write([]byte("q"))
	time.Sleep(200 * time.Millisecond)
	input.Write([]byte("\r"))
	input.Close()
	select {
	case e = <-done:
		finished = true
		if e != nil {
			t.Fatal("menu exit failed", e)
		}
	case <-time.After(8 * time.Second):
		t.Fatal("menu exit timed out")
	}
	after, e := s.Status()
	if e != nil || after.ID != state.ID || !after.ListenerReady {
		t.Fatal("leaving menu stopped detached session")
	}
	if !strings.Contains(output.String(), "Main menu") || !strings.Contains(output.String(), "Leave running and exit") || strings.Contains(output.String(), "synthetic-upstream-password") {
		t.Fatal("menu did not render safe explicit lifetime")
	}
	e = s.StopSession("stale-confirmation")
	var de *domain.Error
	if !errors.As(e, &de) || de.Code != 6 {
		t.Fatal("stale stop confirmation accepted")
	}
	if e = s.StopSession(""); !errors.As(e, &de) || de.Code != 6 {
		t.Fatal("empty confirmation became a wildcard")
	}
	after, e = s.Status()
	if e != nil || after.ID != state.ID || !after.ListenerReady {
		t.Fatal("stale confirmation stopped active session")
	}
	if e = s.StopSession(state.ID); e != nil {
		t.Fatal(e)
	}
	after, e = s.Status()
	if e != nil || after.State != "idle" {
		t.Fatal("menu-created session not cleaned")
	}
}
