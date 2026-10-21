// Package datadir owns the on-disk data directory: layout, the format marker, the exclusive instance lock
// and the os.Root through which every file operation goes.
package datadir

import (
	"errors"
	"fmt"
	"io/fs"
	"os"
	"path/filepath"
	"runtime"
	"strconv"
	"strings"
)

// Layout constants (relative to the data directory).
const (
	FormatFile = "FORMAT"
	LockFile   = ".lock"
	DBFile     = "picberth.db"
	Objects    = "objects"
	Trash      = "trash"
	Tmp        = "tmp"
	ThumbsDir  = "cache/thumbs"
	Quarantine = "quarantine"
	Snapshots  = "snapshots"

	formatName   = "picberth-data"
	formatMajor  = 1
	longPathWarn = 240
)

// FormatContent is the content of the FORMAT marker.
var FormatContent = fmt.Sprintf("%s %d\n", formatName, formatMajor)

// ErrLocked is returned when another process holds the instance lock.
var ErrLocked = errors.New("data directory is in use by another PicBerth process")

// Options controls how a data directory is opened.
type Options struct {
	// Create makes the directory (and parents) when missing.
	Create bool
	// RequireEmpty fails unless the directory holds nothing but the lock file (used by restore).
	RequireEmpty bool
}

// Dir is an opened, locked data directory. Close releases the lock.
type Dir struct {
	path     string
	root     *os.Root
	lock     *os.File
	warnings []string
}

// Open locks and validates the directory, creating the layout as needed.
func Open(path string, opts Options) (*Dir, error) {
	abs, err := filepath.Abs(path)
	if err != nil {
		return nil, err
	}
	if strings.ContainsAny(abs, "?#") {
		return nil, fmt.Errorf("data directory path %q must not contain '?' or '#'", abs)
	}
	if st, err := os.Stat(abs); err != nil {
		if !errors.Is(err, fs.ErrNotExist) || !opts.Create {
			return nil, fmt.Errorf("data directory %s: %w", abs, err)
		}
		if err := os.MkdirAll(abs, 0o700); err != nil {
			return nil, fmt.Errorf("creating data directory: %w", err)
		}
	} else if !st.IsDir() {
		return nil, fmt.Errorf("data directory %s is not a directory", abs)
	}
	root, err := os.OpenRoot(abs)
	if err != nil {
		return nil, fmt.Errorf("opening data directory: %w", err)
	}
	d := &Dir{path: abs, root: root}
	lock, err := root.OpenFile(LockFile, os.O_RDWR|os.O_CREATE, 0o600)
	if err != nil {
		root.Close()
		return nil, fmt.Errorf("opening lock file: %w", err)
	}
	if err := tryLock(lock); err != nil {
		lock.Close()
		root.Close()
		if errors.Is(err, ErrLocked) {
			return nil, fmt.Errorf("%w (%s): stop the running server before using offline commands", ErrLocked, abs)
		}
		return nil, fmt.Errorf("locking data directory: %w", err)
	}
	d.lock = lock
	if err := d.checkFormat(opts); err != nil {
		d.Close()
		return nil, err
	}
	for _, sub := range []string{Objects, Trash, Tmp, ThumbsDir, Quarantine} {
		if err := root.MkdirAll(sub, 0o700); err != nil {
			d.Close()
			return nil, fmt.Errorf("creating %s: %w", sub, err)
		}
	}
	d.computeWarnings()
	return d, nil
}

func (d *Dir) checkFormat(opts Options) error {
	entries, err := d.listTop()
	if err != nil {
		return err
	}
	if opts.RequireEmpty && len(entries) > 0 {
		return fmt.Errorf("data directory %s is not empty", d.path)
	}
	data, err := d.root.ReadFile(FormatFile)
	switch {
	case err == nil:
		name, ver, ok := strings.Cut(strings.TrimSpace(string(data)), " ")
		major, convErr := strconv.Atoi(ver)
		if !ok || name != formatName || convErr != nil {
			return fmt.Errorf("%s: unrecognised FORMAT file", d.path)
		}
		if major != formatMajor {
			return fmt.Errorf("%s: data format %d is not supported by this build (supports %d)", d.path, major, formatMajor)
		}
		return nil
	case errors.Is(err, fs.ErrNotExist):
		if len(entries) > 0 {
			return fmt.Errorf("%s is not empty and has no FORMAT file: refusing to use it as a PicBerth data directory", d.path)
		}
		return d.writeFile(FormatFile, []byte(FormatContent))
	default:
		return fmt.Errorf("reading FORMAT: %w", err)
	}
}

// listTop returns the top-level entry names except the lock file.
func (d *Dir) listTop() ([]string, error) {
	f, err := d.root.Open(".")
	if err != nil {
		return nil, err
	}
	defer f.Close()
	ents, err := f.ReadDir(-1)
	if err != nil {
		return nil, err
	}
	var names []string
	for _, e := range ents {
		if e.Name() != LockFile {
			names = append(names, e.Name())
		}
	}
	return names, nil
}

func (d *Dir) writeFile(name string, data []byte) error {
	f, err := d.root.OpenFile(name, os.O_WRONLY|os.O_CREATE|os.O_TRUNC, 0o600)
	if err != nil {
		return err
	}
	if _, err := f.Write(data); err != nil {
		f.Close()
		return err
	}
	if err := f.Sync(); err != nil {
		f.Close()
		return err
	}
	return f.Close()
}

func (d *Dir) computeWarnings() {
	// Longest relative path: snapshots/<ts>/objects/ab/cd/<id>.avif is longer, but only live paths matter here.
	longest := len(d.path) + 1 + len("objects/ab/cd/") + 20 + len(".webp")
	if runtime.GOOS == "windows" && longest > longPathWarn {
		d.warnings = append(d.warnings, fmt.Sprintf("data directory path is long (%d chars before the file name); Windows path limits may be hit", len(d.path)))
	}
	if runtime.GOOS != "windows" {
		if st, err := d.root.Stat("."); err == nil && st.Mode().Perm()&0o077 != 0 {
			d.warnings = append(d.warnings, fmt.Sprintf("data directory permissions %v are wider than 0700", st.Mode().Perm()))
		}
	}
}

// Warnings lists non-fatal findings from opening the directory.
func (d *Dir) Warnings() []string { return d.warnings }

// Path is the absolute path of the directory.
func (d *Dir) Path() string { return d.path }

// Root is the os.Root all file operations must use.
func (d *Dir) Root() *os.Root { return d.root }

// DBPath is the absolute path of the SQLite database file.
func (d *Dir) DBPath() string { return filepath.Join(d.path, DBFile) }

// Abs returns the absolute path of a validated relative slash-separated key. Callers must only pass
// server-generated keys, never user input.
func (d *Dir) Abs(rel string) string { return filepath.Join(d.path, filepath.FromSlash(rel)) }

// FreeSpace reports free (available to this user) and total bytes of the volume holding the directory.
func (d *Dir) FreeSpace() (free, total uint64, err error) { return freeSpace(d.path) }

// Close releases the lock and the root. It is safe to call more than once.
func (d *Dir) Close() error {
	var err error
	if d.lock != nil {
		unlock(d.lock)
		err = d.lock.Close()
		d.lock = nil
	}
	if d.root != nil {
		if e := d.root.Close(); err == nil {
			err = e
		}
		d.root = nil
	}
	return err
}
