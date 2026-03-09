package main

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

func main() {
	src := "test_clone_src.txt"
	dst := "test_clone_dst.txt"
	
	os.WriteFile(src, []byte("hello world, testing block cloning"), 0644)
	defer os.Remove(src)
	defer os.Remove(dst)
	
	s, err := os.Open(src)
	if err != nil {
		fmt.Println("Error opening src:", err)
		return
	}
	defer s.Close()

	stat, _ := s.Stat()
	size := stat.Size()

	d, err := os.Create(dst)
	if err != nil {
		fmt.Println("Error creating dst:", err)
		return
	}
	defer d.Close()

	sRc, _ := s.SyscallConn()
	var sourceHandle windows.Handle
	sRc.Control(func(fd uintptr) { sourceHandle = windows.Handle(fd) })

	dRc, _ := d.SyscallConn()
	var destHandle windows.Handle
	dRc.Control(func(fd uintptr) { destHandle = windows.Handle(fd) })

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
		fmt.Printf("DeviceIoControl failed: %v\n", err)
	} else {
		fmt.Println("Block cloning succeeded!")
	}
}
