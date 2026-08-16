//go:build windows

package system

import (
	"fmt"
	"os"

	"golang.org/x/sys/windows"
)

func openFileNoFollow(path string, flags int, mode os.FileMode) (*os.File, error) {
	pathPtr, err := windows.UTF16PtrFromString(path)
	if err != nil {
		return nil, err
	}

	access := uint32(windows.GENERIC_READ)
	switch flags & (os.O_WRONLY | os.O_RDWR) {
	case os.O_WRONLY:
		access = windows.GENERIC_WRITE
	case os.O_RDWR:
		access = windows.GENERIC_READ | windows.GENERIC_WRITE
	}
	if flags&os.O_APPEND != 0 {
		access &^= windows.GENERIC_WRITE
		access |= windows.FILE_APPEND_DATA
	}

	creation := uint32(windows.OPEN_EXISTING)
	switch {
	case flags&(os.O_CREATE|os.O_EXCL) == os.O_CREATE|os.O_EXCL:
		creation = windows.CREATE_NEW
	case flags&(os.O_CREATE|os.O_TRUNC) == os.O_CREATE|os.O_TRUNC:
		creation = windows.CREATE_ALWAYS
	case flags&os.O_CREATE != 0:
		creation = windows.OPEN_ALWAYS
	case flags&os.O_TRUNC != 0:
		creation = windows.TRUNCATE_EXISTING
	}

	attrs := uint32(windows.FILE_ATTRIBUTE_NORMAL | windows.FILE_FLAG_OPEN_REPARSE_POINT)
	if mode.Perm()&0200 == 0 {
		attrs |= windows.FILE_ATTRIBUTE_READONLY
	}
	handle, err := windows.CreateFile(pathPtr, access,
		windows.FILE_SHARE_READ|windows.FILE_SHARE_WRITE, nil, creation, attrs, 0)
	if err != nil {
		return nil, err
	}

	var info windows.ByHandleFileInformation
	if err := windows.GetFileInformationByHandle(handle, &info); err != nil {
		_ = windows.CloseHandle(handle)
		return nil, err
	}
	if info.FileAttributes&windows.FILE_ATTRIBUTE_REPARSE_POINT != 0 {
		_ = windows.CloseHandle(handle)
		return nil, fmt.Errorf("refusing to open final-component reparse point: %s", path)
	}
	return os.NewFile(uintptr(handle), path), nil
}
