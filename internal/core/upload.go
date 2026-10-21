package core

import (
	"context"
	"crypto/sha256"
	"encoding/hex"
	"errors"
	"fmt"
	"hash"
	"io"
	"slices"
	"strings"
	"time"
	"unicode/utf8"

	"picberth/internal/blob"
	"picberth/internal/ids"
	"picberth/internal/imaging"
	"picberth/internal/meta"
)

// Upload is one in-progress upload. The HTTP layer streams the multipart body into WriteFile, may set the
// visibility (the part can arrive before or after the file), then calls Commit. Close must always be called.
type Upload struct {
	s          *Service
	p          Principal
	tmp        *blob.Temp
	tmpClosed  bool
	sum        hash.Hash
	size       int64
	filename   string
	haveFile   bool
	visibility string
	tags       []string // normalized, sorted
	release    func()
	done       bool
}

// UploadResult is the outcome of a committed upload.
type UploadResult struct {
	Image     ImageView
	Duplicate bool // an existing image was returned; nothing new was stored
}

// SourceReadError wraps an error from the request body so the caller can tell client/transport problems
// (e.g. http.MaxBytesError) from local I/O failures.
type SourceReadError struct{ Err error }

func (e *SourceReadError) Error() string { return "reading upload: " + e.Err.Error() }
func (e *SourceReadError) Unwrap() error { return e.Err }

// watermark is the free space that must remain: max(1 GiB, 2% of the volume).
func watermark(total uint64) uint64 {
	w := total / 50
	if w < 1<<30 {
		w = 1 << 30
	}
	return w
}

// UploadTimeout is how long an upload of up to maxBytes may take at a minimum rate of 64 KiB/s (at least 2 min).
func UploadTimeout(maxBytes int64) time.Duration {
	d := time.Duration(maxBytes/(64<<10)) * time.Second
	if d < 2*time.Minute {
		d = 2 * time.Minute
	}
	return d
}

// BeginUpload authorizes the principal, applies the upload rate limit, the concurrency cap and the disk
// watermark, and creates the temp file.
func (s *Service) BeginUpload(ctx context.Context, p Principal) (*Upload, error) {
	if p.Kind == KindAnonymous {
		return nil, unauthorized()
	}
	if !p.CanUpload() {
		return nil, forbidden("This token cannot upload.")
	}
	var key string
	switch p.Kind {
	case KindToken:
		key = "t:" + itoa(p.TokenID)
	case KindSession:
		key = "u:" + itoa(p.UserID)
	}
	if key != "" {
		if ok, wait := s.uploadLimit.Allow(key); !ok {
			return nil, rateLimited(wait)
		}
	}
	select {
	case s.uploadSem <- struct{}{}:
	default:
		return nil, rateLimited(2 * time.Second)
	}
	u := &Upload{s: s, p: p, sum: sha256.New(), release: func() { <-s.uploadSem }}
	if free, total, err := s.DiskFree(); err != nil {
		s.log.Warn("could not determine free disk space", "err", err)
	} else if free < watermark(total) {
		u.Close()
		return nil, newErr(CodeInsufficientStorage, "The server is low on disk space and cannot accept uploads right now.")
	}
	tmp, err := s.blobs.CreateTemp()
	if err != nil {
		u.Close()
		if blob.IsDiskFull(err) {
			return nil, newErr(CodeInsufficientStorage, "The server is out of disk space.")
		}
		return nil, err
	}
	u.tmp = tmp
	return u, nil
}

// SetVisibility sets the requested visibility ("public" or "private").
func (u *Upload) SetVisibility(v string) error {
	if v != "public" && v != "private" {
		return validation("visibility must be public or private.")
	}
	u.visibility = v
	return nil
}

// AddTag adds one tag from a `tag` form part. The whole set is validated on every call, so an invalid or
// surplus tag fails the request before any bytes are committed.
func (u *Upload) AddTag(raw string) error {
	tags, err := normalizeTags(append(slices.Clone(u.tags), raw))
	if err != nil {
		return err
	}
	u.tags = tags
	return nil
}

