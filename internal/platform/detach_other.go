//go:build !linux

package platform

import (
	"os"
	"os/exec"
	"tunnelctl/internal/domain"
)

func ConfigureDetached(*exec.Cmd) {}
func OpenSelfExecutable() (*os.File, error) {
	return nil, domain.Fail(3, "Detached execution is unsupported on this platform.", "Native lifecycle adapters are required.")
}
