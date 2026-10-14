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
	"sync/atomic"
	"testing"
	"tunnelctl/internal/domain"
	"tunnelctl/internal/subscription"
)

func TestSubscriptionCLIThroughRealProxy(t *testing.T) {
	f := newFixture(t, "aes-256-gcm")
	var broken atomic.Bool
	feed := httptest.NewTLSServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if broken.Load() {
			fmt.Fprint(w, "malformed feed")
			return
		}
		fmt.Fprintf(w, "ss://aes-256-gcm:synthetic-upstream-password@127.0.0.1:%d#from-source", f.upstream)
	}))
	defer feed.Close()
	ca := filepath.Join(private(t), "ca.pem")
	cert, e := x509.ParseCertificate(feed.TLS.Certificates[0].Certificate[0])
	if e != nil {
		t.Fatal(e)
	}
	if e = os.WriteFile(ca, pem.EncodeToMemory(&pem.Block{Type: "CERTIFICATE", Bytes: cert.Raw}), 0600); e != nil {
		t.Fatal(e)
	}
	f.command(feed.URL+"/?token=synthetic-only", 0, "subscription", "add", "--name", "local-source", "--url-stdin", "--allow-private-cidr", "127.0.0.1/32", "--ca-file", ca)
	preview := f.command("", 0, "subscription", "update", "local-source", "--preview")
	var diff subscription.Diff
	if e = json.Unmarshal(preview.Data, &diff); e != nil {
		t.Fatal(e)
	}
	if diff.Committed || len(diff.Added) != 1 {
		t.Fatal("preview did not stage one profile")
	}
	f.command("", 0, "subscription", "update", "local-source", "--if-preview", diff.Token)
	f.command("", 0, "profile", "select", "from-source")
	port := freePort(t)
	run := f.start(port, "--duration", "30s")
	f.ready(port)
	selected := f.command("", 0, "profile", "show", "from-source", "--redacted")
	var profile domain.PublicProfile
	if e = json.Unmarshal(selected.Data, &profile); e != nil || f.state().ProfileID != profile.ID {
		t.Fatal("proxy did not use the imported subscription profile")
	}
	target := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) { fmt.Fprint(w, "subscription-proxy-response") }))
	defer target.Close()
	body, e := request(t, port, target.URL, false)
	if e != nil || body != "subscription-proxy-response" {
		t.Fatal("subscription proxy failed", e)
	}
	id := f.state().ID
	before, e := os.ReadFile(filepath.Join(f.root, "config", "profiles.json"))
	if e != nil {
		t.Fatal(e)
	}
	broken.Store(true)
	f.command("", 2, "subscription", "update", "local-source", "--accept-removals")
	after, e := os.ReadFile(filepath.Join(f.root, "config", "profiles.json"))
	if e != nil || string(before) != string(after) {
		t.Fatal("failed fetch altered healthy snapshot")
	}
	f.command("", 0, "subscription", "remove", "local-source")
	if f.state().ID != id {
		t.Fatal("source operation replaced active session")
	}
	f.command("", 0, "stop")
	run.wait(t, 0)
	f.command("", 0, "profile", "show", "from-source", "--redacted")
}
