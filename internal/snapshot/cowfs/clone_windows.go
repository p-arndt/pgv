//go:build windows
// +build windows

package cowfs

import (
	"fmt"
	"os"
	"unsafe"

	"golang.org/x/sys/windows"
)

const FSCTL_DUPLICATE_EXTENTS_TO_FILE = 0x98344

type duplicateExtentsData struct {
	FileHandle       windows.Handle
	SourceFileOffset int64
	TargetFileOffset int64
	ByteCount        int64
}

func cloneFile(src, dst string) error {
	s, err := os.Open(src)
	if err != nil {
		return err
	}
	defer s.Close()

	stat, err := s.Stat()
	if err != nil {
		return err
	}
	size := stat.Size()

	d, err := os.Create(dst)
	if err != nil {
		return err
	}
	defer d.Close()

	sRc, err := s.SyscallConn()
	if err != nil {
		return err
	}
	var sourceHandle windows.Handle
	sRc.Control(func(fd uintptr) {
		sourceHandle = windows.Handle(fd)
	})

	dRc, err := d.SyscallConn()
	if err != nil {
		return err
	}
	var destHandle windows.Handle
	dRc.Control(func(fd uintptr) {
		destHandle = windows.Handle(fd)
	})

	if size > 0 {
		data := duplicateExtentsData{
			FileHandle:       sourceHandle,
			SourceFileOffset: 0,
			TargetFileOffset: 0,
			ByteCount:        size,
		}

		var bytesReturned uint32
		err = windows.DeviceIoControl(
			destHandle,
			FSCTL_DUPLICATE_EXTENTS_TO_FILE,
			(*byte)(unsafe.Pointer(&data)),
			uint32(unsafe.Sizeof(data)),
			nil,
			0,
			&bytesReturned,
			nil,
		)
		if err != nil {
			return fmt.Errorf("DeviceIoControl FSCTL_DUPLICATE_EXTENTS_TO_FILE failed: %w", err)
		}
	}

	return nil
}
