package meta

import (
	"context"
	"database/sql"
	"errors"
)

// ImagePatch is a set of changes applied to one record in a single transaction; nil fields are left alone.
type ImagePatch struct {
	Name       *string
	FolderID   *string // meaningful only with MoveFolder
	MoveFolder bool
}

// PatchImage applies p to a non-purged record; false when there is no such record. A move re-verifies
// the destination folder (existence and ownership) inside the same write transaction, so an image can
// never be moved into a folder deleted concurrently. ErrNotFound is returned in that case.
func (db *DB) PatchImage(ctx context.Context, id string, p ImagePatch) (bool, error) {
	var ok bool
	err := db.WriteTx(ctx, func(tx *sql.Tx) error {
		var ownerID int64
		err := tx.QueryRowContext(ctx, `SELECT owner_id FROM images WHERE id = ? AND state != 'purged'`, id).Scan(&ownerID)
		if errors.Is(err, sql.ErrNoRows) {
			return nil
		}
		if err != nil {
			return err
		}
		if p.MoveFolder && p.FolderID != nil {
			var one int
			err := tx.QueryRowContext(ctx, `SELECT 1 FROM folders WHERE id = ? AND owner_id = ?`,
				*p.FolderID, ownerID).Scan(&one)
			if errors.Is(err, sql.ErrNoRows) {
				return ErrNotFound
			}
			if err != nil {
				return err
			}
		}
		ok = true
		if p.Name != nil {
			if _, err := tx.ExecContext(ctx, `UPDATE images SET original_name = ? WHERE id = ?`, *p.Name, id); err != nil {
				return err
			}
		}
		if p.MoveFolder {
			if _, err := tx.ExecContext(ctx, `UPDATE images SET folder_id = ? WHERE id = ?`, p.FolderID, id); err != nil {
				return err
			}
		}
		return nil
	})
	return ok && err == nil, err
}

// SourceCount is the number of images from one source; token sources carry the token's id and name.
type SourceCount struct {
	Source    string // web | token | cli
	TokenID   *int64
	TokenName string
	Count     int64
}

// Facets are the counts over a state's visible set.
type Facets struct {
	Total, Bytes int64
	Exts         map[string]int64 // stored extension -> count
	Sources      []SourceCount
}

// FacetQuery selects the set Facets counts over.
type FacetQuery struct {
	State   string
	TokenID *int64
	OwnerID *int64
}

// ImageFacets counts a state's images by format and source.
func (db *DB) ImageFacets(ctx context.Context, q FacetQuery) (Facets, error) {
	where := "i.state = ?"
	args := []any{q.State}
	if q.TokenID != nil {
		where += " AND i.token_id = ?"
		args = append(args, *q.TokenID)
	}
	if q.OwnerID != nil {
		where += " AND i.owner_id = ?"
		args = append(args, *q.OwnerID)
	}
	f := Facets{Exts: map[string]int64{}}

	rows, err := db.r.QueryContext(ctx, `SELECT i.ext, i.source, i.token_id, COALESCE(t.name, ''),
		COUNT(*), COALESCE(SUM(i.size), 0)`+imageFrom+` WHERE `+where+
		` GROUP BY i.ext, i.source, i.token_id, t.name`, args...)
	if err != nil {
		return f, err
	}
	defer rows.Close()
	type srcKey struct {
		source string
		token  int64
	}
	srcIdx := map[srcKey]int{}
	for rows.Next() {
		var ext, source, tname string
		var tid *int64
		var n, size int64
		if err := rows.Scan(&ext, &source, &tid, &tname, &n, &size); err != nil {
			return f, err
		}
		f.Total += n
		f.Bytes += size
		f.Exts[ext] += n
		k := srcKey{source: source}
		if source == "token" && tid != nil {
			k.token = *tid
		}
		i, seen := srcIdx[k]
		if !seen {
			i = len(f.Sources)
			srcIdx[k] = i
			f.Sources = append(f.Sources, SourceCount{Source: source, TokenID: tid, TokenName: tname})
		}
		f.Sources[i].Count += n
	}
	return f, rows.Err()
}
