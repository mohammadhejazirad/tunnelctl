// Command packaging orchestrates local builds. It never creates tags or releases.
package main

import (
	"crypto/sha256"
	"encoding/hex"
	"encoding/json"
	"flag"
	"fmt"
	"io"
	"os"
	"os/exec"
	"path/filepath"
	"sort"
	"strings"
	"time"
	"tunnelctl/internal/hygiene"
	"tunnelctl/internal/packaging"
)

func main() {
	mode := flag.String("mode", "build", "build, versions, verify-tag, finalize or verify-assets")
	tag := flag.String("tag", "", "Version tag")
	out := flag.String("out", "dist/packages", "New build directory or asset directory")
	dry := flag.Bool("dry-run", false, "Allow an untagged checkout; never publish")
	flag.Parse()
	if e := run(*mode, *tag, *out, *dry); e != nil {
		fmt.Fprintln(os.Stderr, "Packaging failed:", e)
		os.Exit(1)
	}
}
func command(name string, args ...string) error {
	c := exec.Command(name, args...)
	c.Stdout = os.Stdout
	c.Stderr = os.Stderr
	c.Stdin = os.Stdin
	return c.Run()
}
func git(args ...string) (string, error) {
	b, e := exec.Command("git", args...).Output()
	return strings.TrimSpace(string(b)), e
}
func writeJSON(path string, v any) error {
	b, e := json.MarshalIndent(v, "", "  ")
	if e != nil {
		return e
	}
	return os.WriteFile(path, append(b, '\n'), 0644)
}
func copyFile(src, dst string) error {
	st, e := os.Lstat(src)
	if e != nil {
		return e
	}
	if !st.Mode().IsRegular() || hygiene.PrivatePath(src) {
		return fmt.Errorf("non-public build input rejected")
	}
	b, e := os.ReadFile(src)
	if e != nil {
		return e
	}
	f, e := os.OpenFile(dst, os.O_CREATE|os.O_EXCL|os.O_WRONLY, st.Mode().Perm())
	if e != nil {
		return e
	}
	_, e = f.Write(b)
	ce := f.Close()
	if e != nil {
		return e
	}
	return ce
}
func run(mode, tag, out string, dry bool) error {
	v, e := packaging.Parse(tag)
	if e != nil {
		return e
	}
	if mode == "versions" {
		return json.NewEncoder(os.Stdout).Encode(v)
	}
	if mode == "finalize" || mode == "verify-assets" {
		return finalize(v, out, mode == "verify-assets")
	}
	commit, e := git("rev-parse", "HEAD")
	if e != nil {
		return e
	}
	status, e := git("status", "--porcelain")
	if e != nil {
		return e
	}
	if !dry {
		actual, e := git("rev-parse", "refs/tags/"+tag+"^{commit}")
		if e != nil || actual != commit || status != "" {
			return fmt.Errorf("release requires a clean tree and a tag resolving to exactly HEAD")
		}
	} else if status != "" {
		commit += "-dirty"
	}
	if mode == "verify-tag" {
		fmt.Println("Tag/commit and native version contract verified:", tag)
		return nil
	}
	if mode != "build" {
		return fmt.Errorf("unknown packaging mode")
	}
	out, e = filepath.Abs(out)
	if e != nil {
		return e
	}
	if _, e = os.Lstat(out); !os.IsNotExist(e) {
		return fmt.Errorf("output must not exist; packages are never overwritten")
	}
	if e = os.MkdirAll(filepath.Join(out, "assets"), 0755); e != nil {
		return e
	}
	public := filepath.Join(out, "public")
	if e = command("go", "run", "./cmd/release", "--tag", tag, "--stage-public", "--out", public); e != nil {
		return e
	}
	date, e := git("show", "-s", "--format=%cI", "HEAD")
	if e != nil {
		return e
	}
	if _, e = time.Parse(time.RFC3339, date); e != nil {
		return e
	}
	for k, val := range map[string]string{"PACKAGE_WORK": out, "PUBLIC_STAGE": public, "RELEASE_VERSION": v.Semver, "NATIVE_VERSION": v.Native, "RELEASE_TAG": v.Tag, "RELEASE_COMMIT": commit, "RELEASE_DATE": date} {
		if e = os.Setenv(k, val); e != nil {
			return e
		}
	}
	config, e := os.ReadFile(".goreleaser.yaml")
	if e != nil {
		return e
	}
	config = []byte(strings.Replace(string(config), "dist: dist/goreleaser", fmt.Sprintf("dist: %q", filepath.Join(out, "goreleaser")), 1))
	archiveFiles, packageFiles, e := publicConfiguration(public)
	if e != nil {
		return e
	}
	for marker, replacement := range map[string]string{"    files: [] # GENERATED_PUBLIC_ARCHIVE_FILES": archiveFiles, "    contents: [] # GENERATED_PUBLIC_PACKAGE_FILES": packageFiles} {
		if strings.Count(string(config), marker) != 1 {
			return fmt.Errorf("public packaging configuration marker missing/duplicated")
		}
		config = []byte(strings.Replace(string(config), marker, replacement, 1))
	}
	configPath := filepath.Join(out, "goreleaser.yaml")
	if e = os.WriteFile(configPath, config, 0644); e != nil {
		return e
	}
	if e = command("goreleaser", "check", "--config", configPath); e != nil {
		return e
	}
	// GoReleaser's snapshot packaging engine supplies the mapped native version.
	// Publication is disabled; the independent gate above validates actual tags.
	if e = command("goreleaser", "release", "--config", configPath, "--snapshot", "--skip=publish", "--parallelism=2"); e != nil {
		return e
	}
	assets := filepath.Join(out, "assets")
	for _, n := range packaging.Expected(v.Semver) {
		if strings.HasSuffix(n, ".tar.gz") || strings.HasSuffix(n, ".zip") || strings.HasSuffix(n, ".deb") || strings.HasSuffix(n, ".rpm") || strings.HasSuffix(n, ".apk") || strings.HasSuffix(n, ".pkg.tar.zst") {
			if e = copyFile(filepath.Join(out, "goreleaser", n), filepath.Join(assets, n)); e != nil {
				return e
			}
		}
	}
	if e = copyFile(filepath.Join(public, "sbom.cdx.json"), filepath.Join(assets, "sbom.cdx.json")); e != nil {
		return e
	}
	if e = writeJSON(filepath.Join(assets, "release-manifest.json"), map[string]any{"tag": v.Tag, "version": v.Semver, "native_version": v.Native, "commit": commit, "build_date": date, "dry_run": dry, "core_bundled": false, "approved_proxy_targets": []string{"linux/amd64"}, "experimental_install_targets": []string{"linux/arm64", "darwin/amd64", "darwin/arm64", "windows/amd64"}, "signing": "unsigned; GitHub provenance is separate", "tools": map[string]string{"goreleaser": "2.18.3", "nfpm_embedded": "2.47.0", "appimagetool": "1.9.1", "wix": "5.0.2"}}); e != nil {
		return e
	}
	notes, e := os.ReadFile("CHANGELOG.md")
	if e != nil {
		return e
	}
	header := "# " + v.Tag + "\n\nOnly Linux amd64 proxy runtime is approved. Other target installers are experimental. No TUN, service or bundled core. Unsigned installers; no macOS notarization.\n\nNative package version: " + v.Native + ". Binary version: " + v.Semver + ".\n\n"
	if e = os.WriteFile(filepath.Join(assets, "release-notes.md"), append([]byte(header), notes...), 0644); e != nil {
		return e
	}
	for _, arch := range []string{"amd64", "arm64"} {
		if e = archBuild(v, assets, arch); e != nil {
			return e
		}
	}
	fmt.Println("Portable/native Linux packages prepared in", assets, "; native MSI/pkg and AppImages must pass their separate gates.")
	return nil
}

