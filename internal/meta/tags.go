package meta

import (
	"context"
	"database/sql"
	"errors"
	"sort"
)

// insertTags stores already-normalized, de-duplicated tags for an image.
func insertTags(ctx context.Context, tx *sql.Tx, id string, tags []string) error {
	for _, t := range tags {
		if _, err := tx.ExecContext(ctx, `INSERT INTO image_tags(image_id, tag) VALUES (?, ?)`, id, t); err != nil {
			return err
		}
	}
	return nil
}

// TagsFor loads the tags of many images in as few queries as possible. Every requested id is present in
// the result; tags are sorted ascending.
func (db *DB) TagsFor(ctx context.Context, imageIDs []string) (map[string][]string, error) {
	out := make(map[string][]string, len(imageIDs))
	for _, id := range imageIDs {
		out[id] = []string{}
	}
	const chunk = 500
	for start := 0; start < len(imageIDs); start += chunk {
		end := min(start+chunk, len(imageIDs))
		part := imageIDs[start:end]
		args := make([]any, len(part))
		for i, id := range part {
			args[i] = id
		}
		rows, err := db.r.QueryContext(ctx, `SELECT image_id, tag FROM image_tags WHERE image_id IN (`+placeholders(len(part))+`)`, args...)
		if err != nil {
			return nil, err
		}
		for rows.Next() {
			var id, tag string
			if err := rows.Scan(&id, &tag); err != nil {
				rows.Close()
				return nil, err
			}
			out[id] = append(out[id], tag)
		}
		if err := rows.Err(); err != nil {
			rows.Close()
			return nil, err
		}
		rows.Close()
	}
	for _, tags := range out {
		sort.Strings(tags)
	}
	return out, nil
}

// ImagePatch is a set of changes applied to one record in a single transaction; nil fields are left alone.
type ImagePatch struct {
	Visibility *string
	Name       *string
	Tags       *[]string // replaces the whole set (normalized, de-duplicated)
}

// PatchImage applies p to a non-purged record; false when there is no such record.
func (db *DB) PatchImage(ctx context.Context, id string, p ImagePatch) (bool, error) {
	var ok bool
	err := db.WriteTx(ctx, func(tx *sql.Tx) error {
		var one int
		err := tx.QueryRowContext(ctx, `SELECT 1 FROM images WHERE id = ? AND state != 'purged'`, id).Scan(&one)
		if errors.Is(err, sql.ErrNoRows) {
			return nil
		}
		if err != nil {
			return err
		}
		ok = true
		if p.Visibility != nil {
			if _, err := tx.ExecContext(ctx, `UPDATE images SET visibility = ? WHERE id = ?`, *p.Visibility, id); err != nil {
				return err
			}
		}
		if p.Name != nil {
			if _, err := tx.ExecContext(ctx, `UPDATE images SET original_name = ? WHERE id = ?`, *p.Name, id); err != nil {
				return err
			}
		}
		if p.Tags != nil {
			if _, err := tx.ExecContext(ctx, `DELETE FROM image_tags WHERE image_id = ?`, id); err != nil {
				return err
			}
			return insertTags(ctx, tx, id, *p.Tags)
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

// TagCount is the number of images carrying a tag.
type TagCount struct {
	Tag   string
	Count int64
}

// Facets are the counts over a state's visible set.
type Facets struct {
	Total, Bytes int64
	Exts         map[string]int64 // stored extension -> count
	Visibility   map[string]int64
	Sources      []SourceCount
	Tags         []TagCount // count desc, tag asc
}

// FacetQuery selects the set Facets counts over.
type FacetQuery struct {
	State   string
	TokenID *int64
	OwnerID *int64
}

// ImageFacets counts a state's images by format, visibility, source and tag.
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
	f := Facets{Exts: map[string]int64{}, Visibility: map[string]int64{}}

	rows, err := db.r.QueryContext(ctx, `SELECT i.ext, i.visibility, i.source, i.token_id, COALESCE(t.name, ''),
		COUNT(*), COALESCE(SUM(i.size), 0)`+imageFrom+` WHERE `+where+
		` GROUP BY i.ext, i.visibility, i.source, i.token_id, t.name`, args...)
	if err != nil {
		return f, err
	}
	type srcKey struct {
		source string
		token  int64
	}
	srcIdx := map[srcKey]int{}
	for rows.Next() {
		var ext, vis, source, tname string
		var tid *int64
		var n, size int64
		if err := rows.Scan(&ext, &vis, &source, &tid, &tname, &n, &size); err != nil {
			rows.Close()
			return f, err
		}
		f.Total += n
		f.Bytes += size
		f.Exts[ext] += n
		f.Visibility[vis] += n
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
	if err := rows.Err(); err != nil {
		rows.Close()
		return f, err
	}
	rows.Close()

	trows, err := db.r.QueryContext(ctx, `SELECT g.tag, COUNT(*) FROM image_tags g JOIN images i ON i.id = g.image_id WHERE `+where+
		` GROUP BY g.tag ORDER BY COUNT(*) DESC, g.tag ASC`, args...)
	if err != nil {
		return f, err
	}
	defer trows.Close()
	for trows.Next() {
		var tc TagCount
		if err := trows.Scan(&tc.Tag, &tc.Count); err != nil {
			return f, err
		}
		f.Tags = append(f.Tags, tc)
	}
	return f, trows.Err()
}
