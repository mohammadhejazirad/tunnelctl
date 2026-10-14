package main

import (
	"strings"
	"testing"
)

func TestSensitivePathsAndValues(t *testing.T) {
	for _, path := range []string{"links_and_subs.txt", "docs/result.private.json", "private-results/report.txt", "bin/tunnelctl", ".env.local", "config/profiles.schema1.backup.json", "state/events.json", "secret.pem"} {
		if inspect(path, []byte("ordinary")) == nil {
			t.Fatal("sensitive path accepted", path)
		}
	}
	token := "ghp_" + strings.Repeat("a", 36)
	if e := inspect("README.md", []byte(token)); e == nil || strings.Contains(e.Error(), token) {
		t.Fatal("credential detected incorrectly or echoed")
	}
	if inspect("notes.md", []byte("ss"+"://aes-256-gcm:secret@host:1234")) == nil {
		t.Fatal("private link accepted")
	}
	if inspect("fixture_test.go", []byte("ss"+"://synthetic-only")) != nil {
		t.Fatal("synthetic fixture refused")
	}
}