// Generate one entry per already-allowlisted public file. Tree-wide mode=0644
// also removes execute permission from directories in nFPM, and recursive archive
// globs can flatten copies unexpectedly. Explicit entries avoid both pitfalls,
// independent of the source filesystem's permission emulation.
func publicConfiguration(stage string) (string, string, error) {
	var archives, packages strings.Builder
	archives.WriteString("    files:\n")
	packages.WriteString("    contents:\n")
	e := filepath.WalkDir(stage, func(path string, entry os.DirEntry, e error) error {
		if e != nil {
			return e
		}
		rel, e := filepath.Rel(stage, path)
		if e != nil {
			return e
		}
		if hygiene.PrivatePath(rel) || entry.Type()&os.ModeSymlink != 0 {
			return fmt.Errorf("non-public staging member rejected")
		}
		dst := "/usr/share/doc/tunnelctl"
		if rel != "." {
			dst += "/" + filepath.ToSlash(rel)
		}
		if entry.IsDir() {
			fmt.Fprintf(&packages, "      - dst: %q\n        type: dir\n        file_info: {mode: 0755, owner: root, group: root}\n", dst)
			return nil
		}
		if !entry.Type().IsRegular() {
			return fmt.Errorf("nonregular public staging member rejected")
		}
		fmt.Fprintf(&archives, "      - src: %q\n        dst: %q\n        strip_parent: true\n        info: {mode: 0644, owner: root, group: root}\n", path, filepath.ToSlash(filepath.Dir(rel)))
		fmt.Fprintf(&packages, "      - src: %q\n        dst: %q\n        file_info: {mode: 0644, owner: root, group: root}\n", path, dst)
		return nil
	})
	return archives.String(), packages.String(), e
}
func archBuild(v packaging.Version, out, arch string) error {
	archive := "tunnelctl_" + v.Semver + "_linux_" + arch + ".tar.gz"
	b, e := os.ReadFile(filepath.Join(out, archive))
	if e != nil {
		return e
	}
	sum := sha256.Sum256(b)
	nativeArch := "x86_64"
	if arch == "arm64" {
		nativeArch = "aarch64"
	}
	text := fmt.Sprintf(`# Maintainer: Mohammad Hejazirad <mohammadhejazirad@users.noreply.github.com>
# Generated for this exact release archive. Not an AUR publication.
pkgname=tunnelctl
pkgver=%s
pkgrel=1
pkgdesc='Terminal proxy client; see SUPPORT.txt for target limitations'
arch=('%s')
url='https://github.com/mohammadhejazirad/tunnelctl'
license=('Apache-2.0')
options=('!strip' '!debug')
source=('%s::https://github.com/mohammadhejazirad/tunnelctl/releases/download/%s/%s')
sha256sums=('%x')
package() {
  install -Dm755 "$srcdir/tunnelctl" "$pkgdir/usr/bin/tunnelctl"
  install -Dm644 "$srcdir/LICENSE" "$pkgdir/usr/share/licenses/tunnelctl/LICENSE"
  install -d "$pkgdir/usr/share/doc/tunnelctl"
  cp -R "$srcdir/docs" "$srcdir/THIRD_PARTY_LICENSES" "$srcdir/examples" "$pkgdir/usr/share/doc/tunnelctl/"
  for doc in LICENSE NOTICE README.md CHANGELOG.md SECURITY.md compatibility.json SUPPORT.txt sbom.cdx.json THIRD-PARTY-NOTICES.txt; do
    install -m644 "$srcdir/$doc" "$pkgdir/usr/share/doc/tunnelctl/$doc"
  done
}
`, v.Native, nativeArch, archive, v.Tag, archive, sum)
	return os.WriteFile(filepath.Join(out, "tunnelctl_"+v.Semver+"_linux_"+arch+".PKGBUILD"), []byte(text), 0644)
}
func finalize(v packaging.Version, out string, verify bool) error {
	names := packaging.Expected(v.Semver)
	sort.Strings(names)
	entries, e := os.ReadDir(out)
	if e != nil {
		return e
	}
	want := map[string]bool{"checksums.txt": true}
	for _, n := range names {
		want[n] = true
	}
	for _, entry := range entries {
		if !want[entry.Name()] || !entry.Type().IsRegular() || hygiene.PrivatePath(entry.Name()) {
			return fmt.Errorf("unexpected/non-public asset rejected: %s", entry.Name())
		}
	}
	var checks strings.Builder
	for _, n := range names {
		f, e := os.Open(filepath.Join(out, n))
		if e != nil {
			return fmt.Errorf("required asset missing: %s", n)
		}
		h := sha256.New()
		_, e = io.Copy(h, f)
		ce := f.Close()
		if e != nil {
			return e
		}
		if ce != nil {
			return ce
		}
		fmt.Fprintf(&checks, "%s  %s\n", hex.EncodeToString(h.Sum(nil)), n)
	}
	var manifest struct {
		Tag, Version  string
		NativeVersion string `json:"native_version"`
		DryRun        bool   `json:"dry_run"`
	}
	b, e := os.ReadFile(filepath.Join(out, "release-manifest.json"))
	if e != nil {
		return e
	}
	if e = json.Unmarshal(b, &manifest); e != nil || manifest.Tag != v.Tag || manifest.Version != v.Semver || manifest.NativeVersion != v.Native {
		return fmt.Errorf("asset manifest/tag mismatch")
	}
	if verify {
		b, e := os.ReadFile(filepath.Join(out, "checksums.txt"))
		if e != nil {
			return e
		}
		if string(b) != checks.String() {
			return fmt.Errorf("asset checksums/set mismatch")
		}
		return nil
	}
	return os.WriteFile(filepath.Join(out, "checksums.txt"), []byte(checks.String()), 0644)
}
