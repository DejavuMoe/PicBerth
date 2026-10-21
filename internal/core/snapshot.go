package core

import (
	"context"
	"crypto/sha256"
	"encoding/hex"
	"encoding/json"
	"errors"
	"fmt"
	"io"
	"io/fs"
	"log/slog"
	"os"
	"path"
	"path/filepath"
	"runtime"
	"time"

	"picberth/internal/blob"
	"picberth/internal/buildinfo"
	"picberth/internal/datadir"
	"picberth/internal/meta"
)

const snapshotFormat = "picberth-snapshot 1"

// Manifest is MANIFEST.json inside a snapshot directory.
type Manifest struct {
	Format        string `json:"format"`
	DataFormat    string `json:"dataFormat"`
	Version       string `json:"version"`
	SchemaVersion int    `json:"schemaVersion"`
	CreatedAt     string `json:"createdAt"`
	Images        int    `json:"images"`
	Bytes         int64  `json:"bytes"`
}

// SnapshotInfo is the result of a snapshot.
type SnapshotInfo struct {
	Path      string `json:"path"`
	Images    int    `json:"images"`
	Bytes     int64  `json:"bytes"`
	CreatedAt string `json:"createdAt"`
}

// Snapshot writes a consistent, verified copy of the library below parent (the configured snapshot
// directory when empty) following docs/architecture.md §5. On any failure nothing is left behind.
func (s *Service) Snapshot(ctx context.Context, p Principal, parent string) (info SnapshotInfo, err error) {
	if !p.IsAdmin() {
		return SnapshotInfo{}, s.denyAdmin(p)
	}
	if parent == "" {
		parent = s.cfg.SnapshotDir
	}
	parent, err = filepath.Abs(parent)
	if err != nil {
		return SnapshotInfo{}, err
	}

	s.purgeMu.Lock() // pause purging; uploads and deletes continue
	defer s.purgeMu.Unlock()

	if err := os.MkdirAll(parent, 0o700); err != nil {
		return SnapshotInfo{}, fmt.Errorf("creating snapshot directory: %w", err)
	}
	created := s.now().UTC()
	var dest string
	for i := 0; ; i++ {
		name := created.Format("20060102T150405Z")
		if i > 0 {
			name += fmt.Sprintf("-%d", i)
		}
		dest = filepath.Join(parent, name)
		if err = os.Mkdir(dest, 0o700); err == nil {
			break
		}
		if !errors.Is(err, fs.ErrExist) || i >= 20 {
			return SnapshotInfo{}, fmt.Errorf("creating snapshot directory: %w", err)
		}
	}
	defer func() {
		if err != nil {
			if rmErr := os.RemoveAll(dest); rmErr != nil {
				s.log.Warn("could not remove failed snapshot", "path", dest, "err", rmErr)
			}
		}
	}()

	droot, err := os.OpenRoot(dest)
	if err != nil {
		return SnapshotInfo{}, err
	}
	defer droot.Close()

	dbDest := filepath.Join(dest, datadir.DBFile)
	if err = s.db.VacuumInto(ctx, dbDest); err != nil {
		return SnapshotInfo{}, fmt.Errorf("vacuum into snapshot: %w", err)
	}
	if err = syncPath(dbDest); err != nil {
		return SnapshotInfo{}, err
	}
	snapDB, err := meta.OpenReadOnly(dbDest)
	if err != nil {
		return SnapshotInfo{}, err
	}
	defer snapDB.Close()
	if problems, perr := meta.Integrity(ctx, snapDB); perr != nil {
		return SnapshotInfo{}, perr
	} else if len(problems) > 0 {
		return SnapshotInfo{}, fmt.Errorf("snapshot database failed verification: %v", problems)
	}

	var images int
	var bytes int64
	err = meta.ForEachLiveImage(ctx, snapDB, func(r meta.ImageRef) error {
		if err := ctx.Err(); err != nil {
			return err
		}
		if !blob.ValidKey(r.ObjectKey) {
			return fmt.Errorf("record %s has an invalid object key", r.ID)
		}
		if err := s.copyVerified(droot, r); err != nil {
			return err
		}
		images++
		bytes += r.Size
		return nil
	})
	if err != nil {
		return SnapshotInfo{}, err
	}
	if err = snapDB.Close(); err != nil {
		return SnapshotInfo{}, err
	}

	mf := Manifest{Format: snapshotFormat, DataFormat: datadir.FormatContent[:len(datadir.FormatContent)-1], Version: buildinfo.Version,
		SchemaVersion: meta.SchemaVersion(), CreatedAt: created.Format(time.RFC3339), Images: images, Bytes: bytes}
	data, _ := json.MarshalIndent(mf, "", "  ")
	if err = writeSynced(droot, "MANIFEST.json", data); err != nil {
		return SnapshotInfo{}, err
	}
	if err = syncPath(dest); err != nil {
		return SnapshotInfo{}, err
	}

	info = SnapshotInfo{Path: dest, Images: images, Bytes: bytes, CreatedAt: mf.CreatedAt}
	last, _ := json.Marshal(map[string]any{"at": mf.CreatedAt, "path": dest, "images": images, "bytes": bytes})
	if kerr := s.db.KVSet(ctx, "last_snapshot", string(last)); kerr != nil {
		s.log.Warn("could not record snapshot", "err", kerr)
	}
	s.audit(ctx, p, "snapshot", dest, fmt.Sprintf("%d images, %d bytes", images, bytes))
	return info, nil
}

