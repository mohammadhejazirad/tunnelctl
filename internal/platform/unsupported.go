//go:build !linux

package platform

import (
	"net"
	"os"
	"os/exec"
	"tunnelctl/internal/domain"
)

func Supported() bool             { return false }
func CheckOwner(os.FileInfo) bool { return false }
func Lock(*os.File) error {
	return domain.Fail(3, "This platform has no validated private storage or session implementation.", "Use Linux; native macOS and Windows support is deferred.")
}
func Unlock(*os.File) error          { return nil }
func ConfigureChild(*exec.Cmd)       {}
func Terminate(p *os.Process) error  { return p.Kill() }
func PeerAllowed(*net.UnixConn) bool { return false }
func StopSignals() []os.Signal       { return []os.Signal{os.Interrupt} }
func OpenPrivate(path string, flags int) (*os.File, error) {
	return nil, domain.Fail(3, "Private files are unsupported on this platform.", "Use Linux.")
}
func ListenerOwned(int, uint16) bool { return false }
