package meta

import (
	"context"
	"database/sql"
	"errors"
	"strings"
)

// Folder is one node of the owner's folder tree. IDs are opaque and never derived from names.
type Folder struct {
	ID        string
	OwnerID   int64
	ParentID  *string
	Name      string
	NameKey   string // lower-cased name, unique among siblings
	Depth     int    // 1..3
	CreatedAt int64
}

var (
	// ErrFolderExists is returned when a sibling already uses the name (case-insensitively).
	ErrFolderExists = errors.New("meta: a sibling folder already uses that name")
	// ErrFolderDepth is returned when a write would push the folder or its subtree below level 3.
	ErrFolderDepth = errors.New("meta: a folder would be deeper than 3 levels")
	// ErrFolderNotEmpty is returned when a folder still has subfolders or active images.
	ErrFolderNotEmpty = errors.New("meta: the folder has subfolders or active images")
	// ErrFolderCycle is returned when a folder would move into itself or one of its descendants.
	ErrFolderCycle = errors.New("meta: a folder cannot move inside itself")
)

const folderCols = `f.id, f.owner_id, f.parent_id, f.name, f.name_key, f.depth, f.created_at`

func scanFolder(sc scanner) (*Folder, error) {
	var f Folder
	if err := sc.Scan(&f.ID, &f.OwnerID, &f.ParentID, &f.Name, &f.NameKey, &f.Depth, &f.CreatedAt); err != nil {
		if errors.Is(err, sql.ErrNoRows) {
			return nil, ErrNotFound
		}
		return nil, err
	}
	return &f, nil
}

// GetFolder returns a folder by id, regardless of owner (callers enforce ownership).
func (db *DB) GetFolder(ctx context.Context, id string) (*Folder, error) {
	return scanFolder(db.r.QueryRowContext(ctx, `SELECT `+folderCols+` FROM folders f WHERE f.id = ?`, id))
}

// ListFolders returns every folder of ownerID (all owners when nil), ordered by name then id.
func (db *DB) ListFolders(ctx context.Context, ownerID *int64) ([]Folder, error) {
	q := `SELECT ` + folderCols + ` FROM folders f`
	var args []any
	if ownerID != nil {
		q += ` WHERE f.owner_id = ?`
		args = append(args, *ownerID)
	}
	q += ` ORDER BY f.name COLLATE NOCASE, f.id`
	rows, err := db.r.QueryContext(ctx, q, args...)
	if err != nil {
		return nil, err
	}
	defer rows.Close()
	var out []Folder
	for rows.Next() {
		f, err := scanFolder(rows)
		if err != nil {
			return nil, err
		}
		out = append(out, *f)
	}
	return out, rows.Err()
}

// FolderActiveCounts returns active, direct image counts by folder id.
func (db *DB) FolderActiveCounts(ctx context.Context, ownerID *int64) (map[string]int64, error) {
	q := `SELECT i.folder_id, COUNT(*) FROM images i WHERE i.state = 'active' AND i.folder_id IS NOT NULL`
	var args []any
	if ownerID != nil {
		q += ` AND i.owner_id = ?`
		args = append(args, *ownerID)
	}
	q += ` GROUP BY i.folder_id`
	rows, err := db.r.QueryContext(ctx, q, args...)
	if err != nil {
		return nil, err
	}
	defer rows.Close()
	out := map[string]int64{}
	for rows.Next() {
		var id string
		var n int64
		if err := rows.Scan(&id, &n); err != nil {
			return nil, err
		}
		out[id] = n
	}
	return out, rows.Err()
}

// folderNameTaken reports whether a sibling of parentID (nil = root) already uses nameKey. excludeID
// lets a rename keep its own name.
func folderNameTaken(ctx context.Context, q queryer, ownerID int64, parentID *string, nameKey, excludeID string) (bool, error) {
	parent := ""
	if parentID != nil {
		parent = *parentID
	}
	var n int
	err := q.QueryRowContext(ctx,
		`SELECT COUNT(*) FROM folders WHERE owner_id = ? AND COALESCE(parent_id, '') = ? AND name_key = ? AND id != ?`,
		ownerID, parent, nameKey, excludeID).Scan(&n)
	return n > 0, err
}

// FolderNameTaken reports whether a sibling of parentID (nil = root) already uses nameKey.
func (db *DB) FolderNameTaken(ctx context.Context, ownerID int64, parentID *string, nameKey, excludeID string) (bool, error) {
	return folderNameTaken(ctx, db.r, ownerID, parentID, nameKey, excludeID)
}

