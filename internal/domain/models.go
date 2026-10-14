package domain

import (
	"crypto/rand"
	"encoding/hex"
	"fmt"
	"net"
	"strconv"
	"strings"
	"time"
	"unicode"
	"unicode/utf8"
)

const SchemaVersion = 3
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
	Credentials string   `json:"credentials"`
	Selected    bool     `json:"selected"`
	Tags        []string `json:"tags,omitempty"`
	Favorite    bool     `json:"favorite"`
}

func (p Profile) Public(selected bool) PublicProfile {
	return PublicProfile{ID: p.ID, Name: SafeName(p.Name), Protocol: p.Protocol, Host: p.Host, Port: p.Port, Method: p.Method, Credentials: "[REDACTED]", Selected: selected, Tags: append([]string(nil), p.Tags...), Favorite: p.Favorite}
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
func (p Profile) Validate() error {
	if !ValidHost(p.Host) || p.Port == 0 {
		return Fail(2, "Invalid profile endpoint.", "Use a valid ASCII hostname or IPv4/IPv6 address and a port from 1 to 65535.")
	}
	if p.Protocol != "shadowsocks" || !SupportedMethod(p.Method) {
		return Fail(3, "This profile capability is unsupported by the pinned sing-box adapter.", "Use SIP002 Shadowsocks with aes-128-gcm, aes-256-gcm or chacha20-ietf-poly1305 and no plugin.")
	}
	if p.Password == "" || len(p.Password) > 4096 || !utf8.ValidString(p.Password) {
		return Fail(2, "Invalid credential length or encoding.", "Provide a nonempty UTF-8 credential of at most 4096 bytes.")
	}
	for _, c := range p.Password {
		if unicode.IsControl(c) {
			return Fail(2, "Control characters are not allowed in credentials.", "Use a printable credential.")
		}
	}
	return nil
}
func Endpoint(p Profile) string { return net.JoinHostPort(p.Host, strconv.Itoa(int(p.Port))) }
func Unexpected() error {
	return Fail(1, "The operation failed.", "Check private storage permissions and run doctor.")
}
func (s Session) String() string {
	return fmt.Sprintf("%s; listener=%s; process_running=%t; listener_ready=%t; connection_tested=%t", s.State, s.Listener, s.ProcessRunning, s.ListenerReady, s.ConnectionTested)
}
