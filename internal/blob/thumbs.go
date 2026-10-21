package blob

import (
	"errors"
	"io/fs"
	"os"
	"path"

	"picberth/internal/datadir"
	"picberth/internal/ids"
)

// ThumbExts are the extensions a thumbnail can have.
var ThumbExts = []string{"jpg", "png"}

func thumbPath(id, ext string) (string, error) {
	if !ids.ValidImageID(id) || (ext != "jpg" && ext != "png") {
		return "", ErrInvalidKey
	}
	return datadir.ThumbsDir + "/" + id[0:2] + "/" + id + "." + ext, nil
}

// WriteThumb writes cache/thumbs/ab/<id>.<ext> atomically (temp file in tmp/, then rename).
// Thumbnails are derived data, so no fsync is needed.
func (s *Store) WriteThumb(id, ext string, data []byte) error {
	dst, err := thumbPath(id, ext)
	if err != nil {
		return err
	}
	tmp, err := s.CreateTemp()
	if err != nil {
		return err
	}
	_, werr := tmp.Write(data)
	cerr := tmp.Close()
	if werr != nil || cerr != nil {
		s.RemoveTemp(tmp.Name())
		return errors.Join(werr, cerr)
	}
	if err := s.root.MkdirAll(path.Dir(dst), 0o700); err != nil {
		s.RemoveTemp(tmp.Name())
		return err
	}
	if err := retry(func() error { return s.root.Rename(tmp.Name(), dst) }); err != nil {
		s.RemoveTemp(tmp.Name())
		return err
	}
	return nil
}

// OpenThumb opens a thumbnail for reading.
func (s *Store) OpenThumb(id, ext string) (*os.File, error) {
	p, err := thumbPath(id, ext)
	if err != nil {
		return nil, err
	}
	return s.root.Open(p)
}

// RemoveThumbs deletes every thumbnail variant of an image.
func (s *Store) RemoveThumbs(id string) error {
	var errs []error
	for _, ext := range ThumbExts {
		p, err := thumbPath(id, ext)
		if err != nil {
			return err
		}
		if err := retry(func() error { return s.root.Remove(p) }); err != nil && !errors.Is(err, fs.ErrNotExist) {
			errs = append(errs, err)
		}
	}
	return errors.Join(errs...)
}
