// Command release builds and packages local artifacts. It never pushes tags or publishes.
package main

import (
	"archive/tar"
	"archive/zip"
	"bytes"
	"compress/gzip"
	"crypto/sha256"
	"encoding/hex"
	"encoding/json"
	"flag"
	"fmt"
	"io"
	"os"
	"os/exec"
	"path/filepath"
	"runtime"
	"sort"
	"strings"
	"time"
	"tunnelctl/internal/buildinfo"
)

type target struct {
	OS        string `json:"os"`
	Arch      string `json:"arch"`
	Supported bool   `json:"runtime_supported"`
}

var targets = []target{{"linux", "amd64", true}, {"linux", "arm64", false}, {"darwin", "amd64", false}, {"darwin", "arm64", false}, {"windows", "amd64", false}}

func main() {
	tag := flag.String("tag", "", "Semantic release tag")
	out := flag.String("out", "dist", "New empty output directory")
	dry := flag.Bool("dry-run", false, "Permit an untagged working tree; never publish")
	flag.Parse()
	if e := run(*tag, *out, *dry); e != nil {
		fmt.Fprintln(os.Stderr, "Release preparation failed:", e)
		os.Exit(1)
	}
}
func run(tag, out string, dry bool) error {
	version, e := buildinfo.FromTag(tag)
	if e != nil {
		return e
	}
	if entries, e := os.ReadDir(out); e == nil && len(entries) > 0 {
		return fmt.Errorf("output directory must be empty; existing assets are never overwritten")
	}
	commit, e := exec.Command("git", "rev-parse", "HEAD").Output()
	if e != nil {
		return fmt.Errorf("a local Git commit is required")
	}
	commitText := strings.TrimSpace(string(commit))
	status, e := exec.Command("git", "status", "--porcelain").Output()
	if e != nil {
		return fmt.Errorf("Git worktree inspection failed: %w", e)
	}
	if !dry {
		actual, e := exec.Command("git", "describe", "--exact-match", "--tags", "HEAD").Output()
		if e != nil || strings.TrimSpace(string(actual)) != tag || len(status) > 0 {
			return fmt.Errorf("release requires a clean tree checked out at the exact supplied tag")
		}
	} else if len(status) > 0 {
		commitText += "-dirty"
	}
	built := time.Now().UTC().Format(time.RFC3339)
	if epoch := os.Getenv("SOURCE_DATE_EPOCH"); epoch != "" {
		var seconds int64
		if _, e := fmt.Sscan(epoch, &seconds); e != nil {
			return e
		}
		built = time.Unix(seconds, 0).UTC().Format(time.RFC3339)
	}
	if e = os.MkdirAll(out, 0755); e != nil {
		return e
	}
	goTool := os.Getenv("RELEASE_GO")
	if goTool == "" {
		goTool = "go"
	}
	stage, e := os.MkdirTemp("", "tunnelctl-release-")
	if e != nil {
		return e
	}
	defer os.RemoveAll(stage)
	modules, e := moduleInventory(goTool)
	if e != nil {
		return fmt.Errorf("Go module inventory failed: %w", e)
	}
	licenses := filepath.Join(stage, "THIRD_PARTY_LICENSES")
	if e = collectLicenses(modules, licenses); e != nil {
		return e
	}
	files := []string{}
	for _, t := range targets {
		suffix := ""
		if !t.Supported {
			suffix = "_compile-only"
		}
		base := "tunnelctl_" + version + "_" + t.OS + "_" + t.Arch + suffix
		dir := filepath.Join(stage, base)
		if e = os.Mkdir(dir, 0700); e != nil {
			return e
		}
		binary := "tunnelctl"
		if t.OS == "windows" {
			binary += ".exe"
		}
		binaryPath := filepath.Join(dir, binary)
		ld := "-s -w -X tunnelctl/internal/buildinfo.Version=" + version + " -X tunnelctl/internal/buildinfo.Tag=" + tag + " -X tunnelctl/internal/buildinfo.Commit=" + commitText + " -X tunnelctl/internal/buildinfo.BuildDate=" + built
		cmd := exec.Command(goTool, "build", "-buildvcs=false", "-trimpath", "-ldflags", ld, "-o", binaryPath, "./cmd/tunnelctl")
		cmd.Env = append(os.Environ(), "CGO_ENABLED=0", "GOOS="+t.OS, "GOARCH="+t.Arch)
		cmd.Stdout = os.Stdout
		cmd.Stderr = os.Stderr
		if e = cmd.Run(); e != nil {
			return e
		}
		for _, name := range []string{"LICENSE", "NOTICE", "README.md", "CHANGELOG.md", "SECURITY.md", "compatibility.json"} {
			if e = copyFile(name, filepath.Join(dir, name)); e != nil {
				return e
			}
		}
		if e = copyTree("docs", filepath.Join(dir, "docs")); e != nil {
			return e
		}
		if e = copyTree(licenses, filepath.Join(dir, "THIRD_PARTY_LICENSES")); e != nil {
			return e
		}
		if e = copyTree("examples", filepath.Join(dir, "examples")); e != nil {
			return e
		}
		if e = writeCompatibility(filepath.Join(dir, "compatibility.json"), version, tag, commitText, built, dry, t); e != nil {
			return e
		}
		support := "Runtime unsupported: compilation artifact only. Do not deploy as a VPN/proxy client.\n"
		if t.Supported {
			support = "Tested Linux amd64 proxy; TUN unsupported. External verified sing-box required.\n"
		}
		if e = os.WriteFile(filepath.Join(dir, "SUPPORT.txt"), []byte(support), 0644); e != nil {
			return e
		}
		extension := ".tar.gz"
		if t.OS == "windows" {
			extension = ".zip"
		}
		path := filepath.Join(out, base+extension)
		if e = archive(dir, path, t.OS == "windows"); e != nil {
			return e
		}
		files = append(files, path)
	}
	decoder := json.NewDecoder(strings.NewReader(string(modules)))
	components := []map[string]any{}
	for {
		var m struct{ Path, Version, Sum string }
		if e = decoder.Decode(&m); e == io.EOF {
			break
		} else if e != nil {
			return e
		}
		entry := map[string]any{"type": "library", "name": m.Path}
		if m.Version != "" {
			entry["version"] = m.Version
		}
		components = append(components, entry)
	}
	sbom := map[string]any{"bomFormat": "CycloneDX", "specVersion": "1.6", "version": 1, "metadata": map[string]any{"timestamp": built, "component": map[string]any{"type": "application", "name": "tunnelctl", "version": version}}, "components": components}
	sbomPath := filepath.Join(out, "sbom.cdx.json")
	if e = writeJSON(sbomPath, sbom); e != nil {
		return e
	}
	files = append(files, sbomPath)
	manifest := map[string]any{"tag": tag, "version": version, "commit": commitText, "build_date": built, "dry_run": dry, "core_bundled": false, "targets": targets}
	manifestPath := filepath.Join(out, "release-manifest.json")
	if e = writeJSON(manifestPath, manifest); e != nil {
		return e
	}
	files = append(files, manifestPath)
	notesPath := filepath.Join(out, "release-notes.md")
	notes, e := os.ReadFile("docs/release-notes-0.2.0.md")
	if e != nil {
		return e
	}
	if e = os.WriteFile(notesPath, append([]byte("# "+tag+"\n\n"), notes...), 0644); e != nil {
		return e
	}
	files = append(files, notesPath)
	sort.Strings(files)
	var checks strings.Builder
	for _, path := range files {
		f, e := os.Open(path)
		if e != nil {
			return e
		}
		h := sha256.New()
		_, e = io.Copy(h, f)
		f.Close()
		if e != nil {
			return e
		}
		fmt.Fprintf(&checks, "%s  %s\n", hex.EncodeToString(h.Sum(nil)), filepath.Base(path))
	}
	return os.WriteFile(filepath.Join(out, "checksums.txt"), []byte(checks.String()), 0644)
}

