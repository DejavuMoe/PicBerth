package core

import (
	"context"
	"encoding/base64"
	"encoding/json"
	"errors"
	"html"
	"os"
	"path/filepath"
	"slices"
	"sort"
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
	ID        string      `json:"id"`
	URL       string      `json:"url"`
	Filename  string      `json:"filename"`
	FolderID  *string     `json:"folderId"`
	Mime      string      `json:"mime"`
	Ext       string      `json:"ext"`
	Size      int64       `json:"size"`
	Width     int         `json:"width"`
	Height    int         `json:"height"`
	Frames    int         `json:"frames"`
	SHA256    string      `json:"sha256"`
	State     string      `json:"state"`
	HasGPS    bool        `json:"hasGps"`
	HasThumb  bool        `json:"hasThumb"`
	Broken    bool        `json:"broken"`
	Duplicate bool        `json:"duplicate"`
	Source    ImageSource `json:"source"`
	CreatedAt string      `json:"createdAt"`
	TrashedAt *string     `json:"trashedAt"`
	Links     ImageLinks  `json:"links"`
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
		ID: im.ID, URL: url, Filename: im.OriginalName, FolderID: im.FolderID, Mime: im.Mime, Ext: im.Ext, Size: im.Size,
		Width: im.Width, Height: im.Height, Frames: im.Frames, SHA256: im.SHA256,
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

// ListParams are the parsed list query parameters. Folder is empty (every folder), "root" or a folder id;
// Recursive widens an explicit folder to its whole subtree.
type ListParams struct {
	State     string
	Q         string
	Formats   []string // repeated format values (each may also be comma-separated); any-of
	Sources   []string // any of: "web", "cli", "token:<id>"
	TokenID   *int64
	Folder    string
	Recursive bool
	Since     *time.Time // created at or after
	Until     *time.Time // created before
	Sort      string
	Limit     int
	Cursor    string
}

type cursorToken struct {
	Sort      string `json:"s"`
	State     string `json:"t"`
	Folder    string `json:"f,omitempty"`
	Recursive bool   `json:"r,omitempty"`
	Key       int64  `json:"k"`
	StrKey    string `json:"n,omitempty"` // name sort only
	ID        string `json:"i"`
}

const maxFilterValues = 50

// parseExts turns format values (repeated and/or comma-separated) into stored extensions.
func parseExts(values ...[]string) ([]string, error) {
	var exts []string
	for _, vs := range values {
		for _, v := range vs {
			for _, name := range strings.Split(v, ",") {
				name = strings.TrimSpace(name)
				if name == "" {
					continue
				}
				f, ok := imaging.ParseFormat(name)
				if !ok {
					return nil, validation("format must be one of jpeg, png, gif, webp, avif.")
				}
				if !slices.Contains(exts, f.Ext()) {
					exts = append(exts, f.Ext())
				}
			}
		}
	}
	return exts, nil
}

func parseSources(in []string) ([]meta.SourceFilter, error) {
	var out []meta.SourceFilter
	for _, v := range in {
		v = strings.TrimSpace(v)
		var sf meta.SourceFilter
		switch {
		case v == "web" || v == "cli":
			sf.Kind = v
		case strings.HasPrefix(v, "token:"):
			n, err := strconv.ParseUint(strings.TrimPrefix(v, "token:"), 10, 63)
			if err != nil || n < 1 {
				return nil, validation("source token ids must be positive integers (token:<id>).")
			}
			sf = meta.SourceFilter{Kind: "token", TokenID: int64(n)}
		default:
			return nil, validation("source must be web, cli or token:<id>.")
		}
		if !slices.Contains(out, sf) {
			out = append(out, sf)
		}
	}
	if len(out) > maxFilterValues {
		return nil, validation("At most %d sources can be combined.", maxFilterValues)
	}
	return out, nil
}

// listScope is the part of the library a principal may query: its owner and, for upload-only tokens,
// its own token. Callers must already have checked that the principal may read at all.
func (p Principal) listScope() (owner, token *int64) {
	if !p.CanReadAll() { // upload-only token: its own uploads, nothing else
		tid := p.TokenID
		token = &tid
	}
	if p.Kind == KindToken || p.Kind == KindSession {
		owner = &p.UserID
	}
	return owner, token
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
	case "new", "old", "large", "small", "name":
	default:
		return nil, "", validation("sort must be new, old, large, small or name.")
	}
	if lp.Limit == 0 {
		lp.Limit = 60
	}
	if lp.Limit < 1 || lp.Limit > 200 {
		return nil, "", validation("limit must be between 1 and 200.")
	}
	q := meta.ListQuery{State: lp.State, Q: strings.TrimSpace(lp.Q), TokenID: lp.TokenID, Sort: lp.Sort, Limit: lp.Limit}
	if len(q.Q) > 200 {
		return nil, "", validation("q is too long.")
	}
	var err error
	if q.Exts, err = parseExts(lp.Formats); err != nil {
		return nil, "", err
	}
	if q.Sources, err = parseSources(lp.Sources); err != nil {
		return nil, "", err
	}
	if lp.Since != nil {
		ms := lp.Since.UnixMilli()
		q.Since = &ms
	}
	if lp.Until != nil {
		ms := lp.Until.UnixMilli()
		q.Until = &ms
	}
	if lp.Folder != "" {
		if lp.Folder == "root" {
			if !lp.Recursive {
				q.FolderRoot = true
			}
		} else {
			f, err := s.resolveFolder(ctx, p, lp.Folder)
			if err != nil {
				return nil, "", err
			}
			if lp.Recursive {
				idsInTree, _, err := s.db.FolderSubtree(ctx, f.ID)
				if err != nil {
					return nil, "", err
				}
				q.FolderIDs = idsInTree
			} else {
				q.FolderID = &f.ID
			}
		}
	}
	owner, token := p.listScope()
	if token != nil {
		q.TokenID = token
	}
	q.OwnerID = owner
	if lp.Cursor != "" {
		raw, err := base64.RawURLEncoding.DecodeString(lp.Cursor)
		var ct cursorToken
		if err != nil || json.Unmarshal(raw, &ct) != nil || ct.Sort != lp.Sort || ct.State != lp.State ||
			ct.Folder != lp.Folder || ct.Recursive != lp.Recursive || !ids.ValidImageID(ct.ID) {
			return nil, "", validation("Invalid cursor.")
		}
		q.After = &meta.Cursor{Key: ct.Key, StrKey: ct.StrKey, ID: ct.ID}
	}
	rows, err := s.db.ListImages(ctx, q)
	if err != nil {
		return nil, "", err
	}
	next := ""
	if len(rows) > lp.Limit {
		rows = rows[:lp.Limit]
		last := rows[len(rows)-1]
		ct := cursorToken{Sort: lp.Sort, State: lp.State, Folder: lp.Folder, Recursive: lp.Recursive, Key: last.CreatedAt, ID: last.ID}
		switch lp.Sort {
		case "large", "small":
			ct.Key = last.Size
		case "name":
			ct.Key, ct.StrKey = 0, last.OriginalName
		}
		b, _ := json.Marshal(ct)
		next = base64.RawURLEncoding.EncodeToString(b)
	}
	out := make([]ImageView, len(rows))
	for i := range rows {
		out[i] = s.view(&rows[i])
	}
	return out, next, nil
}

