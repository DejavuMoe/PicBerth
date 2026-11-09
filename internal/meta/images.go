package meta

import (
	"context"
	"database/sql"
	"errors"
	"strings"
)

// Image states.
const (
	StateActive  = "active"
	StateTrashed = "trashed"
	StatePurged  = "purged"
)

// Image is one image record (or tombstone).
type Image struct {
	ID           string
	OwnerID      int64
	TokenID      *int64
	TokenName    string // joined from tokens; empty when not uploaded by a token
	FolderID     *string
	Source       string
	ObjectKey    string
	OriginalName string
	Mime         string
	Ext          string
	Size         int64
	Width        int
	Height       int
	Frames       int
	SHA256       string
	HasGPS       bool
	State        string
	Thumb        string
	CreatedAt    int64
	TrashedAt    *int64
	PurgedAt     *int64
	BrokenAt     *int64
}

const imageCols = `i.id, i.owner_id, i.token_id, COALESCE(t.name, ''), i.folder_id, i.source, i.object_key, i.original_name, i.mime, i.ext,
	i.size, i.width, i.height, i.frames, i.sha256, i.has_gps, i.state, i.thumb,
	i.created_at, i.trashed_at, i.purged_at, i.broken_at`

const imageFrom = ` FROM images i LEFT JOIN tokens t ON t.id = i.token_id`

type scanner interface{ Scan(...any) error }

func scanImage(sc scanner) (*Image, error) {
	var im Image
	var gps int
	err := sc.Scan(&im.ID, &im.OwnerID, &im.TokenID, &im.TokenName, &im.FolderID, &im.Source, &im.ObjectKey, &im.OriginalName, &im.Mime, &im.Ext,
		&im.Size, &im.Width, &im.Height, &im.Frames, &im.SHA256, &gps, &im.State, &im.Thumb,
		&im.CreatedAt, &im.TrashedAt, &im.PurgedAt, &im.BrokenAt)
	if err != nil {
		if errors.Is(err, sql.ErrNoRows) {
			return nil, ErrNotFound
		}
		return nil, err
	}
	im.HasGPS = gps != 0
	return &im, nil
}

func b2i(b bool) int {
	if b {
		return 1
	}
	return 0
}

// GetImage returns any record (including tombstones) by id.
func (db *DB) GetImage(ctx context.Context, id string) (*Image, error) {
	return scanImage(db.r.QueryRowContext(ctx, `SELECT `+imageCols+imageFrom+` WHERE i.id = ?`, id))
}

// GetImageByKey returns the record owning an object key.
func (db *DB) GetImageByKey(ctx context.Context, key string) (*Image, error) {
	return scanImage(db.r.QueryRowContext(ctx, `SELECT `+imageCols+imageFrom+` WHERE i.object_key = ?`, key))
}

// DupScope is the record set duplicate detection searches: one folder of the owner's active, healthy
// images, restricted to one uploader token when the caller may only see its own uploads. A nil FolderID
// means the root, which counts as a folder of its own.
type DupScope struct {
	OwnerID  int64
	TokenID  *int64
	FolderID *string
}

func (s DupScope) where() (string, []any) {
	cond := "i.owner_id = ?"
	args := []any{s.OwnerID}
	if s.TokenID != nil {
		cond += " AND i.token_id = ?"
		args = append(args, *s.TokenID)
	}
	if s.FolderID == nil {
		cond += " AND i.folder_id IS NULL"
	} else {
		cond += " AND i.folder_id = ?"
		args = append(args, *s.FolderID)
	}
	return cond, args
}

func findActiveDuplicate(ctx context.Context, q queryer, scope DupScope, sha string) (*Image, error) {
	cond, args := scope.where()
	args = append(args, sha)
	return scanImage(q.QueryRowContext(ctx,
		`SELECT `+imageCols+imageFrom+` WHERE `+cond+` AND i.sha256 = ? AND i.state = 'active' AND i.broken_at IS NULL ORDER BY i.created_at, i.id LIMIT 1`, args...))
}

type queryer interface {
	QueryRowContext(ctx context.Context, query string, args ...any) *sql.Row
}

