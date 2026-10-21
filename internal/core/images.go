package core

import (
	"context"
	"encoding/base64"
	"encoding/json"
	"errors"
	"html"
	"os"
	"path/filepath"
	"strconv"
	"strings"
	"time"

	"picberth/internal/blob"
	"picberth/internal/ids"
	"picberth/internal/imaging"
	"picberth/internal/meta"
)

func fmtTime(ms int64) string { return time.UnixMilli(ms).UTC().Format(time.RFC3339) }

func fmtTimePtr(ms *int64) *string {
	if ms == nil {
		return nil
	}
	s := fmtTime(*ms)
	return &s
}

func itoa(n int64) string { return strconv.FormatInt(n, 10) }

// ImageView is the API representation of an image (docs/api.md §3).
type ImageView struct {
	ID         string      `json:"id"`
	URL        string      `json:"url"`
	Filename   string      `json:"filename"`
	Mime       string      `json:"mime"`
	Ext        string      `json:"ext"`
	Size       int64       `json:"size"`
	Width      int         `json:"width"`
	Height     int         `json:"height"`
	Frames     int         `json:"frames"`
	SHA256     string      `json:"sha256"`
	Visibility string      `json:"visibility"`
	State      string      `json:"state"`
	HasGPS     bool        `json:"hasGps"`
	HasThumb   bool        `json:"hasThumb"`
	Broken     bool        `json:"broken"`
	Duplicate  bool        `json:"duplicate"`
	Source     ImageSource `json:"source"`
	CreatedAt  string      `json:"createdAt"`
	TrashedAt  *string     `json:"trashedAt"`
	Links      ImageLinks  `json:"links"`
}

// ImageSource says how an image arrived.
type ImageSource struct {
	Kind      string `json:"kind"`
	TokenID   *int64 `json:"tokenId,omitempty"`
	TokenName string `json:"tokenName,omitempty"`
}

// ImageLinks are ready-made embed snippets.
type ImageLinks struct {
	URL      string `json:"url"`
	Markdown string `json:"markdown"`
	HTML     string `json:"html"`
	BBCode   string `json:"bbcode"`
}

func (s *Service) view(im *meta.Image) ImageView {
	url := s.ImageURL(im.ID, im.Ext)
	alt := strings.TrimSuffix(im.OriginalName, filepath.Ext(im.OriginalName))
	if alt == "" {
		alt = "image"
	}
	v := ImageView{
		ID: im.ID, URL: url, Filename: im.OriginalName, Mime: im.Mime, Ext: im.Ext, Size: im.Size,
		Width: im.Width, Height: im.Height, Frames: im.Frames, SHA256: im.SHA256, Visibility: im.Visibility,
		State: im.State, HasGPS: im.HasGPS, HasThumb: im.Thumb == "jpg" || im.Thumb == "png", Broken: im.BrokenAt != nil,
		Source:    ImageSource{Kind: im.Source},
		CreatedAt: fmtTime(im.CreatedAt), TrashedAt: fmtTimePtr(im.TrashedAt),
		Links: ImageLinks{
			URL:      url,
			Markdown: "![" + mdEscape(alt) + "](" + url + ")",
			HTML:     `<img src="` + html.EscapeString(url) + `" alt="` + html.EscapeString(alt) + `" width="` + strconv.Itoa(im.Width) + `" height="` + strconv.Itoa(im.Height) + `">`,
			BBCode:   "[img]" + url + "[/img]",
		},
	}
	if im.Source == "token" {
		v.Source.TokenID = im.TokenID
		v.Source.TokenName = im.TokenName
	}
	return v
}

func mdEscape(s string) string {
	var b strings.Builder
	for _, r := range s {
		switch r {
		case '\\', '[', ']', '*', '_', '`', '<', '>', '!', '(', ')':
			b.WriteByte('\\')
		}
		b.WriteRune(r)
	}
	return b.String()
}

func areaFor(state string) blob.Area {
	if state == meta.StateTrashed {
		return blob.Trash
	}
	return blob.Objects
}

// getVisible loads a record the principal may see, else not-found.
func (s *Service) getVisible(ctx context.Context, p Principal, id string) (*meta.Image, error) {
	if p.Kind == KindAnonymous {
		return nil, unauthorized()
	}
	if !ids.ValidImageID(id) {
		return nil, notFound()
	}
	im, err := s.db.GetImage(ctx, id)
	if err != nil {
		if errors.Is(err, meta.ErrNotFound) {
			return nil, notFound()
		}
		return nil, err
	}
	if !p.canSee(im) {
		return nil, notFound()
	}
	return im, nil
}

