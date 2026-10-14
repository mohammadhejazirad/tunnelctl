package singbox

import (
	"context"
	"crypto/sha256"
	"encoding/hex"
	"encoding/json"
	"errors"
	"io"
	"net"
	"os"
	"os/exec"
	"path/filepath"
	"runtime"
	"strconv"
	"strings"
	"sync"
	"time"
	"tunnelctl/internal/core"
	"tunnelctl/internal/domain"
	"tunnelctl/internal/platform"
)

const Version = "1.14.3"

// Filled from the SHA-256 verified upstream linux-amd64 archive, see compatibility.json.
const LinuxAMD64SHA256 = "768d31fa7f68debc067bb4626539498a1e8751d36a7b99e1a54bee6a23815f93"

type Adapter struct {
	binary *os.File
	path   string
}

func Open(path string) (*Adapter, error) {
	if runtime.GOOS != "linux" || runtime.GOARCH != "amd64" {
		return nil, domain.Fail(3, "The approved core artifact is limited to Linux amd64.", "Validate a native core artifact and lifecycle adapter before enabling this platform.")
	}
	if !filepath.IsAbs(path) {
		return nil, domain.Fail(2, "The core executable path must be absolute.", "Pass --core /absolute/path/to/sing-box.")
	}
	resolved, e := filepath.EvalSymlinks(path)
	if e != nil {
		return nil, missing()
	}
	f, e := platform.OpenPrivate(resolved, os.O_RDONLY)
	if e != nil {
		return nil, missing()
	}
	st, e := f.Stat()
	if e != nil || !st.Mode().IsRegular() || st.Mode().Perm()&0022 != 0 || st.Mode().Perm()&0111 == 0 {
		f.Close()
		return nil, domain.Fail(4, "The core executable must be a regular executable file and not group/world writable.", "Use the verified pinned core in a protected location.")
	}
	if !platform.UnprivilegedExecutable(f) {
		f.Close()
		return nil, domain.Fail(4, "Proxy mode refuses set-ID or file-capability core executables.", "Use an unprivileged verified core; do not grant persistent capabilities for proxy mode.")
	}
	h := sha256.New()
	if _, e = io.Copy(h, f); e != nil {
		f.Close()
		return nil, missing()
	}
	if hex.EncodeToString(h.Sum(nil)) != LinuxAMD64SHA256 {
		f.Close()
		return nil, domain.Fail(3, "The core artifact does not match the approved linux-amd64 SHA-256.", "Use the exact sing-box 1.14.3 artifact documented in docs/core-versions.md.")
	}
	return &Adapter{f, resolved}, nil
}
func missing() error {
	return domain.Fail(2, "Cannot open the core executable.", "Install the pinned core manually and pass its absolute path with --core.")
}
func (a *Adapter) Close() error { return a.binary.Close() }
func (a *Adapter) command(config *os.File, args ...string) *exec.Cmd {
	fd := "3"
	files := []*os.File{a.binary}
	if config != nil {
		fd = "4"
		files = []*os.File{config, a.binary}
	}
	cmd := exec.Command("/proc/self/fd/"+fd, args...)
	cmd.Args[0] = a.path
	cmd.ExtraFiles = files
	cmd.Stdout = io.Discard
	cmd.Stderr = io.Discard
	platform.ConfigureChild(cmd)
	return cmd
}

type boundedOutput struct{ b strings.Builder }

func (b *boundedOutput) Write(p []byte) (int, error) {
	n := len(p)
	left := 4096 - b.b.Len()
	if left > 0 {
		if len(p) > left {
			p = p[:left]
		}
		b.b.Write(p)
	}
	return n, nil
}
func checked(ctx context.Context, cmd *exec.Cmd) error {
	if e := ctx.Err(); e != nil {
		return e
	}
	if e := cmd.Start(); e != nil {
		return e
	}
	done := make(chan error, 1)
	go func() { done <- cmd.Wait() }()
	select {
	case e := <-done:
		return e
	case <-ctx.Done():
		cmd.Process.Kill()
		<-done
		return ctx.Err()
	}
}
func (a *Adapter) Version(ctx context.Context) (string, error) {
	cmd := a.command(nil, "version")
	var out boundedOutput
	cmd.Stdout = &out
	if e := checked(ctx, cmd); e != nil {
		return "", domain.Fail(3, "Cannot probe the pinned core version.", "Check executable permissions and the core artifact.")
	}
	line := strings.Split(out.b.String(), "\n")[0]
	if strings.TrimSpace(line) != "sing-box version "+Version {
		return "", domain.Fail(3, "The core schema version is unsupported.", "Use the exact pinned sing-box release.")
	}
	return Version, nil
}
func (a *Adapter) Capabilities() core.Capabilities {
	return core.Capabilities{Protocol: "shadowsocks", Methods: []string{"aes-128-gcm", "aes-256-gcm", "chacha20-ietf-poly1305"}, Mode: "proxy", Transport: "tcp", Statistics: false}
}

