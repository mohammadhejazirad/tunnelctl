//go:build linux

package subscription

import (
	"compress/gzip"
	"context"
	"crypto/x509"
	"encoding/base64"
	"encoding/json"
	"encoding/pem"
	"fmt"
	"net"
	"net/http"
	"net/http/httptest"
	"net/netip"
	"os"
	"path/filepath"
	"strings"
	"sync"
	"sync/atomic"
	"testing"
	"time"
	"tunnelctl/internal/domain"
	"tunnelctl/internal/store"
)

const linkA = "ss://aes-256-gcm:synthetic-alpha@127.0.0.1:8388#alpha"
const linkB = "ss://aes-128-gcm:synthetic-beta@[::1]:8389#beta"

func manager(t *testing.T, server *httptest.Server) Manager {
	t.Helper()
	root := t.TempDir()
	os.Chmod(root, 0700)
	s, e := store.New(root)
	if e != nil {
		t.Fatal(e)
	}
	ca := filepath.Join(root, "ca.pem")
	cert, e := x509.ParseCertificate(server.TLS.Certificates[0].Certificate[0])
	if e != nil {
		t.Fatal(e)
	}
	if e = os.WriteFile(ca, pem.EncodeToMemory(&pem.Block{Type: "CERTIFICATE", Bytes: cert.Raw}), 0600); e != nil {
		t.Fatal(e)
	}
	m := Manager{Store: s}
	if _, e = m.Add("controlled", strings.NewReader(server.URL+"/feed?token=synthetic-secret"), []string{"127.0.0.1/32"}, ca); e != nil {
		t.Fatal(e)
	}
	return m
}
func snapshot(t *testing.T, m Manager) string {
	t.Helper()
	b, e := os.ReadFile(filepath.Join(m.Store.Paths.Config, "profiles.json"))
	if e != nil {
		t.Fatal(e)
	}
	return string(b)
}
func TestHTTPSAtomicUpdatesAndIdentity(t *testing.T) {
	var mu sync.Mutex
	body := linkA + "\n" + linkB
	etag := "\"one\""
	last := "Wed, 14 Oct 2026 00:00:00 GMT"
	fail := false
	var conditional atomic.Bool
	server := httptest.NewTLSServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		mu.Lock()
		defer mu.Unlock()
		if r.Header.Get("If-None-Match") == etag && r.Header.Get("If-Modified-Since") == last {
			conditional.Store(true)
			w.WriteHeader(304)
			return
		}
		if fail {
			w.WriteHeader(500)
			return
		}
		w.Header().Set("ETag", etag)
		w.Header().Set("Last-Modified", last)
		fmt.Fprint(w, body)
	}))
	defer server.Close()
	m := manager(t, server)
	ctx := context.Background()
	before := snapshot(t, m)
	d, e := m.Update(ctx, "controlled", true, false, "")
	if e != nil || len(d.Added) != 2 || d.Committed || snapshot(t, m) != before {
		t.Fatal("preview changed private state", e)
	}
	d, e = m.Update(ctx, "controlled", false, false, d.Token)
	if e != nil || !d.Committed {
		t.Fatal(e)
	}
	v, e := m.Store.Load()
	if e != nil {
		t.Fatal(e)
	}
	alpha := v.Profiles[0]
	beta := v.Profiles[1]
	if e = m.Store.Update(func(v *domain.Settings) error {
		v.Selected = alpha.ID
		v.Profiles[0].Name = "user label"
		v.Profiles[0].UserNamed = true
		v.Profiles[0].Tags = []string{"personal"}
		v.Profiles[0].Favorite = true
		return nil
	}); e != nil {
		t.Fatal(e)
	}
	before = snapshot(t, m)
	d, e = m.Update(ctx, "controlled", false, false, "")
	if e != nil || !d.NotModified || !conditional.Load() || snapshot(t, m) != before {
		t.Fatal("conditional update failed", e)
	}
	mu.Lock()
	etag = "\"two\""
	body = strings.Replace(linkA, "#alpha", "#server-renamed", 1)
	mu.Unlock()
	d, e = m.Update(ctx, "controlled", true, false, "")
	if e != nil || len(d.Removed) != 1 {
		t.Fatal("missing removal preview", e)
	}
	if _, e = m.Update(ctx, "controlled", false, false, ""); e == nil || snapshot(t, m) != before {
		t.Fatal("unapproved removal committed")
	}
	if _, e = m.Update(ctx, "controlled", false, true, d.Token); e != nil {
		t.Fatal(e)
	}
	v, _ = m.Store.Load()
	if len(v.Profiles) != 1 || v.Profiles[0].ID != alpha.ID || v.Profiles[0].Name != "user label" || v.Selected != alpha.ID {
		t.Fatal("stable identity/name/selection lost")
	}
	if !v.Profiles[0].Favorite || len(v.Profiles[0].Tags) != 1 || v.Profiles[0].Tags[0] != "personal" {
		t.Fatal("subscription refresh lost user metadata")
	}
	mu.Lock()
	etag = "\"three\""
	body = linkB
	mu.Unlock()
	d, e = m.Update(ctx, "controlled", false, false, "")
	if e != nil || len(d.RetainedSelected) != 1 {
		t.Fatal("selected removal not retained", e)
	}
	v, _ = m.Store.Load()
	if v.Selected != alpha.ID || len(v.Profiles) != 2 {
		t.Fatal("selected snapshot lost")
	}
	_ = beta
	before = snapshot(t, m)
	for _, bad := range []string{"", "{\"raw\":true}", linkA + "\nunsupported://token", "%%%%"} {
		mu.Lock()
		etag = "bad"
		body = bad
		mu.Unlock()
		if _, e = m.Update(ctx, "controlled", false, true, ""); e == nil || snapshot(t, m) != before {
			t.Fatal("invalid update changed last good state")
		}
	}
	mu.Lock()
	fail = true
	etag = "fail"
	mu.Unlock()
	if _, e = m.Update(ctx, "controlled", false, true, ""); e == nil || snapshot(t, m) != before {
		t.Fatal("HTTP failure changed healthy data")
	}
	public, e := m.List()
	if e != nil {
		t.Fatal(e)
	}
	b, _ := json.Marshal(public)
	if strings.Contains(string(b), "synthetic-secret") || strings.Contains(string(b), "token=") {
		t.Fatal("URL leaked")
	}
	if _, e = m.Rename("controlled", "renamed"); e != nil {
		t.Fatal(e)
	}
	if e = m.Remove("renamed"); e != nil {
		t.Fatal(e)
	}
	v, _ = m.Store.Load()
	for _, p := range v.Profiles {
		if p.SubscriptionID != "" {
			t.Fatal("source removal did not retain standalone profiles")
		}
	}
}
func TestSSRFRedirectTLSAndLimits(t *testing.T) {
	var destinationHits atomic.Int32
	other := httptest.NewTLSServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) { destinationHits.Add(1) }))
	defer other.Close()
	mode := "ok"
	server := httptest.NewTLSServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		switch mode {
		case "redirect":
			http.Redirect(w, r, other.URL, 302)
		case "downgrade":
			http.Redirect(w, r, "http://127.0.0.1/", 302)
		case "loop":
			http.Redirect(w, r, "/loop", 302)
		case "gzip":
			w.Header().Set("Content-Encoding", "gzip")
			gz := gzip.NewWriter(w)
			gz.Write([]byte(strings.Repeat("a", MaxExpanded+1)))
			gz.Close()
		default:
			fmt.Fprint(w, linkA)
		}
	}))
	defer server.Close()
	m := manager(t, server)
	v, _ := m.Store.Load()
	source := v.Subscriptions[0]
	source.AllowedCIDRs = nil
	if _, e := (Fetcher{}).Fetch(context.Background(), source); e == nil {
		t.Fatal("private source bypassed SSRF policy")
	}
	source = v.Subscriptions[0]
	source.CAFile = ""
	if _, e := (Fetcher{}).Fetch(context.Background(), source); e == nil {
		t.Fatal("untrusted TLS certificate accepted")
	}
	before := snapshot(t, m)
	for _, value := range []string{"redirect", "downgrade", "loop", "gzip"} {
		mode = value
		if _, e := m.Update(context.Background(), "controlled", false, true, ""); e == nil || snapshot(t, m) != before {
			t.Fatal("unsafe response committed", value)
		}
	}
	if destinationHits.Load() != 0 {
		t.Fatal("cross-origin request leaked credentials")
	}
	mode = "ok"
	m.Fetcher.Lookup = func(context.Context, string) ([]net.IPAddr, error) {
		return []net.IPAddr{{IP: net.ParseIP("8.8.8.8")}, {IP: net.ParseIP("169.254.169.254")}}, nil
	}
	if _, e := m.Update(context.Background(), "controlled", false, true, ""); e == nil {
		t.Fatal("mixed public/private DNS answers accepted")
	}
	for _, value := range []string{"http://example.com/feed", "https://127.0.0.1/#fragment", "https://example.com:0/feed", "https://example.com/\xff"} {
		source.URL = value
		if ValidateSource(source) == nil {
			t.Fatal("invalid URL accepted")
		}
	}
	for _, ip := range []string{"127.0.0.1", "::1", "169.254.169.254", "10.0.0.1", "100.100.100.200", "::ffff:127.0.0.1", "64:ff9b::a9fe:a9fe"} {
		if allowed(netip.MustParseAddr(ip), nil) {
			t.Fatal("unsafe IP accepted")
		}
	}
}
func TestConcurrentUpdateCancellationAndRecovery(t *testing.T) {
	entered := make(chan struct{}, 1)
	release := make(chan struct{})
	server := httptest.NewTLSServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		select {
		case entered <- struct{}{}:
		default:
		}
		select {
		case <-release:
			fmt.Fprint(w, linkA)
		case <-r.Context().Done():
		}
	}))
	defer server.Close()
	m := manager(t, server)
	before := snapshot(t, m)
	done := make(chan error, 1)
	ctx, cancel := context.WithCancel(context.Background())
	go func() { _, e := m.Update(ctx, "controlled", false, false, ""); done <- e }()
	<-entered
	if _, e := m.Update(context.Background(), "controlled", false, false, ""); e == nil {
		t.Fatal("concurrent update lock bypassed")
	}
	cancel()
	select {
	case e := <-done:
		if e == nil {
			t.Fatal("cancel did not fail")
		}
	case <-time.After(3 * time.Second):
		t.Fatal("cancel did not terminate")
	}
	if snapshot(t, m) != before {
		t.Fatal("canceled update changed state")
	}
	close(release)
	if _, e := m.Update(context.Background(), "controlled", false, false, ""); e != nil {
		t.Fatal("recovery after cancel failed", e)
	}
}
func TestParserBounds(t *testing.T) {
	for _, enc := range []*base64.Encoding{base64.StdEncoding, base64.RawStdEncoding, base64.URLEncoding, base64.RawURLEncoding} {
		p, e := Parse([]byte(enc.EncodeToString([]byte(linkA + "\n" + linkB))))
		if e != nil || len(p) != 2 {
			t.Fatal("envelope rejected", e)
		}
	}
	if _, e := Parse([]byte(strings.Repeat(linkA+"\n", MaxProfiles+1))); e == nil {
		t.Fatal("input count unbounded")
	}
	if _, e := Parse([]byte(strings.Repeat("a", MaxExpanded+1))); e == nil {
		t.Fatal("decode unbounded")
	}
}

