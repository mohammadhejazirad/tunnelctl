//go:build linux

package session

import (
	"context"
	"errors"
	"net"
	"os"
	"path/filepath"
	"testing"
	"time"
	"tunnelctl/internal/core"
	"tunnelctl/internal/domain"
)

type testAdapter struct {
	reject  bool
	hang    bool
	started bool
	p       *testProcess
}

func (*testAdapter) Version(context.Context) (string, error)              { return "test", nil }
func (*testAdapter) Capabilities() core.Capabilities                      { return core.Capabilities{} }
func (*testAdapter) Render(domain.Profile, core.Listener) ([]byte, error) { return []byte(`{}`), nil }
func (a *testAdapter) Validate(context.Context, *os.File) error {
	if a.reject {
		return domain.Fail(3, "Synthetic validation rejection.", "")
	}
	return nil
}
func (a *testAdapter) Start(*os.File) (core.Process, error) {
	a.started = true
	a.p = &testProcess{done: make(chan struct{}), hang: a.hang}
	return a.p, nil
}
func (*testAdapter) Close() error { return nil }

type testProcess struct {
	done    chan struct{}
	hang    bool
	stopped bool
}

func (p *testProcess) Done() <-chan struct{} { return p.done }
func (*testProcess) ExitError() error        { return nil }
func (p *testProcess) Ready(ctx context.Context, _ core.Listener) error {
	if p.hang {
		<-ctx.Done()
		return domain.Fail(5, "Synthetic readiness deadline.", "")
	}
	return nil
}
func (*testProcess) ListenerReady(core.Listener) bool { return true }
func (p *testProcess) Stop(time.Duration) error {
	if !p.stopped {
		p.stopped = true
		close(p.done)
	}
	return nil
}
func setup(t *testing.T) (*Supervisor, Options) {
	t.Helper()
	dir := t.TempDir()
	os.Chmod(dir, 0700)
	l, e := net.Listen("tcp", "127.0.0.1:0")
	if e != nil {
		t.Fatal(e)
	}
	port := l.Addr().(*net.TCPAddr).Port
	l.Close()
	return &Supervisor{Runtime: dir}, Options{Listener: core.Listener{Port: uint16(port)}, Timeout: 30 * time.Millisecond}
}
func TestRejectedConfigAndReadinessDeadlineCleanUp(t *testing.T) {
	for _, reject := range []bool{true, false} {
		s, o := setup(t)
		a := &testAdapter{reject: reject, hang: !reject}
		e := s.Run(context.Background(), a, o, func(domain.Session) error { return nil })
		var de *domain.Error
		if !errors.As(e, &de) || reject && de.Code != 3 || !reject && de.Code != 5 {
			t.Fatal("wrong failure", e)
		}
		if reject && a.started || !reject && !a.p.stopped {
			t.Fatal("unsafe process lifecycle")
		}
		if v, e := s.Status(); e != nil || v.State != "idle" {
			t.Fatal("failed session left resources", e)
		}
		files, _ := os.ReadDir(s.Runtime)
		for _, file := range files {
			if file.Name() != "session.lock" {
				t.Fatal("sensitive temporary file leaked")
			}
		}
	}
}
func TestCleanupConflictIsReported(t *testing.T) {
	s, o := setup(t)
	a := &testAdapter{}
	e := s.Run(context.Background(), a, o, func(domain.Session) error {
		p := filepath.Join(s.Runtime, "session.json")
		if e := os.Remove(p); e != nil {
			return e
		}
		if e := os.Mkdir(p, 0700); e != nil {
			return e
		}
		return domain.Fail(1, "Synthetic operation failure.", "")
	})
	if e == nil || a.p == nil || !a.p.stopped {
		t.Fatal("expected stop and cleanup failure")
	}
	var found bool
	var inspect func(error)
	inspect = func(e error) {
		if de, ok := e.(*domain.Error); ok && de.Code == 7 {
			found = true
		}
		if joined, ok := e.(interface{ Unwrap() []error }); ok {
			for _, part := range joined.Unwrap() {
				inspect(part)
			}
		}
	}
	inspect(e)
	if !found {
		t.Fatal("cleanup failure hidden", e)
	}
	if st, e := os.Stat(filepath.Join(s.Runtime, "session.json")); e != nil || !st.IsDir() {
		t.Fatal("unowned conflicting resource removed")
	}
}
