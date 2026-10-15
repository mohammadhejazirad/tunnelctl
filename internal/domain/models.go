package domain

import (
	"crypto/rand"
	"encoding/base64"
	"encoding/hex"
	"fmt"
	"net"
	"strconv"
	"strings"
	"time"
	"unicode"
	"unicode/utf8"
)

const SchemaVersion = 4
const OutputSchemaVersion = 1

type Error struct {
	Code        int    `json:"code"`
	Message     string `json:"message"`
	Remediation string `json:"remediation"`
}

func (e *Error) Error() string                         { return e.Message }
func Fail(code int, message, remediation string) error { return &Error{code, message, remediation} }

// Profile is internal, secret-bearing data. Never serialize it in public output.
type Profile struct {
	ID             string    `json:"id"`
	Name           string    `json:"name"`
	Protocol       string    `json:"protocol"`
	Host           string    `json:"host"`
	Port           uint16    `json:"port"`
	Method         string    `json:"method"`
	Password       string    `json:"password"`
	Transport      string    `json:"transport,omitempty"`
	Security       string    `json:"security,omitempty"`
	ServerName     string    `json:"server_name,omitempty"`
	HostHeader     string    `json:"host_header,omitempty"`
	Path           string    `json:"path,omitempty"`
	ServiceName    string    `json:"service_name,omitempty"`
	Flow           string    `json:"flow,omitempty"`
	Fingerprint    string    `json:"fingerprint,omitempty"`
	PublicKey      string    `json:"public_key,omitempty"`
	ShortID        string    `json:"short_id,omitempty"`
	ALPN           []string  `json:"alpn,omitempty"`
	AlterID        int       `json:"alter_id,omitempty"`
	CreatedAt      time.Time `json:"created_at"`
	SubscriptionID string    `json:"subscription_id,omitempty"`
	UserNamed      bool      `json:"user_named,omitempty"`
	Tags           []string  `json:"tags,omitempty"`
	Favorite       bool      `json:"favorite,omitempty"`
}
type PublicProfile struct {
	ID          string   `json:"id"`
	Name        string   `json:"name"`
	Protocol    string   `json:"protocol"`
	Host        string   `json:"host"`
	Port        uint16   `json:"port"`
	Method      string   `json:"method"`
	Transport   string   `json:"transport,omitempty"`
	Security    string   `json:"security,omitempty"`
	Credentials string   `json:"credentials"`
	Selected    bool     `json:"selected"`
	Tags        []string `json:"tags,omitempty"`
	Favorite    bool     `json:"favorite"`
}

func (p Profile) Public(selected bool) PublicProfile {
	return PublicProfile{ID: p.ID, Name: SafeName(p.Name), Protocol: p.Protocol, Host: p.Host, Port: p.Port, Method: p.Method, Transport: p.Transport, Security: p.Security, Credentials: "[REDACTED]", Selected: selected, Tags: append([]string(nil), p.Tags...), Favorite: p.Favorite}
}

type Settings struct {
	SchemaVersion int            `json:"schema_version"`
	Selected      string         `json:"selected"`
	Profiles      []Profile      `json:"profiles"`
	Revision      uint64         `json:"revision,omitempty"`
	Subscriptions []Subscription `json:"subscriptions,omitempty"`
	Preferences   Preferences    `json:"preferences"`
}
type Session struct {
	ID               string     `json:"id,omitempty"`
	State            string     `json:"state"`
	ProfileID        string     `json:"profile_id,omitempty"`
	CoreVersion      string     `json:"core_version,omitempty"`
	Listener         string     `json:"listener,omitempty"`
	StartedAt        time.Time  `json:"started_at,omitempty"`
	Deadline         *time.Time `json:"deadline,omitempty"`
	ProcessRunning   bool       `json:"process_running"`
	ListenerReady    bool       `json:"listener_ready"`
	ConnectionTested bool       `json:"connection_tested"`
	Statistics       string     `json:"statistics"`
}

func NewID() (string, error) {
	var b [16]byte
	if _, e := rand.Read(b[:]); e != nil {
		return "", Fail(1, "Cannot generate a random identifier.", "Check the operating system random source.")
	}
	return hex.EncodeToString(b[:]), nil
}