func TestSourceIDPrecedesNameAlias(t *testing.T) {
	v := domain.Settings{Subscriptions: []domain.Subscription{{ID: "first-id", Name: "second-id"}, {ID: "second-id", Name: "second-name"}}}
	s, e := resolve(v, "second-id")
	if e != nil || s.ID != "second-id" {
		t.Fatal("display name overrode an exact source ID")
	}
}

func TestConservativeAddressPolicy(t *testing.T) {
	for _, ip := range []string{"0.1.2.3", "127.1.2.3", "169.254.169.254", "100.100.100.200", "192.88.99.1", "::ffff:127.0.0.1", "::7f00:1", "64:ff9b::7f00:1", "100::1", "2001:2::1", "2001:20::1", "2002:7f00:1::1", "3fff::1"} {
		if allowed(netip.MustParseAddr(ip), nil) {
			t.Fatal("reserved address permitted", ip)
		}
	}
	for _, ip := range []string{"8.8.8.8", "2606:4700:4700::1111"} {
		if !allowed(netip.MustParseAddr(ip), nil) {
			t.Fatal("public address denied", ip)
		}
	}
	if !allowed(netip.MustParseAddr("127.0.0.1"), []string{"127.0.0.1/32"}) || allowed(netip.MustParseAddr("127.0.0.2"), []string{"127.0.0.1/32"}) {
		t.Fatal("private exception is not narrow")
	}
}

