// Package blob stores image bytes in the data directory through a single os.Root. Object keys are
// server-generated ("ab/cd/<id>.<ext>") and validated before use; user input never reaches a path.
package blob

import (
	"errors"
	"fmt"
	"io/fs"
	"os"
	"path"
	"regexp"
	"strconv"
	"strings"
	"time"

	"picberth/internal/datadir"
	"picberth/internal/ids"
)

// Area is a top-level directory objects live in.
type Area string

const (
	Objects Area = datadir.Objects
	Trash   Area = datadir.Trash
)

// Other returns the other area.
func (a Area) Other() Area {
	if a == Objects {
		return Trash
	}
	return Objects
}

var keyRE = regexp.MustCompile(`^[0-9a-z]{2}/[0-9a-z]{2}/[0-9a-z]{20}\.[0-9a-z]{1,8}$`)

// ValidKey reports whether key has the exact shape ObjectKey produces.
func ValidKey(key string) bool {
	if !keyRE.MatchString(key) {
		return false
	}
	// the shard directories must be the first characters of the file name
	return key[0:2] == key[6:8] && key[3:5] == key[8:10]
}

// ErrInvalidKey is returned when a key does not have the server-generated shape.
var ErrInvalidKey = errors.New("blob: invalid object key")

// ErrExists is returned when a commit would overwrite an existing object.
var ErrExists = errors.New("blob: object already exists")

// Store is the file store.
type Store struct {
	root *os.Root
}

// New wraps an opened data-directory root.
func New(root *os.Root) *Store { return &Store{root: root} }

// Root exposes the underlying root (snapshot and restore copy through it).
func (s *Store) Root() *os.Root { return s.root }

func (s *Store) rel(a Area, key string) (string, error) {
	if !ValidKey(key) {
		return "", ErrInvalidKey
	}
	return string(a) + "/" + key, nil
}

// Temp is an in-progress upload file in tmp/.
type Temp struct {
	*os.File
	name string
}

// Name is the slash-separated path of the temp file relative to the data directory.
func (t *Temp) Name() string { return t.name }

// CreateTemp creates tmp/up-<random>.part, exclusively.
func (s *Store) CreateTemp() (*Temp, error) {
	for i := 0; i < 5; i++ {
		name := datadir.Tmp + "/up-" + ids.RandomHex(12) + ".part"
		f, err := s.root.OpenFile(name, os.O_RDWR|os.O_CREATE|os.O_EXCL, 0o600)
		if err == nil {
			return &Temp{File: f, name: name}, nil
		}
		if !errors.Is(err, fs.ErrExist) {
			return nil, err
		}
	}
	return nil, errors.New("blob: could not create a unique temp file")
}

// RemoveTemp deletes a temp file (best effort, retried on Windows sharing violations).
func (s *Store) RemoveTemp(name string) error {
	if !strings.HasPrefix(name, datadir.Tmp+"/") || strings.Contains(name[len(datadir.Tmp)+1:], "/") {
		return ErrInvalidKey
	}
	return ignoreNotExist(retry(func() error { return s.root.Remove(name) }))
}

// CommitTemp renames a closed, fsynced temp file to objects/key and makes the rename durable.
// It never replaces an existing object.
func (s *Store) CommitTemp(tmpName, key string) error {
	dst, err := s.rel(Objects, key)
	if err != nil {
		return err
	}
	if !strings.HasPrefix(tmpName, datadir.Tmp+"/") {
		return ErrInvalidKey
	}
	if _, err := s.root.Lstat(dst); err == nil {
		return ErrExists
	}
	dir := path.Dir(dst)
	if err := s.root.MkdirAll(dir, 0o700); err != nil {
		return err
	}
	if err := retry(func() error { return s.root.Rename(tmpName, dst) }); err != nil {
		return err
	}
	return s.syncDir(dir)
}

// Move relocates an object between areas (delete = Objects→Trash, restore = Trash→Objects). It is
// idempotent: if the source is gone and the destination exists it succeeds. If neither exists it
// returns an error wrapping fs.ErrNotExist.
func (s *Store) Move(from, to Area, key string) error {
	src, err := s.rel(from, key)
	if err != nil {
		return err
	}
	dst, _ := s.rel(to, key)
	if _, err := s.root.Lstat(src); err != nil {
		if errors.Is(err, fs.ErrNotExist) {
			if _, err2 := s.root.Lstat(dst); err2 == nil {
				return nil
			}
		}
		return err
	}
	if _, err := s.root.Lstat(dst); err == nil {
		return fmt.Errorf("blob: %s already exists: %w", dst, ErrExists)
	}
	dir := path.Dir(dst)
	if err := s.root.MkdirAll(dir, 0o700); err != nil {
		return err
	}
	if err := retry(func() error { return s.root.Rename(src, dst) }); err != nil {
		return err
	}
	if err := s.syncDir(dir); err != nil {
		return err
	}
	return s.syncDir(path.Dir(src))
}

