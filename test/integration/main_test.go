//go:build linux

package integration

import (
	"fmt"
	"os"
	"path/filepath"
	"strings"
	"testing"
)

func TestMain(m *testing.M) {
	// Detached supervisors intentionally suppress raw diagnostics. Capture race
	// reports privately so a late detached-child race cannot silently pass CI.
	reports, e := os.MkdirTemp("", "tunnelctl-child-race-")
	if e != nil {
		fmt.Fprintln(os.Stderr, "Cannot create private child race-report storage.")
		os.Exit(1)
	}
	if e = os.Setenv("GORACE", strings.TrimSpace(os.Getenv("GORACE")+" log_path="+filepath.Join(reports, "report")+" halt_on_error=1")); e != nil {
		os.RemoveAll(reports)
		os.Exit(1)
	}
	code := m.Run()
	entries, e := os.ReadDir(reports)
	if e != nil {
		fmt.Fprintln(os.Stderr, "Cannot inspect child race-report storage.")
		code = 1
	}
	if len(entries) > 0 {
		fmt.Fprintln(os.Stderr, "Child-process race evidence detected; private reports retained at", reports)
		code = 1
	} else if e = os.RemoveAll(reports); e != nil {
		fmt.Fprintln(os.Stderr, "Cannot clean private child race-report storage.")
		code = 1
	}
	os.Exit(code)
}
