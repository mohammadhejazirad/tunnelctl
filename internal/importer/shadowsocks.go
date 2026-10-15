package importer

import (
	"bytes"
	"encoding/base64"
	"encoding/json"
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
	data, e := io.ReadAll(io.LimitReader(r, MaxLinkBytes+1))
	if e != nil {
		return domain.Profile{}, domain.Fail(2, "Cannot read profile input.", "Provide a readable private file, clipboard value or stdin.")
	}
	if len(data) > MaxLinkBytes {
		return domain.Profile{}, domain.Fail(2, "Profile input exceeds 64 KiB.", "Import one share link at a time.")
	}
	if !utf8.Valid(data) {
		return domain.Profile{}, invalid("share")
	}
	s := strings.TrimSpace(string(data))
	if strings.ContainsAny(s, "\r\n\t") {
		return domain.Profile{}, invalid("share")
	}
	switch {
	case strings.HasPrefix(s, "ss://"):
		return parseShadowsocks(s)
	case strings.HasPrefix(s, "vless://"):
		return parseVLESS(s)
	case strings.HasPrefix(s, "vmess://"):
		return parseVMess(s)
	case strings.HasPrefix(s, "trojan://"):
		return parseTrojan(s)
	default:
		return domain.Profile{}, domain.Fail(3, "The share-link protocol is unsupported.", "Use a supported Shadowsocks, VLESS, VMess or Trojan share link.")
	}
}

func parseShadowsocks(s string) (domain.Profile, error) {
	authority := strings.SplitN(strings.TrimPrefix(s, "ss://"), "#", 2)[0]
	if strings.Count(authority, "@") != 1 {
		return domain.Profile{}, invalid("Shadowsocks")
	}
	u, e := url.Parse(s)
	if e != nil || u.Scheme != "ss" || u.User == nil || u.Opaque != "" || u.ForceQuery {
		return domain.Profile{}, invalid("Shadowsocks")
	}
	if !utf8.ValidString(u.Fragment) || (u.Path != "" && u.Path != "/") {
		return domain.Profile{}, invalid("Shadowsocks")
	}
	if u.RawQuery != "" {
		return domain.Profile{}, domain.Fail(3, "Shadowsocks plugins and query extensions are unsupported.", "Use a SIP002 link without plugin or transport parameters.")
	}
	host, port, e := endpoint(u)
	if e != nil {
		return domain.Profile{}, invalid("Shadowsocks")
	}
	var method, password string
	if pass, ok := u.User.Password(); ok {
		method, password = u.User.Username(), pass
	} else {
		decoded, e := decodeBase64(u.User.Username(), 8192)
		if e != nil {
			return domain.Profile{}, invalid("Shadowsocks")
		}
		var ok bool
		method, password, ok = strings.Cut(string(decoded), ":")
		if !ok {
			return domain.Profile{}, invalid("Shadowsocks")
		}
	}
	return finish(domain.Profile{Name: name(u.Fragment, "Shadowsocks"), Protocol: "shadowsocks", Host: host, Port: port, Method: method, Password: password})
}

func parseVLESS(s string) (domain.Profile, error) {
	u, e := url.Parse(s)
	if e != nil || u.Scheme != "vless" || u.User == nil || u.User.Username() == "" || u.Opaque != "" || (u.Path != "" && u.Path != "/") {
		return domain.Profile{}, invalid("VLESS")
	}
	if _, ok := u.User.Password(); ok {
		return domain.Profile{}, invalid("VLESS")
	}
	host, port, e := endpoint(u)
	if e != nil {
		return domain.Profile{}, invalid("VLESS")
	}
	q, e := query(u, "encryption", "flow", "type", "security", "sni", "fp", "pbk", "sid", "path", "host", "serviceName", "alpn", "headerType", "mode", "insecure", "allowInsecure", "skip-cert-verify")
	if e != nil || insecure(q) {
		return domain.Profile{}, unsafeOrInvalid("VLESS", e)
	}
	transportName, e := transport(q.Get("type"))
	if e != nil || unsupportedHeader(q.Get("headerType")) || unsupportedGRPCMode(q.Get("mode"), transportName) {
		return domain.Profile{}, invalid("VLESS")
	}
	security := strings.ToLower(q.Get("security"))
	if security == "" {
		security = "none"
	}
	method := strings.ToLower(q.Get("encryption"))
	if method == "" {
		method = "none"
	}
	p := domain.Profile{
		Name: name(u.Fragment, "VLESS"), Protocol: "vless", Host: host, Port: port,
		Method: method, Password: strings.ToLower(u.User.Username()), Transport: transportName,
		Security: security, ServerName: q.Get("sni"), HostHeader: q.Get("host"), Path: q.Get("path"),
		ServiceName: q.Get("serviceName"), Flow: q.Get("flow"), Fingerprint: strings.ToLower(q.Get("fp")),
		PublicKey: q.Get("pbk"), ShortID: q.Get("sid"), ALPN: splitALPN(q.Get("alpn")),
	}
	defaultServerName(&p)
	return finish(p)
}