// Remove deletes an object; a missing file is not an error.
func (s *Store) Remove(a Area, key string) error {
	p, err := s.rel(a, key)
	if err != nil {
		return err
	}
	return ignoreNotExist(retry(func() error { return s.root.Remove(p) }))
}

// Stat returns the file info of an object.
func (s *Store) Stat(a Area, key string) (fs.FileInfo, error) {
	p, err := s.rel(a, key)
	if err != nil {
		return nil, err
	}
	return s.root.Lstat(p)
}

// Open opens an object for reading.
func (s *Store) Open(a Area, key string) (*os.File, error) {
	p, err := s.rel(a, key)
	if err != nil {
		return nil, err
	}
	return s.root.Open(p)
}

// Quarantine moves an object found without a record into quarantine/<unix-ms>-<name> and returns that path.
func (s *Store) Quarantine(a Area, key string, now time.Time) (string, error) {
	src, err := s.rel(a, key)
	if err != nil {
		return "", err
	}
	return s.quarantinePath(src, path.Base(key), now)
}

// QuarantineRaw quarantines an arbitrary (but already root-relative) path found by a directory walk,
// for files whose names do not even look like object keys.
func (s *Store) QuarantineRaw(rel string, now time.Time) (string, error) {
	return s.quarantinePath(rel, strings.ReplaceAll(rel, "/", "_"), now)
}

func (s *Store) quarantinePath(src, name string, now time.Time) (string, error) {
	for i := 0; i < 100; i++ {
		dst := datadir.Quarantine + "/" + strconv.FormatInt(now.UnixMilli(), 10) + "-" + name
		if i > 0 {
			dst += "." + strconv.Itoa(i)
		}
		if _, err := s.root.Lstat(dst); err == nil {
			continue
		}
		if err := s.root.MkdirAll(datadir.Quarantine, 0o700); err != nil {
			return "", err
		}
		if err := retry(func() error { return s.root.Rename(src, dst) }); err != nil {
			return "", err
		}
		return dst, nil
	}
	return "", errors.New("blob: could not choose a quarantine name")
}

// File is an entry found by Walk.
type File struct {
	Rel     string // path relative to the data directory, slash-separated
	Key     string // the part below the area; empty when it does not look like an object key
	Size    int64
	ModTime time.Time
	Regular bool
}

// Walk visits every non-directory below an area.
func (s *Store) Walk(a Area, fn func(File) error) error {
	return fs.WalkDir(s.root.FS(), string(a), func(p string, d fs.DirEntry, err error) error {
		if err != nil {
			if errors.Is(err, fs.ErrNotExist) {
				return nil
			}
			return err
		}
		if d.IsDir() {
			return nil
		}
		f := File{Rel: p, Regular: d.Type().IsRegular()}
		if info, err := d.Info(); err == nil {
			f.Size, f.ModTime = info.Size(), info.ModTime()
		}
		if key := strings.TrimPrefix(p, string(a)+"/"); ValidKey(key) {
			f.Key = key
		}
		return fn(f)
	})
}

// TempFile describes a leftover in tmp/.
type TempFile struct {
	Name    string
	ModTime time.Time
}

// ListTemp lists files in tmp/.
func (s *Store) ListTemp() ([]TempFile, error) {
	d, err := s.root.Open(datadir.Tmp)
	if err != nil {
		return nil, err
	}
	defer d.Close()
	ents, err := d.ReadDir(-1)
	if err != nil {
		return nil, err
	}
	var out []TempFile
	for _, e := range ents {
		if e.IsDir() {
			continue
		}
		var mt time.Time
		if info, err := e.Info(); err == nil {
			mt = info.ModTime()
		}
		out = append(out, TempFile{Name: datadir.Tmp + "/" + e.Name(), ModTime: mt})
	}
	return out, nil
}

func ignoreNotExist(err error) error {
	if errors.Is(err, fs.ErrNotExist) {
		return nil
	}
	return err
}