// Facets are the library counts served by GET /api/v1/images/facets.
type Facets struct {
	Total   int64            `json:"total"`
	Bytes   int64            `json:"bytes"`
	Formats map[string]int64 `json:"formats"`
	Sources []FacetSource    `json:"sources"`
}

// FacetSource counts the images of one source (web, cli or a single token).
type FacetSource struct {
	Key       string `json:"key"`
	Kind      string `json:"kind"`
	TokenID   *int64 `json:"tokenId,omitempty"`
	TokenName string `json:"tokenName,omitempty"`
	Count     int64  `json:"count"`
}

// ImageFacets counts the principal's whole visible set for a state (default active), ignoring list filters.
func (s *Service) ImageFacets(ctx context.Context, p Principal, state string) (Facets, error) {
	if p.Kind == KindAnonymous {
		return Facets{}, unauthorized()
	}
	if !p.CanReadAll() && !p.canReadOwnUploads() {
		return Facets{}, forbidden("This token cannot list images.")
	}
	if state == "" {
		state = meta.StateActive
	}
	if state != meta.StateActive && state != meta.StateTrashed {
		return Facets{}, validation("state must be active or trashed.")
	}
	if state == meta.StateTrashed && !p.CanManage() {
		return Facets{}, forbidden("Listing the trash requires the manage scope.")
	}
	owner, token := p.listScope()
	mf, err := s.db.ImageFacets(ctx, meta.FacetQuery{State: state, OwnerID: owner, TokenID: token})
	if err != nil {
		return Facets{}, err
	}
	f := Facets{
		Total: mf.Total, Bytes: mf.Bytes,
		Formats: map[string]int64{},
		Sources: []FacetSource{},
	}
	for _, fm := range imaging.AllFormats {
		f.Formats[string(fm)] = mf.Exts[fm.Ext()]
	}
	var web, cli *FacetSource
	var tokens []FacetSource
	for _, sc := range mf.Sources {
		switch sc.Source {
		case "web":
			web = &FacetSource{Key: "web", Kind: "web", Count: sc.Count}
		case "cli":
			cli = &FacetSource{Key: "cli", Kind: "cli", Count: sc.Count}
		case "token":
			if sc.TokenID != nil {
				tokens = append(tokens, FacetSource{Key: "token:" + itoa(*sc.TokenID), Kind: "token", TokenID: sc.TokenID, TokenName: sc.TokenName, Count: sc.Count})
			}
		}
	}
	sort.Slice(tokens, func(i, j int) bool {
		if tokens[i].Count != tokens[j].Count {
			return tokens[i].Count > tokens[j].Count
		}
		return *tokens[i].TokenID < *tokens[j].TokenID
	})
	if web != nil {
		f.Sources = append(f.Sources, *web)
	}
	f.Sources = append(f.Sources, tokens...)
	if cli != nil {
		f.Sources = append(f.Sources, *cli)
	}
	return f, nil
}

