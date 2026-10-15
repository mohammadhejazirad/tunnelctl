package main

import (
	"encoding/xml"
	"os"
	"os/exec"
	"path/filepath"
	"runtime"
	"strings"
	"testing"
	"tunnelctl/internal/packaging"
)

func TestWiXAutoGuidComponentsHaveSingleFile(t *testing.T) {
	type component struct {
		ID    string     `xml:"Id,attr"`
		GUID  string     `xml:"Guid,attr"`
		Files []struct{} `xml:"File"`
	}
	var manifest struct {
		Components []component `xml:"Package>StandardDirectory>Directory>Component"`
	}
	b, err := os.ReadFile("../../packaging/windows/Product.wxs")
	if err != nil {
		t.Fatal(err)
	}
	if err = xml.Unmarshal(b, &manifest); err != nil {
		t.Fatal(err)
	}
	if len(manifest.Components) == 0 {
		t.Fatal("WiX manifest contains no components")
	}
	for _, c := range manifest.Components {
		if c.GUID == "*" && len(c.Files) != 1 {
			t.Fatalf("WiX auto-guid component %q must own exactly one file, got %d", c.ID, len(c.Files))
		}
	}
}

func TestAggregateRejectsMissingUnexpectedAndTampered(t *testing.T) {
	v, _ := packaging.Parse("v0.2.0-rc.1")
	d := t.TempDir()
	if finalize(v, d, false) == nil {
		t.Fatal("missing artifacts accepted")
	}
	for _, n := range packaging.Expected(v.Semver) {
		if e := os.WriteFile(filepath.Join(d, n), []byte("synthetic-public-asset"), 0600); e != nil {
			t.Fatal(e)
		}
	}
	if e := writeJSON(filepath.Join(d, "release-manifest.json"), map[string]string{"tag": v.Tag, "version": v.Semver, "native_version": v.Native}); e != nil {
		t.Fatal(e)
	}
	if e := finalize(v, d, false); e != nil {
		t.Fatal(e)
	}
	if e := finalize(v, d, true); e != nil {
		t.Fatal(e)
	}
	if e := os.WriteFile(filepath.Join(d, "links_and_subs.txt"), []byte("synthetic-forbidden-file"), 0600); e != nil {
		t.Fatal(e)
	}
	if finalize(v, d, true) == nil {
		t.Fatal("private unexpected file accepted")
	}
	if e := os.Remove(filepath.Join(d, "links_and_subs.txt")); e != nil {
		t.Fatal(e)
	}
	if e := os.WriteFile(filepath.Join(d, "sbom.cdx.json"), []byte("tampered"), 0600); e != nil {
		t.Fatal(e)
	}
	if finalize(v, d, true) == nil {
		t.Fatal("tampering accepted")
	}
}

func TestPublicConfigurationPreservesPathsAndDirectoryModes(t *testing.T) {
	d := t.TempDir()
	if e := os.Mkdir(filepath.Join(d, "docs"), 0700); e != nil {
		t.Fatal(e)
	}
	if e := os.WriteFile(filepath.Join(d, "docs", "public.md"), []byte("synthetic"), 0600); e != nil {
		t.Fatal(e)
	}
	a, p, e := publicConfiguration(d)
	if e != nil || !strings.Contains(a, `dst: "docs"`) || !strings.Contains(p, `dst: "/usr/share/doc/tunnelctl/docs/public.md"`) || !strings.Contains(p, "mode: 0755") || !strings.Contains(p, "mode: 0644") {
		t.Fatal(a, p, e)
	}
	if e := os.WriteFile(filepath.Join(d, "links_and_subs.txt"), []byte("synthetic"), 0600); e != nil {
		t.Fatal(e)
	}
	if _, _, e = publicConfiguration(d); e == nil {
		t.Fatal("private public-stage contamination accepted")
	}
}

func TestExactTaggedCommitAndDirtyGate(t *testing.T) {
	d := t.TempDir()
	old, e := os.Getwd()
	if e != nil {
		t.Fatal(e)
	}
	if e = os.Chdir(d); e != nil {
		t.Fatal(e)
	}
	t.Cleanup(func() {
		if e := os.Chdir(old); e != nil {
			t.Fatal(e)
		}
	})
	gitCommand := func(args ...string) {
		t.Helper()
		c := exec.Command("git", args...)
		if b, e := c.CombinedOutput(); e != nil {
			t.Fatal(e, string(b))
		}
	}
	gitCommand("init")
	if e = os.WriteFile("public.txt", []byte("synthetic-public"), 0600); e != nil {
		t.Fatal(e)
	}
	gitCommand("add", "public.txt")
	gitCommand("-c", "user.name=Packaging Test", "-c", "user.email=packaging@example.invalid", "commit", "-m", "synthetic fixture")
	if run("verify-tag", "v0.2.0-rc.1", "", false) == nil {
		t.Fatal("missing tag accepted")
	}
	gitCommand("tag", "v0.2.0-rc.1")
	if e = run("verify-tag", "v0.2.0-rc.1", "", false); e != nil {
		t.Fatal(e)
	}
	if e = os.WriteFile("public.txt", []byte("changed"), 0600); e != nil {
		t.Fatal(e)
	}
	if run("verify-tag", "v0.2.0-rc.1", "", false) == nil {
		t.Fatal("dirty release accepted")
	}
	gitCommand("add", "public.txt")
	gitCommand("-c", "user.name=Packaging Test", "-c", "user.email=packaging@example.invalid", "commit", "-m", "other commit")
	if run("verify-tag", "v0.2.0-rc.1", "", false) == nil {
		t.Fatal("tag from other commit accepted")
	}
	if e = run("verify-tag", "v0.2.0-rc.1", "", true); e != nil {
		t.Fatal("explicit dry-run refused", e)
	}
}

