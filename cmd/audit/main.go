// Command audit performs a narrow repository hygiene check, not a complete secret scan.
package main

import (
	"bytes"
	"fmt"
	"os"
	"os/exec"
	"path/filepath"
	"regexp"
	"strings"
)

var credential = regexp.MustCompile(`-----BEGIN (?:RSA |EC |OPENSSH )?PRIVATE KEY-----|gh[pousr]_[A-Za-z0-9]{30,}|github_pat_[A-Za-z0-9_]{40,}|AKIA[A-Z0-9]{16}`)
var secretLink = regexp.MustCompile(`(?:ss|vless|vmess|trojan)://[^\s"'` + "`" + `]+|https?://[^\s"']+[?&](?:token|password|secret|key)=[^\s"']+`)

func main() {
	if e := audit(); e != nil {
		fmt.Fprintln(os.Stderr, e)
		os.Exit(1)
	}
}
func audit() error {
	files, e := exec.Command("git", "ls-files", "-z", "--cached", "--others", "--exclude-standard").Output()
	if e != nil {
		return e
	}
	for _, name := range bytes.Split(files, []byte{0}) {
		if len(name) == 0 {
			continue
		}
		path := string(name)
		data, e := os.ReadFile(path)
		if e != nil {
			return fmt.Errorf("cannot inspect repository path: %s", path)
		}
		if e = inspect(path, data); e != nil {
			return e
		}
	}
	fmt.Println("Repository hygiene checks passed; synthetic fixtures remain allowed. This is not a comprehensive secret scan.")
	return nil
}

func inspect(path string, data []byte) error {
	base := filepath.Base(path)
	if strings.HasPrefix(path, "bin/") || strings.HasPrefix(path, "dist/") || strings.HasPrefix(path, ".local/") || base == ".env" || strings.HasPrefix(base, ".env.") || (strings.HasPrefix(base, "profiles.") && strings.HasSuffix(base, ".json")) || base == "session.json" || base == "events.json" || base == "control.sock" || strings.HasPrefix(base, ".core-config-") || strings.HasPrefix(base, ".atomic-") || strings.HasSuffix(base, ".key") || strings.HasSuffix(base, ".pem") {
		return fmt.Errorf("sensitive/generated path is not permitted: %s", path)
	}
	if credential.Match(data) {
		return fmt.Errorf("possible credential detected in %s (value withheld)", path)
	}
	fixture := strings.HasSuffix(path, "_test.go") || path == "examples/shadowsocks.txt"
	if !fixture && secretLink.Match(data) {
		return fmt.Errorf("possible private link detected in %s (value withheld)", path)
	}
	return nil
}