func writeCompatibility(path, version, tag, commit, built string, dry bool, t target) error {
	data, e := os.ReadFile("compatibility.json")
	if e != nil {
		return e
	}
	var manifest map[string]any
	if e = json.Unmarshal(data, &manifest); e != nil {
		return e
	}
	application, ok := manifest["application"].(map[string]any)
	if !ok {
		return fmt.Errorf("compatibility application record missing")
	}
	application["version"] = version
	application["release_tag"] = tag
	application["commit"] = commit
	application["build_date"] = built
	application["dry_run"] = dry
	application["artifact_target"] = t.OS + "/" + t.Arch
	application["runtime_supported"] = t.Supported
	if dry {
		application["release_status"] = "local dry-run; not published"
	} else {
		application["release_status"] = "exact clean tagged build; publication controlled externally"
	}
	return writeJSON(path, manifest)
}

// Inventory only modules actually linked by the release targets, not unrelated
// dependency-test modules in the complete module graph.
func moduleInventory(goTool string) ([]byte, error) {
	type module struct {
		Path, Version, Dir, Sum string
		Main                    bool
	}
	seen := map[string]module{}
	for _, target := range targets {
		cmd := exec.Command(goTool, "list", "-deps", "-json", "./cmd/tunnelctl")
		cmd.Env = append(os.Environ(), "CGO_ENABLED=0", "GOOS="+target.OS, "GOARCH="+target.Arch)
		cmd.Stderr = os.Stderr
		data, e := cmd.Output()
		if e != nil {
			return nil, e
		}
		decoder := json.NewDecoder(bytes.NewReader(data))
		for {
			var pkg struct{ Module *module }
			if e := decoder.Decode(&pkg); e == io.EOF {
				break
			} else if e != nil {
				return nil, e
			}
			if pkg.Module != nil && !pkg.Module.Main {
				seen[pkg.Module.Path] = *pkg.Module
			}
		}
	}
	keys := []string{}
	for key := range seen {
		keys = append(keys, key)
	}
	sort.Strings(keys)
	var buffer bytes.Buffer
	encoder := json.NewEncoder(&buffer)
	for _, key := range keys {
		if e := encoder.Encode(seen[key]); e != nil {
			return nil, e
		}
	}
	return buffer.Bytes(), nil
}
func writeJSON(path string, v any) error {
	b, e := json.MarshalIndent(v, "", "  ")
	if e != nil {
		return e
	}
	return os.WriteFile(path, b, 0644)
}