// Remove whole ANSI/OSC sequences (including their payload), C0/C1 and bidi controls.
func SafeName(s string) string {
	r := []rune(s)
	var b strings.Builder
	for i := 0; i < len(r); i++ {
		c := r[i]
		if c == 27 || c == 0x9b || c == 0x9d {
			kind := c
			if c == 27 {
				i++
				if i >= len(r) {
					break
				}
				kind = r[i]
			}
			if kind == '[' || kind == 0x9b {
				for i++; i < len(r); i++ {
					if r[i] >= 0x40 && r[i] <= 0x7e {
						break
					}
				}
			} else if kind == ']' || kind == 0x9d || kind == 'P' || kind == '_' || kind == '^' {
				for i++; i < len(r); i++ {
					if r[i] == 7 || r[i] == 0x9c {
						break
					}
					if r[i] == 27 && i+1 < len(r) && r[i+1] == '\\' {
						i++
						break
					}
				}
			}
			continue
		}
		if unicode.IsControl(c) || unicode.Is(unicode.Cf, c) || c == unicode.ReplacementChar {
			continue
		}
		b.WriteRune(c)
	}
	return strings.TrimSpace(b.String())
}
func ValidHost(s string) bool {
	if ip := net.ParseIP(s); ip != nil {
		return !ip.IsUnspecified() && !ip.IsMulticast()
	}
	if len(s) == 0 || len(s) > 253 || strings.ContainsAny(s, ":%/@\\") {
		return false
	}
	s = strings.TrimSuffix(s, ".")
	for _, part := range strings.Split(s, ".") {
		if len(part) == 0 || len(part) > 63 || part[0] == '-' || part[len(part)-1] == '-' {
			return false
		}
		for _, c := range part {
			if !(c >= 'a' && c <= 'z' || c >= 'A' && c <= 'Z' || c >= '0' && c <= '9' || c == '-') {
				return false
			}
		}
	}
	return true
}
func SupportedMethod(s string) bool {
	return s == "aes-128-gcm" || s == "aes-256-gcm" || s == "chacha20-ietf-poly1305"
}

func validUUID(s string) bool {
	if len(s) != 36 || s[8] != '-' || s[13] != '-' || s[18] != '-' || s[23] != '-' {
		return false
	}
	compact := strings.ReplaceAll(s, "-", "")
	_, err := hex.DecodeString(compact)
	return err == nil && len(compact) == 32
}

func printableBounded(s string, limit int) bool {
	if len(s) > limit || !utf8.ValidString(s) {
		return false
	}
	for _, c := range s {
		if unicode.IsControl(c) {
			return false
		}
	}
	return true
}

func supportedTransport(s string) bool {
	return s == "" || s == "tcp" || s == "ws" || s == "grpc" || s == "http" || s == "httpupgrade"
}

func supportedFingerprint(s string) bool {
	switch s {
	case "", "chrome", "firefox", "edge", "safari", "ios", "android", "random", "randomized":
		return true
	default:
		return false
	}
}

func (p Profile) validateTransport() error {
	if !supportedTransport(p.Transport) || !printableBounded(p.Path, 4096) || !printableBounded(p.ServiceName, 1024) || !printableBounded(p.HostHeader, 1024) {
		return Fail(3, "This profile transport is unsupported.", "Use TCP, WebSocket, gRPC, HTTP or HTTP Upgrade with bounded printable options.")
	}
	if p.Transport == "grpc" && p.ServiceName == "" {
		return Fail(2, "The gRPC service name is missing.", "Use a share link containing serviceName.")
	}
	if (p.Transport == "ws" || p.Transport == "http" || p.Transport == "httpupgrade") && p.Path != "" && !strings.HasPrefix(p.Path, "/") {
		return Fail(2, "The transport path is invalid.", "Use an absolute path beginning with '/'.")
	}
	return nil
}

