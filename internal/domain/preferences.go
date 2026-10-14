package domain

import (
	"net/url"
	"path/filepath"
	"strings"
	"time"
)

// Preferences contains defaults for future operations, never applied session state.
// Authentication credentials remain in their existing private file; only its path is stored.
type Preferences struct {
	ListenerAddress    string `json:"listener_address"`
	ListenerPort       uint16 `json:"listener_port"`
	Duration           string `json:"duration"`
	AuthFile           string `json:"auth_file"`
	PreferredMode      string `json:"preferred_mode"`
	Core               string `json:"core"`
	CorePath           string `json:"core_path"`
	Timeout            string `json:"timeout"`
	Color              bool   `json:"color"`
	ProbeTarget        string `json:"probe_target"`
	SubscriptionPolicy string `json:"subscription_policy"`
}

func DefaultPreferences() Preferences {
	return Preferences{ListenerAddress: "127.0.0.1", ListenerPort: 2080, Duration: "0s", PreferredMode: "proxy", Core: "sing-box", Timeout: "10s", Color: true, SubscriptionPolicy: "manual"}
}

func (p Preferences) Validate() error {
	invalid := func() error {
		return Fail(2, "Invalid application settings.", "Use a loopback listener on port 1024–65535, nonnegative duration, timeout in (0,2m], and absolute private file/core paths.")
	}
	if p.ListenerAddress != "127.0.0.1" || p.ListenerPort < 1024 {
		return invalid()
	}
	if p.PreferredMode != "proxy" || p.Core != "sing-box" || p.SubscriptionPolicy != "manual" {
		return Fail(3, "The requested settings capability is unavailable.", "Only Proxy, sing-box and manual subscription updates are implemented; host routing/DNS/IPv6 settings are not changed.")
	}
	d, e := time.ParseDuration(p.Duration)
	if e != nil || d < 0 {
		return invalid()
	}
	d, e = time.ParseDuration(p.Timeout)
	if e != nil || d <= 0 || d > 2*time.Minute {
		return invalid()
	}
	for _, path := range []string{p.AuthFile, p.CorePath} {
		if path != "" && (!filepath.IsAbs(path) || len(path) > 4096 || SafeName(path) != path) {
			return invalid()
		}
	}
	if p.ProbeTarget != "" {
		u, e := url.Parse(p.ProbeTarget)
		if e != nil || len(p.ProbeTarget) > 4096 || SafeName(p.ProbeTarget) != p.ProbeTarget || u.Scheme != "https" || u.Hostname() == "" || u.User != nil || u.RawQuery != "" || u.ForceQuery || u.Fragment != "" || u.Opaque != "" {
			return Fail(2, "Invalid probe destination.", "Use HTTPS without credentials, query or fragment. No request is made when saving.")
		}
	}
	return nil
}

func ValidateTags(tags []string) error {
	if len(tags) > 32 {
		return Fail(2, "Too many profile tags.", "Use at most 32 tags of 1–64 printable bytes.")
	}
	seen := map[string]bool{}
	for _, tag := range tags {
		if tag == "" || len(tag) > 64 || SafeName(tag) != tag || strings.Contains(tag, ",") || seen[tag] {
			return Fail(2, "Invalid profile tags.", "Use unique tags of 1–64 printable bytes without terminal controls.")
		}
		seen[tag] = true
	}
	return nil
}
