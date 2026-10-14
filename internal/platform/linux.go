//go:build linux

package platform

import (
	"fmt"
	"net"
	"os"
	"os/exec"
	"strconv"
	"strings"
	"syscall"
	"tunnelctl/internal/domain"
)

func Supported() bool { return true }
func CheckOwner(info os.FileInfo) bool {
	st, ok := info.Sys().(*syscall.Stat_t)
	return ok && int(st.Uid) == os.Getuid()
}
func Lock(f *os.File) error {
	if e := syscall.Flock(int(f.Fd()), syscall.LOCK_EX|syscall.LOCK_NB); e != nil {
		return domain.Fail(6, "Another operation owns the lock.", "Wait for the current operation or stop the active session.")
	}
	return nil
}
func Unlock(f *os.File) error { return syscall.Flock(int(f.Fd()), syscall.LOCK_UN) }
func ConfigureChild(cmd *exec.Cmd) {
	cmd.SysProcAttr = &syscall.SysProcAttr{Setpgid: true, Pdeathsig: syscall.SIGKILL}
}
func Terminate(p *os.Process) error { return p.Signal(syscall.SIGTERM) }
func PeerAllowed(c *net.UnixConn) bool {
	raw, e := c.SyscallConn()
	if e != nil {
		return false
	}
	allowed := false
	e = raw.Control(func(fd uintptr) {
		cred, err := syscall.GetsockoptUcred(int(fd), syscall.SOL_SOCKET, syscall.SO_PEERCRED)
		allowed = err == nil && int(cred.Uid) == os.Getuid()
	})
	return e == nil && allowed
}
func StopSignals() []os.Signal { return []os.Signal{os.Interrupt, syscall.SIGTERM, syscall.SIGHUP} }
func OpenPrivate(path string, flags int) (*os.File, error) {
	fd, e := syscall.Open(path, flags|syscall.O_NOFOLLOW|syscall.O_CLOEXEC, 0600)
	if e != nil {
		return nil, e
	}
	return os.NewFile(uintptr(fd), path), nil
}
func ListenerOwned(pid int, port uint16) bool {
	data, e := os.ReadFile("/proc/" + strconv.Itoa(pid) + "/net/tcp")
	if e != nil {
		return false
	}
	target := fmt.Sprintf("0100007F:%04X", port)
	inode := ""
	for _, line := range strings.Split(string(data), "\n") {
		f := strings.Fields(line)
		if len(f) > 9 && f[1] == target && f[3] == "0A" {
			inode = f[9]
			break
		}
	}
	if inode == "" {
		return false
	}
	fds, e := os.ReadDir("/proc/" + strconv.Itoa(pid) + "/fd")
	if e != nil {
		return false
	}
	for _, fd := range fds {
		link, e := os.Readlink("/proc/" + strconv.Itoa(pid) + "/fd/" + fd.Name())
		if e == nil && link == "socket:["+inode+"]" {
			return true
		}
	}
	return false
}
