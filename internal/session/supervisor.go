package session

import (
	"context"
	"encoding/json"
	"errors"
	"net"
	"os"
	"path/filepath"
	"runtime"
	"strconv"
	"sync"
	"time"
	"tunnelctl/internal/core"
	"tunnelctl/internal/domain"
	"tunnelctl/internal/platform"
	"tunnelctl/internal/store"
)

type Options struct {
	Profile  domain.Profile
	Listener core.Listener
	Duration time.Duration
	Timeout  time.Duration
}
type Supervisor struct {
	Runtime  string
	LogPath  string
	events   []Event
	logErr   error
	mu       sync.Mutex
	state    domain.Session
	process  core.Process
	listener core.Listener
}

func (s *Supervisor) snapshot() domain.Session {
	s.mu.Lock()
	defer s.mu.Unlock()
	v := s.state
	if s.process != nil {
		select {
		case <-s.process.Done():
			v.ProcessRunning = false
			v.ListenerReady = false
		default:
			v.ProcessRunning = true
			v.ListenerReady = v.ListenerReady && s.process.ListenerReady(s.listener)
		}
	}
	return v
}
func (s *Supervisor) stateTo(state string) {
	s.mu.Lock()
	defer s.mu.Unlock()
	s.state.State = state
	if s.LogPath != "" {
		s.events = append(s.events, Event{time.Now().UTC(), s.state.ID, state})
		if len(s.events) > 256 {
			s.events = s.events[len(s.events)-256:]
		}
		if e := store.AtomicJSON(s.LogPath, s.events); e != nil {
			s.logErr = domain.Fail(7, "Cannot persist private session events.", "Check state storage; session cleanup was still attempted.")
		}
	}
}
func (s *Supervisor) Run(ctx context.Context, a core.Adapter, o Options, onReady func(domain.Session) error) (err error) {
	// Keep the Linux parent-death signal tied to this supervisor's living OS thread.
	runtime.LockOSThread()
	defer runtime.UnlockOSThread()
	defer func() { s.mu.Lock(); err = errors.Join(err, s.logErr); s.mu.Unlock() }()
	lock, e := store.Acquire(filepath.Join(s.Runtime, "session.lock"))
	if e != nil {
		return e
	}
	defer func() { err = errors.Join(err, store.Release(lock)) }()
	if e = s.cleanup(); e != nil {
		return e
	}
	defer func() { err = errors.Join(err, s.cleanup()) }()
	ctx, cancel := context.WithCancel(ctx)
	defer cancel()
	id, e := domain.NewID()
	if e != nil {
		return e
	}
	s.state = domain.Session{ID: id, State: "preflight", ProfileID: o.Profile.ID, Statistics: "Unavailable"}
	s.listener = o.Listener
	s.stateTo("preflight")
	server, e := s.serve(cancel)
	if e != nil {
		return e
	}
	defer func() {
		if e := server.Close(); e != nil && !errors.Is(e, net.ErrClosed) {
			err = errors.Join(err, domain.Fail(7, "Cannot close the session control socket.", "Inspect private runtime resources and retry stop."))
		}
	}()
	if e = store.AtomicJSON(filepath.Join(s.Runtime, "session.json"), s.snapshot()); e != nil {
		return e
	}
	port := net.JoinHostPort("127.0.0.1", strconv.Itoa(int(o.Listener.Port)))
	reservation, e := net.Listen("tcp", port)
	if e != nil {
		return domain.Fail(6, "The loopback listener port is unavailable.", "Choose another --listen-port or stop the process already using that port.")
	}
	defer reservation.Close()
	preCtx, preCancel := context.WithTimeout(ctx, o.Timeout)
	defer preCancel()
	version, e := a.Version(preCtx)
	if e != nil {
		if ctx.Err() != nil {
			return nil
		}
		return e
	}
	b, e := a.Render(o.Profile, o.Listener)
	if e != nil {
		return e
	}
	f, e := os.CreateTemp(s.Runtime, ".core-config-")
	if e != nil {
		return domain.Unexpected()
	}
	defer func() {
		if e := f.Close(); e != nil {
			err = errors.Join(err, domain.Fail(7, "Cannot close the private core configuration descriptor.", "Inspect operating system file resources."))
		}
	}()
	// Remove the pathname immediately. The open inode is inherited as descriptor 3.
	if e = os.Remove(f.Name()); e != nil {
		return domain.Fail(7, "Cannot unlink the private core configuration.", "Inspect the private runtime directory before retrying.")
	}
	if _, e = f.Write(b); e != nil {
		return domain.Unexpected()
	}
	if e = f.Sync(); e != nil {
		return domain.Unexpected()
	}
	s.stateTo("prepared")
	if e = a.Validate(preCtx, f); e != nil {
		if ctx.Err() != nil {
			return nil
		}
		return e
	}
	if ctx.Err() != nil {
		return nil
	}
	s.stateTo("starting")
	if e = reservation.Close(); e != nil {
		return domain.Unexpected()
	}
	p, e := a.Start(f)
	if e != nil {
		return e
	}
	s.mu.Lock()
	s.process = p
	s.state.ProcessRunning = true
	s.state.CoreVersion = version
	s.state.Listener = port
	s.mu.Unlock()
	defer func() { s.stateTo("stopping"); err = errors.Join(err, p.Stop(3*time.Second)); s.stateTo("idle") }()
	readyCtx, readyCancel := context.WithTimeout(ctx, o.Timeout)
	defer readyCancel()
	if e = p.Ready(readyCtx, o.Listener); e != nil {
		if ctx.Err() != nil {
			return nil
		}
		return e
	}
	if ctx.Err() != nil {
		return nil
	}
	now := time.Now().UTC()
	s.mu.Lock()
	s.state.State = "listener_ready"
	s.state.ListenerReady = true
	s.state.StartedAt = now
	if o.Duration > 0 {
		deadline := now.Add(o.Duration)
		s.state.Deadline = &deadline
	}
	s.mu.Unlock()
	s.stateTo("listener_ready")
	if e = store.AtomicJSON(filepath.Join(s.Runtime, "session.json"), s.snapshot()); e != nil {
		return e
	}
	if e = onReady(s.snapshot()); e != nil {
		return e
	}
	var deadline <-chan time.Time
	var timer *time.Timer
	if o.Duration > 0 {
		timer = time.NewTimer(time.Until(now.Add(o.Duration)))
		defer timer.Stop()
		deadline = timer.C
	}
	select {
	case <-ctx.Done():
		return nil
	case <-deadline:
		return nil
	case <-p.Done():
		if e = p.ExitError(); e != nil {
			return e
		}
		return domain.Fail(5, "The core stopped before the session ended.", "Check the core and upstream availability.")
	}
}
func (s *Supervisor) serve(cancel context.CancelFunc) (*net.UnixListener, error) {
	path := filepath.Join(s.Runtime, "control.sock")
	ln, e := net.ListenUnix("unix", &net.UnixAddr{Name: path, Net: "unix"})
	if e != nil {
		return nil, domain.Fail(6, "Cannot create the private session control socket.", "Use a shorter private --data-dir and check permissions.")
	}
	ln.SetUnlinkOnClose(false)
	if e = os.Chmod(path, 0600); e != nil {
		ln.Close()
		return nil, domain.Unexpected()
	}
	go func() {
		limit := make(chan struct{}, 32)
		for {
			c, e := ln.AcceptUnix()
			if e != nil {
				return
			}
			select {
			case limit <- struct{}{}:
			default:
				c.Close()
				continue
			}
			go func(c *net.UnixConn) {
				defer func() { <-limit }()
				defer c.Close()
				c.SetDeadline(time.Now().Add(2 * time.Second))
				if !platform.PeerAllowed(c) {
					return
				}
				var req struct {
					Action    string `json:"action"`
					SessionID string `json:"session_id,omitempty"`
				}
				if store.DecodeStrict(c, 1024, &req) != nil {
					return
				}
				if req.Action == "stop" {
					if req.SessionID != "" && req.SessionID != s.snapshot().ID {
						json.NewEncoder(c).Encode(s.snapshot())
						return
					}
					s.stateTo("stopping")
					cancel()
				} else if req.Action != "status" {
					return
				}
				json.NewEncoder(c).Encode(s.snapshot())
			}(c)
		}
	}()
	return ln, nil
}
func (s *Supervisor) cleanup() error {
	for _, name := range []string{"control.sock", "session.json"} {
		path := filepath.Join(s.Runtime, name)
		st, e := os.Lstat(path)
		if os.IsNotExist(e) {
			continue
		}
		if e != nil || !platform.CheckOwner(st) || st.Mode()&os.ModeSymlink != 0 {
			return domain.Fail(7, "Cannot safely clean up an owned runtime resource.", "Inspect runtime resource ownership; unrelated resources are never removed.")
		}
		if name == "control.sock" && st.Mode()&os.ModeSocket == 0 || name == "session.json" && !st.Mode().IsRegular() {
			return domain.Fail(7, "An unexpected file occupies an owned runtime resource name.", "Inspect the private runtime directory manually.")
		}
		if e = os.Remove(path); e != nil {
			return domain.Fail(7, "Session cleanup is incomplete.", "Check permissions in the private runtime directory and retry stop.")
		}
	}
	return nil
}
func (s *Supervisor) request(action string) (domain.Session, error) {
	return s.requestID(action, "")
}
func (s *Supervisor) requestID(action, id string) (domain.Session, error) {
	c, e := net.DialTimeout("unix", filepath.Join(s.Runtime, "control.sock"), 500*time.Millisecond)
	if e != nil {
		return domain.Session{}, e
	}
	defer c.Close()
	c.SetDeadline(time.Now().Add(2 * time.Second))
	uc, ok := c.(*net.UnixConn)
	if !ok || !platform.PeerAllowed(uc) {
		return domain.Session{}, domain.Fail(4, "The session control peer is not the current user.", "Inspect private runtime directory ownership.")
	}
	if e = json.NewEncoder(c).Encode(struct {
		Action    string `json:"action"`
		SessionID string `json:"session_id,omitempty"`
	}{action, id}); e != nil {
		return domain.Session{}, e
	}
	if e = uc.CloseWrite(); e != nil {
		return domain.Session{}, e
	}
	var v domain.Session
	e = store.DecodeStrict(c, 8192, &v)
	return v, e
}
func (s *Supervisor) Status() (value domain.Session, err error) {
	if v, e := s.request("status"); e == nil {
		return v, nil
	}
	lock, e := store.Acquire(filepath.Join(s.Runtime, "session.lock"))
	if e != nil {
		return domain.Session{}, domain.Fail(6, "The supervisor owns the session but its control socket is unavailable.", "Wait briefly and retry; no process is signaled using a stored PID.")
	}
	defer func() { err = errors.Join(err, store.Release(lock)) }()
	for _, name := range []string{"control.sock", "session.json"} {
		if _, e = os.Lstat(filepath.Join(s.Runtime, name)); e == nil {
			return domain.Session{State: "cleanup_required", Statistics: "Unavailable"}, nil
		}
		if !os.IsNotExist(e) {
			return domain.Session{}, domain.Fail(7, "Cannot inspect private runtime resources.", "Check filesystem availability and permissions before retrying.")
		}
	}
	return domain.Session{State: "idle", Statistics: "Unavailable"}, nil
}
func (s *Supervisor) Stop() (err error) {
	return s.stop("")
}