func TestStagedRevisionConflictAndCommitFailurePreserveData(t *testing.T) {
	for _, failure := range []string{"rename", "remove", "write-permission"} {
		t.Run(failure, func(t *testing.T) {
			entered := make(chan struct{}, 1)
			release := make(chan struct{})
			server := httptest.NewTLSServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) { entered <- struct{}{}; <-release; fmt.Fprint(w, linkA) }))
			defer server.Close()
			unblock := sync.OnceFunc(func() { close(release) })
			defer unblock()
			m := manager(t, server)
			type stagedResult struct {
				Diff Diff
				Err  error
			}
			done := make(chan stagedResult, 1)
			go func() {
				d, e := m.Update(context.Background(), "controlled", false, false, "")
				done <- stagedResult{d, e}
			}()
			<-entered
			switch failure {
			case "rename":
				if _, e := m.Rename("controlled", "user-renamed"); e != nil {
					t.Fatal(e)
				}
			case "remove":
				if e := m.Remove("controlled"); e != nil {
					t.Fatal(e)
				}
			case "write-permission":
				if e := os.Chmod(m.Store.Paths.Config, 0500); e != nil {
					t.Fatal(e)
				}
				defer os.Chmod(m.Store.Paths.Config, 0700)
			}
			before := snapshot(t, m)
			unblock()
			select {
			case result := <-done:
				if result.Err == nil {
					t.Fatal("staged conflict/failure was ignored")
				}
				if result.Diff.Committed {
					t.Fatal("failed storage write reported a committed snapshot")
				}
			case <-time.After(3 * time.Second):
				t.Fatal("staged failure did not terminate")
			}
			if snapshot(t, m) != before {
				t.Fatal("failed stage overwrote last good/concurrent change")
			}
		})
	}
}