// ImagePatch is the set of changes PatchImage applies; nil fields are left alone. MoveFolder with a nil
// FolderID moves the image to the root.
type ImagePatch struct {
	Filename   *string
	FolderID   *string
	MoveFolder bool
}

// sameFolder compares two optional folder ids.
func sameFolder(a, b *string) bool {
	if a == nil || b == nil {
		return a == nil && b == nil
	}
	return *a == *b
}

// folderLabel renders a folder reference for audit details.
func folderLabel(id *string) string {
	if id == nil {
		return "root"
	}
	return *id
}

// PatchImage changes the display name and/or folder of an image in one transaction.
func (s *Service) PatchImage(ctx context.Context, p Principal, id string, patch ImagePatch) (ImageView, error) {
	if p.Kind == KindAnonymous {
		return ImageView{}, unauthorized()
	}
	if !p.CanManage() {
		return ImageView{}, forbidden("Changing an image requires the manage scope.")
	}
	if patch.Filename == nil && !patch.MoveFolder {
		return ImageView{}, validation("Provide at least one of filename or folderId.")
	}
	var mp meta.ImagePatch
	if patch.Filename != nil {
		name, err := normalizeFilename(*patch.Filename)
		if err != nil {
			return ImageView{}, err
		}
		mp.Name = &name
	}
	if patch.MoveFolder {
		if patch.FolderID != nil {
			f, err := s.resolveFolder(ctx, p, *patch.FolderID)
			if err != nil {
				return ImageView{}, err
			}
			mp.FolderID = &f.ID
		}
		mp.MoveFolder = true
	}
	im, err := s.getVisible(ctx, p, id)
	if err != nil {
		return ImageView{}, err
	}
	unlock := s.lockImage(id)
	defer unlock.Unlock()
	ok, err := s.db.PatchImage(ctx, id, mp)
	if err != nil {
		if errors.Is(err, meta.ErrNotFound) { // moved into a folder deleted concurrently
			return ImageView{}, notFound()
		}
		return ImageView{}, err
	}
	if !ok {
		return ImageView{}, notFound()
	}
	if mp.Name != nil && im.OriginalName != *mp.Name {
		s.audit(ctx, p, "image.rename", id, *mp.Name)
		im.OriginalName = *mp.Name
	}
	if mp.MoveFolder && !sameFolder(im.FolderID, mp.FolderID) {
		s.audit(ctx, p, "image.move", id, folderLabel(im.FolderID)+" -> "+folderLabel(mp.FolderID))
		im.FolderID = mp.FolderID
	}
	return s.view(im), nil
}

