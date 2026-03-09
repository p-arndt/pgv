//go:build linux
// +build linux

package cowfs

import (
	"fmt"
	"os"

	"golang.org/x/sys/unix"
)

func cloneFile(src, dst string) error {
	s, err := os.Open(src)
	if err != nil {
		return err
	}
	defer s.Close()

	d, err := os.Create(dst)
	if err != nil {
		return err
	}
	defer d.Close()

	err = unix.IoctlSetInt(int(d.Fd()), unix.FICLONE, int(s.Fd()))
	if err != nil {
		return fmt.Errorf("FICLONE failed: %w", err)
	}
	return nil
}
