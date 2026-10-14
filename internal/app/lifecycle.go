package app

import (
	"context"
	"encoding/json"
	"errors"
	"io"
	"net"
	"os"
	"os/exec"
	"path/filepath"
	"strings"
	"time"
	"tunnelctl/internal/core"
	"tunnelctl/internal/core/singbox"
	"tunnelctl/internal/domain"
	"tunnelctl/internal/platform"
	"tunnelctl/internal/session"
	"tunnelctl/internal/store"
)

func (s *Service) Logs() ([]session.Event, error) {
	return session.ReadEvents(filepath.Join(s.Store.Paths.State, "events.json"))
}
func (s *Service) Start(ctx context.Context, key string, l core.Listener, duration time.Duration) (domain.Session, error) {
	executable, e := os.Executable()
	if e != nil {
		return domain.Session{}, domain.Unexpected()
	}
	self, e := platform.OpenSelfExecutable()
	if e != nil {
		return domain.Session{}, e
	}
	defer self.Close()
	st, e := self.Stat()
	if e != nil || !st.Mode().IsRegular() || st.Mode().Perm()&0022 != 0 || !platform.UnprivilegedExecutable(self) {
		return domain.Session{}, domain.Fail(4, "Detached operation requires a protected application executable.", "Install or extract tunnelctl on a filesystem supporting Unix executable permissions.")
	}
	r, w, e := os.Pipe()
	if e != nil {
		return domain.Session{}, domain.Unexpected()
	}
	defer r.Close()
	// Credentials travel over the private inherited pipe, never process arguments.
	input, inWrite, e := os.Pipe()
	if e != nil {
		w.Close()
		return domain.Session{}, domain.Unexpected()
	}
	defer inWrite.Close()
	// Reexec the running inode, not a replaceable installation pathname.
	cmd := exec.Command("/proc/self/fd/5", "--core", s.CorePath, "--timeout", s.Timeout.String(), "__supervise")
	cmd.Args[0] = executable
	cmd.ExtraFiles = []*os.File{w, input, self}
	cmd.Stdout = io.Discard
	cmd.Stderr = io.Discard
	platform.ConfigureDetached(cmd)
	if e = cmd.Start(); e != nil {
		w.Close()
		input.Close()
		return domain.Session{}, domain.Fail(1, "Cannot launch the detached supervisor.", "Check application permissions and platform support.")
	}
	w.Close()
	input.Close()
	waited := make(chan error, 1)
	go func() { waited <- cmd.Wait() }()
	abort := func() {
		platform.Terminate(cmd.Process)
		select {
		case <-waited:
		case <-time.After(4 * time.Second):
			cmd.Process.Kill()
			<-waited
		}
	}
	request := DetachedRequest{Key: key, Listener: l, Duration: duration, Paths: s.Store.Paths}
	if e = json.NewEncoder(inWrite).Encode(request); e != nil {
		abort()
		return domain.Session{}, domain.Unexpected()
	}
	inWrite.Close()
	type reply struct {
		Session domain.Session `json:"session"`
		Error   *domain.Error  `json:"error,omitempty"`
	}
	ch := make(chan reply, 1)
	go func() {
		var v reply
		if store.DecodeStrict(r, 8192, &v) != nil {
			v.Error = &domain.Error{Code: 1, Message: "Detached supervisor failed before readiness.", Remediation: "Run doctor or foreground run."}
		}
		ch <- v
	}()
	select {
	case v := <-ch:
		if v.Error != nil {
			return domain.Session{}, v.Error
		}
		if ctx.Err() != nil {
			return domain.Session{}, errors.Join(domain.Fail(5, "Detached start was canceled.", "Startup has been stopped."), s.StopSession(v.Session.ID))
		}
		return v.Session, nil
	case <-ctx.Done():
		abort()
		return domain.Session{}, domain.Fail(5, "Detached start was canceled.", "Retry status to reconcile startup resources.")
	case <-time.After(2*s.Timeout + 5*time.Second):
		abort()
		return domain.Session{}, domain.Fail(5, "Detached startup timed out.", "Run status and doctor before retrying.")
	}
}

type DetachedRequest struct {
	Key      string        `json:"key"`
	Listener core.Listener `json:"listener"`
	Duration time.Duration `json:"duration"`
	Paths    store.Paths   `json:"paths"`
}