type config struct {
	Log       logOptions `json:"log"`
	Inbounds  []inbound  `json:"inbounds"`
	Outbounds []outbound `json:"outbounds"`
	Route     route      `json:"route"`
}
type logOptions struct {
	Disabled bool `json:"disabled"`
}
type user struct {
	Username string `json:"username"`
	Password string `json:"password"`
}
type inbound struct {
	Type   string `json:"type"`
	Tag    string `json:"tag"`
	Listen string `json:"listen"`
	Port   uint16 `json:"listen_port"`
	Users  []user `json:"users,omitempty"`
}
type outbound struct {
	Type     string `json:"type"`
	Tag      string `json:"tag"`
	Server   string `json:"server"`
	Port     uint16 `json:"server_port"`
	Method   string `json:"method"`
	Password string `json:"password"`
	Network  string `json:"network"`
}
type route struct {
	Final string `json:"final"`
}

func (a *Adapter) Render(p domain.Profile, l core.Listener) ([]byte, error) {
	if e := p.Validate(); e != nil {
		return nil, e
	}
	if l.Port < 1024 {
		return nil, domain.Fail(2, "The listener port must be between 1024 and 65535.", "Choose an unprivileged loopback port.")
	}
	in := inbound{Type: "mixed", Tag: "local", Listen: "127.0.0.1", Port: l.Port}
	if l.Username != "" {
		in.Users = []user{{l.Username, l.Password}}
	}
	return json.Marshal(config{logOptions{true}, []inbound{in}, []outbound{{"shadowsocks", "upstream", p.Host, p.Port, p.Method, p.Password, "tcp"}}, route{"upstream"}})
}
func (a *Adapter) Validate(ctx context.Context, f *os.File) error {
	if e := checked(ctx, a.command(f, "check", "-c", "/proc/self/fd/3")); e != nil {
		return domain.Fail(3, "The pinned core rejected the generated configuration.", "Run doctor and check the supported profile capability; core output is suppressed to protect secrets.")
	}
	return nil
}

type process struct {
	cmd  *exec.Cmd
	done chan struct{}
	mu   sync.Mutex
	exit error
}

func (a *Adapter) Start(f *os.File) (core.Process, error) {
	cmd := a.command(f, "run", "-c", "/proc/self/fd/3")
	if e := cmd.Start(); e != nil {
		return nil, domain.Fail(1, "Cannot launch the core process.", "Check the verified executable and OS restrictions.")
	}
	p := &process{cmd: cmd, done: make(chan struct{})}
	go func() { e := cmd.Wait(); p.mu.Lock(); p.exit = e; p.mu.Unlock(); close(p.done) }()
	return p, nil
}
func (p *process) Done() <-chan struct{} { return p.done }
func (p *process) ListenerReady(l core.Listener) bool {
	select {
	case <-p.done:
		return false
	default:
		return platform.ListenerOwned(p.cmd.Process.Pid, l.Port)
	}
}
func (p *process) ExitError() error {
	p.mu.Lock()
	defer p.mu.Unlock()
	if p.exit != nil {
		return domain.Fail(5, "The core exited unexpectedly.", "Check endpoint availability and supported profile settings; sensitive core logs are suppressed.")
	}
	return nil
}
func (p *process) Ready(ctx context.Context, l core.Listener) error {
	tick := time.NewTicker(40 * time.Millisecond)
	defer tick.Stop()
	for {
		select {
		case <-ctx.Done():
			return domain.Fail(5, "The local proxy did not become ready before the deadline.", "Check the listener port, permissions and core compatibility.")
		case <-p.done:
			return domain.Fail(5, "The core exited before listener readiness.", "Check for a port conflict and run doctor.")
		case <-tick.C:
			if !platform.ListenerOwned(p.cmd.Process.Pid, l.Port) {
				continue
			}
			c, e := net.DialTimeout("tcp", net.JoinHostPort("127.0.0.1", strconv.Itoa(int(l.Port))), 200*time.Millisecond)
			if e != nil {
				continue
			}
			c.SetDeadline(time.Now().Add(300 * time.Millisecond))
			method := byte(0)
			if l.Username != "" {
				method = 2
			}
			_, e = c.Write([]byte{5, 1, method})
			var response [2]byte
			if e == nil {
				_, e = io.ReadFull(c, response[:])
			}
			c.Close()
			if e == nil && response == [2]byte{5, method} {
				select {
				case <-p.done:
					continue
				default:
					return nil
				}
			}
		}
	}
}
func (p *process) Stop(timeout time.Duration) error {
	select {
	case <-p.done:
		return nil
	default:
	}
	if e := platform.Terminate(p.cmd.Process); e != nil && !errors.Is(e, os.ErrProcessDone) { // Signal ownership is the live child handle, never a persisted PID.
		select {
		case <-p.done:
			return nil
		default:
		}
	}
	timer := time.NewTimer(timeout)
	defer timer.Stop()
	select {
	case <-p.done:
		return nil
	case <-timer.C:
	}
	if e := p.cmd.Process.Kill(); e != nil && !errors.Is(e, os.ErrProcessDone) {
		return domain.Fail(7, "Cannot terminate the owned core.", "Inspect the private session with status; do not kill processes by name.")
	}
	select {
	case <-p.done:
		return nil
	case <-time.After(timeout):
		return domain.Fail(7, "The owned core did not exit after forced termination.", "Inspect the operating system process state.")
	}
}
