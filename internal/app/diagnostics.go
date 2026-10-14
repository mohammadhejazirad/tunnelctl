package app

import (
	"context"
	"crypto/tls"
	"errors"
	"io"
	"net"
	"net/http"
	"net/url"
	"os"
	"strings"
	"time"
	"tunnelctl/internal/core"
	"tunnelctl/internal/core/singbox"
	"tunnelctl/internal/domain"
	"tunnelctl/internal/store"
)

// CheckProfile validates a generated configuration with the approved core, without
// opening a listener or contacting the upstream. Config bytes never leave here.
func (s *Service) CheckProfile(ctx context.Context, key string, l core.Listener) (err error) {
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
	defer func() {
		if a.Close() != nil {
			err = errors.Join(err, domain.Fail(7, "Cannot close the verified core.", "Inspect local file resources."))
		}
	}()
	ctx, cancel := context.WithTimeout(ctx, s.Timeout)
	defer cancel()
	if _, e = a.Version(ctx); e != nil {
		return e
	}
	b, e := a.Render(p, l)
	if e != nil {
		return e
	}
	f, e := os.CreateTemp(s.Store.Paths.Runtime, ".core-config-")
	if e != nil {
		return domain.Unexpected()
	}
	defer func() {
		if f.Close() != nil {
			err = errors.Join(err, domain.Fail(7, "Cannot close generated configuration.", "Inspect local file resources."))
		}
	}()
	if e = os.Remove(f.Name()); e != nil {
		return domain.Fail(7, "Cannot unlink generated configuration.", "Inspect private runtime resources.")
	}
	if _, e = f.Write(b); e != nil {
		return domain.Unexpected()
	}
	if e = f.Sync(); e != nil {
		return domain.Unexpected()
	}
	return a.Validate(ctx, f)
}

type ProbeResult struct {
	ConnectionTested bool `json:"connection_tested"`
	HTTPStatus       int  `json:"http_status"`
}

func (s *Service) ProbeSession(ctx context.Context, expectedID, target, authFile string) (ProbeResult, error) {
	state, e := s.Status()
	if e != nil {
		return ProbeResult{}, e
	}
	if expectedID == "" || state.ID != expectedID || !state.ListenerReady {
		return ProbeResult{}, domain.Fail(6, "No matching ready session is available.", "Refresh status and explicitly retry the request.")
	}
	l, e := AuthFile(authFile, core.Listener{})
	if e != nil {
		return ProbeResult{}, e
	}
	r, e := Probe(ctx, state.Listener, target, l)
	if e != nil {
		return r, e
	}
	current, e := s.Status()
	if e != nil {
		return ProbeResult{}, e
	}
	if current.ID != state.ID || !current.ListenerReady {
		return ProbeResult{}, domain.Fail(6, "The session changed during the probe.", "Refresh status; this result was not persisted.")
	}
	return r, nil
}

// Probe makes one explicit bounded HTTPS request via the specified loopback proxy.
// It never bypasses the proxy, follows redirects or persists fabricated metrics.
func Probe(ctx context.Context, listener, target string, l core.Listener) (ProbeResult, error) {
	u, e := url.Parse(target)
	if e != nil || len(target) > 4096 || u.Scheme != "https" || u.Hostname() == "" || u.User != nil || u.Fragment != "" || u.RawQuery != "" || !domain.ValidHost(u.Hostname()) || strings.ContainsAny(target, "\r\n\t") {
		return ProbeResult{}, domain.Fail(2, "Probe target must be an HTTPS URL without credentials, query or fragment.", "Use a small explicitly chosen HTTPS destination.")
	}
	host, _, e := net.SplitHostPort(listener)
	if e != nil || host != "127.0.0.1" {
		return ProbeResult{}, domain.Fail(2, "Probe requires an owned loopback listener.", "Start proxy mode first.")
	}
	proxy, _ := url.Parse("http://" + listener)
	if l.Username != "" {
		proxy.User = url.UserPassword(l.Username, l.Password)
	}
	ctx, cancel := context.WithTimeout(ctx, 10*time.Second)
	defer cancel()
	tr := &http.Transport{Proxy: http.ProxyURL(proxy), DisableCompression: true, DisableKeepAlives: true, TLSHandshakeTimeout: 5 * time.Second, ResponseHeaderTimeout: 8 * time.Second, MaxResponseHeaderBytes: 16 * 1024, TLSClientConfig: &tls.Config{MinVersion: tls.VersionTLS12}}
	defer tr.CloseIdleConnections()
	client := &http.Client{Transport: tr, CheckRedirect: func(*http.Request, []*http.Request) error { return http.ErrUseLastResponse }}
	req, e := http.NewRequestWithContext(ctx, http.MethodGet, target, nil)
	if e != nil {
		return ProbeResult{}, domain.Unexpected()
	}
	res, e := client.Do(req)
	if e != nil {
		return ProbeResult{}, domain.NetworkFail(domain.FailureCategory(e), "The explicit proxy request failed; upstream authentication cannot be inferred from a timeout.")
	}
	defer res.Body.Close()
	// Bound bandwidth. Do not print response bodies or headers.
	_, e = io.Copy(io.Discard, io.LimitReader(res.Body, 4096))
	if e != nil {
		return ProbeResult{}, domain.NetworkFail(domain.FailureCategory(e), "The explicit proxy response could not be read.")
	}
	if res.StatusCode < 200 || res.StatusCode >= 300 {
		return ProbeResult{HTTPStatus: res.StatusCode}, domain.NetworkFail("http_response", "The probe destination returned a non-success HTTP response.")
	}
	return ProbeResult{true, res.StatusCode}, nil
}