// MoveImage moves one image to another folder (nil = root).
func (s *Service) MoveImage(ctx context.Context, p Principal, id string, folderID *string) (ImageView, error) {
	if p.Kind == KindAnonymous {
		return ImageView{}, unauthorized()
	}
	if !p.CanManage() {
		return ImageView{}, forbidden("Moving an image requires the manage scope.")
	}
	im, err := s.getVisible(ctx, p, id)
	if err != nil {
		return ImageView{}, err
	}
	if err := s.checkFolderRef(ctx, p, folderID); err != nil {
		return ImageView{}, err
	}
	return s.applyMove(ctx, p, im, folderID)
}

// checkFolderRef validates a folder reference without locking (nil and "root" mean the root).
func (s *Service) checkFolderRef(ctx context.Context, p Principal, folderID *string) error {
	if folderID == nil || *folderID == "" || *folderID == "root" {
		return nil
	}
	_, err := s.resolveFolder(ctx, p, *folderID)
	return err
}

// applyMove is MoveImage with an already-authorized, already-resolved folder reference.
func (s *Service) applyMove(ctx context.Context, p Principal, im *meta.Image, folderID *string) (ImageView, error) {
	var target *string
	if folderID != nil && *folderID != "" && *folderID != "root" {
		f, err := s.resolveFolder(ctx, p, *folderID)
		if err != nil {
			return ImageView{}, err
		}
		target = &f.ID
	}
	unlock := s.lockImage(im.ID)
	defer unlock.Unlock()
	ok, err := s.db.PatchImage(ctx, im.ID, meta.ImagePatch{FolderID: target, MoveFolder: true})
	if err != nil {
		if errors.Is(err, meta.ErrNotFound) { // moved into a folder deleted concurrently
			return ImageView{}, notFound()
		}
		return ImageView{}, err
	}
	if !ok {
		return ImageView{}, notFound()
	}
	if !sameFolder(im.FolderID, target) {
		s.audit(ctx, p, "image.move", im.ID, folderLabel(im.FolderID)+" -> "+folderLabel(target))
		im.FolderID = target
	}
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

// Restore brings an image back from the trash (to the root when its folder was deleted meanwhile).
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

// Batch applies one action to up to 500 images. Per-image failures are reported, not fatal. For move,
// folderID is the destination (nil = root) and is resolved once for the whole batch.
func (s *Service) Batch(ctx context.Context, p Principal, action string, idList []string, folderID *string) ([]BatchResult, error) {
	if p.Kind == KindAnonymous {
		return nil, unauthorized()
	}
	if !p.CanManage() {
		return nil, forbidden("Batch operations require the manage scope.")
	}
	switch action {
	case "trash", "restore", "purge", "move":
	default:
		return nil, validation("action must be trash, restore, purge or move.")
	}
	if len(idList) == 0 || len(idList) > 500 {
		return nil, validation("ids must contain between 1 and 500 entries.")
	}
	if action == "move" {
		if err := s.checkFolderRef(ctx, p, folderID); err != nil {
			return nil, err
		}
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
		case "move":
			im, gerr := s.getVisible(ctx, p, id)
			if gerr != nil {
				err = gerr
			} else {
				_, err = s.applyMove(ctx, p, im, folderID)
			}
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

// OpenRaw opens the original bytes for a principal allowed to read this image.
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

// LookupPublic finds an active, healthy image by id and canonical extension. Every active image is public.
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
	if im.State != meta.StateActive || im.BrokenAt != nil || im.Ext != ext {
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