// WriteFile streams the file part into the temp file while hashing, enforcing maxUploadBytes.
func (u *Upload) WriteFile(ctx context.Context, r io.Reader, filename string) error {
	if u.haveFile {
		return validation("Exactly one file part is allowed.")
	}
	u.haveFile = true
	u.filename = filename
	limit := u.s.Settings().MaxUploadBytes
	buf := make([]byte, 32<<10)
	for {
		if err := ctx.Err(); err != nil {
			return err
		}
		n, rerr := r.Read(buf)
		if n > 0 {
			u.size += int64(n)
			if u.size > limit {
				return newErr(CodeTooLarge, "The file exceeds the %s limit.", humanBytes(limit))
			}
			u.sum.Write(buf[:n])
			if _, err := u.tmp.Write(buf[:n]); err != nil {
				if blob.IsDiskFull(err) {
					return newErr(CodeInsufficientStorage, "The server ran out of disk space.")
				}
				return err
			}
		}
		if rerr == io.EOF {
			return nil
		}
		if rerr != nil {
			return &SourceReadError{Err: rerr}
		}
	}
}

// HasFile reports whether a file part has been received.
func (u *Upload) HasFile() bool { return u.haveFile }

func (u *Upload) closeTmp() error {
	if u.tmp == nil || u.tmpClosed {
		return nil
	}
	u.tmpClosed = true
	return u.tmp.Close()
}

// Close discards the temp file (if still present) and releases the concurrency slot. Idempotent.
func (u *Upload) Close() {
	if u.done {
		return
	}
	u.done = true
	if u.tmp != nil {
		u.closeTmp()
		if err := u.s.blobs.RemoveTemp(u.tmp.Name()); err != nil {
			u.s.log.Warn("could not remove upload temp file", "file", u.tmp.Name(), "err", err)
		}
	}
	if u.release != nil {
		u.release()
	}
}

// Commit validates the received file and registers it (docs/architecture.md §4).
func (u *Upload) Commit(ctx context.Context) (UploadResult, error) {
	s := u.s
	if !u.haveFile {
		return UploadResult{}, validation("The request must contain a file part named \"file\".")
	}
	st := s.Settings()

	head := make([]byte, 64)
	n, err := u.tmp.ReadAt(head, 0)
	if err != nil && !errors.Is(err, io.EOF) {
		return UploadResult{}, err
	}
	sniffed, ok := s.Sniff(head[:n])
	if !ok {
		return UploadResult{}, newErr(CodeUnsupportedFormat, "The file is not a supported image (allowed: %s).", strings.Join(st.AllowedFormats, ", "))
	}
	if !st.allows(sniffed) {
		return UploadResult{}, newErr(CodeUnsupportedFormat, "The %s format is not allowed (allowed: %s).", sniffed, strings.Join(st.AllowedFormats, ", "))
	}
	info, err := s.Inspect(u.tmp.File, u.size, imaging.Limits{MaxPixels: st.MaxPixels})
	switch {
	case err == nil:
	case errors.Is(err, imaging.ErrTooManyPixels):
		return UploadResult{}, newErr(CodeTooManyPixels, "The image has more than %d pixels per frame.", st.MaxPixels)
	case errors.Is(err, imaging.ErrInvalid):
		return UploadResult{}, newErr(CodeInvalidImage, "The image is damaged or not valid.")
	case errors.Is(err, imaging.ErrUnsupported):
		return UploadResult{}, newErr(CodeUnsupportedFormat, "The file is not a supported image.")
	default:
		return UploadResult{}, fmt.Errorf("inspecting upload: %w", err)
	}
	if info.Format != sniffed {
		return UploadResult{}, newErr(CodeInvalidImage, "The image content does not match its format.")
	}
	if info.Width <= 0 || info.Height <= 0 {
		return UploadResult{}, newErr(CodeInvalidImage, "The image has no dimensions.")
	}

	sha := hex.EncodeToString(u.sum.Sum(nil))
	owner := u.p.UserID
	if st.Dedupe {
		if dup, err := s.db.FindActiveDuplicate(ctx, owner, sha); err == nil {
			v, err := s.viewTagged(ctx, dup)
			if err != nil {
				return UploadResult{}, err
			}
			v.Duplicate = true
			return UploadResult{Image: v, Duplicate: true}, nil
		} else if !errors.Is(err, meta.ErrNotFound) {
			return UploadResult{}, err
		}
	}

	vis := u.visibility
	if vis == "" {
		vis = st.DefaultVisibility
	}
	id := ids.NewImageID()
	ext := info.Format.Ext()
	frames := info.Frames
	if frames < 1 {
		frames = 1
	}
	rec := meta.Image{
		ID: id, OwnerID: owner, Source: sourceKind(u.p), ObjectKey: ids.ObjectKey(id, ext),
		OriginalName: SanitizeFilename(u.filename, ext), Mime: info.Format.MIME(), Ext: ext, Size: u.size,
		Width: info.Width, Height: info.Height, Frames: frames, SHA256: sha, HasGPS: info.HasGPS,
		Visibility: vis, CreatedAt: s.nowMs(),
	}
	if u.p.Kind == KindToken {
		tid := u.p.TokenID
		rec.TokenID = &tid
	}

	// Durable file first: fsync, close, rename (never over an existing object), directory sync.
	if err := u.tmp.Sync(); err != nil {
		if blob.IsDiskFull(err) {
			return UploadResult{}, newErr(CodeInsufficientStorage, "The server ran out of disk space.")
		}
		return UploadResult{}, err
	}
	if err := u.closeTmp(); err != nil {
		return UploadResult{}, err
	}

	// From here on a client disconnect must not abort between the rename and the INSERT.
	cctx := context.WithoutCancel(ctx)
	s.commitMu.RLock()
	defer s.commitMu.RUnlock()
	if err := s.blobs.CommitTemp(u.tmp.Name(), rec.ObjectKey); err != nil {
		if blob.IsDiskFull(err) {
			return UploadResult{}, newErr(CodeInsufficientStorage, "The server ran out of disk space.")
		}
		return UploadResult{}, err
	}
	if err := s.fault(FaultUploadAfterRename); err != nil {
		return UploadResult{}, err
	}
	dup, err := s.db.InsertImageWithTags(cctx, rec, st.Dedupe, u.tags)
	if err != nil {
		if rmErr := s.blobs.Remove(blob.Objects, rec.ObjectKey); rmErr != nil {
			s.log.Warn("could not remove object after failed insert (the check will quarantine it)", "key", rec.ObjectKey, "err", rmErr)
		}
		return UploadResult{}, err
	}
	if dup != nil { // lost a race with an identical concurrent upload
		if rmErr := s.blobs.Remove(blob.Objects, rec.ObjectKey); rmErr != nil {
			s.log.Warn("could not remove duplicate object", "key", rec.ObjectKey, "err", rmErr)
		}
		v, err := s.viewTagged(cctx, dup)
		if err != nil {
			return UploadResult{}, err
		}
		v.Duplicate = true
		return UploadResult{Image: v, Duplicate: true}, nil
	}
	s.enqueueThumb(id)
	im := rec
	im.State = meta.StateActive
	im.TokenName = u.p.TokenName
	v := s.view(&im)
	if u.tags != nil {
		v.Tags = u.tags
	}
	return UploadResult{Image: v}, nil
}