// CreateFolder inserts a folder inside one write transaction. The parent's existence, ownership and
// depth and the sibling-name uniqueness are re-checked there, so a concurrent move or delete cannot
// leave the new folder under a missing parent or below level 3. Depth is computed here and overrides
// f.Depth.
func (db *DB) CreateFolder(ctx context.Context, f Folder) (*Folder, error) {
	var out *Folder
	err := db.WriteTx(ctx, func(tx *sql.Tx) error {
		depth := 1
		if f.ParentID != nil {
			var parentDepth int
			err := tx.QueryRowContext(ctx, `SELECT depth FROM folders WHERE id = ? AND owner_id = ?`,
				*f.ParentID, f.OwnerID).Scan(&parentDepth)
			if errors.Is(err, sql.ErrNoRows) {
				return ErrNotFound
			}
			if err != nil {
				return err
			}
			depth = parentDepth + 1
			if depth > 3 {
				return ErrFolderDepth
			}
		}
		taken, err := folderNameTaken(ctx, tx, f.OwnerID, f.ParentID, f.NameKey, "")
		if err != nil {
			return err
		}
		if taken {
			return ErrFolderExists
		}
		f.Depth = depth
		if _, err := tx.ExecContext(ctx,
			`INSERT INTO folders(id, owner_id, parent_id, name, name_key, depth, created_at) VALUES (?, ?, ?, ?, ?, ?, ?)`,
			f.ID, f.OwnerID, f.ParentID, f.Name, f.NameKey, f.Depth, f.CreatedAt); err != nil {
			if isUniqueViolation(err) {
				return ErrFolderExists
			}
			return err
		}
		out = &f
		return nil
	})
	if err != nil {
		return nil, err
	}
	return out, nil
}

func isUniqueViolation(err error) bool {
	return err != nil && strings.Contains(err.Error(), "UNIQUE constraint failed")
}

// FolderUpdate is one atomic rename and/or move. OwnerID must match the folder's owner; ParentID is
// meaningful only with Move (nil = root). NameKey is the lower-cased new name when Name is set.
type FolderUpdate struct {
	OwnerID  int64
	Name     *string
	NameKey  *string
	Move     bool
	ParentID *string
}

// UpdateFolder applies a rename and/or move inside one write transaction. Ownership, the destination's
// existence and ownership, sibling-name uniqueness, cycles and the depth of the whole subtree are all
// checked against the transaction's snapshot, so concurrent moves cannot create a cycle or depth > 3.
// Every descendant's depth shifts by the same delta. Returns ErrNotFound, ErrFolderExists,
// ErrFolderDepth or ErrFolderCycle.
func (db *DB) UpdateFolder(ctx context.Context, id string, u FolderUpdate) (*Folder, error) {
	var out *Folder
	err := db.WriteTx(ctx, func(tx *sql.Tx) error {
		f, err := scanFolder(tx.QueryRowContext(ctx, `SELECT `+folderCols+` FROM folders f WHERE f.id = ?`, id))
		if err != nil {
			return err // ErrNotFound or a real error
		}
		if f.OwnerID != u.OwnerID {
			return ErrNotFound
		}
		nameKey := f.NameKey
		if u.NameKey != nil {
			nameKey = *u.NameKey
		}
		parentID := f.ParentID
		newDepth := f.Depth
		var subtree []string
		if u.Move {
			parentID = u.ParentID
			newDepth = 1
			if u.ParentID != nil {
				var parentDepth int
				err := tx.QueryRowContext(ctx, `SELECT depth FROM folders WHERE id = ? AND owner_id = ?`,
					*u.ParentID, u.OwnerID).Scan(&parentDepth)
				if errors.Is(err, sql.ErrNoRows) {
					return ErrNotFound
				}
				if err != nil {
					return err
				}
				newDepth = parentDepth + 1
			}
			var maxDepth int
			subtree, maxDepth, err = folderSubtree(ctx, tx, id)
			if err != nil {
				return err
			}
			// The subtree includes the folder itself, so a self-parent is a cycle too.
			for _, did := range subtree {
				if u.ParentID != nil && did == *u.ParentID {
					return ErrFolderCycle
				}
			}
			if newDepth+(maxDepth-f.Depth) > 3 {
				return ErrFolderDepth
			}
		}
		taken, err := folderNameTaken(ctx, tx, u.OwnerID, parentID, nameKey, id)
		if err != nil {
			return err
		}
		if taken {
			return ErrFolderExists
		}
		if u.Name != nil {
			if _, err := tx.ExecContext(ctx, `UPDATE folders SET name = ?, name_key = ? WHERE id = ?`,
				*u.Name, *u.NameKey, id); err != nil {
				if isUniqueViolation(err) {
					return ErrFolderExists
				}
				return err
			}
		}
		if u.Move {
			delta := int64(newDepth - f.Depth)
			if _, err := tx.ExecContext(ctx, `UPDATE folders SET parent_id = ?, depth = ? WHERE id = ?`,
				u.ParentID, newDepth, id); err != nil {
				if isUniqueViolation(err) {
					return ErrFolderExists
				}
				return err
			}
			for _, did := range subtree {
				if did == id {
					continue // the folder itself already has newDepth
				}
				if _, err := tx.ExecContext(ctx, `UPDATE folders SET depth = depth + ? WHERE id = ?`, delta, did); err != nil {
					return err
				}
			}
		}
		updated, err := scanFolder(tx.QueryRowContext(ctx, `SELECT `+folderCols+` FROM folders f WHERE f.id = ?`, id))
		if err != nil {
			return err
		}
		out = updated
		return nil
	})
	if err != nil {
		return nil, err
	}
	return out, nil
}

