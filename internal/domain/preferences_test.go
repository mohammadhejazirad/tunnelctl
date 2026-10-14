package domain

import "testing"

func TestPreferencesRejectUnsupportedAndUnsafeDefaults(t *testing.T) {
	if e := DefaultPreferences().Validate(); e != nil {
		t.Fatal(e)
	}
	edits := []func(*Preferences){
		func(p *Preferences) { p.ListenerAddress = "0.0.0.0" },
		func(p *Preferences) { p.ListenerPort = 80 },
		func(p *Preferences) { p.PreferredMode = "tun" },
		func(p *Preferences) { p.Core = "xray" },
		func(p *Preferences) { p.SubscriptionPolicy = "scheduled" },
		func(p *Preferences) { p.CorePath = "./relative" },
		func(p *Preferences) { p.AuthFile = "./relative" },
		func(p *Preferences) { p.CorePath = "/private/\x1b]52;c;injected\a" },
		func(p *Preferences) { p.Duration = "-1s" },
		func(p *Preferences) { p.Timeout = "0s" },
		func(p *Preferences) { p.Timeout = "3m" },
		func(p *Preferences) { p.ProbeTarget = "http://example.invalid" },
		func(p *Preferences) { p.ProbeTarget = "https://user:synthetic@example.invalid" },
		func(p *Preferences) { p.ProbeTarget = "https://example.invalid/?token=synthetic" },
		func(p *Preferences) { p.ProbeTarget = "https://example.invalid/#synthetic" },
	}
	for i, edit := range edits {
		p := DefaultPreferences()
		edit(&p)
		if p.Validate() == nil {
			t.Fatalf("unsafe/unsupported settings accepted: case %d", i)
		}
	}
	for _, tags := range [][]string{{""}, {"duplicate", "duplicate"}, {"comma,tag"}, {"bad\x1b[31m"}} {
		if ValidateTags(tags) == nil {
			t.Fatal("invalid metadata accepted")
		}
	}
}
