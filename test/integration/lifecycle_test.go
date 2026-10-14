//go:build linux

package integration

import (
	"bytes"
	"encoding/json"
	"fmt"
	"net/http"
	"net/http/httptest"
	"os/exec"
	"strings"
	"testing"
	"tunnelctl/internal/domain"
)

func TestDetachedExec(t *testing.T) {
	f := newFixture(t, "aes-256-gcm")
	target := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) { fmt.Fprint(w, "isolated-child-response") }))
	defer target.Close()
	port := freePort(t)
	r := f.command("", 0, "start", "--profile", "controlled", "--listen-port", fmt.Sprint(port), "--duration", "30s")
	var state domain.Session
	if e := json.Unmarshal(r.Data, &state); e != nil {
		t.Fatal(e)
	}
	f.ready(port)
	t.Cleanup(func() { exec.Command(f.bin, f.args("stop")...).Run() })
	body, e := request(t, port, target.URL, false)
	if e != nil || body != "isolated-child-response" {
		t.Fatal("detached proxy failed", e)
	}
	cmd := exec.Command(f.bin, f.args("exec", "--", "curl", "--silent", "--show-error", "--fail", target.URL)...)
	var out, diagnostics bytes.Buffer
	cmd.Stdout = &out
	cmd.Stderr = &diagnostics
	if e = cmd.Run(); e != nil {
		t.Fatal("isolated child execution failed", e, out.String(), diagnostics.String())
	}
	if !strings.Contains(diagnostics.String(), "isolated-child-response") {
		t.Fatal("child did not complete proxy request")
	}
	if f.state().ID != state.ID {
		t.Fatal("exec replaced the unrelated detached session")
	}
	f.command("", 23, "exec", "--", "sh", "-c", "exit 23")
	if f.state().ID != state.ID {
		t.Fatal("failed child stopped unrelated session")
	}
	f.command("", 0, "logs")
	f.command("", 0, "stop")
	f.command("", 0, "stop")
	if f.state().State != "idle" {
		t.Fatal("detached cleanup failed")
	}
}
