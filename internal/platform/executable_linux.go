//go:build linux

package platform

import (
	"golang.org/x/sys/unix"
	"os"
)

// Proxy operation must not acquire privileges from set-ID bits or file capabilities.
func UnprivilegedExecutable(f *os.File) bool {
	st, e := f.Stat()
	if e != nil || st.Mode()&(os.ModeSetuid|os.ModeSetgid) != 0 {
		return false
	}
	size, e := unix.Fgetxattr(int(f.Fd()), "security.capability", nil)
	return (e == nil && size == 0) || e == unix.ENODATA || e == unix.ENOTSUP
}
