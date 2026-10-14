package subscription

import (
	"compress/gzip"
	"context"
	"crypto/tls"
	"crypto/x509"
	"io"
	"net"
	"net/http"
	"net/netip"
	"net/url"
	"os"
	"strings"
	"time"
	"tunnelctl/internal/domain"
	"tunnelctl/internal/store"
	"unicode"
	"unicode/utf8"
)

type FetchResult struct {
	Body         []byte
	ETag         string
	LastModified string
	NotModified  bool
}
type Fetcher struct {
	Lookup func(context.Context, string) ([]net.IPAddr, error)
}

func ValidateSource(s domain.Subscription) error {
	u, e := url.Parse(s.URL)
	if e != nil || !utf8.ValidString(s.URL) || u.Scheme != "https" || u.Hostname() == "" || u.Fragment != "" || u.Opaque != "" || len(s.URL) > 16384 || !domain.ValidHost(u.Hostname()) {
		return domain.Fail(2, "Subscription URL must be a valid HTTPS endpoint.", "Provide its private URL through stdin; fragments and non-HTTPS schemes are unsupported.")
	}
	for _, c := range s.URL {
		if unicode.IsControl(c) || unicode.IsSpace(c) {
			return domain.Fail(2, "Subscription URL contains invalid characters.", "Provide a correctly percent-encoded HTTPS URL.")
		}
	}
	if p := u.Port(); p != "" {
		if port, e := net.LookupPort("tcp", p); e != nil || port < 1 || port > 65535 {
			return domain.Fail(2, "Invalid subscription endpoint port.", "Use a port between 1 and 65535.")
		}
	}
	for _, s := range s.AllowedCIDRs {
		p, e := netip.ParsePrefix(s)
		if e != nil || p != p.Masked() || p.Bits() == 0 {
			return domain.Fail(2, "Invalid private source CIDR exception.", "Use a specific canonical IP network; catch-all CIDRs are prohibited.")
		}
	}
	return nil
}
func forbidden(ip netip.Addr) bool {
	ip = ip.Unmap()
	if !ip.IsGlobalUnicast() || ip.IsPrivate() || ip.IsLoopback() || ip.IsLinkLocalUnicast() {
		return true
	}
	// IPv6 outside the currently assigned global-unicast block is conservative
	// deny-by-default; this includes compatible, translation and discard prefixes.
	if ip.Is6() && !netip.MustParsePrefix("2000::/3").Contains(ip) {
		return true
	}
	for _, s := range []string{"0.0.0.0/8", "100.64.0.0/10", "192.0.0.0/24", "192.0.2.0/24", "192.88.99.0/24", "198.18.0.0/15", "198.51.100.0/24", "203.0.113.0/24", "240.0.0.0/4", "2001::/23", "2001:db8::/32", "2002::/16", "3fff::/20"} {
		if netip.MustParsePrefix(s).Contains(ip) {
			return true
		}
	}
	return false
}
func allowed(ip netip.Addr, prefixes []string) bool {
	ip = ip.Unmap()
	if ip.IsUnspecified() || ip.IsMulticast() {
		return false
	}
	if !forbidden(ip) {
		return true
	}
	for _, s := range prefixes {
		p, e := netip.ParsePrefix(s)
		if e == nil && p.Contains(ip) {
			return true
		}
	}
	return false
}
func (f Fetcher) Fetch(ctx context.Context, s domain.Subscription) (FetchResult, error) {
	if e := ValidateSource(s); e != nil {
		return FetchResult{}, e
	}
	ctx, cancel := context.WithTimeout(ctx, 20*time.Second)
	defer cancel()
	var roots *x509.CertPool
	if s.CAFile != "" {
		file, e := store.OpenPrivate(s.CAFile, os.O_RDONLY)
		if e != nil {
			return FetchResult{}, networkError("Cannot read the private subscription CA file.")
		}
		b, e := io.ReadAll(io.LimitReader(file, 1024*1024+1))
		file.Close()
		if e != nil || len(b) > 1024*1024 {
			return FetchResult{}, networkError("Invalid subscription CA file.")
		}
		roots, _ = x509.SystemCertPool()
		if roots == nil {
			roots = x509.NewCertPool()
		}
		if !roots.AppendCertsFromPEM(b) {
			return FetchResult{}, networkError("Invalid subscription CA certificate.")
		}
	}
	lookup := f.Lookup
	if lookup == nil {
		lookup = net.DefaultResolver.LookupIPAddr
	}
	dialer := net.Dialer{Timeout: 5 * time.Second}
	tr := &http.Transport{Proxy: nil, DisableCompression: true, DisableKeepAlives: true, MaxResponseHeaderBytes: 64 * 1024, TLSHandshakeTimeout: 5 * time.Second, ResponseHeaderTimeout: 10 * time.Second, TLSClientConfig: &tls.Config{MinVersion: tls.VersionTLS12, RootCAs: roots}}
	tr.DialContext = func(ctx context.Context, network, address string) (net.Conn, error) {
		host, port, e := net.SplitHostPort(address)
		if e != nil {
			return nil, networkError("Invalid subscription destination.")
		}
		ips, e := lookup(ctx, host)
		if e != nil || len(ips) == 0 || len(ips) > 64 {
			return nil, networkError("Subscription hostname resolution failed.")
		}
		// Validate every resolved answer, then dial that literal. There is no second DNS lookup.
		for _, a := range ips {
			ip, ok := netip.AddrFromSlice(a.IP)
			if !ok || !allowed(ip, s.AllowedCIDRs) {
				return nil, networkError("Subscription destination is blocked by the private-address policy.")
			}
		}
		for _, a := range ips {
			c, e := dialer.DialContext(ctx, "tcp", net.JoinHostPort(a.IP.String(), port))
			if e == nil {
				tcp, ok := c.RemoteAddr().(*net.TCPAddr)
				if !ok {
					c.Close()
					return nil, networkError("Cannot verify subscription peer address.")
				}
				ip, ok := netip.AddrFromSlice(tcp.IP)
				if !ok || !allowed(ip, s.AllowedCIDRs) {
					c.Close()
					return nil, networkError("Subscription peer address is blocked.")
				}
				return c, nil
			}
		}
		return nil, networkError("Cannot connect to the subscription server.")
	}
	defer tr.CloseIdleConnections()
	original, _ := url.Parse(s.URL)
	client := &http.Client{Transport: tr, CheckRedirect: func(req *http.Request, via []*http.Request) error {
		if len(via) > 3 || req.URL.Scheme != "https" || !strings.EqualFold(req.URL.Host, original.Host) || (req.URL.User != nil && (original.User == nil || req.URL.User.String() != original.User.String())) {
			return networkError("Subscription redirect is outside the allowed HTTPS origin or redirect limit.")
		}
		return nil
	}}
	req, e := http.NewRequestWithContext(ctx, http.MethodGet, s.URL, nil)
	if e != nil {
		return FetchResult{}, networkError("Cannot prepare subscription request.")
	}
	req.Header.Set("Accept-Encoding", "gzip")
	if s.ETag != "" {
		req.Header.Set("If-None-Match", s.ETag)
	}
	if s.LastModified != "" {
		req.Header.Set("If-Modified-Since", s.LastModified)
	}
	res, e := client.Do(req)
	if e != nil {
		if ctx.Err() != nil {
			return FetchResult{}, networkError("Subscription request was canceled or timed out.")
		}
		return FetchResult{}, networkError("Subscription fetch failed; check TLS, DNS and source policy.")
	}
	defer res.Body.Close()
	if res.StatusCode == http.StatusNotModified {
		return FetchResult{NotModified: true}, nil
	}
	if res.StatusCode != http.StatusOK {
		return FetchResult{}, networkError("Subscription server returned an unsuccessful response.")
	}
	if res.ContentLength > MaxWire {
		return FetchResult{}, parseError()
	}
	limited := &io.LimitedReader{R: res.Body, N: MaxWire + 1}
	var reader io.Reader = limited
	switch res.Header.Get("Content-Encoding") {
	case "", "identity":
	case "gzip":
		gz, e := gzip.NewReader(limited)
		if e != nil {
			return FetchResult{}, parseError()
		}
		defer gz.Close()
		reader = gz
	default:
		return FetchResult{}, parseError()
	}
	b, e := io.ReadAll(io.LimitReader(reader, MaxExpanded+1))
	if e != nil || len(b) > MaxExpanded || limited.N <= 0 {
		return FetchResult{}, parseError()
	}
	etag, last := res.Header.Get("ETag"), res.Header.Get("Last-Modified")
	if len(etag) > 4096 || len(last) > 4096 {
		return FetchResult{}, parseError()
	}
	return FetchResult{Body: b, ETag: etag, LastModified: last}, nil
}
func networkError(message string) error {
	return domain.Fail(5, message, "Check the source, TLS trust and explicit private CIDR exceptions; the last good snapshot is preserved.")
}
