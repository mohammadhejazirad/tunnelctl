package app

import (
	"context"
	"time"
	"tunnelctl/internal/core/singbox"
)

// VerifyCore does not connect or change host networking.
func (s *Service) VerifyCore(ctx context.Context) (map[string]any, error) {
	a, e := singbox.Open(s.CorePath)
	if e != nil {
		return nil, e
	}
	defer a.Close()
	ctx, cancel := context.WithTimeout(ctx, s.Timeout)
	defer cancel()
	version, e := a.Version(ctx)
	if e != nil {
		return nil, e
	}
	return map[string]any{"name": "sing-box", "version": version, "sha256": singbox.LinuxAMD64SHA256, "integrity": "matched approved executable digest", "capabilities": a.Capabilities(), "verified_at": time.Now().UTC()}, nil
}