func parseTrojan(s string) (domain.Profile, error) {
	u, e := url.Parse(s)
	if e != nil || u.Scheme != "trojan" || u.User == nil || u.User.Username() == "" || u.Opaque != "" || (u.Path != "" && u.Path != "/") {
		return domain.Profile{}, invalid("Trojan")
	}
	if _, ok := u.User.Password(); ok {
		return domain.Profile{}, invalid("Trojan")
	}
	host, port, e := endpoint(u)
	if e != nil {
		return domain.Profile{}, invalid("Trojan")
	}
	q, e := query(u, "type", "security", "sni", "fp", "pbk", "sid", "path", "host", "serviceName", "alpn", "headerType", "mode", "insecure", "allowInsecure", "skip-cert-verify")
	if e != nil || insecure(q) {
		return domain.Profile{}, unsafeOrInvalid("Trojan", e)
	}
	transportName, e := transport(q.Get("type"))
	if e != nil || unsupportedHeader(q.Get("headerType")) || unsupportedGRPCMode(q.Get("mode"), transportName) {
		return domain.Profile{}, invalid("Trojan")
	}
	security := strings.ToLower(q.Get("security"))
	if security == "" {
		security = "tls"
	}
	p := domain.Profile{
		Name: name(u.Fragment, "Trojan"), Protocol: "trojan", Host: host, Port: port,
		Method: "tls", Password: u.User.Username(), Transport: transportName, Security: security,
		ServerName: q.Get("sni"), HostHeader: q.Get("host"), Path: q.Get("path"),
		ServiceName: q.Get("serviceName"), Fingerprint: strings.ToLower(q.Get("fp")),
		PublicKey: q.Get("pbk"), ShortID: q.Get("sid"), ALPN: splitALPN(q.Get("alpn")),
	}
	defaultServerName(&p)
	return finish(p)
}

func parseVMess(s string) (domain.Profile, error) {
	raw, e := decodeBase64(strings.TrimPrefix(s, "vmess://"), MaxLinkBytes)
	if e != nil || !utf8.Valid(raw) {
		return domain.Profile{}, invalid("VMess")
	}
	dec := json.NewDecoder(bytes.NewReader(raw))
	dec.UseNumber()
	var values map[string]json.RawMessage
	if e = dec.Decode(&values); e != nil || len(values) == 0 || dec.Decode(&struct{}{}) != io.EOF {
		return domain.Profile{}, invalid("VMess")
	}
	allowed := map[string]bool{"v": true, "ps": true, "add": true, "port": true, "id": true, "aid": true, "scy": true, "net": true, "type": true, "host": true, "path": true, "tls": true, "sni": true, "alpn": true, "fp": true, "skip-cert-verify": true, "allowInsecure": true}
	for key := range values {
		if !allowed[key] {
			return domain.Profile{}, domain.Fail(3, "The VMess link contains unsupported options.", "Export a standard VMess link using TCP, WebSocket, gRPC, HTTP or HTTP Upgrade.")
		}
	}
	if rawBool(values["skip-cert-verify"]) || rawBool(values["allowInsecure"]) {
		return domain.Profile{}, insecureError()
	}
	host, e := rawString(values["add"])
	if e != nil {
		return domain.Profile{}, invalid("VMess")
	}
	portText, e := rawString(values["port"])
	if e != nil {
		return domain.Profile{}, invalid("VMess")
	}
	n, e := strconv.ParseUint(portText, 10, 16)
	if e != nil || n == 0 {
		return domain.Profile{}, invalid("VMess")
	}
	transportName, e := rawString(values["net"])
	if e != nil {
		return domain.Profile{}, invalid("VMess")
	}
	transportName, e = transport(transportName)
	if e != nil {
		return domain.Profile{}, invalid("VMess")
	}
	headerType, _ := rawString(values["type"])
	if unsupportedHeader(headerType) {
		return domain.Profile{}, invalid("VMess")
	}
	id, _ := rawString(values["id"])
	method, _ := rawString(values["scy"])
	if method == "" {
		method = "auto"
	}
	aidText, _ := rawString(values["aid"])
	alterID := 0
	if aidText != "" {
		value, err := strconv.ParseUint(aidText, 10, 16)
		if err != nil {
			return domain.Profile{}, invalid("VMess")
		}
		alterID = int(value)
	}
	security, _ := rawString(values["tls"])
	security = strings.ToLower(security)
	if security == "" {
		security = "none"
	}
	profileName, _ := rawString(values["ps"])
	sni, _ := rawString(values["sni"])
	hostHeader, _ := rawString(values["host"])
	path, _ := rawString(values["path"])
	fingerprint, _ := rawString(values["fp"])
	alpn, _ := rawString(values["alpn"])
	p := domain.Profile{
		Name: name(profileName, "VMess"), Protocol: "vmess", Host: strings.ToLower(host), Port: uint16(n),
		Method: strings.ToLower(method), Password: strings.ToLower(id), Transport: transportName, Security: security,
		ServerName: sni, HostHeader: hostHeader, Path: path, Fingerprint: strings.ToLower(fingerprint),
		ALPN: splitALPN(alpn), AlterID: alterID,
	}
	if transportName == "grpc" {
		p.ServiceName = path
		p.Path = ""
	}
	defaultServerName(&p)
	return finish(p)
}

