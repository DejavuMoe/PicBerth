//go:build !windows

package blob

import (
	"errors"
	"syscall"
)

func transient(error) bool { return false }

// IsDiskFull reports whether err means the volume has no space left (or quota exceeded).
func IsDiskFull(err error) bool {
	return errors.Is(err, syscall.ENOSPC) || errors.Is(err, syscall.EDQUOT)
}

// syncDir fsyncs a directory so a rename inside it survives power loss.
func (s *Store) syncDir(rel string) error {
	d, err := s.root.Open(rel)
	if err != nil {
		return err
	}
	defer d.Close()
	if err := d.Sync(); err != nil && !errors.Is(err, syscall.EINVAL) {
		return err
	}
	return nil
}