func TestPublicCopyRefusesSymlinkAndPrivate(t *testing.T) {
	d := t.TempDir()
	src := filepath.Join(d, "links_and_subs.txt")
	if e := os.WriteFile(src, []byte("synthetic"), 0600); e != nil {
		t.Fatal(e)
	}
	if copyFile(src, filepath.Join(d, "output")) == nil {
		t.Fatal("private copy accepted")
	}
	if e := os.Symlink(src, filepath.Join(d, "link")); e != nil {
		t.Skip("symlinks unavailable")
	}
	if copyFile(filepath.Join(d, "link"), filepath.Join(d, "output")) == nil {
		t.Fatal("symlink accepted")
	}
}

// Exercise the actual publisher control flow with a synthetic gh transport.
// Asset verification remains the real Go implementation; no remote API is used.
func TestPublisherNeverPromotesIncompleteOrExistingRelease(t *testing.T) {
	if runtime.GOOS == "windows" {
		t.Skip("publisher runs on the Linux release runner")
	}
	for _, tool := range []string{"bash", "jq"} {
		if _, e := exec.LookPath(tool); e != nil {
			t.Skip("publisher test prerequisite unavailable: " + tool)
		}
	}
	root, e := filepath.Abs("../..")
	if e != nil {
		t.Fatal(e)
	}
	for _, scenario := range []string{"success", "stable-success", "existing", "api-failure", "upload-failure", "tampered-download", "manual"} {
		t.Run(scenario, func(t *testing.T) {
			tag := "v0.2.0-rc.1"
			if scenario == "stable-success" {
				tag = "v0.2.0"
			}
			v, _ := packaging.Parse(tag)
			wantSuccess := scenario == "success" || scenario == "stable-success"
			d := t.TempDir()
			assets := filepath.Join(d, "assets")
			bin := filepath.Join(d, "bin")
			remote := filepath.Join(d, "remote")
			for _, dir := range []string{assets, bin, remote} {
				if e := os.Mkdir(dir, 0700); e != nil {
					t.Fatal(e)
				}
			}
			for _, name := range packaging.Expected(v.Semver) {
				if e := os.WriteFile(filepath.Join(assets, name), []byte("synthetic-public-asset"), 0600); e != nil {
					t.Fatal(e)
				}
			}
			if e := writeJSON(filepath.Join(assets, "release-manifest.json"), map[string]string{"tag": v.Tag, "version": v.Semver, "native_version": v.Native}); e != nil {
				t.Fatal(e)
			}
			if e := finalize(v, assets, false); e != nil {
				t.Fatal(e)
			}
			fake := `#!/usr/bin/env bash
set -euo pipefail
printf '%s\n' "$2" >> "$PUBLISH_LOG"
case "$2" in
  list)
    if [[ "$PUBLISH_SCENARIO" == api-failure ]]; then exit 1; fi
    if [[ "$PUBLISH_SCENARIO" == existing ]]; then printf '[{"tagName":"v0.2.0-rc.1"}]'; else printf '[]'; fi ;;
  create)
    [[ " $* " == *' --draft '* ]]
    [[ " $* " != *' --clobber '* ]]
    if [[ "$PUBLISH_SCENARIO" == upload-failure ]]; then exit 1; fi
    for arg in "$@"; do if [[ -f "$arg" ]]; then cp "$arg" "$PUBLISH_REMOTE/"; fi; done ;;
  download)
    destination=''
    while [[ $# -gt 0 ]]; do
      if [[ "$1" == --dir ]]; then shift; destination=$1; fi
      shift
    done
    test -n "$destination"
    cp "$PUBLISH_REMOTE/"* "$destination/"
    if [[ "$PUBLISH_SCENARIO" == tampered-download ]]; then printf 'tampered' >> "$destination/sbom.cdx.json"; fi ;;
  edit)
    [[ " $* " == *' --draft=false '* ]]
    if [[ "$PUBLISH_SCENARIO" == stable-success ]]; then
      [[ " $* " == *' --prerelease=false '* && " $* " == *' --latest=true '* ]]
    else
      test "$PUBLISH_SCENARIO" == success
      [[ " $* " == *' --prerelease=true '* && " $* " == *' --latest=false '* ]]
    fi ;;
  *) exit 2 ;;
esac
`
			if e := os.WriteFile(filepath.Join(bin, "gh"), []byte(fake), 0700); e != nil {
				t.Fatal(e)
			}
			log := filepath.Join(d, "calls")
			c := exec.Command("bash", "packaging/publish.sh", assets, v.Tag)
			c.Dir = root
			c.Env = append(os.Environ(), "PATH="+bin+string(os.PathListSeparator)+os.Getenv("PATH"), "PUBLISH_SCENARIO="+scenario, "PUBLISH_REMOTE="+remote, "PUBLISH_LOG="+log, "GITHUB_EVENT_NAME=push", "GITHUB_REF=refs/tags/"+v.Tag)
			if scenario == "manual" {
				c.Env = append(c.Env, "GITHUB_EVENT_NAME=workflow_dispatch")
			}
			output, e := c.CombinedOutput()
			if (e == nil) != wantSuccess {
				t.Fatalf("unexpected publisher result: %v\n%s", e, output)
			}
			calls, _ := os.ReadFile(log)
			promoted := strings.Contains(string(calls), "edit\n")
			if promoted != wantSuccess {
				t.Fatal("incorrect promotion", string(calls))
			}
			if (scenario == "existing" || scenario == "api-failure" || scenario == "manual") && strings.Contains(string(calls), "create\n") {
				t.Fatal("draft created without authorization or after existing/API failure")
			}
		})
	}
}
