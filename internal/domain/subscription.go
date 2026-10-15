package domain

import (
	"crypto/sha256"
	"encoding/hex"
	"encoding/json"
	"time"
)

type Subscription struct {
	ID           string     `json:"id"`
	Name         string     `json:"name"`
	URL          string     `json:"url"`
	AllowedCIDRs []string   `json:"allowed_cidrs,omitempty"`
	CAFile       string     `json:"ca_file,omitempty"`
	Revision     uint64     `json:"revision"`
	PreviewKey   string     `json:"preview_key"`
	ETag         string     `json:"etag,omitempty"`
	LastModified string     `json:"last_modified,omitempty"`
	UpdatedAt    *time.Time `json:"updated_at,omitempty"`
}
type PublicSubscription struct {
	ID           string     `json:"id"`
	Name         string     `json:"name"`
	URL          string     `json:"url"`
	Profiles     int        `json:"profiles"`
	Revision     uint64     `json:"revision"`
	UpdatedAt    *time.Time `json:"updated_at,omitempty"`
	AllowedCIDRs []string   `json:"allowed_cidrs,omitempty"`
}

func (s Subscription) Public(count int) PublicSubscription {
	return PublicSubscription{s.ID, SafeName(s.Name), "[REDACTED]", count, s.Revision, s.UpdatedAt, s.AllowedCIDRs}
}

// Identity is only used inside private persistence/transactions, never in public output.
func (p Profile) Identity() string {
	b, _ := json.Marshal([]any{
		p.Protocol, p.Host, p.Port, p.Method, p.Password, p.Transport, p.Security,
		p.ServerName, p.HostHeader, p.Path, p.ServiceName, p.Flow, p.Fingerprint,
		p.PublicKey, p.ShortID, p.ALPN, p.AlterID,
	})
	h := sha256.Sum256(b)
	return hex.EncodeToString(h[:])
}