// copyVerified places one referenced file into the snapshot (hard link, falling back to a copy) and
// compares size and SHA-256 with the recorded values.
func (s *Service) copyVerified(droot *os.Root, r meta.ImageRef) error {
	area := blob.Objects
	if r.State == meta.StateTrashed {
		area = blob.Trash
	}
	// A restore or trash after the VACUUM may have moved the file to the other area; look in both.
	var srcAbs string
	for _, a := range []blob.Area{area, area.Other()} {
		if _, err := s.blobs.Stat(a, r.ObjectKey); err == nil {
			srcAbs = s.dir.Abs(string(a) + "/" + r.ObjectKey)
			break
		}
	}
	if srcAbs == "" {
		return fmt.Errorf("image %s: file %s is missing", r.ID, r.ObjectKey)
	}
	rel := string(area) + "/" + r.ObjectKey
	if err := droot.MkdirAll(path.Dir(rel), 0o700); err != nil {
		return err
	}
	dstAbs := filepath.Join(droot.Name(), filepath.FromSlash(rel))
	if err := os.Link(srcAbs, dstAbs); err != nil {
		if cerr := copyFile(srcAbs, droot, rel); cerr != nil {
			return fmt.Errorf("image %s: cannot link or copy %s: %w", r.ID, r.ObjectKey, cerr)
		}
	}
	size, sum, err := hashFile(droot, rel)
	if err != nil {
		return fmt.Errorf("image %s: %w", r.ID, err)
	}
	if size != r.Size || sum != r.SHA256 {
		return fmt.Errorf("image %s: file %s does not match its recorded size/hash (corrupted?)", r.ID, r.ObjectKey)
	}
	return nil
}

func copyFile(srcAbs string, droot *os.Root, rel string) error {
	in, err := os.Open(srcAbs)
	if err != nil {
		return err
	}
	defer in.Close()
	out, err := droot.OpenFile(rel, os.O_WRONLY|os.O_CREATE|os.O_EXCL, 0o600)
	if err != nil {
		return err
	}
	if _, err := io.Copy(out, in); err != nil {
		out.Close()
		return err
	}
	if err := out.Sync(); err != nil {
		out.Close()
		return err
	}
	return out.Close()
}

func hashFile(root *os.Root, rel string) (int64, string, error) {
	f, err := root.Open(rel)
	if err != nil {
		return 0, "", err
	}
	defer f.Close()
	h := sha256.New()
	n, err := io.Copy(h, f)
	if err != nil {
		return 0, "", err
	}
	return n, hex.EncodeToString(h.Sum(nil)), nil
}

