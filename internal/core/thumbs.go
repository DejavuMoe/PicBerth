package core

import (
	"context"
	"errors"
	"fmt"
	"os"
	"time"

	"picberth/internal/imaging"
	"picberth/internal/meta"
)

// enqueueThumb schedules thumbnail generation. When the queue is full the image is picked up by the next
// backfill (images with an empty thumb state are always retried).
func (s *Service) enqueueThumb(id string) {
	select {
	case s.thumbQ <- id:
	default:
		s.thumbDirty.Store(true)
	}
}

// runThumbWorker is the single thumbnail worker (concurrency 1): a startup backfill, then queue items.
func (s *Service) runThumbWorker() {
	defer s.wg.Done()
	s.backfillThumbs()
	tick := time.NewTicker(time.Minute)
	defer tick.Stop()
	for {
		select {
		case <-s.jobCtx.Done():
			return
		case id := <-s.thumbQ:
			s.thumbOne(id)
		case <-tick.C:
			if s.thumbDirty.Swap(false) {
				s.backfillThumbs()
			}
		}
	}
}

func (s *Service) backfillThumbs() {
	for {
		pending, err := s.db.PendingThumbs(s.jobCtx, 200)
		if err != nil || len(pending) == 0 {
			if err != nil && s.jobCtx.Err() == nil {
				s.log.Warn("thumbnail backfill failed", "err", err)
			}
			return
		}
		progressed := false
		for _, id := range pending {
			if s.jobCtx.Err() != nil {
				return
			}
			if s.thumbOne(id) {
				progressed = true
			}
		}
		if !progressed { // everything pending is unreadable right now; the check job deals with it
			return
		}
	}
}

// ProcessThumbnails synchronously generates every pending thumbnail (tests and offline use).
func (s *Service) ProcessThumbnails(ctx context.Context) {
	for {
		pending, err := s.db.PendingThumbs(ctx, 200)
		if err != nil || len(pending) == 0 {
			return
		}
		progressed := false
		for _, id := range pending {
			if s.thumbOne(id) {
				progressed = true
			}
		}
		if !progressed {
			return
		}
	}
}

// thumbOne generates one thumbnail and records the outcome. It returns true when the image's thumb state
// was settled (so loops make progress).
func (s *Service) thumbOne(id string) bool {
	ctx := s.jobCtx
	im, err := s.db.GetImage(ctx, id)
	if err != nil || im.State != meta.StateActive || im.Thumb != "" || im.BrokenAt != nil {
		return err == nil && (im.State != meta.StateActive || im.Thumb != "")
	}
	f, err := s.openObject(im)
	if err != nil {
		if !errors.Is(err, os.ErrNotExist) {
			s.log.Warn("thumbnail: cannot open image", "id", id, "err", err)
		}
		return false
	}
	defer f.Close()
	st, err := f.Stat()
	if err != nil {
		return false
	}

	outcome, data := "none", []byte(nil)
	info, err := s.Inspect(f, st.Size(), imaging.Limits{MaxPixels: maxPixels})
	if err == nil && int64(info.Width)*int64(info.Height) <= thumbMaxPixels {
		var ext string
		data, ext, err = s.safeThumbnail(f, st.Size(), info)
		if err == nil && (ext == "jpg" || ext == "png") && len(data) > 0 {
			outcome = ext
		} else if err != nil && !errors.Is(err, imaging.ErrUnsupported) {
			s.log.Debug("thumbnail not generated", "id", id, "err", err)
		}
	}
	if outcome != "none" {
		if err := s.blobs.WriteThumb(id, outcome, data); err != nil {
			s.log.Warn("thumbnail: write failed", "id", id, "err", err)
			return false
		}
	}
	ok, err := s.db.SetThumb(ctx, id, outcome)
	if err != nil {
		s.log.Warn("thumbnail: recording outcome failed", "id", id, "err", err)
		return false
	}
	if !ok && outcome != "none" { // purged while we worked
		s.blobs.RemoveThumbs(id)
	}
	return true
}

// safeThumbnail shields the process from decoder panics on hostile input.
func (s *Service) safeThumbnail(f *os.File, size int64, info imaging.Info) (data []byte, ext string, err error) {
	defer func() {
		if r := recover(); r != nil {
			data, ext, err = nil, "", fmt.Errorf("thumbnail decoder panicked: %v", r)
		}
	}()
	return s.Thumbnail(f, size, info, thumbMaxDim, thumbMaxPixels)
}
