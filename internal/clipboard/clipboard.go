package clipboard

import (
	"bytes"
	"context"
	"errors"
	"io"
	"os"
	"os/exec"
	"runtime"
	"strings"
	"tunnelctl/internal/domain"
)

const maxBytes = 10 * 1024 * 1024

var lookPath = exec.LookPath
var commandContext = exec.CommandContext

type command struct {
	name string
	args []string
}

type boundedBuffer struct {
	bytes.Buffer
	overflow bool
}

func (b *boundedBuffer) Write(p []byte) (int, error) {
	n := len(p)
	left := maxBytes + 1 - b.Len()
	if left > 0 {
		if len(p) > left {
			p = p[:left]
		}
		_, _ = b.Buffer.Write(p)
	}
	if b.Len() > maxBytes || n > left {
		b.overflow = true
	}
	return n, nil
}

func candidates() []command {
	switch runtime.GOOS {
	case "darwin":
		return []command{{"pbpaste", nil}}
	case "windows":
		return []command{{"powershell.exe", []string{"-NoProfile", "-NonInteractive", "-Command", "Get-Clipboard -Raw"}}}
	default:
		out := []command{}
		if os.Getenv("WAYLAND_DISPLAY") != "" {
			out = append(out, command{"wl-paste", []string{"--no-newline"}})
		}
		if os.Getenv("DISPLAY") != "" {
			out = append(out, command{"xclip", []string{"-selection", "clipboard", "-out"}}, command{"xsel", []string{"--clipboard", "--output"}})
		}
		return append(out, command{"wl-paste", []string{"--no-newline"}}, command{"xclip", []string{"-selection", "clipboard", "-out"}}, command{"xsel", []string{"--clipboard", "--output"}}, command{"powershell.exe", []string{"-NoProfile", "-NonInteractive", "-Command", "Get-Clipboard -Raw"}})
	}
}

func Read(ctx context.Context) (string, error) {
	var last error
	seen := map[string]bool{}
	for _, candidate := range candidates() {
		key := candidate.name + "\x00" + strings.Join(candidate.args, "\x00")
		if seen[key] {
			continue
		}
		seen[key] = true
		path, e := lookPath(candidate.name)
		if e != nil {
			last = e
			continue
		}
		var output boundedBuffer
		cmd := commandContext(ctx, path, candidate.args...)
		cmd.Stdout = &output
		cmd.Stderr = io.Discard
		if e = cmd.Run(); e != nil {
			last = e
			continue
		}
		if output.overflow {
			return "", domain.Fail(2, "Clipboard content exceeds 10 MiB.", "Copy one share link or subscription URL and retry.")
		}
		value := strings.TrimSpace(output.String())
		if value == "" {
			last = errors.New("empty clipboard")
			continue
		}
		return value, nil
	}
	if errors.Is(ctx.Err(), context.Canceled) || errors.Is(ctx.Err(), context.DeadlineExceeded) {
		return "", ctx.Err()
	}
	_ = last
	return "", domain.Fail(2, "The system clipboard could not be read.", "Install wl-clipboard, xclip or xsel on Linux, then press v again; manual paste with i remains available.")
}
