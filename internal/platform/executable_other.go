//go:build !linux

package platform

import "os"

func UnprivilegedExecutable(*os.File) bool { return false }
