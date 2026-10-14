//go:build linux

package integration

import (
	"bytes"
	"fmt"
	"os"
	"os/exec"
	"strings"
	"testing"
	"time"
)

func TestTUIPseudoTerminal(t *testing.T) {
	f := newFixture(t, "aes-256-gcm")
	script, e := exec.LookPath("script")
	if e != nil {
		t.Skip("util-linux script required for PTY smoke test")
	}
	// Only generated local fixture paths enter this test-owned shell command.
	command := fmt.Sprintf("%q --data-dir %q --core %q --no-color tui", f.bin, f.root, f.core)
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
