//go:build linux

package integration

import (
	"crypto/x509"
	"encoding/base64"
	"encoding/json"
	"encoding/pem"
	"fmt"
	"io"
	"net/http"
	"net/http/httptest"
	"os"
	"os/exec"
	"path/filepath"
	"testing"
	"tunnelctl/internal/platform"
)

func newURIProtocolFixture(t *testing.T, protocol string) *fixture {
	t.Helper()
	bin, core := os.Getenv("TUNNELCTL_TEST_BIN"), os.Getenv("TUNNELCTL_TEST_CORE")
	if bin == "" || core == "" {
		t.Skip("set TUNNELCTL_TEST_BIN and TUNNELCTL_TEST_CORE to run real-core acceptance tests")
	}
	binaryBytes, e := os.ReadFile(bin)
	if e != nil {
		t.Fatal(e)
	}
	bin = filepath.Join(private(t), "tunnelctl")
	if e = os.WriteFile(bin, binaryBytes, 0700); e != nil {
		t.Fatal(e)
	}
	f := &fixture{t: t, bin: bin, core: core, root: private(t), method: protocol, upstream: freePort(t)}
	uuid := "11111111-2222-4333-8444-555555555555"
	var inbound map[string]any
	var link string
	switch protocol {
	case "vless":
		inbound = map[string]any{"type": "vless", "listen": "127.0.0.1", "listen_port": f.upstream, "users": []any{map[string]any{"uuid": uuid}}}
		link = fmt.Sprintf("vless://%s@127.0.0.1:%d?encryption=none&security=none&type=tcp#controlled", uuid, f.upstream)
	case "vmess":
		inbound = map[string]any{"type": "vmess", "listen": "127.0.0.1", "listen_port": f.upstream, "users": []any{map[string]any{"uuid": uuid, "alterId": 0}}}
		encoded, _ := json.Marshal(map[string]any{"v": "2", "ps": "controlled", "add": "127.0.0.1", "port": f.upstream, "id": uuid, "aid": 0, "scy": "auto", "net": "tcp", "type": "none", "tls": ""})
		link = "vmess://" + base64.RawStdEncoding.EncodeToString(encoded)
	case "trojan":
		certificateSource := httptest.NewTLSServer(http.HandlerFunc(func(http.ResponseWriter, *http.Request) {}))
		certificateSource.Close()
		certificate := certificateSource.TLS.Certificates[0]
		leaf, err := x509.ParseCertificate(certificate.Certificate[0])
		if err != nil {
			t.Fatal(err)
		}
		keyBytes, err := x509.MarshalPKCS8PrivateKey(certificate.PrivateKey)
		if err != nil {
			t.Fatal(err)
		}
		certPath := filepath.Join(private(t), "trojan-cert.pem")
		keyPath := filepath.Join(private(t), "trojan-key.pem")
		caPath := filepath.Join(private(t), "trojan-ca.pem")
		certPEM := pem.EncodeToMemory(&pem.Block{Type: "CERTIFICATE", Bytes: leaf.Raw})
		if e = os.WriteFile(certPath, certPEM, 0600); e != nil {
			t.Fatal(e)
		}
		if e = os.WriteFile(caPath, certPEM, 0600); e != nil {
			t.Fatal(e)
		}
		if e = os.WriteFile(keyPath, pem.EncodeToMemory(&pem.Block{Type: "PRIVATE KEY", Bytes: keyBytes}), 0600); e != nil {
			t.Fatal(e)
		}
		t.Setenv("SSL_CERT_FILE", caPath)
		inbound = map[string]any{"type": "trojan", "listen": "127.0.0.1", "listen_port": f.upstream, "users": []any{map[string]any{"password": "synthetic-trojan-password"}}, "tls": map[string]any{"enabled": true, "certificate_path": certPath, "key_path": keyPath}}
		link = fmt.Sprintf("trojan://synthetic-trojan-password@127.0.0.1:%d?security=tls&sni=example.com&type=tcp#controlled", f.upstream)
	default:
		t.Fatal("unsupported synthetic protocol")
	}
	config := map[string]any{"log": map[string]any{"disabled": true}, "inbounds": []any{inbound}, "outbounds": []any{map[string]any{"type": "direct", "tag": "direct"}}, "route": map[string]any{"final": "direct"}}
	b, _ := json.Marshal(config)
	path := filepath.Join(private(t), "server.json")
	if e = os.WriteFile(path, b, 0600); e != nil {
		t.Fatal(e)
	}
	if e = exec.Command(core, "check", "-c", path).Run(); e != nil {
		t.Fatal("controlled server configuration rejected", e)
	}
	server := exec.Command(core, "run", "-c", path)
	server.Stdout = io.Discard
	server.Stderr = io.Discard
	platform.ConfigureChild(server)
	if e = server.Start(); e != nil {
		t.Fatal(e)
	}
	t.Cleanup(func() { server.Process.Kill(); server.Wait() })
	waitFor(t, func() bool { return platform.ListenerOwned(server.Process.Pid, f.upstream) }, "controlled "+protocol+" listener")
	f.command("", 0, "init")
	f.command(link, 0, "profile", "import", "--stdin")
	f.command("", 0, "profile", "select", "controlled")
	return f
}

func TestVLESSVMessAndTrojanRealProxy(t *testing.T) {
	for _, protocol := range []string{"vless", "vmess", "trojan"} {
		t.Run(protocol, func(t *testing.T) {
			f := newURIProtocolFixture(t, protocol)
			target := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, _ *http.Request) { fmt.Fprint(w, protocol+"-response") }))
			defer target.Close()
			port := freePort(t)
			running := f.start(port)
			f.ready(port)
			body, e := request(t, port, target.URL, false)
			if e != nil || body != protocol+"-response" {
				t.Fatal("real proxy request failed", e)
			}
			f.command("", 0, "stop")
			running.wait(t, 0)
		})
	}
}
