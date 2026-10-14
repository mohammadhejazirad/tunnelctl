//go:build linux

package integration

import (
	"crypto/x509"
	"encoding/json"
	"encoding/pem"
	"fmt"
	"net/http"
	"net/http/httptest"
	"os"
	"os/exec"
	"path/filepath"
	"strings"
	"testing"
	"tunnelctl/internal/privatetest"
)

func TestPrivateHarnessSyntheticRealProxy(t *testing.T) {
	harness := os.Getenv("TUNNELCTL_PRIVATE_TEST_BIN")
	if harness == "" {
		t.Skip("set development private-test binary path for synthetic harness acceptance")
	}
	f := newFixture(t, "aes-256-gcm")
	target := httptest.NewTLSServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) { fmt.Fprint(w, "small-synthetic-HTTPS-response") }))
	defer target.Close()
	cert, e := x509.ParseCertificate(target.TLS.Certificates[0].Certificate[0])
	if e != nil {
		t.Fatal(e)
	}
	ca := filepath.Join(private(t), "target-ca.pem")
	os.WriteFile(ca, pem.EncodeToMemory(&pem.Block{Type: "CERTIFICATE", Bytes: cert.Raw}), 0600)
	sample := filepath.Join(private(t), "synthetic-only.txt")
	os.WriteFile(sample, []byte(fmt.Sprintf("ss://aes-256-gcm:synthetic-upstream-password@127.0.0.1:%d#synthetic-name", f.upstream)), 0600)
	cmd := exec.Command(harness, "--file", sample, "--core", f.core, "--connect", "--probe-url", target.URL, "--max-samples", "1", "--max-profiles", "1", "--max-connections", "1", "--timeout", "10s")
	// Explicit trust for this synthetic target only. TLS verification remains on.
	cmd.Env = append(os.Environ(), "SSL_CERT_FILE="+ca)
	output, e := cmd.CombinedOutput()
	if e != nil {
		t.Fatal("synthetic harness failed; output withheld", e)
	}
	if strings.Contains(string(output), "synthetic-upstream-password") || strings.Contains(string(output), "synthetic-name") || strings.Contains(string(output), target.URL) || strings.Contains(string(output), sample) || strings.Contains(string(output), "\x1b") {
		t.Fatal("harness leaked source/target metadata")
	}
	var r privatetest.Report
	if json.Unmarshal(output, &r) != nil || !r.Passed || r.Cleanup != "passed" || len(r.Samples) != 1 {
		t.Fatal("invalid anonymous harness report")
	}
	stages := map[string]string{}
	for _, s := range r.Samples[0].Stages {
		stages[s.Name] = s.Result
	}
	for _, name := range []string{"parse_and_import", "core_compatibility", "generated_config_check", "listener_ready", "real_https_proxy_request", "stop", "session_cleanup"} {
		if stages[name] != "passed" {
			t.Fatal("distinct harness stage did not pass", name)
		}
	}
}
