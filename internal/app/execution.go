package app

import (
	"os"
	"path/filepath"
	"strings"
	"tunnelctl/internal/domain"
)

// The pinned AppImage runtime's transient extractor forks/waits without
// forwarding directed signals. Refuse sessions there rather than pretending
// the extra process has normal foreground/dead-supervisor semantics.
func checkSessionExecution() error {
	if os.Getenv("APPIMAGE") != "" && (os.Getenv("APPIMAGE_EXTRACT_AND_RUN") != "" || strings.HasPrefix(filepath.Base(os.Getenv("APPDIR")), "appimage_extracted_")) {
		return domain.Fail(3, "Transient AppImage extraction cannot safely supervise sessions.", "Use --appimage-extract in a stable private directory and run squashfs-root/AppRun, or install a native package. Do not use --appimage-extract-and-run for sessions.")
	}
	return nil
}
