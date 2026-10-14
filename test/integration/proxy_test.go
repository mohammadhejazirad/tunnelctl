//go:build linux

package integration

import (
	"bytes"
	"crypto/tls"
	"encoding/json"
	"fmt"
	"io"
	"net"
	"net/http"
	"net/http/httptest"
	"net/url"
	"os"
	"os/exec"
	"path/filepath"
	"strconv"
	"strings"
	"sync/atomic"
	"syscall"
	"testing"
	"time"
	"tunnelctl/internal/domain"
	"tunnelctl/internal/platform"
)

type result struct {
	SchemaVersion int             `json:"schema_version"`
	OK            bool            `json:"ok"`
	Data          json.RawMessage `json:"data"`
	Error         *domain.Error   `json:"error"`
}
type fixture struct {
	t                       *testing.T
	bin, core, root, method string
	upstream                uint16
}

func private(t *testing.T) string {
	t.Helper()
	p := t.TempDir()
	if e := os.Chmod(p, 0700); e != nil {
		t.Fatal(e)
	}
	return p
}
func freePort(t *testing.T) uint16 {
	t.Helper()
	l, e := net.Listen("tcp", "127.0.0.1:0")
	if e != nil {
		t.Fatal(e)
	}
	p := l.Addr().(*net.TCPAddr).Port
	l.Close()
	return uint16(p)
}
func newFixture(t *testing.T, method string) *fixture {
	t.Helper()
	bin, core := os.Getenv("TUNNELCTL_TEST_BIN"), os.Getenv("TUNNELCTL_TEST_CORE")
	if bin == "" || core == "" {
		t.Skip("set TUNNELCTL_TEST_BIN and TUNNELCTL_TEST_CORE to run real-core acceptance tests")
	}
	// Detached supervision needs a protected executable; the development source
	// drive may not implement Unix permissions. Stage the exact built bytes privately.
	binaryBytes, e := os.ReadFile(bin)
	if e != nil {
		t.Fatal(e)
	}
	bin = filepath.Join(private(t), "tunnelctl")
	if e = os.WriteFile(bin, binaryBytes, 0700); e != nil {
		t.Fatal(e)
	}
	f := &fixture{t, bin, core, private(t), method, freePort(t)}
	config := map[string]any{"log": map[string]any{"disabled": true}, "inbounds": []any{map[string]any{"type": "shadowsocks", "listen": "127.0.0.1", "listen_port": f.upstream, "method": method, "password": "synthetic-upstream-password", "network": "tcp"}}, "outbounds": []any{map[string]any{"type": "direct", "tag": "direct"}}, "route": map[string]any{"final": "direct"}}
	b, e := json.Marshal(config)
	if e != nil {
		t.Fatal(e)
	}
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
	waitFor(t, func() bool { return platform.ListenerOwned(server.Process.Pid, f.upstream) }, "controlled Shadowsocks listener")
	f.command("", 0, "init")
	f.command(fmt.Sprintf("ss://%s:synthetic-upstream-password@127.0.0.1:%d#controlled", method, f.upstream), 0, "profile", "import", "--stdin")
	f.command("", 0, "profile", "select", "controlled")
	f.command("", 0, "doctor")
	f.command("", 0, "core", "verify")
	return f
}
func (f *fixture) args(args ...string) []string {
	return append([]string{"--data-dir", f.root, "--core", f.core, "--json", "--timeout", "3s"}, args...)
}
func (f *fixture) command(input string, expected int, args ...string) result {
	f.t.Helper()
	cmd := exec.Command(f.bin, f.args(args...)...)
	cmd.Stdin = strings.NewReader(input)
	var out, errout bytes.Buffer
	cmd.Stdout = &out
	cmd.Stderr = &errout
	e := cmd.Run()
	code := 0
	if e != nil {
		var ok bool
		ex, ok := e.(*exec.ExitError)
		if !ok {
			f.t.Fatal(e)
		}
		code = ex.ExitCode()
	}
	if code != expected {
		f.t.Fatalf("command %v returned %d, expected %d; output=%s", args, code, expected, out.String())
	}
	var r result
	if e = json.Unmarshal(out.Bytes(), &r); e != nil || r.SchemaVersion != 1 || r.OK != (expected == 0) {
		f.t.Fatalf("invalid JSON result: %s", out.String())
	}
	if strings.Contains(out.String(), "synthetic-upstream-password") || errout.Len() != 0 {
		f.t.Fatal("secret or stderr leak")
	}
	return r
}
func (f *fixture) state() domain.Session {
	f.t.Helper()
	r := f.command("", 0, "status")
	var s domain.Session
	if e := json.Unmarshal(r.Data, &s); e != nil {
		f.t.Fatal(e)
	}
	return s
}

type running struct {
	cmd      *exec.Cmd
	done     chan error
	out      bytes.Buffer
	consumed bool
}

