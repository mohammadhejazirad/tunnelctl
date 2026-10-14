//go:build linux

package cli

import (
	"bytes"
	"encoding/json"
	"os"
	"path/filepath"
	"strings"
	"testing"
)

func TestCLIJSONAndRedaction(t *testing.T) {
	root := t.TempDir()
	os.Chmod(root, 0700)
	var out, diagnostics bytes.Buffer
	run := func(args []string, input string) (int, envelope) {
		out.Reset()
		diagnostics.Reset()
		code := Execute(append([]string{"--data-dir", root, "--json"}, args...), strings.NewReader(input), &out, &diagnostics)
		var result envelope
		if e := json.Unmarshal(out.Bytes(), &result); e != nil {
			t.Fatalf("invalid JSON: %s (%v)", out.String(), e)
		}
		if diagnostics.Len() != 0 || strings.Contains(out.String(), "synthetic-secret") || strings.Contains(out.String(), "\x1b") {
			t.Fatal("unsafe output")
		}
		return code, result
	}
	if code, v := run([]string{"init"}, ""); code != 0 || !v.OK {
		t.Fatal("init failed")
	}
	if code, v := run([]string{"profile", "import", "--stdin"}, "ss://aes-256-gcm:synthetic-secret@127.0.0.1:8388#%1B%5D52%3Bc%3Binjected%07safe"); code != 0 || !v.OK {
		t.Fatal("import failed", v.Error)
	}
	if code, _ := run([]string{"profile", "list"}, ""); code != 0 || !strings.Contains(out.String(), "safe") || strings.Contains(out.String(), "injected") {
		t.Fatal("name sanitization failed")
	}
	if code, _ := run([]string{"profile", "show", "safe", "--redacted"}, ""); code != 0 || !strings.Contains(out.String(), "[REDACTED]") {
		t.Fatal("show failed")
	}
	if code, _ := run([]string{"profile", "select", "safe"}, ""); code != 0 {
		t.Fatal("select failed")
	}
	if code, _ := run([]string{"profile", "import", "--stdin"}, "ss://aes-256-gcm:synthetic-secret@127.0.0.1:8388"); code != 6 {
		t.Fatal("duplicate identity accepted")
	}
	if code, _ := run([]string{"profile", "import"}, ""); code != 2 {
		t.Fatal("invalid input accepted")
	}
	file := filepath.Join(root, "synthetic-input.txt")
	if e := os.WriteFile(file, []byte("ss://aes-128-gcm:synthetic-secret@127.0.0.1:8389#file-import"), 0600); e != nil {
		t.Fatal(e)
	}
	if code, _ := run([]string{"profile", "import", "--file", file}, ""); code != 0 {
		t.Fatal("private file import failed")
	}
	os.Chmod(file, 0644)
	if code, _ := run([]string{"profile", "import", "--file", file}, ""); code != 4 {
		t.Fatal("public input file accepted")
	}
	if code, _ := run([]string{"--synthetic-secret"}, ""); code != 2 {
		t.Fatal("invalid flag accepted")
	}
	if code, _ := run([]string{"status"}, ""); code != 0 {
		t.Fatal("idle status failed")
	}
	if code, _ := run([]string{"stop"}, ""); code != 0 {
		t.Fatal("idempotent stop failed")
	}
	if code, _ := run([]string{"run", "--mode", "tun"}, ""); code != 3 {
		t.Fatal("TUN should be unsupported")
	}
}