func writeSynced(root *os.Root, name string, data []byte) error {
	f, err := root.OpenFile(name, os.O_WRONLY|os.O_CREATE|os.O_EXCL, 0o600)
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

// syncPath fsyncs a file, or a directory on platforms that allow it (Windows cannot fsync directories).
func syncPath(p string) error {
	st, err := os.Stat(p)
	if err != nil {
		return err
	}
	if st.IsDir() {
		if runtime.GOOS == "windows" {
			return nil
		}
		d, err := os.Open(p)
		if err != nil {
			return err
		}
		defer d.Close()
		return d.Sync()
	}
	f, err := os.OpenFile(p, os.O_RDWR, 0) // FlushFileBuffers needs write access
	if err != nil {
		return err
	}
	defer f.Close()
	return f.Sync()
}

// RestoreError carries the check report so the CLI can print what was wrong.
type RestoreError struct {
	Report *CheckReport
	Err    error
}

func (e *RestoreError) Error() string { return e.Err.Error() }
func (e *RestoreError) Unwrap() error { return e.Err }

// Restore rebuilds a data directory from a snapshot and verifies it with a full check. The target must be
// empty or absent. Any issue at all (even a repaired one) is an error; on failure the directory is emptied
// again.
func Restore(ctx context.Context, from, dataDir string, cfg Config, log *slog.Logger) (rep *CheckReport, err error) {
	from, err = filepath.Abs(from)
	if err != nil {
		return nil, err
	}
	sroot, err := os.OpenRoot(from)
	if err != nil {
		return nil, fmt.Errorf("opening snapshot: %w", err)
	}
	defer sroot.Close()
	raw, err := sroot.ReadFile("MANIFEST.json")
	if err != nil {
		return nil, fmt.Errorf("reading MANIFEST.json (is %s a snapshot?): %w", from, err)
	}
	var mf Manifest
	if err := json.Unmarshal(raw, &mf); err != nil || mf.Format != snapshotFormat {
		return nil, errors.New("MANIFEST.json is not a PicBerth snapshot manifest")
	}
	if mf.SchemaVersion > meta.SchemaVersion() {
		return nil, fmt.Errorf("%w: snapshot schema %d, this build knows %d", meta.ErrSchemaTooNew, mf.SchemaVersion, meta.SchemaVersion())
	}

	dir, err := datadir.Open(dataDir, datadir.Options{Create: true, RequireEmpty: true})
	if err != nil {
		return nil, err
	}
	defer dir.Close()
	defer func() {
		if err != nil { // the directory was empty before: leave it empty again
			cleanDir(dir)
		}
	}()

	if err = copyTree(ctx, sroot, dir.Root(), datadir.DBFile); err != nil {
		return nil, err
	}
	for _, area := range []string{datadir.Objects, datadir.Trash} {
		if _, serr := sroot.Stat(area); serr != nil {
			continue
		}
		if err = copyTree(ctx, sroot, dir.Root(), area); err != nil {
			return nil, err
		}
	}

	db, err := meta.Open(ctx, dir.DBPath())
	if err != nil {
		return nil, fmt.Errorf("opening restored database: %w", err)
	}
	defer db.Close()
	var problems []string
	if problems, err = meta.Integrity(ctx, db.Reader()); err != nil {
		return nil, err
	}
	if len(problems) > 0 {
		err = fmt.Errorf("restored database failed verification: %v", problems)
		return nil, err
	}
	svc, err := New(db, dir, cfg, log)
	if err != nil {
		return nil, err
	}
	defer svc.Close()
	rep, err = svc.Check(ctx, CLIPrincipal(), CheckFull)
	if err != nil {
		return nil, err
	}
	if len(rep.Issues) > 0 {
		err = &RestoreError{Report: rep, Err: fmt.Errorf("restored data directory has %d issue(s)", len(rep.Issues))}
		return rep, err
	}
	if mf.Images != int(countLive(ctx, db)) {
		err = &RestoreError{Report: rep, Err: fmt.Errorf("snapshot manifest lists %d images but the database holds a different number", mf.Images)}
		return rep, err
	}
	return rep, nil
}

func countLive(ctx context.Context, db *meta.DB) int64 {
	st, err := db.ImageStats(ctx)
	if err != nil {
		return -1
	}
	return st.Active + st.Trashed
}

// cleanDir removes everything inside a data directory except the lock file.
func cleanDir(dir *datadir.Dir) {
	f, err := dir.Root().Open(".")
	if err != nil {
		return
	}
	ents, _ := f.ReadDir(-1)
	f.Close()
	for _, e := range ents {
		if e.Name() == datadir.LockFile {
			continue
		}
		_ = dir.Root().RemoveAll(e.Name())
	}
}

// copyTree copies a file or directory (relative name) from one root to another, with fsync.
func copyTree(ctx context.Context, src, dst *os.Root, name string) error {
	st, err := src.Lstat(name)
	if err != nil {
		return err
	}
	switch {
	case st.IsDir():
		if err := dst.MkdirAll(name, 0o700); err != nil {
			return err
		}
		d, err := src.Open(name)
		if err != nil {
			return err
		}
		ents, err := d.ReadDir(-1)
		d.Close()
		if err != nil {
			return err
		}
		for _, e := range ents {
			if err := copyTree(ctx, src, dst, path.Join(name, e.Name())); err != nil {
				return err
			}
		}
		return nil
	case st.Mode().IsRegular():
		if err := ctx.Err(); err != nil {
			return err
		}
		in, err := src.Open(name)
		if err != nil {
			return err
		}
		defer in.Close()
		if dir := path.Dir(name); dir != "." {
			if err := dst.MkdirAll(dir, 0o700); err != nil {
				return err
			}
		}
		out, err := dst.OpenFile(name, os.O_WRONLY|os.O_CREATE|os.O_EXCL, 0o600)
		if err != nil {
			return err
		}
		if _, err := io.Copy(out, in); err != nil {
			out.Close()
			return err
		}
		if err := out.Sync(); err != nil {
			out.Close()
			return err
		}
		return out.Close()
	default:
		return fmt.Errorf("%s: unsupported file type in snapshot", name)
	}
}