func (f *fixture) start(port uint16, args ...string) *running {
	f.t.Helper()
	args = append([]string{"run", "--mode", "proxy", "--listen-port", strconv.Itoa(int(port))}, args...)
	r := &running{cmd: exec.Command(f.bin, f.args(args...)...), done: make(chan error, 1)}
	r.cmd.Stdout = &r.out
	r.cmd.Stderr = io.Discard
	if e := r.cmd.Start(); e != nil {
		f.t.Fatal(e)
	}
	go func() { r.done <- r.cmd.Wait() }()
	f.t.Cleanup(func() {
		if !r.consumed {
			r.cmd.Process.Kill()
			select {
			case <-r.done:
			case <-time.After(5 * time.Second):
			}
		}
	})
	return r
}
func (r *running) wait(t *testing.T, expected int) {
	t.Helper()
	select {
	case e := <-r.done:
		r.consumed = true
		code := 0
		if e != nil {
			if x, ok := e.(*exec.ExitError); ok {
				code = x.ExitCode()
			} else {
				t.Fatal(e)
			}
		}
		if code != expected {
			t.Fatalf("run returned %d, expected %d: %s", code, expected, r.out.String())
		}
		var v result
		if json.Unmarshal(r.out.Bytes(), &v) != nil || v.OK != (expected == 0) {
			t.Fatalf("invalid run JSON: %s", r.out.String())
		}
	case <-time.After(12 * time.Second):
		t.Fatal("foreground run did not exit")
	}
}
func waitFor(t *testing.T, fn func() bool, what string) {
	t.Helper()
	deadline := time.Now().Add(5 * time.Second)
	for time.Now().Before(deadline) {
		if fn() {
			return
		}
		time.Sleep(40 * time.Millisecond)
	}
	t.Fatalf("timeout waiting for %s", what)
}
func (f *fixture) ready(port uint16) {
	waitFor(f.t, func() bool {
		cmd := exec.Command(f.bin, f.args("status")...)
		out, e := cmd.Output()
		if e != nil {
			return false
		}
		var r result
		if json.Unmarshal(out, &r) != nil || !r.OK {
			return false
		}
		var s domain.Session
		if json.Unmarshal(r.Data, &s) != nil {
			return false
		}
		return s.State == "listener_ready" && s.ListenerReady && s.ProcessRunning && s.Listener == fmt.Sprintf("127.0.0.1:%d", port) && !s.ConnectionTested
	}, "owned proxy readiness")
}
func request(t *testing.T, port uint16, target string, auth bool) (string, error) {
	return requestScheme(t, "http", port, target, auth, nil)
}
func requestScheme(t *testing.T, scheme string, port uint16, target string, auth bool, tlsConfig *tls.Config) (string, error) {
	t.Helper()
	u, _ := url.Parse(fmt.Sprintf("%s://127.0.0.1:%d", scheme, port))
	if auth {
		u.User = url.UserPassword("local-user", "synthetic-local-password")
	}
	tr := &http.Transport{Proxy: http.ProxyURL(u), TLSClientConfig: tlsConfig}
	defer tr.CloseIdleConnections()
	client := &http.Client{Transport: tr, Timeout: 2 * time.Second}
	res, e := client.Get(target)
	if e != nil {
		return "", e
	}
	defer res.Body.Close()
	if res.StatusCode != 200 {
		return "", fmt.Errorf("status %d", res.StatusCode)
	}
	b, e := io.ReadAll(io.LimitReader(res.Body, 1024))
	return string(b), e
}
func TestSOCKSHostnameAndVerifiedTLS(t *testing.T) {
	f := newFixture(t, "aes-256-gcm")
	f.command(fmt.Sprintf("ss://aes-256-gcm:synthetic-upstream-password@localhost:%d#hostname", f.upstream), 0, "profile", "import", "--stdin")
	port := freePort(t)
	r := f.start(port, "--profile", "hostname")
	f.ready(port)
	target := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) { fmt.Fprint(w, "hostname-response") }))
	defer target.Close()
	body, e := requestScheme(t, "socks5h", port, strings.Replace(target.URL, "127.0.0.1", "localhost", 1), false, nil)
	if e != nil || body != "hostname-response" {
		t.Fatal("SOCKS5 proxy-side hostname request failed", e)
	}
	tlsTarget := httptest.NewTLSServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) { fmt.Fprint(w, "verified-tls-response") }))
	defer tlsTarget.Close()
	trusted := tlsTarget.Client().Transport.(*http.Transport).TLSClientConfig.Clone()
	if trusted.InsecureSkipVerify {
		t.Fatal("test trust must retain certificate validation")
	}
	body, e = requestScheme(t, "http", port, tlsTarget.URL, false, trusted)
	if e != nil || body != "verified-tls-response" {
		t.Fatal("HTTP CONNECT with verified TLS failed", e)
	}
	f.command("", 0, "stop")
	r.wait(t, 0)
}
func TestRealProxyMethodsAndSafeStop(t *testing.T) {
	for _, method := range []string{"aes-128-gcm", "aes-256-gcm", "chacha20-ietf-poly1305"} {
		t.Run(method, func(t *testing.T) {
			f := newFixture(t, method)
			var requests atomic.Int32
			target := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) { requests.Add(1); fmt.Fprint(w, "controlled-response") }))
			defer target.Close()
			port := freePort(t)
			auth := filepath.Join(private(t), "auth.json")
			if e := os.WriteFile(auth, []byte(`{"username":"local-user","password":"synthetic-local-password"}`), 0600); e != nil {
				t.Fatal(e)
			}
			r := f.start(port, "--duration", "15s", "--proxy-auth-file", auth)
			f.ready(port)
			if _, e := request(t, port, target.URL, false); e == nil || requests.Load() != 0 {
				t.Fatal("unauthenticated local client reached target")
			}
			body, e := request(t, port, target.URL, true)
			if e != nil || body != "controlled-response" || requests.Load() != 1 {
				t.Fatal("real proxy request failed", e)
			}
			f.command("", 6, "run", "--listen-port", strconv.Itoa(int(freePort(t))))
			f.command("", 0, "stop")
			r.wait(t, 0)
			f.command("", 0, "stop")
			if f.state().State != "idle" {
				t.Fatal("session did not return idle")
			}
			for _, name := range []string{"session.json", "control.sock"} {
				if _, e := os.Lstat(filepath.Join(f.root, "runtime", name)); !os.IsNotExist(e) {
					t.Fatal("runtime resource leaked")
				}
			}
		})
	}
}
func TestPortCollisionDurationAndSignals(t *testing.T) {
	f := newFixture(t, "aes-256-gcm")
	held, e := net.Listen("tcp", "127.0.0.1:0")
	if e != nil {
		t.Fatal(e)
	}
	f.command("", 6, "run", "--listen-port", strconv.Itoa(held.Addr().(*net.TCPAddr).Port))
	held.Close()
	r := f.start(freePort(t), "--duration", "200ms")
	r.wait(t, 0)
	if f.state().State != "idle" {
		t.Fatal("duration cleanup failed")
	}
	for _, sig := range []os.Signal{os.Interrupt, syscall.SIGTERM, syscall.SIGHUP} {
		port := freePort(t)
		r = f.start(port)
		f.ready(port)
		if e = r.cmd.Process.Signal(sig); e != nil {
			t.Fatal(e)
		}
		r.wait(t, 0)
		if f.state().State != "idle" {
			t.Fatal("signal cleanup failed")
		}
	}
}
func childPID(t *testing.T, pid int) int {
	t.Helper()
	tasks, e := os.ReadDir(fmt.Sprintf("/proc/%d/task", pid))
	if e != nil {
		t.Fatal(e)
	}
	for _, task := range tasks {
		b, _ := os.ReadFile(fmt.Sprintf("/proc/%d/task/%s/children", pid, task.Name()))
		for _, v := range strings.Fields(string(b)) {
			p, _ := strconv.Atoi(v)
			if p > 0 {
				return p
			}
		}
	}
	t.Fatal("owned core child not found")
	return 0
}
func TestCoreCrashAndSupervisorDeath(t *testing.T) {
	f := newFixture(t, "aes-256-gcm")
	port := freePort(t)
	r := f.start(port)
	f.ready(port)
	pid := childPID(t, r.cmd.Process.Pid)
	p, e := os.FindProcess(pid)
	if e != nil {
		t.Fatal(e)
	}
	if e = p.Kill(); e != nil {
		t.Fatal(e)
	}
	r.wait(t, 5)
	if f.state().State != "idle" {
		t.Fatal("core crash cleanup failed")
	}
	port = freePort(t)
	r = f.start(port)
	f.ready(port)
	if e = r.cmd.Process.Kill(); e != nil {
		t.Fatal(e)
	}
	select {
	case <-r.done:
		r.consumed = true
	case <-time.After(5 * time.Second):
		t.Fatal("supervisor kill timeout")
	}
	waitFor(t, func() bool {
		l, e := net.Listen("tcp", fmt.Sprintf("127.0.0.1:%d", port))
		if e != nil {
			return false
		}
		l.Close()
		return true
	}, "parent-death core termination")
	if f.state().State != "cleanup_required" {
		t.Fatal("stale state was not reported")
	}
	f.command("", 0, "stop")
	f.command("", 0, "stop")
	if f.state().State != "idle" {
		t.Fatal("stale cleanup failed")
	}
}
func TestWrongUpstreamCredentialNeverClaimsConnection(t *testing.T) {
	f := newFixture(t, "aes-256-gcm")
	f.command(fmt.Sprintf("ss://aes-256-gcm:wrong-synthetic-password@127.0.0.1:%d#wrong", f.upstream), 0, "profile", "import", "--stdin")
	f.command("", 0, "profile", "select", "wrong")
	// Run using the selected wrong profile, rather than the fixture's named profile.
	port := freePort(t)
	r := f.start(port, "--profile", "wrong")
	f.ready(port)
	var hits atomic.Int32
	target := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) { hits.Add(1) }))
	defer target.Close()
	if _, e := request(t, port, target.URL, false); e == nil || hits.Load() != 0 {
		t.Fatal("wrong upstream credentials unexpectedly reached target")
	}
	if f.state().ConnectionTested {
		t.Fatal("fabricated connection success")
	}
	f.command("", 0, "stop")
	r.wait(t, 0)
}
