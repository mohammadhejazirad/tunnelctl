// Package packaging defines the release artifact and native version contract.
package packaging

import (
	"fmt"
	"strconv"
	"strings"
	"tunnelctl/internal/buildinfo"
)

type Version struct {
	Tag        string `json:"tag"`
	Semver     string `json:"version"`
	Native     string `json:"native_version"`
	Prerelease bool   `json:"prerelease"`
}

// Parse uses one monotonic numeric version for every native installer, including
// MSI's three-component/16-bit-build constraint. The binary retains real SemVer.
func Parse(tag string) (Version, error) {
	v, err := buildinfo.FromTag(tag)
	if err != nil {
		return Version{}, err
	}
	parts := strings.SplitN(v, "-", 2)
	n := strings.Split(parts[0], ".")
	nums := [3]int{}
	for i := range n {
		nums[i], err = strconv.Atoi(n[i])
		if err != nil {
			return Version{}, fmt.Errorf("native version component overflow")
		}
	}
	if nums[0] > 255 || nums[1] > 255 || nums[2] > 64 {
		return Version{}, fmt.Errorf("native packages require major/minor <=255 and patch <=64")
	}
	rank := 999
	if len(parts) == 2 {
		pre := strings.Split(parts[1], ".")
		if len(pre) != 2 {
			return Version{}, fmt.Errorf("native prerelease must be alpha.N, beta.N or rc.N (1 <= N <= 199)")
		}
		i, e := strconv.Atoi(pre[1])
		if e != nil || i < 1 || i > 199 {
			return Version{}, fmt.Errorf("prerelease number must be 1..199")
		}
		switch pre[0] {
		case "alpha":
			rank = i
		case "beta":
			rank = 200 + i
		case "rc":
			rank = 400 + i
		default:
			return Version{}, fmt.Errorf("native prerelease channel must be alpha, beta or rc")
		}
	}
	return Version{tag, v, fmt.Sprintf("%d.%d.%d", nums[0], nums[1], nums[2]*1000+rank), len(parts) == 2}, nil
}

func Expected(v string) []string {
	names := []string{}
	for _, arch := range []string{"amd64", "arm64"} {
		for _, ext := range []string{"tar.gz", "deb", "rpm", "pkg.tar.zst", "apk", "AppImage", "PKGBUILD"} {
			names = append(names, "tunnelctl_"+v+"_linux_"+arch+"."+ext)
		}
		for _, ext := range []string{"tar.gz", "pkg"} {
			names = append(names, "tunnelctl_"+v+"_darwin_"+arch+"."+ext)
		}
	}
	for _, ext := range []string{"zip", "msi"} {
		names = append(names, "tunnelctl_"+v+"_windows_amd64."+ext)
	}
	return append(names, "sbom.cdx.json", "release-manifest.json", "release-notes.md")
}
