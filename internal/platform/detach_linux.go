//go:build linux

package platform

import (
	"os"
	"os/exec"
	"syscall"
)

func ConfigureDetached(cmd *exec.Cmd) { cmd.SysProcAttr = &syscall.SysProcAttr{Setsid: true} }

func OpenSelfExecutable() (*os.File, error) { return os.Open("/proc/self/exe") }