// FindActiveDuplicate returns the earliest active, healthy image in scope with the same content hash.
func (db *DB) FindActiveDuplicate(ctx context.Context, scope DupScope, sha string) (*Image, error) {
	return findActiveDuplicate(ctx, db.r, scope, sha)
}

// InsertImage stores a new record. The destination folder is re-verified inside the same write
// transaction (existence and ownership), so a record can never land in a folder deleted concurrently.
// A non-nil scope enables dedupe inside the same transaction: when an active image in scope with the
// same hash exists it is returned as dup and nothing is written.
func (db *DB) InsertImage(ctx context.Context, im Image, dedupe *DupScope) (dup *Image, err error) {
	err = db.WriteTx(ctx, func(tx *sql.Tx) error {
		if im.FolderID != nil {
			var one int
			err := tx.QueryRowContext(ctx, `SELECT 1 FROM folders WHERE id = ? AND owner_id = ?`,
				*im.FolderID, im.OwnerID).Scan(&one)
			if errors.Is(err, sql.ErrNoRows) {
				return ErrNotFound
			}
			if err != nil {
				return err
			}
		}
		if dedupe != nil {
			d, err := findActiveDuplicate(ctx, tx, *dedupe, im.SHA256)
			if err == nil {
				dup = d
				return nil
			} else if !errors.Is(err, ErrNotFound) {
				return err
			}
		}
		_, err := tx.ExecContext(ctx,
			`INSERT INTO images(id, owner_id, token_id, folder_id, source, object_key, original_name, mime, ext, size, width, height, frames,
			                    sha256, has_gps, state, thumb, created_at)
			 VALUES (?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?, 'active', '', ?)`,
			im.ID, im.OwnerID, im.TokenID, im.FolderID, im.Source, im.ObjectKey, im.OriginalName, im.Mime, im.Ext, im.Size, im.Width, im.Height,
			im.Frames, im.SHA256, b2i(im.HasGPS), im.CreatedAt)
		return err
	})
	return dup, err
}

// MarkTrashed moves an active record to trashed; false when it was not active.
func (db *DB) MarkTrashed(ctx context.Context, id string, now int64) (bool, error) {
	return db.exec1(ctx, `UPDATE images SET state = 'trashed', trashed_at = ? WHERE id = ? AND state = 'active'`, now, id)
}

// MarkRestored moves a trashed record back to active; false when it was not trashed.
func (db *DB) MarkRestored(ctx context.Context, id string) (bool, error) {
	return db.exec1(ctx, `UPDATE images SET state = 'active', trashed_at = NULL WHERE id = ? AND state = 'trashed'`, id)
}

// MarkPurged turns a trashed record into a tombstone and drops its folder; false when it was not trashed.
func (db *DB) MarkPurged(ctx context.Context, id string, now int64) (bool, error) {
	return db.exec1(ctx, `UPDATE images SET state = 'purged', purged_at = ?, folder_id = NULL WHERE id = ? AND state = 'trashed'`, now, id)
}

func (db *DB) exec1(ctx context.Context, q string, args ...any) (bool, error) {
	res, err := db.w.ExecContext(ctx, q, args...)
	if err != nil {
		return false, err
	}
	n, err := res.RowsAffected()
	return n > 0, err
}

// SetThumb records the thumbnail outcome ("jpg", "png" or "none") unless the record was purged.
func (db *DB) SetThumb(ctx context.Context, id, thumb string) (bool, error) {
	return db.exec1(ctx, `UPDATE images SET thumb = ? WHERE id = ? AND state != 'purged'`, thumb, id)
}

// SetBroken sets or clears the broken marker (nil clears). It reports whether a row changed.
func (db *DB) SetBroken(ctx context.Context, id string, at *int64) (bool, error) {
	if at == nil {
		return db.exec1(ctx, `UPDATE images SET broken_at = NULL WHERE id = ? AND broken_at IS NOT NULL`, id)
	}
	return db.exec1(ctx, `UPDATE images SET broken_at = ? WHERE id = ? AND broken_at IS NULL`, *at, id)
}