// GetImage returns one image the principal may see.
func (s *Service) GetImage(ctx context.Context, p Principal, id string) (ImageView, error) {
	im, err := s.getVisible(ctx, p, id)
	if err != nil {
		return ImageView{}, err
	}
	return s.view(im), nil
}

// ListParams are the parsed list query parameters.
type ListParams struct {
	State      string
	Q          string
	Format     string
	Visibility string
	GPS        bool
	TokenID    *int64
	Sort       string
	Limit      int
	Cursor     string
}

type cursorToken struct {
	Sort  string `json:"s"`
	State string `json:"t"`
	Key   int64  `json:"k"`
	ID    string `json:"i"`
}

// ListImages returns a page of images visible to the principal.
func (s *Service) ListImages(ctx context.Context, p Principal, lp ListParams) ([]ImageView, string, error) {
	if p.Kind == KindAnonymous {
		return nil, "", unauthorized()
	}
	if !p.CanReadAll() && !p.canReadOwnUploads() {
		return nil, "", forbidden("This token cannot list images.")
	}
	if lp.State == "" {
		lp.State = meta.StateActive
	}
	if lp.State != meta.StateActive && lp.State != meta.StateTrashed {
		return nil, "", validation("state must be active or trashed.")
	}
	if lp.State == meta.StateTrashed && !p.CanManage() {
		return nil, "", forbidden("Listing the trash requires the manage scope.")
	}
	if lp.Sort == "" {
		lp.Sort = "new"
	}
	switch lp.Sort {
	case "new", "old", "large", "small":
	default:
		return nil, "", validation("sort must be new, old, large or small.")
	}
	if lp.Limit == 0 {
		lp.Limit = 60
	}
	if lp.Limit < 1 || lp.Limit > 200 {
		return nil, "", validation("limit must be between 1 and 200.")
	}
	if lp.Visibility != "" && lp.Visibility != "public" && lp.Visibility != "private" {
		return nil, "", validation("visibility must be public or private.")
	}
	q := meta.ListQuery{State: lp.State, Q: strings.TrimSpace(lp.Q), Visibility: lp.Visibility, GPS: lp.GPS, TokenID: lp.TokenID, Sort: lp.Sort, Limit: lp.Limit}
	if len(q.Q) > 200 {
		return nil, "", validation("q is too long.")
	}
	if lp.Format != "" {
		f, ok := imaging.ParseFormat(lp.Format)
		if !ok {
			return nil, "", validation("format must be one of jpeg, png, gif, webp, avif.")
		}
		q.Ext = f.Ext()
	}
	if !p.CanReadAll() { // upload-only token: its own uploads, nothing else
		tid := p.TokenID
		q.TokenID = &tid
	}
	if p.Kind == KindToken || p.Kind == KindSession {
		q.OwnerID = &p.UserID
	}
	if lp.Cursor != "" {
		raw, err := base64.RawURLEncoding.DecodeString(lp.Cursor)
		var ct cursorToken
		if err != nil || json.Unmarshal(raw, &ct) != nil || ct.Sort != lp.Sort || ct.State != lp.State || !ids.ValidImageID(ct.ID) {
			return nil, "", validation("Invalid cursor.")
		}
		q.After = &meta.Cursor{Key: ct.Key, ID: ct.ID}
	}
	rows, err := s.db.ListImages(ctx, q)
	if err != nil {
		return nil, "", err
	}
	next := ""
	if len(rows) > lp.Limit {
		rows = rows[:lp.Limit]
		last := rows[len(rows)-1]
		key := last.CreatedAt
		if lp.Sort == "large" || lp.Sort == "small" {
			key = last.Size
		}
		b, _ := json.Marshal(cursorToken{Sort: lp.Sort, State: lp.State, Key: key, ID: last.ID})
		next = base64.RawURLEncoding.EncodeToString(b)
	}
	out := make([]ImageView, len(rows))
	for i := range rows {
		out[i] = s.view(&rows[i])
	}
	return out, next, nil
}

