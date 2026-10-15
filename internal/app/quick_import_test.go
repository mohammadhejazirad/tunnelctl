//go:build linux

package app

import (
	"encoding/base64"
	"os"
	"strings"
	"testing"
	"time"
)

func quickService(t *testing.T) *Service {
	t.Helper()
	root := t.TempDir()
	if e := os.Chmod(root, 0700); e != nil {
		t.Fatal(e)
	}
	service, e := New(root, "/missing/core", time.Second)
	if e != nil {
		t.Fatal(e)
	}
	if e = service.Init(); e != nil {
		t.Fatal(e)
	}
	return service
}

func TestQuickImportDetectsProfileFeedAndSubscription(t *testing.T) {
	service := quickService(t)
	uuid := "11111111-2222-4333-8444-555555555555"
	result, e := service.QuickImport("vless://" + uuid + "@example.com:443?encryption=none&security=tls&sni=example.com#one")
	if e != nil || result.Kind != "profile" || result.Added != 1 {
		t.Fatal("single profile detection failed", e)
	}
	feed := strings.Join([]string{
		"ss://aes-256-gcm:synthetic@127.0.0.1:8388#two",
		"trojan://synthetic@example.net:443?security=tls&sni=example.net#three",
	}, "\n")
	result, e = service.QuickImport(base64.StdEncoding.EncodeToString([]byte(feed)))
	if e != nil || result.Kind != "profiles" || result.Added != 2 {
		t.Fatal("feed detection failed", e)
	}
	result, e = service.QuickImport("https://example.org/subscription?token=synthetic")
	if e != nil || result.Kind != "subscription" || result.Name != "example.org" {
		t.Fatal("subscription URL detection failed", e)
	}
	profiles, _ := service.List()
	sources, _ := service.Subscriptions().List()
	if len(profiles) != 3 || len(sources) != 1 || !profiles[0].Selected {
		t.Fatal("quick import did not persist expected entries")
	}
}
