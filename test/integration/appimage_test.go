//go:build linux

package integration

import (
	"bytes"
	"encoding/json"
	"os"
	"os/exec"
	"path/filepath"
	"strings"
	"testing"
)

func TestAppImageStreamsExitAndPersistentData(t *testing.T) {
	path, tag := os.Getenv("TUNNELCTL_TEST_APPIMAGE"), os.Getenv("TUNNELCTL_RELEASE_TAG")
	if path == "" || tag == "" {
		t.Skip("set explicit AppImage path and tag; no private sample is read")
	}
	dir := t.TempDir()
	run := func(input string, args ...string) ([]byte, []byte, error) {
		c := exec.Command(path, append([]string{"--appimage-extract-and-run"}, args...)...)
		c.Dir = dir
		c.Stdin = strings.NewReader(input)
		var stdout, stderr bytes.Buffer
		c.Stdout = &stdout
		c.Stderr = &stderr
		err := c.Run()
		return stdout.Bytes(), stderr.Bytes(), err
	}
	b, _, e := run("", "version", "--json")
	if e != nil {
		t.Fatal(e)
	}
	var info struct{ Data struct{ Tag string } }
	if json.Unmarshal(b, &info) != nil || info.Data.Tag != tag {
		t.Fatal("AppImage version mismatch")
	}
	data := filepath.Join(dir, "data")
	if _, _, e = run("", "--data-dir", data, "init"); e != nil {
		t.Fatal(e)
	}
	secret := "synthetic-appimage-only"
	b, stderr, e := run("ss://aes-256-gcm:"+secret+"@127.0.0.1:8388#appimage-preserved", "--data-dir", data, "profile", "import", "--stdin", "--json")
	if e != nil || !json.Valid(b) || bytes.Contains(append(b, stderr...), []byte(secret)) {
		t.Fatal("stdin/JSON/redaction failure", e)
	}
	b, _, e = run("", "--data-dir", data, "profile", "list", "--json")
	if e != nil || !bytes.Contains(b, []byte("appimage-preserved")) {
		t.Fatal("AppImage data did not survive mount lifecycle")
	}
	b, stderr, e = run("", "not-a-real-command")
	if e == nil || len(stderr) == 0 || len(b) != 0 {
		t.Fatal("exit status or stdout/stderr forwarding broken")
	}
	for _, command := range []string{"run", "start", "exec"} {
		args := []string{"--data-dir", data, "--json", command}
		if command == "exec" {
			args = append(args, "--", "true")
		}
		b, _, e = run("", args...)
		x, ok := e.(*exec.ExitError)
		if !ok || x.ExitCode() != 3 || !bytes.Contains(b, []byte("Transient AppImage extraction")) {
			t.Fatal("transient session execution not explicitly refused", command, e)
		}
	}
	// Signal, duration, crash and real proxy traffic run against stable extracted
	// payload in packaging/appimage/test.sh; this retains the upstream regression.
}
