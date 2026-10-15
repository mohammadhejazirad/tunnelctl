package subscription

import (
	"bytes"
	"encoding/base64"
	"errors"
	"strings"
	"tunnelctl/internal/domain"
	"tunnelctl/internal/importer"
	"unicode/utf8"
)

const MaxWire = 5 * 1024 * 1024
const MaxExpanded = 10 * 1024 * 1024
const MaxProfiles = 10000

func Parse(b []byte) ([]domain.Profile, error) {
	if len(b) > MaxExpanded {
		return nil, parseError()
	}
	b = bytes.TrimSpace(b)
	if bytes.HasPrefix(b, []byte("{")) || bytes.HasPrefix(b, []byte("[")) || bytes.HasPrefix(b, []byte("proxies:")) || bytes.HasPrefix(b, []byte("proxy-providers:")) {
		return nil, domain.Fail(3, "Structured core/Clash subscriptions are unsupported.", "Use a supported share-link list; raw remote configurations are never executed.")
	}
	first, _, _ := bytes.Cut(b, []byte("\n"))
	if !bytes.Contains(first, []byte("://")) {
		compact := bytes.Map(func(r rune) rune {
			if r == '\r' || r == '\n' || r == '\t' || r == ' ' {
				return -1
			}
			return r
		}, b)
		var decoded []byte
		var e error
		for _, enc := range []*base64.Encoding{base64.StdEncoding.Strict(), base64.RawStdEncoding.Strict(), base64.URLEncoding.Strict(), base64.RawURLEncoding.Strict()} {
			decoded, e = enc.DecodeString(string(compact))
			if e == nil {
				break
			}
		}
		if e != nil || len(decoded) > MaxExpanded {
			return nil, parseError()
		}
		b = bytes.TrimSpace(decoded)
	}
	if !utf8.Valid(b) || len(b) == 0 {
		return nil, parseError()
	}
	out := []domain.Profile{}
	identities := map[string]bool{}
	count := 0
	for _, line := range bytes.Split(b, []byte("\n")) {
		line = bytes.TrimSpace(line)
		if len(line) == 0 {
			continue
		}
		count++
		if count > MaxProfiles {
			return nil, parseError()
		}
		p, e := importer.Parse(strings.NewReader(string(line)))
		if e != nil {
			var de *domain.Error
			if errors.As(e, &de) && de.Code == 3 {
				return nil, domain.Fail(3, "Subscription contains an unsupported profile capability.", "Use documented ss, vless, vmess or trojan options; the previous snapshot is preserved.")
			}
			return nil, domain.Fail(2, "Subscription contains an invalid or unsupported share link.", "Use nonempty lists of supported ss, vless, vmess or trojan links; the previous snapshot is preserved.")
		}
		key := p.Identity()
		if !identities[key] {
			identities[key] = true
			out = append(out, p)
		}
	}
	if len(out) == 0 {
		return nil, parseError()
	}
	return out, nil
}
func parseError() error {
	return domain.Fail(2, "The subscription is empty, malformed or exceeds its limits.", "Use UTF-8 newline-separated supported share links or a recognized Base64 envelope; the last good snapshot is preserved.")
}
