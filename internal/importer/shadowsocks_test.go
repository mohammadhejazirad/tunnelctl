package importer

import (
	"encoding/base64"
	"encoding/json"
	"strings"
	"testing"
	"tunnelctl/internal/domain"
)

func TestSIP002(t *testing.T) {
	for _, enc := range []*base64.Encoding{base64.StdEncoding, base64.RawStdEncoding, base64.URLEncoding, base64.RawURLEncoding} {
		s := "ss://" + enc.EncodeToString([]byte("aes-256-gcm:synthetic:password")) + "@[::1]:443/#%D9%86%D8%A7%D9%85"
		p, e := Parse(strings.NewReader(s))
		if e != nil {
			t.Fatal(e)
		}
		if p.Host != "::1" || p.Port != 443 || p.Password != "synthetic:password" || p.Name != "نام" || p.Method != "aes-256-gcm" {
			t.Fatal("important profile fields were not preserved")
		}
	}
	p, e := Parse(strings.NewReader("ss://chacha20-ietf-poly1305:synthetic%40%3Asecret@EXAMPLE.COM:8388#demo"))
	if e != nil {
		t.Fatal(e)
	}
	if p.Password != "synthetic@:secret" || p.Host != "example.com" {
		t.Fatal("percent encoding or canonical host lost")
	}
}

func TestVLESSVMessAndTrojan(t *testing.T) {
	uuid := "11111111-2222-4333-8444-555555555555"
	vless := "vless://" + uuid + "@example.com:443?encryption=none&security=reality&type=grpc&serviceName=edge&sni=cdn.example.com&fp=chrome&pbk=7E7FjYek1uZt0Wj8KpQDz-yKS0PjWRqRR6B2s8Lh_GU&sid=abcd#vless-demo"
	p, e := Parse(strings.NewReader(vless))
	if e != nil || p.Protocol != "vless" || p.Security != "reality" || p.Transport != "grpc" || p.ServiceName != "edge" || p.PublicKey == "" {
		t.Fatal("VLESS fields were not preserved", e)
	}
	vmessJSON, _ := json.Marshal(map[string]any{"v": "2", "ps": "vmess-demo", "add": "EXAMPLE.COM", "port": "443", "id": uuid, "aid": "0", "scy": "auto", "net": "ws", "type": "none", "host": "cdn.example.com", "path": "/socket", "tls": "tls", "sni": "cdn.example.com", "fp": "firefox", "alpn": "h2,http/1.1"})
	vmess := "vmess://" + base64.RawStdEncoding.EncodeToString(vmessJSON)
	p, e = Parse(strings.NewReader(vmess))
	if e != nil || p.Protocol != "vmess" || p.Host != "example.com" || p.Transport != "ws" || p.Path != "/socket" || len(p.ALPN) != 2 {
		t.Fatal("VMess fields were not preserved", e)
	}
	trojan := "trojan://encoded%40password@example.com:443?security=tls&type=ws&path=%2Ftrojan&host=edge.example.com&sni=edge.example.com#trojan-demo"
	p, e = Parse(strings.NewReader(trojan))
	if e != nil || p.Protocol != "trojan" || p.Password != "encoded@password" || p.Transport != "ws" || p.Security != "tls" {
		t.Fatal("Trojan fields were not preserved", e)
	}
}
func TestRejectMalformedAndUnsupported(t *testing.T) {
	tooLargeDecoded := "ss://" + base64.RawURLEncoding.EncodeToString([]byte("aes-256-gcm:"+strings.Repeat("p", 9000))) + "@127.0.0.1:80"
	if _, e := Parse(strings.NewReader(tooLargeDecoded)); e == nil {
		t.Fatal("decoded size limit not enforced")
	}
	if _, e := Parse(strings.NewReader("ss://aes-256-gcm:secret@host:80@other:80")); e == nil {
		t.Fatal("duplicate authority delimiter accepted")
	}
	for _, s := range []string{"", "vmess://synthetic", "vless://bad@example.com:443", "trojan://secret@example.com:443?allowInsecure=1", "ss://bad@127.0.0.1:80", "ss://aes-256-gcm:secret@host:0", "ss://aes-256-gcm:secret@host:65536", "ss://aes-256-gcm:secret@host:80?plugin=unknown", "ss://aes-256-gcm:secret@host:80?security=tls&security=none", "ss://none:secret@host:80", "ss://aes-256-gcm:secret@host:80/extra", "ss://aes-256-gcm:%0Asecret@host:80", "ss://aes-256-gcm:secret@[fe80::1%25eth0]:80", "ss://aes-256-gcm:secret@bad_host:80", "ss://aes-256-gcm:secret@host:80\nss://another", "ss://aes-256-gcm:@host:80", "ss://aes-256-gcm:secret@host:80?", "ss://aes-256-gcm:secret@host:80#%FF", strings.Repeat("a", MaxLinkBytes+1)} {
		_, e := Parse(strings.NewReader(s))
		if e == nil {
			t.Errorf("accepted malformed or unsupported input (length %d)", len(s))
		}
		if e != nil && strings.Contains(e.Error(), "secret") {
			t.Fatal("credential leaked through parser error")
		}
	}
}
func TestNameSanitizationAndRedaction(t *testing.T) {
	s := "نام\x1b]52;c;secret\x07 safe\x1b[31m\u202etext\x00"
	if v := domain.SafeName(s); v != "نام safetext" {
		t.Fatalf("unsafe terminal name: %q", v)
	}
}
func FuzzParse(f *testing.F) {
	f.Add("ss://aes-256-gcm:synthetic@127.0.0.1:8388#test")
	f.Add("vless://11111111-2222-4333-8444-555555555555@example.com:443?encryption=none&security=tls")
	f.Add("ss://malformed")
	f.Fuzz(func(t *testing.T, s string) {
		p, e := Parse(strings.NewReader(s))
		if e == nil {
			if p.Validate() != nil || p.Name != domain.SafeName(p.Name) || len(p.Name) > 256 {
				t.Fatal("invalid accepted profile")
			}
		}
	})
}
