package core

import (
	"context"
	"crypto/sha256"
	"encoding/hex"
	"encoding/json"
	"errors"
	"io"
	"io/fs"
	"time"

	"picberth/internal/blob"
	"picberth/internal/meta"
)

// Check modes.
const (
	CheckQuick = "quick"
	CheckFull  = "full"
)

// Issue kinds (docs/api.md CheckReport).
const (
	IssueMissingFile  = "missing-file"
	IssueSizeMismatch = "size-mismatch"
	IssueHashMismatch = "hash-mismatch"
	IssueOrphanFile   = "orphan-file"
	IssuePendingMove  = "pending-move"
	IssueStaleTemp    = "stale-temp"
)

// Issue is one finding of a consistency check.
type Issue struct {
	Kind     string  `json:"kind"`
	ImageID  *string `json:"imageId"`
	Path     *string `json:"path"`
	Repaired bool    `json:"repaired"`
}

// CheckReport is the result of a consistency check.
type CheckReport struct {
	Mode       string  `json:"mode"`
	StartedAt  string  `json:"startedAt"`
	DurationMs int64   `json:"durationMs"`
	OK         bool    `json:"ok"`
	Checked    int     `json:"checked"`
	Issues     []Issue `json:"issues"`
}

func strp(s string) *string { return &s }

func (r *CheckReport) add(kind, imageID, path string, repaired bool) {
	is := Issue{Kind: kind, Repaired: repaired}
	if imageID != "" {
		is.ImageID = strp(imageID)
	}
	if path != "" {
		is.Path = strp(path)
	}
	r.Issues = append(r.Issues, is)
}

// Check verifies invariants I1–I5 (baseline §6.6), repairing what the protocol allows: pending moves,
// leftovers of purged images, stale temp files and (full mode) orphan files, which go to quarantine/.
// The database decides where a file must be.
func (s *Service) Check(ctx context.Context, p Principal, mode string) (*CheckReport, error) {
	if !p.IsAdmin() {
		return nil, s.denyAdmin(p)
	}
	if mode != CheckQuick && mode != CheckFull {
		return nil, validation("mode must be quick or full.")
	}
	if !s.checkMu.TryLock() {
		return nil, conflict("A check is already running.")
	}
	defer s.checkMu.Unlock()

	start := s.now()
	rep := &CheckReport{Mode: mode, StartedAt: start.UTC().Format(time.RFC3339), Issues: []Issue{}}
	full := mode == CheckFull

	after := ""
	for {
		batch, err := s.db.ImageIDsAfter(ctx, after, 500)
		if err != nil {
			return nil, err
		}
		if len(batch) == 0 {
			break
		}
		for _, id := range batch {
			if err := ctx.Err(); err != nil {
				return nil, err
			}
			if err := s.checkImage(ctx, id, full, rep); err != nil {
				return nil, err
			}
		}
		after = batch[len(batch)-1]
	}
	if err := s.checkFiles(ctx, full, rep); err != nil {
		return nil, err
	}
	if err := s.cleanTemp(rep); err != nil {
		return nil, err
	}

	rep.OK = true
	for _, is := range rep.Issues {
		if !is.Repaired {
			rep.OK = false
		}
	}
	rep.DurationMs = s.now().Sub(start).Milliseconds()
	last, _ := json.Marshal(map[string]any{"at": fmtTime(s.nowMs()), "mode": mode, "ok": rep.OK, "issues": len(rep.Issues)})
	if err := s.db.KVSet(ctx, "last_check", string(last)); err != nil {
		s.log.Warn("could not record check result", "err", err)
	}
	if len(rep.Issues) > 0 || p.Kind != KindCLI || mode == CheckFull {
		s.audit(ctx, p, "check."+mode, "", summarize(rep))
	}
	return rep, nil
}

