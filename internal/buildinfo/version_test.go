package buildinfo

import "testing"

func TestTagValidation(t *testing.T) {
	for _, v := range []string{"v0.2.0", "v0.2.0-rc.1", "v1.0.0"} {
		if _, e := FromTag(v); e != nil {
			t.Fatal(e)
		}
	}
	for _, v := range []string{"0.2.0", "v01.2.3", "v1.2.3-01", "v1.2", "v1.2.3+meta", "v1.2.3/evil"} {
		if _, e := FromTag(v); e == nil {
			t.Fatal("invalid tag accepted")
		}
	}
}

func TestBuildIdentityAndMismatch(t *testing.T) {
	oldVersion, oldTag, oldCommit := Version, Tag, Commit
	defer func() { Version, Tag, Commit = oldVersion, oldTag, oldCommit }()
	Version, Tag, Commit = "0.2.0-rc.1", "v0.2.0-rc.1", "abc123"
	info, e := Current()
	if e != nil || info.Development || !info.Prerelease {
		t.Fatal("release metadata invalid", e)
	}
	Commit = "abc123-dirty"
	info, e = Current()
	if e != nil || !info.Development {
		t.Fatal("dirty dry-run disguised as production")
	}
	Tag = "v0.3.0"
	if _, e = Current(); e == nil {
		t.Fatal("mismatched tag accepted")
	}
	Version, Tag = "0.2.0+build-with-hyphen", ""
	info, e = Current()
	if e != nil || info.Prerelease || !info.Development {
		t.Fatal("build metadata misclassified as prerelease")
	}
}
