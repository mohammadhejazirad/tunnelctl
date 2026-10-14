package packaging

import (
	"strconv"
	"strings"
	"testing"
)

func TestNativeOrdering(t *testing.T) {
	previous := -1
	for _, tag := range []string{"v0.2.0-alpha.1", "v0.2.0-alpha.199", "v0.2.0-beta.1", "v0.2.0-beta.199", "v0.2.0-rc.1", "v0.2.0-rc.199", "v0.2.0", "v0.2.1-alpha.1"} {
		v, e := Parse(tag)
		if e != nil {
			t.Fatal(e)
		}
		n, _ := strconv.Atoi(strings.Split(v.Native, ".")[2])
		if n <= previous {
			t.Fatal("native ordering broken", tag, v.Native)
		}
		previous = n
	}
	for _, tag := range []string{"v0.2.0-dev", "v0.2.0-rc.0", "v0.2.0-rc.200", "v256.0.0", "v0.256.0", "v0.0.65", "v0.2.0-rc.01", "v0.2.0+meta", "0.2.0"} {
		if _, e := Parse(tag); e == nil {
			t.Fatal("accepted unsupported native version", tag)
		}
	}
	if v, e := Parse("v255.255.64"); e != nil || v.Native != "255.255.64999" {
		t.Fatal(v, e)
	}
}

func TestExpectedSet(t *testing.T) {
	seen := map[string]bool{}
	for _, n := range Expected("0.2.0-rc.1") {
		if seen[n] || strings.Contains(n, "/") {
			t.Fatal(n)
		}
		seen[n] = true
	}
	if len(seen) != 23 {
		t.Fatal("incomplete release set", len(seen))
	}
}
