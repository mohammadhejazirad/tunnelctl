package store

import (
	"strings"
	"testing"
)

func TestRejectAmbiguousJSON(t *testing.T) {
	for _, s := range []string{`{"selected":"one","selected":"two"}`, `{"selected":"one"} {}`, `{"unknown":true}`, strings.Repeat("[", 35) + strings.Repeat("]", 35)} {
		var v struct {
			Selected string `json:"selected"`
		}
		if e := DecodeStrict(strings.NewReader(s), 1024, &v); e == nil {
			t.Fatal("ambiguous JSON accepted")
		}
	}
	var v struct {
		Selected string `json:"selected"`
	}
	if e := DecodeStrict(strings.NewReader(`{"selected":"one"}`), 1024, &v); e != nil || v.Selected != "one" {
		t.Fatal("valid JSON rejected", e)
	}
}