func RunDetached(ctx context.Context, corePath string, timeout time.Duration) error {
	w := os.NewFile(3, "readiness")
	r := os.NewFile(4, "request")
	defer r.Close()
	var request DetachedRequest
	if e := store.DecodeStrict(r, 8192, &request); e != nil {
		return domain.Fail(2, "Invalid detached startup request.", "Use start; the supervisor entry point is internal.")
	}
	for _, path := range []string{request.Paths.Config, request.Paths.State, request.Paths.Cache, request.Paths.Runtime} {
		if e := store.PrivateDir(path); e != nil {
			return e
		}
	}
	s := &Service{Store: &store.Store{Paths: request.Paths}, CorePath: corePath, Timeout: timeout}
	ready := false
	e := s.Run(ctx, request.Key, "proxy", request.Listener, request.Duration, func(v domain.Session) error {
		ready = true
		e := json.NewEncoder(w).Encode(struct {
			Session domain.Session `json:"session"`
		}{v})
		w.Close()
		return e
	})
	if !ready {
		var de *domain.Error
		if !errors.As(e, &de) {
			de = &domain.Error{Code: 1, Message: "Detached startup failed.", Remediation: "Run doctor."}
		}
		json.NewEncoder(w).Encode(struct {
			Error *domain.Error `json:"error"`
		}{de})
		w.Close()
	}
	return e
}

type ChildExit struct{ Code int }

func (e *ChildExit) Error() string { return "The child command exited unsuccessfully." }
func (s *Service) Exec(ctx context.Context, key string, args []string, in io.Reader, out, diagnostics io.Writer) (err error) {
	if len(args) == 0 {
		return domain.Fail(2, "A child command is required.", "Use exec -- COMMAND ARGUMENTS.")
	}
	ctx, cancel := context.WithCancel(ctx)
	defer cancel()
	v, e := s.Store.Load()
	if e != nil {
		return e
	}
	p, e := store.Resolve(v, key)
	if e != nil {
		return e
	}
	a, e := singbox.Open(s.CorePath)
	if e != nil {
		return e
	}
	defer a.Close()
	runtimeDir, e := os.MkdirTemp(s.Store.Paths.Runtime, "exec-")
	if e != nil {
		return domain.Unexpected()
	}
	defer func() {
		if e := os.Remove(filepath.Join(runtimeDir, "session.lock")); e != nil && !os.IsNotExist(e) {
			err = domain.Fail(7, "Cannot remove the isolated child session lock.", "Inspect private runtime storage.")
		}
		if e := os.Remove(runtimeDir); e != nil {
			err = domain.Fail(7, "Cannot remove the isolated child runtime directory.", "Inspect private runtime storage.")
		}
	}()
	reservation, e := net.Listen("tcp", "127.0.0.1:0")
	if e != nil {
		return domain.Fail(6, "Cannot reserve a child proxy port.", "Check local socket permissions.")
	}
	port := uint16(reservation.Addr().(*net.TCPAddr).Port)
	reservation.Close()
	supervisor := &session.Supervisor{Runtime: runtimeDir}
	var child *exec.Cmd
	done := make(chan error, 1)
	runErr := supervisor.Run(ctx, a, session.Options{Profile: p, Listener: core.Listener{Port: port}, Timeout: s.Timeout}, func(state domain.Session) error {
		child = exec.Command(args[0], args[1:]...)
		child.Stdin = in
		child.Stdout = out
		child.Stderr = diagnostics
		platform.ConfigureChild(child)
		env := []string{}
		for _, entry := range os.Environ() {
			name := strings.ToUpper(strings.SplitN(entry, "=", 2)[0])
			if name != "HTTP_PROXY" && name != "HTTPS_PROXY" && name != "ALL_PROXY" && name != "NO_PROXY" {
				env = append(env, entry)
			}
		}
		endpoint := "http://" + state.Listener
		for _, name := range []string{"HTTP_PROXY", "HTTPS_PROXY", "http_proxy", "https_proxy"} {
			env = append(env, name+"="+endpoint)
		}
		env = append(env, "ALL_PROXY=socks5h://"+state.Listener, "all_proxy=socks5h://"+state.Listener, "NO_PROXY=", "no_proxy=")
		child.Env = env
		if e := child.Start(); e != nil {
			return domain.Fail(2, "Cannot launch the requested child command.", "Check the command path and permissions.")
		}
		go func() { e := child.Wait(); done <- e; cancel() }()
		return nil
	})
	if child == nil || child.Process == nil {
		return runErr
	}
	cancel()
	var childErr error
	select {
	case childErr = <-done:
	default:
		platform.Terminate(child.Process)
		select {
		case childErr = <-done:
		case <-time.After(3 * time.Second):
			child.Process.Kill()
			childErr = <-done
		}
	}
	if runErr != nil {
		return runErr
	}
	if childErr != nil {
		if x, ok := childErr.(*exec.ExitError); ok {
			code := x.ExitCode()
			if code < 0 {
				code = 130
			}
			return &ChildExit{code}
		}
		return domain.Unexpected()
	}
	return nil
}