func finish(p domain.Profile) (domain.Profile, error) {
	id, e := domain.NewID()
	if e != nil {
		return domain.Profile{}, e
	}
	p.ID = id
	p.Host = strings.ToLower(p.Host)
	p.CreatedAt = time.Now().UTC()
	if e = p.Validate(); e != nil {
		return domain.Profile{}, e
	}
	return p, nil
}

func endpoint(u *url.URL) (string, uint16, error) {
	host, port, e := net.SplitHostPort(u.Host)
	if e != nil {
		return "", 0, e
	}
	n, e := strconv.ParseUint(port, 10, 16)
	if e != nil || n == 0 {
		return "", 0, strconv.ErrSyntax
	}
	return strings.ToLower(host), uint16(n), nil
}

func query(u *url.URL, allowed ...string) (url.Values, error) {
	q, e := url.ParseQuery(u.RawQuery)
	if e != nil {
		return nil, e
	}
	set := map[string]bool{}
	for _, key := range allowed {
		set[key] = true
	}
	for key, values := range q {
		if !set[key] || len(values) != 1 {
			return nil, strconv.ErrSyntax
		}
	}
	return q, nil
}

func transport(value string) (string, error) {
	switch strings.ToLower(value) {
	case "", "tcp", "raw":
		return "tcp", nil
	case "ws", "grpc", "http", "httpupgrade":
		return strings.ToLower(value), nil
	default:
		return "", strconv.ErrSyntax
	}
}

func unsupportedHeader(value string) bool {
	value = strings.ToLower(value)
	return value != "" && value != "none"
}

func unsupportedGRPCMode(value, transportName string) bool {
	value = strings.ToLower(value)
	return value != "" && !(transportName == "grpc" && (value == "gun" || value == "multi"))
}

func insecure(q url.Values) bool {
	for _, key := range []string{"insecure", "allowInsecure", "skip-cert-verify"} {
		if value := strings.ToLower(q.Get(key)); value == "1" || value == "true" || value == "yes" {
			return true
		}
	}
	return false
}

func insecureError() error {
	return domain.Fail(3, "Insecure TLS verification is refused.", "Use a share link with certificate verification enabled.")
}

func unsafeOrInvalid(protocol string, e error) error {
	if e == nil {
		return insecureError()
	}
	return invalid(protocol)
}

func splitALPN(value string) []string {
	if strings.TrimSpace(value) == "" {
		return nil
	}
	parts := strings.Split(value, ",")
	out := make([]string, 0, len(parts))
	for _, part := range parts {
		if part = strings.TrimSpace(part); part != "" {
			out = append(out, part)
		}
	}
	return out
}

func defaultServerName(p *domain.Profile) {
	if (p.Security == "tls" || p.Security == "reality") && p.ServerName == "" {
		p.ServerName = p.Host
	}
}

func name(value, fallback string) string {
	value = domain.SafeName(value)
	if value == "" {
		return fallback
	}
	if len(value) > 256 {
		for len(value) > 256 {
			_, size := utf8.DecodeLastRuneInString(value)
			value = value[:len(value)-size]
		}
	}
	return value
}

func decodeBase64(value string, limit int) ([]byte, error) {
	if len(value) > MaxLinkBytes {
		return nil, strconv.ErrSyntax
	}
	var last error
	for _, enc := range []*base64.Encoding{base64.RawURLEncoding.Strict(), base64.URLEncoding.Strict(), base64.RawStdEncoding.Strict(), base64.StdEncoding.Strict()} {
		decoded, e := enc.DecodeString(value)
		if e == nil {
			if len(decoded) > limit {
				return nil, strconv.ErrRange
			}
			return decoded, nil
		}
		last = e
	}
	return nil, last
}

func rawString(raw json.RawMessage) (string, error) {
	if len(raw) == 0 || bytes.Equal(raw, []byte("null")) {
		return "", nil
	}
	var text string
	if json.Unmarshal(raw, &text) == nil {
		return text, nil
	}
	var number json.Number
	if json.Unmarshal(raw, &number) == nil {
		return number.String(), nil
	}
	return "", strconv.ErrSyntax
}

func rawBool(raw json.RawMessage) bool {
	var value bool
	if json.Unmarshal(raw, &value) == nil {
		return value
	}
	text, _ := rawString(raw)
	text = strings.ToLower(text)
	return text == "1" || text == "true" || text == "yes"
}

func invalid(protocol string) error {
	return domain.Fail(2, "Malformed or unsupported "+protocol+" share link.", "Check the credential, endpoint, transport, TLS and URI encoding without disabling certificate verification.")
}