func summarize(r *CheckReport) string {
	bad := 0
	for _, is := range r.Issues {
		if !is.Repaired {
			bad++
		}
	}
	return "checked " + itoa(int64(r.Checked)) + ", issues " + itoa(int64(len(r.Issues))) + ", unrepaired " + itoa(int64(bad))
}

// checkImage verifies one record. Structural checks run under the per-image lock so they cannot race a
// trash/restore; the (slow) hash runs outside it.
func (s *Service) checkImage(ctx context.Context, id string, full bool, rep *CheckReport) error {
	unlock := s.lockImage(id)
	im, err := s.db.GetImage(ctx, id)
	if err != nil {
		unlock.Unlock()
		if errors.Is(err, meta.ErrNotFound) {
			return nil
		}
		return err
	}
	if im.State == meta.StatePurged {
		unlock.Unlock()
		return nil
	}
	rep.Checked++
	bad, err := s.checkStructure(ctx, im, rep)
	unlock.Unlock()
	if err != nil {
		return err
	}
	if !bad && full {
		bad, err = s.checkHash(ctx, im, rep)
		if err != nil {
			return err
		}
	}
	if !bad && full && im.BrokenAt != nil {
		// Healthy again (restored from backup, say). Only a full check can vouch for the bytes.
		if _, err := s.db.SetBroken(ctx, id, nil); err != nil {
			return err
		}
	}
	return nil
}

func (s *Service) checkStructure(ctx context.Context, im *meta.Image, rep *CheckReport) (bad bool, err error) {
	want := areaFor(im.State)
	st, serr := s.blobs.Stat(want, im.ObjectKey)
	if serr != nil && errors.Is(serr, fs.ErrNotExist) {
		if _, oerr := s.blobs.Stat(want.Other(), im.ObjectKey); oerr == nil {
			mvErr := s.blobs.Move(want.Other(), want, im.ObjectKey)
			if mvErr != nil {
				s.log.Warn("check: pending move failed", "id", im.ID, "err", mvErr)
			}
			rep.add(IssuePendingMove, im.ID, string(want)+"/"+im.ObjectKey, mvErr == nil)
			if mvErr != nil {
				return false, nil // wait for the next run; not a data problem
			}
			st, serr = s.blobs.Stat(want, im.ObjectKey)
		}
	}
	if serr != nil {
		path := string(want) + "/" + im.ObjectKey
		rep.add(IssueMissingFile, im.ID, path, false)
		if errors.Is(serr, fs.ErrNotExist) {
			return true, s.markBroken(ctx, im)
		}
		s.log.Warn("check: cannot stat object", "id", im.ID, "err", serr)
		return true, nil
	}
	if st.Size() != im.Size {
		rep.add(IssueSizeMismatch, im.ID, string(want)+"/"+im.ObjectKey, false)
		return true, s.markBroken(ctx, im)
	}
	return false, nil
}

func (s *Service) markBroken(ctx context.Context, im *meta.Image) error {
	if im.BrokenAt != nil {
		return nil
	}
	now := s.nowMs()
	_, err := s.db.SetBroken(ctx, im.ID, &now)
	return err
}

func (s *Service) checkHash(ctx context.Context, im *meta.Image, rep *CheckReport) (bool, error) {
	f, err := s.openObject(im)
	if err != nil {
		if errors.Is(err, fs.ErrNotExist) {
			return false, nil // moved or purged while we were busy; the next check decides
		}
		return false, err
	}
	defer f.Close()
	h := sha256.New()
	if _, err := io.Copy(h, &ctxReader{ctx: ctx, r: f}); err != nil {
		if ctx.Err() != nil {
			return false, ctx.Err()
		}
		return false, err
	}
	if hex.EncodeToString(h.Sum(nil)) != im.SHA256 {
		rep.add(IssueHashMismatch, im.ID, string(areaFor(im.State))+"/"+im.ObjectKey, false)
		return true, s.markBroken(ctx, im)
	}
	return false, nil
}

