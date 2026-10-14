package importer

import (
	"encoding/base64"
	"io"
	"net"
	"net/url"
	"strconv"
	"strings"
	"time"
	"tunnelctl/internal/domain"
	"unicode/utf8"
)

const MaxLinkBytes = 64 * 1024

func Parse(r io.Reader) (domain.Profile, error) {
	p := domain.Profile{}
	data, e := io.ReadAll(io.LimitReader(r, MaxLinkBytes+1))
	if e != nil {
		return p, domain.Fail(2, "Cannot read profile input.", "Provide a readable private file or stdin.")
	}
	if len(data) > MaxLinkBytes {
		return p, domain.Fail(2, "Profile input exceeds 64 KiB.", "Import one share link at a time.")
	}
	if !utf8.Valid(data) {
		return p, invalid()
	}
	s := strings.TrimSpace(string(data))
	if !strings.HasPrefix(s, "ss://") {
		return p, domain.Fail(3, "Only Shadowsocks SIP002 links are supported in this release.", "Use subscription commands for supported feeds; raw configurations and other protocols are unsupported.")
	}
	if strings.ContainsAny(s, "\r\n\t") {
		return p, invalid()
	}
	authority := strings.SplitN(strings.TrimPrefix(s, "ss://"), "#", 2)[0]
	if strings.Count(authority, "@") != 1 {
		return p, invalid()
	}
	u, e := url.Parse(s)
	if e != nil || u.Scheme != "ss" || u.User == nil || u.Opaque != "" || u.ForceQuery {
		return p, invalid()
	}
	if !utf8.ValidString(u.Fragment) {
		return p, invalid()
	}
	if u.Path != "" && u.Path != "/" {
		return p, domain.Fail(3, "A Shadowsocks path is unsupported.", "Use a SIP002 link without transport extensions.")
	}
	if u.RawQuery != "" {
		return p, domain.Fail(3, "Shadowsocks query parameters and plugins are unsupported.", "Remove unsupported extensions at the source; they are never silently discarded.")
	}
	host, port, e := net.SplitHostPort(u.Host)
	if e != nil {
		return p, invalid()
	}
	n, e := strconv.ParseUint(port, 10, 16)
	if e != nil || n == 0 {
		return p, invalid()
	}
	var method, password string
	if pass, ok := u.User.Password(); ok {
		method = u.User.Username()
		password = pass
	} else {
		encoded := u.User.Username()
		var decoded []byte
		for _, enc := range []*base64.Encoding{base64.RawURLEncoding.Strict(), base64.URLEncoding.Strict(), base64.RawStdEncoding.Strict(), base64.StdEncoding.Strict()} {
			decoded, e = enc.DecodeString(encoded)
			if e == nil {
				break
			}
		}
		if e != nil || len(decoded) > 8192 {
			return p, invalid()
		}
		var ok bool
		method, password, ok = strings.Cut(string(decoded), ":")
		if !ok {
			return p, invalid()
		}
	}
	id, e := domain.NewID()
	if e != nil {
		return p, e
	}
	name := domain.SafeName(u.Fragment)
	if len(name) > 256 {
		return p, domain.Fail(2, "Profile name exceeds 256 bytes.", "Use a shorter name.")
	}
	if name == "" {
		name = "Shadowsocks"
	}
	p = domain.Profile{ID: id, Name: name, Protocol: "shadowsocks", Host: strings.ToLower(host), Port: uint16(n), Method: method, Password: password, CreatedAt: time.Now().UTC()}
	if e = p.Validate(); e != nil {
		return domain.Profile{}, e
	}
	return p, nil
}
func invalid() error {
	return domain.Fail(2, "Malformed Shadowsocks SIP002 link.", "Check the URI encoding, credential, host and port; legacy whole-link Base64 is unsupported.")
}
