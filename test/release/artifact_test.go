package release

import (
	"archive/tar"
	"archive/zip"
	"bytes"
	"compress/gzip"
	"crypto/sha256"
	"debug/buildinfo"
	"encoding/hex"
	"encoding/json"
	"io"
	"os"
	"os/exec"
	"path/filepath"
	"runtime"
	"strings"
	"testing"
	"tunnelctl/internal/hygiene"
)

func TestReleaseArchivesAndInstallation(t *testing.T) {
	dir, tag := os.Getenv("TUNNELCTL_RELEASE_DIR"), os.Getenv("TUNNELCTL_RELEASE_TAG")
	if dir == "" || tag == "" {
		t.Skip("set release directory and expected tag to check packaged artifacts")
	}
	b, e := os.ReadFile(filepath.Join(dir, "checksums.txt"))
	if e != nil {
		t.Fatal(e)
	}
	assets := 0
	for _, line := range strings.Split(strings.TrimSpace(string(b)), "\n") {
		fields := strings.Fields(line)
		if len(fields) != 2 || filepath.Base(fields[1]) != fields[1] {
			t.Fatal("invalid checksum record")
		}
		data, e := os.ReadFile(filepath.Join(dir, fields[1]))
		if e != nil {
			t.Fatal(e)
		}
		sum := sha256.Sum256(data)
		if hex.EncodeToString(sum[:]) != fields[0] {
			t.Fatal("checksum mismatch", fields[1])
		}
		if strings.HasSuffix(fields[1], ".tar.gz") || strings.HasSuffix(fields[1], ".zip") {
			assets++
			inspect(t, filepath.Join(dir, fields[1]), strings.HasSuffix(fields[1], "_linux_amd64.tar.gz"), tag)
		}
	}
	if assets != 5 {
		t.Fatalf("expected five clearly labeled target archives, found %d", assets)
	}
}
func inspect(t *testing.T, path string, native bool, tag string) {
	t.Helper()
	entries := map[string][]byte{}
	modes := map[string]os.FileMode{}
	valid := func(name string) {
		if hygiene.PrivatePath(name) || filepath.Base(name) == "private-test" || filepath.Base(name) == "private-test.exe" {
			t.Fatal("private input/report/tool entered release")
		}
		if name != filepath.ToSlash(filepath.Clean(name)) || strings.HasPrefix(name, "/") || strings.HasPrefix(name, "../") || strings.Contains(name, "\\") {
			t.Fatal("unsafe archive member", name)
		}
	}
	if strings.HasSuffix(path, ".zip") {
		z, e := zip.OpenReader(path)
		if e != nil {
			t.Fatal(e)
		}
		defer z.Close()
		for _, f := range z.File {
			valid(f.Name)
			if f.Mode()&os.ModeSymlink != 0 || f.UncompressedSize64 > 128*1024*1024 {
				t.Fatal("unsafe ZIP member")
			}
			r, e := f.Open()
			if e != nil {
				t.Fatal(e)
			}
			b, e := io.ReadAll(r)
			r.Close()
			if e != nil {
				t.Fatal(e)
			}
			entries[f.Name] = b
			modes[f.Name] = f.Mode()
		}
	} else {
		f, e := os.Open(path)
		if e != nil {
			t.Fatal(e)
		}
		defer f.Close()
		gz, e := gzip.NewReader(f)
		if e != nil {
			t.Fatal(e)
		}
		defer gz.Close()
		r := tar.NewReader(gz)
		for {
			h, e := r.Next()
			if e == io.EOF {
				break
			}
			if e != nil {
				t.Fatal(e)
			}
			valid(h.Name)
			if h.Typeflag != tar.TypeReg || h.Size > 128*1024*1024 {
				t.Fatal("unsafe tar member")
			}
			b, e := io.ReadAll(r)
			if e != nil {
				t.Fatal(e)
			}
			entries[h.Name] = b
			modes[h.Name] = os.FileMode(h.Mode)
		}
	}
	for _, name := range []string{"LICENSE", "NOTICE", "README.md", "CHANGELOG.md", "compatibility.json", "SUPPORT.txt", "docs/recovery.md", "docs/tui.md", "docs/private-testing.md", "THIRD_PARTY_LICENSES/Go_LICENSE"} {
		if len(entries[name]) == 0 {
			t.Fatal("required archive member missing", name)
		}
	}
	var compatibility struct {
		Application struct {
			Version    string
			ReleaseTag string `json:"release_tag"`
		}
		Modules []any
	}
	if e := json.Unmarshal(entries["compatibility.json"], &compatibility); e != nil || compatibility.Application.Version != strings.TrimPrefix(tag, "v") || compatibility.Application.ReleaseTag != tag || len(compatibility.Modules) == 0 {
		t.Fatal("packaged compatibility/version mismatch")
	}
	binaryName := "tunnelctl"
	if strings.HasSuffix(path, ".zip") {
		binaryName = "tunnelctl.exe"
	}
	info, e := buildinfo.Read(bytes.NewReader(entries[binaryName]))
	if e != nil {
		t.Fatal("cannot inspect target Go build metadata", e)
	}
	settings := map[string]string{}
	for _, setting := range info.Settings {
		settings[setting.Key] = setting.Value
	}
	if settings["CGO_ENABLED"] != "0" || !strings.Contains(filepath.Base(path), "_"+settings["GOOS"]+"_"+settings["GOARCH"]) {
		t.Fatal("target or CGO release metadata mismatch")
	}
	// Go deliberately omits -ldflags from build info with -trimpath (issue 52372).
	// Inspect injected strings here; actual version-command equivalence is tested
	// below only on the native target. This does not certify cross-target execution.
	if !bytes.Contains(entries[binaryName], []byte(tag)) || !bytes.Contains(entries[binaryName], []byte(strings.TrimPrefix(tag, "v"))) {
		t.Fatal("expected injected version/tag strings absent from cross-target binary")
	}
	if !native {
		return
	}
	if runtime.GOOS != "linux" || runtime.GOARCH != "amd64" {
		t.Skip("native archive execution requires Linux amd64")
	}
	if modes["tunnelctl"].Perm() != 0755 {
		t.Fatal("invalid Unix executable mode")
	}
	installed := filepath.Join(t.TempDir(), "tunnelctl")
	if e := os.WriteFile(installed, entries["tunnelctl"], 0700); e != nil {
		t.Fatal(e)
	}
	run := func(args ...string) []byte {
		t.Helper()
		b, e := exec.Command(installed, args...).CombinedOutput()
		if e != nil {
			t.Fatal("installed artifact failed", args, e, string(b))
		}
		return b
	}
	var version struct {
		OK   bool
		Data struct{ Application, Tag string }
	}
	if e := json.Unmarshal(run("version", "--json"), &version); e != nil || !version.OK || version.Data.Tag != tag || version.Data.Application != strings.TrimPrefix(tag, "v") {
		t.Fatal("binary/tag mismatch")
	}
	run("--help")
	run("completion", "powershell")
	dataDir := t.TempDir()
	os.Chmod(dataDir, 0700)
	run("--data-dir", dataDir, "init")
	importCommand := exec.Command(installed, "--data-dir", dataDir, "--json", "profile", "import", "--stdin")
	importCommand.Stdin = strings.NewReader("ss://aes-256-gcm:synthetic-release-only@127.0.0.1:8388#preserved-user-profile")
	if b, e := importCommand.CombinedOutput(); e != nil || bytes.Contains(b, []byte("synthetic-release-only")) {
		t.Fatal("private installation import failed or leaked credentials")
	}
	run("--data-dir", dataDir, "status", "--json")
	profilePath := filepath.Join(dataDir, "config", "profiles.json")
	before, e := os.ReadFile(profilePath)
	if e != nil {
		t.Fatal(e)
	}
	// Manual update: replace only the owned executable; do not touch profile state.
	if e = os.WriteFile(installed, entries["tunnelctl"], 0700); e != nil {
		t.Fatal(e)
	}
	run("version", "--json")
	run("--data-dir", dataDir, "stop")
	run("--data-dir", dataDir, "stop")
	// Manual uninstall preserves user data; no service or host networking is installed.
	if e = os.Remove(installed); e != nil {
		t.Fatal(e)
	}
	after, e := os.ReadFile(profilePath)
	if e != nil || string(before) != string(after) {
		t.Fatal("manual uninstall changed data")
	}
}
