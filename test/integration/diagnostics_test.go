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
	"path/filepath"
	"testing"
	"tunnelctl/internal/app"
)

func TestCLIProfileCheckAndExplicitVerifiedProbe(t *testing.T) {
	f := newFixture(t, "aes-256-gcm")
	f.command("", 0, "profile", "check", "controlled")
	port := freePort(t)
	run := f.start(port)
	f.ready(port)
	target := httptest.NewTLSServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) { fmt.Fprint(w, "small-verified-probe") }))
	defer target.Close()
	// No insecure flag. First prove untrusted TLS refusal in a fresh CLI process.
	f.command("", 5, "probe", "--target", target.URL)
	cert, e := x509.ParseCertificate(target.TLS.Certificates[0].Certificate[0])
	if e != nil {
		t.Fatal(e)
	}
	ca := filepath.Join(private(t), "target-ca.pem")
	os.WriteFile(ca, pem.EncodeToMemory(&pem.Block{Type: "CERTIFICATE", Bytes: cert.Raw}), 0600)
	t.Setenv("SSL_CERT_FILE", ca)
	result := f.command("", 0, "probe", "--target", target.URL)
	var probe app.ProbeResult
	if json.Unmarshal(result.Data, &probe) != nil || !probe.ConnectionTested || probe.HTTPStatus != 200 {
		t.Fatal("verified explicit probe failed")
	}
	f.command("", 2, "probe", "--target", target.URL+"/?token=synthetic-secret")
	if f.state().ConnectionTested {
		t.Fatal("probe invented a persistent connectivity claim")
	}
	f.command("", 0, "stop")
	run.wait(t, 0)
}