// Include upstream module license/notice texts, not just module names in an SBOM.
func collectLicenses(modules []byte, destination string) error {
	if e := os.Mkdir(destination, 0755); e != nil {
		return e
	}
	decoder := json.NewDecoder(strings.NewReader(string(modules)))
	for {
		var module struct {
			Path, Version, Dir string
			Main               bool
		}
		if e := decoder.Decode(&module); e == io.EOF {
			break
		} else if e != nil {
			return e
		}
		if module.Main {
			continue
		}
		entries, e := os.ReadDir(module.Dir)
		if e != nil {
			return e
		}
		found := false
		for _, entry := range entries {
			upper := strings.ToUpper(entry.Name())
			if entry.IsDir() || !(strings.HasPrefix(upper, "LICENSE") || strings.HasPrefix(upper, "COPYING") || strings.HasPrefix(upper, "NOTICE")) {
				continue
			}
			name := strings.ReplaceAll(module.Path, "/", "_") + "@" + module.Version + "_" + entry.Name()
			if e := copyFile(filepath.Join(module.Dir, entry.Name()), filepath.Join(destination, name)); e != nil {
				return e
			}
			if !strings.HasPrefix(upper, "NOTICE") {
				found = true
			}
		}
		if !found {
			return fmt.Errorf("module license text missing for %s; distribution review required", module.Path)
		}
	}
	for _, name := range []string{"LICENSE", "PATENTS"} {
		if e := copyFile(filepath.Join(runtime.GOROOT(), name), filepath.Join(destination, "Go_"+name)); e != nil {
			return e
		}
	}
	return nil
}
func copyFile(from, to string) error {
	b, e := os.ReadFile(from)
	if e != nil {
		return e
	}
	return os.WriteFile(to, b, 0644)
}
func copyTree(from, to string) error {
	return filepath.WalkDir(from, func(path string, d os.DirEntry, e error) error {
		if e != nil {
			return e
		}
		rel, e := filepath.Rel(from, path)
		if e != nil {
			return e
		}
		dest := filepath.Join(to, rel)
		if d.IsDir() {
			return os.MkdirAll(dest, 0755)
		}
		if d.Type()&os.ModeSymlink != 0 {
			return fmt.Errorf("symlinks are prohibited in release documents")
		}
		return copyFile(path, dest)
	})
}
func archive(dir, path string, isZip bool) (err error) {
	f, e := os.OpenFile(path, os.O_CREATE|os.O_EXCL|os.O_WRONLY, 0644)
	if e != nil {
		return e
	}
	defer func() {
		if e := f.Close(); err == nil {
			err = e
		}
	}()
	if isZip {
		z := zip.NewWriter(f)
		e = filepath.WalkDir(dir, func(p string, d os.DirEntry, e error) error {
			if e != nil {
				return e
			}
			if d.IsDir() {
				return nil
			}
			rel, e := filepath.Rel(dir, p)
			if e != nil {
				return e
			}
			st, e := d.Info()
			if e != nil {
				return e
			}
			header, e := zip.FileInfoHeader(st)
			if e != nil {
				return e
			}
			header.Name = filepath.ToSlash(rel)
			header.Method = zip.Deflate
			if strings.HasSuffix(rel, ".exe") {
				header.SetMode(0755)
			}
			w, e := z.CreateHeader(header)
			if e != nil {
				return e
			}
			r, e := os.Open(p)
			if e != nil {
				return e
			}
			defer r.Close()
			_, e = io.Copy(w, r)
			return e
		})
		ce := z.Close()
		if e != nil {
			return e
		}
		return ce
	}
	gz := gzip.NewWriter(f)
	tw := tar.NewWriter(gz)
	e = filepath.WalkDir(dir, func(p string, d os.DirEntry, e error) error {
		if e != nil {
			return e
		}
		if d.IsDir() {
			return nil
		}
		rel, e := filepath.Rel(dir, p)
		if e != nil {
			return e
		}
		st, e := d.Info()
		if e != nil {
			return e
		}
		h, e := tar.FileInfoHeader(st, "")
		if e != nil {
			return e
		}
		h.Name = filepath.ToSlash(rel)
		h.Uid = 0
		h.Gid = 0
		h.Uname = ""
		h.Gname = ""
		h.Mode = 0644
		if rel == "tunnelctl" {
			h.Mode = 0755
		}
		if e = tw.WriteHeader(h); e != nil {
			return e
		}
		r, e := os.Open(p)
		if e != nil {
			return e
		}
		defer r.Close()
		_, e = io.Copy(tw, r)
		return e
	})
	te := tw.Close()
	ge := gz.Close()
	if e != nil {
		return e
	}
	if te != nil {
		return te
	}
	return ge
}
