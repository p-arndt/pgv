//go:build !linux && !windows
// +build !linux,!windows

package cowfs

import (
	"fmt"
)

func cloneFile(src, dst string) error {
	return fmt.Errorf("CoW cloning is not supported on this operating system")
}