// StopSession binds confirmation to a session ID at the authenticated supervisor.
func (s *Supervisor) StopSession(id string) (err error) {
	// No confirmed session must never become a wildcard for a new live session.
	if id == "" {
		id = "no-confirmed-session"
	}
	return s.stop(id)
}
func (s *Supervisor) stop(id string) (err error) {
	if v, e := s.requestID("stop", id); e == nil {
		if id != "" && v.ID != id {
			return domain.Fail(6, "The confirmed session has changed.", "Refresh status and confirm again; the new session was not stopped.")
		}
		deadline := time.Now().Add(10 * time.Second)
		for time.Now().Before(deadline) {
			lock, e := store.Acquire(filepath.Join(s.Runtime, "session.lock"))
			if e == nil {
				defer func() { err = errors.Join(err, store.Release(lock)) }()
				return s.cleanup()
			}
			time.Sleep(50 * time.Millisecond)
		}
		return domain.Fail(7, "The supervisor has not completed stop and cleanup.", "Retry status; do not kill unrelated core processes.")
	}
	lock, e := store.Acquire(filepath.Join(s.Runtime, "session.lock"))
	if e != nil {
		return domain.Fail(6, "The active supervisor cannot be contacted safely.", "Send Ctrl+C in its foreground terminal or retry; no stale PID is signaled.")
	}
	defer func() { err = errors.Join(err, store.Release(lock)) }()
	return s.cleanup()
}