// subtreeQueryer is the subset of *sql.DB / *sql.Tx the subtree walk needs.
type subtreeQueryer interface {
	QueryContext(ctx context.Context, query string, args ...any) (*sql.Rows, error)
}

func folderSubtree(ctx context.Context, q subtreeQueryer, id string) (ids []string, maxDepth int, err error) {
	rows, err := q.QueryContext(ctx, `WITH RECURSIVE sub(id, depth) AS (
			SELECT id, depth FROM folders WHERE id = ?
			UNION ALL
			SELECT f.id, f.depth FROM folders f JOIN sub s ON f.parent_id = s.id)
		SELECT id, depth FROM sub`, id)
	if err != nil {
		return nil, 0, err
	}
	defer rows.Close()
	for rows.Next() {
		var sid string
		var depth int
		if err := rows.Scan(&sid, &depth); err != nil {
			return nil, 0, err
		}
		ids = append(ids, sid)
		if depth > maxDepth {
			maxDepth = depth
		}
	}
	return ids, maxDepth, rows.Err()
}

// FolderSubtree returns the folder's id and all descendant ids with the deepest depth in the subtree.
func (db *DB) FolderSubtree(ctx context.Context, id string) (ids []string, maxDepth int, err error) {
	return folderSubtree(ctx, db.r, id)
}

// FolderChildCount counts direct subfolders.
func (db *DB) FolderChildCount(ctx context.Context, id string) (int64, error) {
	var n int64
	err := db.r.QueryRowContext(ctx, `SELECT COUNT(*) FROM folders WHERE parent_id = ?`, id).Scan(&n)
	return n, err
}

// FolderActiveImageCount counts active images directly in the folder (trashed images do not block a delete).
func (db *DB) FolderActiveImageCount(ctx context.Context, id string) (int64, error) {
	var n int64
	err := db.r.QueryRowContext(ctx, `SELECT COUNT(*) FROM images WHERE folder_id = ? AND state = 'active'`, id).Scan(&n)
	return n, err
}

// DeleteFolder removes an empty folder inside one write transaction: ownership, subfolders and active
// images are re-checked there, then the trashed/tombstoned images still recorded in it return to the
// root and the folder row is deleted. A concurrent upload into the folder either commits first (and
// makes the delete fail with ErrFolderNotEmpty) or starts after the delete (and finds no folder).
func (db *DB) DeleteFolder(ctx context.Context, id string, ownerID int64) error {
	return db.WriteTx(ctx, func(tx *sql.Tx) error {
		var one int
		err := tx.QueryRowContext(ctx, `SELECT 1 FROM folders WHERE id = ? AND owner_id = ?`, id, ownerID).Scan(&one)
		if errors.Is(err, sql.ErrNoRows) {
			return ErrNotFound
		}
		if err != nil {
			return err
		}
		var children, images int64
		if err := tx.QueryRowContext(ctx, `SELECT COUNT(*) FROM folders WHERE parent_id = ?`, id).Scan(&children); err != nil {
			return err
		}
		if err := tx.QueryRowContext(ctx, `SELECT COUNT(*) FROM images WHERE folder_id = ? AND state = 'active'`, id).Scan(&images); err != nil {
			return err
		}
		if children > 0 || images > 0 {
			return ErrFolderNotEmpty
		}
		if _, err := tx.ExecContext(ctx, `UPDATE images SET folder_id = NULL WHERE folder_id = ?`, id); err != nil {
			return err
		}
		_, err = tx.ExecContext(ctx, `DELETE FROM folders WHERE id = ?`, id)
		return err
	})
}
