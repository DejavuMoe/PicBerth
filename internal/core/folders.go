package core

import (
	"context"
	"errors"
	"strings"
	"unicode/utf8"

	"picberth/internal/ids"
	"picberth/internal/meta"
)

// FolderView is the API representation of a folder (docs/api.md §3).
type FolderView struct {
	ID         string  `json:"id"`
	Name       string  `json:"name"`
	ParentID   *string `json:"parentId"`
	Depth      int     `json:"depth"`
	ImageCount int64   `json:"imageCount"`
	CreatedAt  string  `json:"createdAt"`
}

// FolderPatch is a partial folder update. ParentID is meaningful only with Move; Move with a nil
// ParentID moves the folder to the root.
type FolderPatch struct {
	Name     *string
	ParentID *string
	Move     bool
}

// normalizeFolderName trims and validates a folder name, returning the case-insensitive uniqueness key.
func normalizeFolderName(raw string) (name, key string, err error) {
	if !utf8.ValidString(raw) {
		return "", "", validation("Folder names must be valid UTF-8 text.")
	}
	name = strings.TrimSpace(raw)
	switch n := utf8Len(name); {
	case n == 0:
		return "", "", validation("A folder name must not be empty.")
	case n > maxFolderRunes:
		return "", "", validation("A folder name must be at most %d characters.", maxFolderRunes)
	}
	if hasControl(name) {
		return "", "", validation("A folder name must not contain control characters.")
	}
	if strings.ContainsAny(name, `/\`) {
		return "", "", validation("A folder name must not contain / or \\.")
	}
	return name, strings.ToLower(name), nil
}

// folderOwnerFor returns the folder owner for a new folder.
func (s *Service) folderOwnerFor(p Principal) (int64, error) {
	if p.Kind == KindAnonymous {
		return 0, unauthorized()
	}
	if !p.CanManage() {
		return 0, forbidden("Changing folders requires the manage scope.")
	}
	if p.UserID == 0 {
		return 0, forbidden("This principal has no user account to own folders.")
	}
	return p.UserID, nil
}

// resolveFolder loads a folder the principal may reference, else not-found. Callers handle the root
// (absent parent) themselves; this is only for explicit ids.
func (s *Service) resolveFolder(ctx context.Context, p Principal, id string) (*meta.Folder, error) {
	if p.Kind == KindAnonymous {
		return nil, unauthorized()
	}
	if !ids.ValidFolderID(id) {
		return nil, notFound()
	}
	f, err := s.db.GetFolder(ctx, id)
	if err != nil {
		if errors.Is(err, meta.ErrNotFound) {
			return nil, notFound()
		}
		return nil, err
	}
	if owner := p.folderOwner(); owner != nil && f.OwnerID != *owner {
		return nil, notFound()
	}
	return f, nil
}

// ListFolders returns every folder of the principal's library as a flat list (upload-only tokens may list).
func (s *Service) ListFolders(ctx context.Context, p Principal) ([]FolderView, error) {
	if p.Kind == KindAnonymous {
		return nil, unauthorized()
	}
	owner := p.folderOwner()
	folders, err := s.db.ListFolders(ctx, owner)
	if err != nil {
		return nil, err
	}
	counts, err := s.db.FolderActiveCounts(ctx, owner)
	if err != nil {
		return nil, err
	}
	out := make([]FolderView, len(folders))
	for i, f := range folders {
		out[i] = folderView(&f, counts[f.ID])
	}
	return out, nil
}

func folderView(f *meta.Folder, imageCount int64) FolderView {
	return FolderView{
		ID: f.ID, Name: f.Name, ParentID: f.ParentID, Depth: f.Depth, ImageCount: imageCount, CreatedAt: fmtTime(f.CreatedAt),
	}
}

func folderError(err error) error {
	switch {
	case errors.Is(err, meta.ErrNotFound):
		return notFound()
	case errors.Is(err, meta.ErrFolderExists):
		return folderExists()
	case errors.Is(err, meta.ErrFolderDepth):
		return folderDepth()
	case errors.Is(err, meta.ErrFolderNotEmpty):
		return folderNotEmpty()
	case errors.Is(err, meta.ErrFolderCycle):
		return validation("A folder cannot move inside one of its own subfolders.")
	default:
		return err
	}
}

// CreateFolder creates a folder under parentID (nil = root).
func (s *Service) CreateFolder(ctx context.Context, p Principal, name string, parentID *string) (FolderView, error) {
	owner, err := s.folderOwnerFor(p)
	if err != nil {
		return FolderView{}, err
	}
	norm, key, err := normalizeFolderName(name)
	if err != nil {
		return FolderView{}, err
	}
	var pid *string
	if parentID != nil && *parentID != "" {
		if !ids.ValidFolderID(*parentID) {
			return FolderView{}, notFound()
		}
		pid = parentID
	}
	f, err := s.db.CreateFolder(ctx, meta.Folder{
		ID: ids.NewFolderID(), OwnerID: owner, ParentID: pid, Name: norm, NameKey: key, CreatedAt: s.nowMs(),
	})
	if err != nil {
		return FolderView{}, folderError(err)
	}
	s.audit(ctx, p, "folder.create", f.ID, f.Name)
	return folderView(f, 0), nil
}

// UpdateFolder renames and/or moves a folder. The ownership, sibling-name, cycle and subtree-depth
// checks and the write all run in one metadata transaction, so concurrent moves cannot create a cycle
// or exceed three levels.
func (s *Service) UpdateFolder(ctx context.Context, p Principal, id string, patch FolderPatch) (FolderView, error) {
	owner, err := s.folderOwnerFor(p)
	if err != nil {
		return FolderView{}, err
	}
	if patch.Name == nil && !patch.Move {
		return FolderView{}, validation("Provide at least one of name or parentId.")
	}
	if !ids.ValidFolderID(id) {
		return FolderView{}, notFound()
	}
	// Pre-read only for the 404 and the audit details; the transaction re-checks everything below.
	f, err := s.resolveFolder(ctx, p, id)
	if err != nil {
		return FolderView{}, err
	}
	mp := meta.FolderUpdate{OwnerID: owner, Move: patch.Move}
	newName := f.Name
	if patch.Name != nil {
		norm, key, err := normalizeFolderName(*patch.Name)
		if err != nil {
			return FolderView{}, err
		}
		newName = norm
		mp.Name, mp.NameKey = &norm, &key
	}
	newParentID := f.ParentID
	if patch.Move {
		if patch.ParentID != nil && *patch.ParentID != "" {
			if !ids.ValidFolderID(*patch.ParentID) {
				return FolderView{}, notFound()
			}
			pid := *patch.ParentID
			newParentID = &pid
		} else {
			newParentID = nil
		}
		mp.ParentID = newParentID
	}
	updated, err := s.db.UpdateFolder(ctx, f.ID, mp)
	if err != nil {
		return FolderView{}, folderError(err)
	}
	if patch.Name != nil && f.Name != newName {
		s.audit(ctx, p, "folder.rename", f.ID, newName)
	}
	if patch.Move && !sameFolder(f.ParentID, newParentID) {
		s.audit(ctx, p, "folder.move", f.ID, folderLabel(f.ParentID)+" -> "+folderLabel(newParentID))
	}
	count, err := s.db.FolderActiveImageCount(ctx, f.ID)
	if err != nil {
		return FolderView{}, err
	}
	return folderView(updated, count), nil
}

// DeleteFolder removes a folder only when it has neither subfolders nor active images. The emptiness
// check, the release of trashed images and the delete run in one metadata transaction, so an upload
// racing the delete either makes it fail or starts after the folder is gone. Trashed images recorded in
// the folder return to the root.
func (s *Service) DeleteFolder(ctx context.Context, p Principal, id string) error {
	owner, err := s.folderOwnerFor(p)
	if err != nil {
		return err
	}
	f, err := s.resolveFolder(ctx, p, id)
	if err != nil {
		return err
	}
	if err := s.db.DeleteFolder(ctx, f.ID, owner); err != nil {
		return folderError(err)
	}
	s.audit(ctx, p, "folder.delete", f.ID, f.Name)
	return nil
}