// ExpiredTrash returns ids of trashed images trashed before cutoff (unix ms), oldest first.
func (db *DB) ExpiredTrash(ctx context.Context, cutoff int64, limit int) ([]string, error) {
	return db.queryIDs(ctx, `SELECT id FROM images WHERE state = 'trashed' AND trashed_at <= ? ORDER BY trashed_at, id LIMIT ?`, cutoff, limit)
}

// PendingThumbs returns active images that have not had a thumbnail attempt yet.
func (db *DB) PendingThumbs(ctx context.Context, limit int) ([]string, error) {
	return db.queryIDs(ctx, `SELECT id FROM images WHERE state = 'active' AND thumb = '' ORDER BY created_at, id LIMIT ?`, limit)
}

// ImageIDsAfter pages through the ids of active and trashed images in id order.
func (db *DB) ImageIDsAfter(ctx context.Context, after string, limit int) ([]string, error) {
	return db.queryIDs(ctx, `SELECT id FROM images WHERE state IN ('active', 'trashed') AND id > ? ORDER BY id LIMIT ?`, after, limit)
}

func (db *DB) queryIDs(ctx context.Context, q string, args ...any) ([]string, error) {
	rows, err := db.r.QueryContext(ctx, q, args...)
	if err != nil {
		return nil, err
	}
	defer rows.Close()
	var out []string
	for rows.Next() {
		var id string
		if err := rows.Scan(&id); err != nil {
			return nil, err
		}
		out = append(out, id)
	}
	return out, rows.Err()
}

// Stats are the aggregate numbers shown on the status page.
type Stats struct {
	Active, Trashed, Broken int64
	Bytes                   int64 // active + trashed
}

// ImageStats aggregates image counts.
func (db *DB) ImageStats(ctx context.Context) (Stats, error) {
	var s Stats
	err := db.r.QueryRowContext(ctx, `SELECT
		COALESCE(SUM(state = 'active'), 0),
		COALESCE(SUM(state = 'trashed'), 0),
		COALESCE(SUM(broken_at IS NOT NULL AND state != 'purged'), 0),
		COALESCE(SUM(CASE WHEN state != 'purged' THEN size ELSE 0 END), 0)
		FROM images`).Scan(&s.Active, &s.Trashed, &s.Broken, &s.Bytes)
	return s, err
}

// Cursor is the keyset position of the last item on a page.
type Cursor struct {
	Key    int64  // created_at or size, depending on the sort
	StrKey string // original_name for the name sort
	ID     string
}

// SourceFilter matches one image source: "web", "cli", or "token" with a token id.
type SourceFilter struct {
	Kind    string
	TokenID int64 // Kind == "token" only
}

// ListQuery filters and orders the image listing. The three folder fields are exclusive: none means
// every folder, FolderRoot means the root only, FolderID one folder, FolderIDs any of a subtree.
type ListQuery struct {
	State      string // active | trashed
	Q          string
	Exts       []string // any of these stored extensions ("jpg", "png", ...)
	Sources    []SourceFilter
	TokenID    *int64 // filter: uploaded by this token
	OwnerID    *int64 // filter: owned by this user
	FolderRoot bool
	FolderID   *string
	FolderIDs  []string
	Since      *int64 // created_at >= Since (unix ms)
	Until      *int64 // created_at < Until (unix ms)
	Sort       string // new | old | large | small | name
	Limit      int
	After      *Cursor
}

func placeholders(n int) string {
	return strings.TrimSuffix(strings.Repeat("?,", n), ",")
}