func (p Profile) validateTLS() error {
	if p.Security != "" && p.Security != "none" && p.Security != "tls" && p.Security != "reality" {
		return Fail(3, "This profile security mode is unsupported.", "Use none, TLS or Reality; insecure certificate verification is never enabled.")
	}
	if !supportedFingerprint(p.Fingerprint) || !printableBounded(p.ServerName, 253) || !printableBounded(p.PublicKey, 512) || !printableBounded(p.ShortID, 64) {
		return Fail(2, "The TLS or Reality options are invalid.", "Check SNI, fingerprint, public key and short ID values.")
	}
	if p.ServerName != "" && !ValidHost(p.ServerName) {
		return Fail(2, "The TLS server name is invalid.", "Use a valid ASCII hostname or IP address for SNI.")
	}
	if p.Security == "reality" && (p.ServerName == "" || p.PublicKey == "") {
		return Fail(2, "Reality requires SNI and a public key.", "Use a complete Reality share link containing sni and pbk.")
	}
	if p.Security == "reality" {
		decoded, e := base64.RawURLEncoding.Strict().DecodeString(p.PublicKey)
		if e != nil || len(decoded) != 32 || len(p.ShortID)%2 != 0 || len(p.ShortID) > 16 {
			return Fail(2, "The Reality key or short ID is invalid.", "Use a 32-byte Base64URL public key and an even hexadecimal short ID of at most 16 characters.")
		}
		if p.ShortID != "" {
			if _, e = hex.DecodeString(p.ShortID); e != nil {
				return Fail(2, "The Reality short ID is invalid.", "Use an even hexadecimal short ID of at most 16 characters.")
			}
		}
	}
	if len(p.ALPN) > 8 {
		return Fail(2, "Too many ALPN values.", "Keep at most eight ALPN identifiers.")
	}
	for _, value := range p.ALPN {
		if value != "h2" && value != "http/1.1" && value != "h3" {
			return Fail(3, "This ALPN value is unsupported.", "Use h2, http/1.1 or h3.")
		}
	}
	return nil
}

func (p Profile) Validate() error {
	if !ValidHost(p.Host) || p.Port == 0 {
		return Fail(2, "Invalid profile endpoint.", "Use a valid ASCII hostname or IPv4/IPv6 address and a port from 1 to 65535.")
	}
	if p.Password == "" || len(p.Password) > 4096 || !utf8.ValidString(p.Password) {
		return Fail(2, "Invalid credential length or encoding.", "Provide a nonempty UTF-8 credential of at most 4096 bytes.")
	}
	for _, c := range p.Password {
		if unicode.IsControl(c) {
			return Fail(2, "Control characters are not allowed in credentials.", "Use a printable credential.")
		}
	}
	switch p.Protocol {
	case "shadowsocks":
		if !SupportedMethod(p.Method) || p.Transport != "" || p.Security != "" {
			return Fail(3, "This Shadowsocks profile is unsupported.", "Use SIP002 Shadowsocks with a supported AEAD method and no plugin or transport extension.")
		}
		return nil
	case "vless":
		if !validUUID(p.Password) || (p.Method != "" && p.Method != "none") || (p.Flow != "" && p.Flow != "xtls-rprx-vision") {
			return Fail(3, "This VLESS profile is unsupported.", "Use a valid UUID, encryption=none and an optional xtls-rprx-vision flow.")
		}
		if p.Flow != "" && p.Transport != "tcp" {
			return Fail(3, "VLESS Vision requires the TCP transport.", "Use type=tcp with flow=xtls-rprx-vision.")
		}
		if p.Flow != "" && p.Security != "tls" && p.Security != "reality" {
			return Fail(3, "VLESS Vision requires TLS or Reality.", "Use security=tls or security=reality with xtls-rprx-vision.")
		}
	case "vmess":
		if !validUUID(p.Password) || p.AlterID < 0 || p.AlterID > 65535 {
			return Fail(2, "The VMess credential or alter ID is invalid.", "Use a valid UUID and alterId from 0 to 65535.")
		}
		switch p.Method {
		case "auto", "none", "zero", "aes-128-cfb", "aes-128-gcm", "chacha20-poly1305":
		default:
			return Fail(3, "This VMess security method is unsupported.", "Use auto, none, zero, aes-128-cfb, aes-128-gcm or chacha20-poly1305.")
		}
	case "trojan":
		if p.Security != "tls" && p.Security != "reality" {
			return Fail(3, "Trojan requires verified TLS or Reality.", "Use a Trojan share link with security=tls or security=reality.")
		}
	default:
		return Fail(3, "This profile protocol is unsupported.", "Use a supported Shadowsocks, VLESS, VMess or Trojan share link.")
	}
	if e := p.validateTransport(); e != nil {
		return e
	}
	return p.validateTLS()
}
func Endpoint(p Profile) string { return net.JoinHostPort(p.Host, strconv.Itoa(int(p.Port))) }
func Unexpected() error {
	return Fail(1, "The operation failed.", "Check private storage permissions and run doctor.")
}
func (s Session) String() string {
	return fmt.Sprintf("%s; listener=%s; process_running=%t; listener_ready=%t; connection_tested=%t", s.State, s.Listener, s.ProcessRunning, s.ListenerReady, s.ConnectionTested)
}
