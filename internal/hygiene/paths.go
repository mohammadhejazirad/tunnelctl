// Package hygiene defines private/generated paths forbidden in source or packages.
package hygiene

import (
	"path/filepath"
	"strings"
)

func PrivatePath(path string) bool {
	for _, part := range strings.Split(filepath.ToSlash(path), "/") {
		if part == "links_and_subs.txt" || part == ".private" || part == "private-results" || part == "private-cache" || strings.HasSuffix(part, ".private.json") || strings.HasSuffix(part, ".private.log") || strings.HasPrefix(part, ".core-config-") || strings.HasPrefix(part, ".atomic-") {
			return true
		}
	}
	return false
}