// SetVisibility changes an image's visibility.
func (s *Service) SetVisibility(ctx context.Context, p Principal, id, vis string) (ImageView, error) {
	if p.Kind == KindAnonymous {
		return ImageView{}, unauthorized()
	}
	if !p.CanManage() {
		return ImageView{}, forbidden("Changing visibility requires the manage scope.")
	}
	if vis != "public" && vis != "private" {
		return ImageView{}, validation("visibility must be public or private.")
	}
	im, err := s.getVisible(ctx, p, id)
	if err != nil {
		return ImageView{}, err
	}
	unlock := s.lockImage(id)
	defer unlock.Unlock()
	ok, err := s.db.SetVisibility(ctx, id, vis)
	if err != nil {
		return ImageView{}, err
	}
	if !ok {
		return ImageView{}, notFound()
	}
	if im.Visibility != vis {
		s.audit(ctx, p, "image.visibility", id, im.Visibility+" -> "+vis)
	}
	im.Visibility = vis
	return s.view(im), nil
}

// Trash moves an image to the trash. Trashing a trashed image is a no-op. With a retention of 0 days
// the image is purged right away.
func (s *Service) Trash(ctx context.Context, p Principal, id string) error {
	if p.Kind == KindAnonymous {
		return unauthorized()
	}
	if !p.CanManage() {
		return forbidden("Deleting requires the manage scope.")
	}
	im, err := s.getVisible(ctx, p, id)
	if err != nil {
		return err
	}
	unlock := s.lockImage(id)
	if im.State == meta.StateTrashed {
		unlock.Unlock()
		return nil
	}
	ok, err := s.db.MarkTrashed(ctx, id, s.nowMs())
	if err != nil {
		unlock.Unlock()
		return err
	}
	if !ok { // lost a race with another delete
		unlock.Unlock()
		return nil
	}
	if err := s.fault(FaultTrashAfterDB); err != nil {
		unlock.Unlock()
		return err
	}
	// The record is committed: the image is already invisible to the public. Moving the file may fail on
	// Windows while a download holds it open; the consistency check completes the move.
	if err := s.blobs.Move(blob.Objects, blob.Trash, im.ObjectKey); err != nil {
		s.log.Warn("trash: file move deferred to the consistency check", "id", id, "err", err)
	}
	s.audit(ctx, p, "image.trash", id, "")
	unlock.Unlock()
	if s.Settings().TrashRetentionDays == 0 {
		return s.purge(ctx, p, id)
	}
	return nil
}

// Restore brings an image back from the trash.
func (s *Service) Restore(ctx context.Context, p Principal, id string) (ImageView, error) {
	if p.Kind == KindAnonymous {
		return ImageView{}, unauthorized()
	}
	if !p.CanManage() {
		return ImageView{}, forbidden("Restoring requires the manage scope.")
	}
	if _, err := s.getVisible(ctx, p, id); err != nil {
		return ImageView{}, err
	}
	unlock := s.lockImage(id)
	defer unlock.Unlock()
	im, err := s.db.GetImage(ctx, id)
	if err != nil {
		return ImageView{}, err
	}
	if im.State != meta.StateTrashed {
		return ImageView{}, conflict("The image is not in the trash.")
	}
	// File first, then record: an active record always has its file in objects/.
	if err := s.blobs.Move(blob.Trash, blob.Objects, im.ObjectKey); err != nil {
		// The trash move may never have happened (deferred on Windows while the file was open);
		// then the file is still where an active record needs it.
		if _, statErr := s.blobs.Stat(blob.Objects, im.ObjectKey); statErr != nil {
			s.log.Error("restore: cannot move file back", "id", id, "err", err)
			return ImageView{}, conflict("The image file could not be restored; run a consistency check.")
		}
	}
	if err := s.fault(FaultRestoreAfterMove); err != nil {
		return ImageView{}, err
	}
	ok, err := s.db.MarkRestored(ctx, id)
	if err != nil {
		return ImageView{}, err
	}
	if !ok {
		return ImageView{}, conflict("The image is not in the trash.")
	}
	s.audit(ctx, p, "image.restore", id, "")
	if im.Thumb == "" {
		s.enqueueThumb(id)
	}
	im.State, im.TrashedAt = meta.StateActive, nil
	return s.view(im), nil
}