func sourceKind(p Principal) string {
	switch p.Kind {
	case KindToken:
		return "token"
	case KindCLI:
		return "cli"
	}
	return "web"
}

func humanBytes(n int64) string {
	switch {
	case n >= 1<<30 && n%(1<<30) == 0:
		return fmt.Sprintf("%d GiB", n>>30)
	case n >= 1<<20:
		if n%(1<<20) == 0 {
			return fmt.Sprintf("%d MiB", n>>20)
		}
		return fmt.Sprintf("%.1f MiB", float64(n)/(1<<20))
	case n >= 1<<10:
		return fmt.Sprintf("%.1f KiB", float64(n)/(1<<10))
	}
	return fmt.Sprintf("%d bytes", n)
}

// SanitizeFilename reduces a client-supplied file name to a harmless display name: base name only,
// no control or invisible formatting characters, at most 255 bytes, falling back to "image.<ext>".
func SanitizeFilename(name, ext string) string {
	name = strings.ToValidUTF8(name, "")
	if i := strings.LastIndexAny(name, `/\`); i >= 0 {
		name = name[i+1:]
	}
	name = strings.TrimSpace(printable(name, 1<<20))
	name = strings.TrimRight(name, ". ")
	if name == "" || name == "." || name == ".." {
		return "image." + ext
	}
	if len(name) > 255 {
		cut := 255
		for cut > 0 && !utf8.RuneStart(name[cut]) {
			cut--
		}
		name = name[:cut]
	}
	name = strings.TrimSpace(name)
	if name == "" {
		return "image." + ext
	}
	return name
}