type ctxReader struct {
	ctx context.Context
	r   io.Reader
}

func (c *ctxReader) Read(p []byte) (int, error) {
	if err := c.ctx.Err(); err != nil {
		return 0, err
	}
	return c.r.Read(p)
}

// checkFiles walks the object areas: files of purged records are deleted; (full) files without any live
// record are quarantined.
func (s *Service) checkFiles(ctx context.Context, full bool, rep *CheckReport) error {
	areas := []blob.Area{blob.Trash}
	if full {
		areas = append(areas, blob.Objects)
	}
	for _, area := range areas {
		var files []blob.File
		err := s.blobs.Walk(area, func(f blob.File) error {
			if err := ctx.Err(); err != nil {
				return err
			}
			files = append(files, f)
			return nil
		})
		if err != nil {
			return err
		}
		for _, f := range files {
			if err := s.checkFile(ctx, area, f, full, rep); err != nil {
				return err
			}
		}
	}
	return nil
}

func (s *Service) checkFile(ctx context.Context, area blob.Area, f blob.File, full bool, rep *CheckReport) error {
	if f.Key == "" || !f.Regular {
		if full { // not something PicBerth ever writes
			if _, err := s.blobs.QuarantineRaw(f.Rel, s.now()); err != nil {
				s.log.Warn("check: cannot quarantine stray file", "path", f.Rel, "err", err)
				rep.add(IssueOrphanFile, "", f.Rel, false)
			} else {
				rep.add(IssueOrphanFile, "", f.Rel, true)
			}
		}
		return nil
	}
	rec, err := s.db.GetImageByKey(ctx, f.Key)
	switch {
	case err == nil && rec.State == meta.StatePurged:
		unlock := s.lockImage(rec.ID)
		defer unlock.Unlock()
		if err := s.blobs.Remove(area, f.Key); err != nil {
			s.log.Warn("check: cannot remove leftover of purged image", "id", rec.ID, "err", err)
			rep.add(IssueOrphanFile, rec.ID, f.Rel, false)
			return nil
		}
		rep.add(IssueOrphanFile, rec.ID, f.Rel, true)
		return nil
	case err == nil:
		return nil // live record: handled by checkImage
	case !errors.Is(err, meta.ErrNotFound):
		return err
	}
	if !full {
		return nil
	}
	// No record. Wait for in-flight uploads (rename → INSERT window), then decide again.
	s.commitMu.Lock()
	defer s.commitMu.Unlock()
	if _, err := s.db.GetImageByKey(ctx, f.Key); err == nil {
		return nil
	} else if !errors.Is(err, meta.ErrNotFound) {
		return err
	}
	if _, err := s.blobs.Quarantine(area, f.Key, s.now()); err != nil {
		if errors.Is(err, fs.ErrNotExist) {
			return nil
		}
		s.log.Warn("check: cannot quarantine orphan", "path", f.Rel, "err", err)
		rep.add(IssueOrphanFile, "", f.Rel, false)
		return nil
	}
	rep.add(IssueOrphanFile, "", f.Rel, true)
	return nil
}

// cleanTemp removes upload leftovers older than the longest possible upload.
func (s *Service) cleanTemp(rep *CheckReport) error {
	list, err := s.blobs.ListTemp()
	if err != nil {
		return err
	}
	maxAge := UploadTimeout(s.cfg.HardMaxUpload) + time.Minute
	now := s.now()
	for _, t := range list {
		if t.ModTime.IsZero() || now.Sub(t.ModTime) < maxAge {
			continue
		}
		if err := s.blobs.RemoveTemp(t.Name); err != nil {
			s.log.Warn("check: cannot remove stale temp file", "file", t.Name, "err", err)
			rep.add(IssueStaleTemp, "", t.Name, false)
			continue
		}
		rep.add(IssueStaleTemp, "", t.Name, true)
	}
	return nil
}