// Purge permanently deletes a trashed image's file; the record stays as a tombstone.
func (s *Service) Purge(ctx context.Context, p Principal, id string) error {
	if p.Kind == KindAnonymous {
		return unauthorized()
	}
	if !p.CanManage() {
		return forbidden("Purging requires the manage scope.")
	}
	if _, err := s.getVisible(ctx, p, id); err != nil {
		return err
	}
	return s.purge(ctx, p, id)
}

func (s *Service) purge(ctx context.Context, p Principal, id string) error {
	s.purgeMu.RLock()
	defer s.purgeMu.RUnlock()
	unlock := s.lockImage(id)
	defer unlock.Unlock()
	im, err := s.db.GetImage(ctx, id)
	if err != nil {
		if errors.Is(err, meta.ErrNotFound) {
			return notFound()
		}
		return err
	}
	if im.State != meta.StateTrashed {
		if im.State == meta.StatePurged {
			return notFound()
		}
		return conflict("Only images in the trash can be purged.")
	}
	// Tombstone first, then bytes.
	ok, err := s.db.MarkPurged(ctx, id, s.nowMs())
	if err != nil {
		return err
	}
	if !ok {
		return conflict("Only images in the trash can be purged.")
	}
	if err := s.fault(FaultPurgeAfterDB); err != nil {
		return err
	}
	s.removeFiles(im)
	s.audit(ctx, p, "image.purge", id, "")
	return nil
}

// removeFiles deletes an image's bytes (wherever they are) and thumbnails; failures are left to the check.
func (s *Service) removeFiles(im *meta.Image) {
	for _, a := range []blob.Area{blob.Trash, blob.Objects} {
		if err := s.blobs.Remove(a, im.ObjectKey); err != nil {
			s.log.Warn("purge: file removal deferred to the consistency check", "id", im.ID, "area", a, "err", err)
		}
	}
	if err := s.blobs.RemoveThumbs(im.ID); err != nil {
		s.log.Warn("purge: thumbnail removal failed", "id", im.ID, "err", err)
	}
}

// PurgeExpired purges trashed images older than the retention period. It returns how many were purged.
func (s *Service) PurgeExpired(ctx context.Context) (int, error) {
	days := s.Settings().TrashRetentionDays
	cutoff := s.now().Add(-time.Duration(days) * 24 * time.Hour).UnixMilli()
	total := 0
	actor := Principal{Kind: KindCLI, Username: "system"}
	for {
		idsBatch, err := s.db.ExpiredTrash(ctx, cutoff, 200)
		if err != nil {
			return total, err
		}
		if len(idsBatch) == 0 {
			return total, nil
		}
		progressed := false
		for _, id := range idsBatch {
			if err := ctx.Err(); err != nil {
				return total, err
			}
			err := s.purge(ctx, actor, id)
			if err == nil {
				total++
				progressed = true
			} else if !errors.Is(err, ErrConflict) && !errors.Is(err, ErrNotFound) {
				s.log.Error("purging expired image failed", "id", id, "err", err)
			} else {
				progressed = true // someone else changed it: it will not be selected again
			}
		}
		if !progressed {
			return total, nil
		}
	}
}

// BatchResult is one entry of a batch response.
type BatchResult struct {
	ID   string `json:"id"`
	OK   bool   `json:"ok"`
	Code string `json:"code,omitempty"`
}

// Batch applies one action to up to 500 images. Per-image failures are reported, not fatal.
func (s *Service) Batch(ctx context.Context, p Principal, action string, idList []string, visibility string) ([]BatchResult, error) {
	if p.Kind == KindAnonymous {
		return nil, unauthorized()
	}
	if !p.CanManage() {
		return nil, forbidden("Batch operations require the manage scope.")
	}
	switch action {
	case "trash", "restore", "purge", "visibility":
	default:
		return nil, validation("action must be trash, restore, purge or visibility.")
	}
	if action == "visibility" && visibility != "public" && visibility != "private" {
		return nil, validation("visibility must be public or private.")
	}
	if len(idList) == 0 || len(idList) > 500 {
		return nil, validation("ids must contain between 1 and 500 entries.")
	}
	seen := make(map[string]bool, len(idList))
	results := make([]BatchResult, 0, len(idList))
	for _, id := range idList {
		if seen[id] {
			continue
		}
		seen[id] = true
		var err error
		switch action {
		case "trash":
			err = s.Trash(ctx, p, id)
		case "restore":
			_, err = s.Restore(ctx, p, id)
		case "purge":
			err = s.Purge(ctx, p, id)
		case "visibility":
			_, err = s.SetVisibility(ctx, p, id, visibility)
		}
		r := BatchResult{ID: id, OK: err == nil}
		if err != nil {
			r.Code = ErrorCode(err)
			if r.Code == CodeInternal {
				s.log.Error("batch item failed", "action", action, "id", id, "err", err)
			}
		}
		results = append(results, r)
	}
	return results, nil
}