func escapeLike(s string) string {
	return strings.NewReplacer(`\`, `\\`, `%`, `\%`, `_`, `\_`).Replace(s)
}

func isHex(s string) bool {
	for i := 0; i < len(s); i++ {
		c := s[i]
		if !(c >= '0' && c <= '9' || c >= 'a' && c <= 'f') {
			return false
		}
	}
	return s != ""
}

// ListImages returns up to Limit+1 rows so the caller can tell whether another page exists.
func (db *DB) ListImages(ctx context.Context, q ListQuery) ([]Image, error) {
	var where []string
	var args []any
	where = append(where, "i.state = ?")
	args = append(args, q.State)
	if q.Q != "" {
		lq := strings.ToLower(q.Q)
		conds := []string{`i.original_name LIKE ? ESCAPE '\'`}
		args = append(args, "%"+escapeLike(q.Q)+"%")
		if len(lq) == 20 {
			conds = append(conds, `i.id = ?`)
			args = append(args, lq)
		}
		if len(lq) >= 8 && len(lq) <= 64 && isHex(lq) {
			conds = append(conds, `i.sha256 LIKE ? ESCAPE '\'`)
			args = append(args, escapeLike(lq)+"%")
		}
		where = append(where, "("+strings.Join(conds, " OR ")+")")
	}
	if len(q.Exts) > 0 {
		where = append(where, "i.ext IN ("+placeholders(len(q.Exts))+")")
		for _, e := range q.Exts {
			args = append(args, e)
		}
	}
	if len(q.Sources) > 0 {
		var conds []string
		for _, sf := range q.Sources {
			if sf.Kind == "token" {
				conds = append(conds, "(i.source = 'token' AND i.token_id = ?)")
				args = append(args, sf.TokenID)
			} else {
				conds = append(conds, "i.source = ?")
				args = append(args, sf.Kind)
			}
		}
		where = append(where, "("+strings.Join(conds, " OR ")+")")
	}
	switch {
	case q.FolderRoot:
		where = append(where, "i.folder_id IS NULL")
	case q.FolderID != nil:
		where = append(where, "i.folder_id = ?")
		args = append(args, *q.FolderID)
	case len(q.FolderIDs) > 0:
		where = append(where, "i.folder_id IN ("+placeholders(len(q.FolderIDs))+")")
		for _, id := range q.FolderIDs {
			args = append(args, id)
		}
	}
	if q.Since != nil {
		where = append(where, "i.created_at >= ?")
		args = append(args, *q.Since)
	}
	if q.Until != nil {
		where = append(where, "i.created_at < ?")
		args = append(args, *q.Until)
	}
	if q.TokenID != nil {
		where = append(where, "i.token_id = ?")
		args = append(args, *q.TokenID)
	}
	if q.OwnerID != nil {
		where = append(where, "i.owner_id = ?")
		args = append(args, *q.OwnerID)
	}

	col, dir := "i.created_at", "DESC"
	switch q.Sort {
	case "old":
		dir = "ASC"
	case "large":
		col = "i.size"
	case "small":
		col, dir = "i.size", "ASC"
	case "name":
		col, dir = "i.original_name COLLATE NOCASE", "ASC"
	}
	if q.After != nil {
		op := "<"
		if dir == "ASC" {
			op = ">"
		}
		where = append(where, "("+col+", i.id) "+op+" (?, ?)")
		if q.Sort == "name" {
			args = append(args, q.After.StrKey, q.After.ID)
		} else {
			args = append(args, q.After.Key, q.After.ID)
		}
	}
	args = append(args, q.Limit+1)

	rows, err := db.r.QueryContext(ctx,
		`SELECT `+imageCols+imageFrom+` WHERE `+strings.Join(where, " AND ")+
			` ORDER BY `+col+` `+dir+`, i.id `+dir+` LIMIT ?`, args...)
	if err != nil {
		return nil, err
	}
	defer rows.Close()
	var out []Image
	for rows.Next() {
		im, err := scanImage(rows)
		if err != nil {
			return nil, err
		}
		out = append(out, *im)
	}
	return out, rows.Err()
}

// ImageRef is the minimal record the snapshot needs.
type ImageRef struct {
	ID, ObjectKey, State, SHA256 string
	Size                         int64
}

// ForEachLiveImage streams id/key/state/hash/size of active and trashed images from d (any database
// handle, e.g. a snapshot opened with OpenReadOnly).
func ForEachLiveImage(ctx context.Context, d *sql.DB, fn func(ImageRef) error) error {
	rows, err := d.QueryContext(ctx, `SELECT id, object_key, state, sha256, size FROM images WHERE state IN ('active', 'trashed') ORDER BY id`)
	if err != nil {
		return err
	}
	defer rows.Close()
	for rows.Next() {
		var r ImageRef
		if err := rows.Scan(&r.ID, &r.ObjectKey, &r.State, &r.SHA256, &r.Size); err != nil {
			return err
		}
		if err := fn(r); err != nil {
			return err
		}
	}
	return rows.Err()
}
