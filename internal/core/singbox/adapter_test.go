//go:build linux

package singbox

import (
	"context"
	"encoding/json"
	"io"
	"os"
	"os/exec"
	"os/signal"
	"path/filepath"
	"sync"
	"syscall"
	"testing"
	"time"
	"tunnelctl/internal/core"
	"tunnelctl/internal/domain"
	"tunnelctl/internal/platform"
)

func profile() domain.Profile {
	return domain.Profile{Protocol: "shadowsocks", Host: "example.com", Port: 8388, Method: "aes-256-gcm", Password: "synthetic"}
}
func TestGeneratedConfiguration(t *testing.T) {
	a := &Adapter{}
	b, e := a.Render(profile(), core.Listener{Port: 2080, Username: "local", Password: "synthetic-local"})
	if e != nil {
		t.Fatal(e)
	}
	var c config
	if e = json.Unmarshal(b, &c); e != nil {
		t.Fatal(e)
	}
	if c.Inbounds[0].Listen != "127.0.0.1" || c.Outbounds[0].Network != "tcp" || c.Route.Final != "upstream" || !c.Log.Disabled || c.Inbounds[0].Users[0].Password != "synthetic-local" {
		t.Fatal("unsafe config")
	}
	if _, e = a.Render(profile(), core.Listener{Port: 80}); e == nil {
		t.Fatal("privileged listener accepted")
	}
}
func TestRealConfigValidation(t *testing.T) {
	path := os.Getenv("TUNNELCTL_TEST_CORE")
	if path == "" {
		t.Skip("set TUNNELCTL_TEST_CORE for real schema validation")
	}
	a, e := Open(path)
	if e != nil {
		t.Fatal(e)
	}
	defer a.Close()
	profiles := []domain.Profile{
		profile(),
		{Protocol: "vless", Host: "example.com", Port: 443, Method: "none", Password: "11111111-2222-4333-8444-555555555555", Transport: "ws", Security: "tls", ServerName: "example.com", HostHeader: "cdn.example.com", Path: "/socket", Fingerprint: "chrome"},
		{Protocol: "vless", Host: "example.com", Port: 443, Method: "none", Password: "11111111-2222-4333-8444-555555555555", Transport: "grpc", Security: "reality", ServerName: "example.com", ServiceName: "edge", Fingerprint: "chrome", PublicKey: "7E7FjYek1uZt0Wj8KpQDz-yKS0PjWRqRR6B2s8Lh_GU", ShortID: "abcd"},
		{Protocol: "vmess", Host: "example.com", Port: 443, Method: "auto", Password: "11111111-2222-4333-8444-555555555555", Transport: "ws", Security: "tls", ServerName: "example.com", Path: "/vmess"},
		{Protocol: "trojan", Host: "example.com", Port: 443, Method: "tls", Password: "synthetic", Transport: "tcp", Security: "tls", ServerName: "example.com"},
	}
	for _, p := range profiles {
		b, e := a.Render(p, core.Listener{Port: 2080})
		if e != nil {
			t.Fatal(e)
		}
		f, e := os.CreateTemp(t.TempDir(), "config-")
		if e != nil {
			t.Fatal(e)
		}
		os.Remove(f.Name())
		if _, e = f.Write(b); e != nil {
			t.Fatal(e)
		}
		ctx, cancel := context.WithTimeout(context.Background(), 3*time.Second)
		e = a.Validate(ctx, f)
		cancel()
		f.Close()
		if e != nil {
			t.Fatalf("core rejected config for %s/%s: %v", p.Protocol, p.Transport, e)
		}
	}
}
func TestRefuseUnapprovedExecutable(t *testing.T) {
	p := filepath.Join(t.TempDir(), "core")
	if e := os.WriteFile(p, []byte("unapproved synthetic executable"), 0700); e != nil {
		t.Fatal(e)
	}
	if _, e := Open(p); e == nil {
		t.Fatal("unapproved executable accepted")
	}
	if _, e := Open("relative"); e == nil {
		t.Fatal("relative executable accepted")
	}
}
func TestTerminationHelper(t *testing.T) {
	if os.Getenv("TUNNELCTL_IGNORE_TERM_HELPER") != "1" {
		return
	}
	signal.Ignore(syscall.SIGTERM)
	os.Stdout.Write([]byte("ready\n"))
	for {
		time.Sleep(time.Hour)
	}
}
func TestGracefulTimeoutEscalatesOnOwnedChild(t *testing.T) {
	cmd := exec.Command(os.Args[0], "-test.run=^TestTerminationHelper$")
	cmd.Env = append(os.Environ(), "TUNNELCTL_IGNORE_TERM_HELPER=1")
	platform.ConfigureChild(cmd)
	r, e := cmd.StdoutPipe()
	if e != nil {
		t.Fatal(e)
	}
	if e = cmd.Start(); e != nil {
		t.Fatal(e)
	}
	t.Cleanup(func() { cmd.Process.Kill() })
	var ready [6]byte
	if _, e = io.ReadFull(r, ready[:]); e != nil {
		t.Fatal(e)
	}
	p := &process{cmd: cmd, done: make(chan struct{}), mu: sync.Mutex{}}
	go func() { e := cmd.Wait(); p.mu.Lock(); p.exit = e; p.mu.Unlock(); close(p.done) }()
	start := time.Now()
	if e = p.Stop(80 * time.Millisecond); e != nil {
		t.Fatal(e)
	}
	if elapsed := time.Since(start); elapsed < 70*time.Millisecond || elapsed > 2*time.Second {
		t.Fatal("graceful timeout did not escalate")
	}
	if e = p.Stop(80 * time.Millisecond); e != nil {
		t.Fatal("repeated stop failed", e)
	}
}
