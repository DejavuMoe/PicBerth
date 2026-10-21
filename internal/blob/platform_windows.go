//go:build windows

package blob

import (
	"errors"
	"io/fs"
	"syscall"
)

const (
	errorAccessDenied    syscall.Errno = 5
	errorSharingViolaton syscall.Errno = 32
	errorLockViolation   syscall.Errno = 33
	errorDiskFull        syscall.Errno = 112
	errorHandleDiskFull  syscall.Errno = 39
)

func transient(err error) bool {
	if errors.Is(err, fs.ErrNotExist) || errors.Is(err, fs.ErrExist) {
		return false
	}
	var no syscall.Errno
	if errors.As(err, &no) {
		return no == errorAccessDenied || no == errorSharingViolaton || no == errorLockViolation
	}
	return false
}

// IsDiskFull reports whether err means the volume has no space left.
func IsDiskFull(err error) bool {
	var no syscall.Errno
	if errors.As(err, &no) {
		return no == errorDiskFull || no == errorHandleDiskFull
	}
	return false
}

// syncDir is a no-op on Windows: directories cannot be opened for fsync; NTFS journals metadata.
func (s *Store) syncDir(string) error { return nil }