// Opened is an image (or thumbnail) file ready to be served to an authenticated principal.
type Opened struct {
	Image       *meta.Image
	File        *os.File
	Size        int64
	ModTime     time.Time
	ContentType string
}

// openObject opens the record's file in its expected area, falling back to the other area (a pending move).
func (s *Service) openObject(im *meta.Image) (*os.File, error) {
	area := areaFor(im.State)
	f, err := s.blobs.Open(area, im.ObjectKey)
	if errors.Is(err, os.ErrNotExist) {
		f, err = s.blobs.Open(area.Other(), im.ObjectKey)
	}
	return f, err
}

func (s *Service) openedFrom(im *meta.Image, f *os.File, ctype string) (*Opened, error) {
	st, err := f.Stat()
	if err != nil {
		f.Close()
		return nil, err
	}
	return &Opened{Image: im, File: f, Size: st.Size(), ModTime: st.ModTime(), ContentType: ctype}, nil
}

// OpenRaw opens the original bytes for a principal allowed to read this image (including private ones).
func (s *Service) OpenRaw(ctx context.Context, p Principal, id string) (*Opened, error) {
	im, err := s.getVisible(ctx, p, id)
	if err != nil {
		return nil, err
	}
	if !p.CanReadAll() && !p.canReadOwnUploads() {
		return nil, forbidden("This token cannot read images.")
	}
	if im.BrokenAt != nil {
		return nil, notFound()
	}
	f, err := s.openObject(im)
	if err != nil {
		if errors.Is(err, os.ErrNotExist) {
			return nil, notFound()
		}
		return nil, err
	}
	return s.openedFrom(im, f, im.Mime)
}

// OpenThumb opens the thumbnail; not-found when the image has none.
func (s *Service) OpenThumb(ctx context.Context, p Principal, id string) (*Opened, error) {
	im, err := s.getVisible(ctx, p, id)
	if err != nil {
		return nil, err
	}
	if !p.CanReadAll() && !p.canReadOwnUploads() {
		return nil, forbidden("This token cannot read images.")
	}
	if im.Thumb != "jpg" && im.Thumb != "png" {
		return nil, notFound()
	}
	f, err := s.blobs.OpenThumb(im.ID, im.Thumb)
	if err != nil {
		if errors.Is(err, os.ErrNotExist) {
			return nil, notFound()
		}
		return nil, err
	}
	ct := "image/jpeg"
	if im.Thumb == "png" {
		ct = "image/png"
	}
	return s.openedFrom(im, f, ct)
}

// PublicInfo is what the public image route needs to answer without touching the file.
type PublicInfo struct {
	ID          string
	Ext         string
	SHA256      string
	ContentType string
	Size        int64
	ModTime     time.Time
	key         string
}

// LookupPublic finds an active, public, healthy image by id and canonical extension.
func (s *Service) LookupPublic(ctx context.Context, id, ext string) (*PublicInfo, error) {
	if !ids.ValidImageID(id) {
		return nil, notFound()
	}
	im, err := s.db.GetImage(ctx, id)
	if err != nil {
		if errors.Is(err, meta.ErrNotFound) {
			return nil, notFound()
		}
		return nil, err
	}
	if im.State != meta.StateActive || im.Visibility != "public" || im.BrokenAt != nil || im.Ext != ext {
		return nil, notFound()
	}
	return &PublicInfo{ID: im.ID, Ext: im.Ext, SHA256: im.SHA256, ContentType: im.Mime, Size: im.Size,
		ModTime: time.UnixMilli(im.CreatedAt).UTC(), key: im.ObjectKey}, nil
}

// OpenPublic opens the bytes of a looked-up public image.
func (s *Service) OpenPublic(info *PublicInfo) (*os.File, error) {
	f, err := s.blobs.Open(blob.Objects, info.key)
	if err != nil {
		if errors.Is(err, os.ErrNotExist) {
			return nil, notFound()
		}
		return nil, err
	}
	return f, nil
}
