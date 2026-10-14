package buildinfo

import (
	"regexp"
	"runtime"
	"strings"
	"tunnelctl/internal/domain"
)

var Version = "0.2.0-dev"
var Commit = "unknown"
var BuildDate = "unknown"
var Tag = ""
var semver = regexp.MustCompile(`^(0|[1-9][0-9]*)\.(0|[1-9][0-9]*)\.(0|[1-9][0-9]*)(?:-([0-9A-Za-z-]+(?:\.[0-9A-Za-z-]+)*))?(?:\+([0-9A-Za-z-]+(?:\.[0-9A-Za-z-]+)*))?$`)

func ValidVersion(v string) bool {
	m := semver.FindStringSubmatch(v)
	if m == nil {
		return false
	}
	for _, id := range strings.Split(m[4], ".") {
		if len(id) > 1 && id[0] == '0' && strings.Trim(id, "0123456789") == "" {
			return false
		}
	}
	return true
}
func FromTag(tag string) (string, error) {
	v := strings.TrimPrefix(tag, "v")
	if !strings.HasPrefix(tag, "v") || !ValidVersion(v) || strings.Contains(v, "+") {
		return "", domain.Fail(2, "Invalid release tag.", "Use a Semantic Versioning tag such as v0.2.0 or v0.2.0-rc.1, without build metadata.")
	}
	return v, nil
}

type Info struct {
	Application   string `json:"application"`
	Commit        string `json:"commit"`
	BuildDate     string `json:"build_date"`
	Tag           string `json:"tag,omitempty"`
	Development   bool   `json:"development"`
	Prerelease    bool   `json:"prerelease"`
	Go            string `json:"go"`
	Platform      string `json:"platform"`
	StorageSchema int    `json:"storage_schema"`
	OutputSchema  int    `json:"output_schema"`
	PinnedCore    string `json:"pinned_core"`
}

func Current() (Info, error) {
	if !ValidVersion(Version) {
		return Info{}, domain.Fail(3, "Binary build version is invalid.", "Rebuild using the documented release tool.")
	}
	if Tag != "" {
		v, e := FromTag(Tag)
		if e != nil || v != Version {
			return Info{}, domain.Fail(3, "Binary version does not match its release tag.", "Discard this artifact and use a correctly built release.")
		}
	}
	return Info{Version, Commit, BuildDate, Tag, Tag == "" || strings.HasSuffix(Commit, "-dirty"), semver.FindStringSubmatch(Version)[4] != "", runtime.Version(), runtime.GOOS + "/" + runtime.GOARCH, domain.SchemaVersion, domain.OutputSchemaVersion, "1.14.3"}, nil
}
