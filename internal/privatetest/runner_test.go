//go:build linux

package privatetest

import (
	"context"
	"crypto/x509"
	"encoding/json"
	"encoding/pem"
	"fmt"
	"net/http"
	"net/http/httptest"
	"os"
	"path/filepath"
	"strings"
	"testing"
	"time"
	"tunnelctl/internal/app"
	"tunnelctl/internal/domain"
)

func TestOptInBoundsAndAnonymousOutput(t *testing.T) {
	if _, e := Run(context.Background(), Options{}); e == nil {
		t.Fatal("implicit test activation allowed")
	}
	// Only synthetic data, outside the source tree. No public test opens real samples.
	path := filepath.Join(t.TempDir(), "synthetic-input.txt")
	secret := "synthetic-token-never-display"
	os.WriteFile(path, []byte("# comment\nvless://"+secret+"@example.invalid:443\nnot-a-link\nmore\n"), 0600)
	o := Options{File: path, MaxSamples: 2, MaxProfiles: 1, MaxConnections: 0, Timeout: time.Second}
	r, e := Run(context.Background(), o)
	if e != nil || r.Cleanup != "passed" || !r.Truncated || r.Passed || len(r.Samples) != 2 {
		t.Fatal("bounded anonymous run failed")
	}
	encoded, e := json.Marshal(r)
	if e != nil || strings.Contains(string(encoded), secret) || strings.Contains(string(encoded), "example.invalid") || strings.Contains(string(encoded), path) {
		t.Fatal("private data leaked")
	}
	if r.Samples[0].ID != "sample-001" || r.Samples[0].Stages[1].Category != "unsupported" {
		t.Fatal("anonymous category missing")
	}
	os.WriteFile(path, []byte(strings.Repeat("a", 64*1024+2)), 0600)
	if _, e = Run(context.Background(), o); e == nil {
		t.Fatal("oversized line accepted")
	}
	os.Remove(path)
	os.Symlink("/dev/null", path)
	if _, e = input(path); e == nil {
		t.Fatal("symlink input allowed")
	}
}
func TestCategoriesNeverIncludeRawErrors(t *testing.T) {
	e := domain.NetworkFail("tls", "TLS validation failed.")
	s := result("fetch", e)
	if s.Category != "tls" || s.Result != "failed" {
		t.Fatal("typed category not preserved")
	}
}

func TestRefetchChecksStableSelectionAcrossSourcesWithSameConnection(t *testing.T) {
	server := httptest.NewTLSServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		fmt.Fprint(w, "ss://aes-256-gcm:synthetic-refetch@127.0.0.1:8388#synthetic")
	}))
	defer server.Close()
	root := t.TempDir()
	os.Chmod(root, 0700)
	s, e := app.New(root, "/not-used", time.Second)
	if e != nil {
		t.Fatal(e)
	}
	cert, _ := x509.ParseCertificate(server.TLS.Certificates[0].Certificate[0])
	ca := filepath.Join(root, "ca.pem")
	os.WriteFile(ca, pem.EncodeToMemory(&pem.Block{Type: "CERTIFICATE", Bytes: cert.Raw}), 0600)
	var sources []string
	for _, name := range []string{"one", "two"} {
		source, e := s.Subscriptions().Add(name, strings.NewReader(server.URL+"/"+name), []string{"127.0.0.1/32"}, ca)
		if e != nil {
			t.Fatal(e)
		}
		if _, e = s.Subscriptions().Update(context.Background(), source.ID, false, false, ""); e != nil {
			t.Fatal(e)
		}
		sources = append(sources, source.ID)
	}
	for _, id := range sources {
		sample := Sample{}
		testRefresh(context.Background(), s, id, &sample)
		for _, stage := range sample.Stages {
			if stage.Result != "passed" {
				t.Fatal("stable scoped refetch check failed", stage.Category)
			}
		}
	}
}
