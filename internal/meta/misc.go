package meta

import (
	"context"
	"database/sql"
	"errors"
	"strings"
)

// AuditEntry is one row of the audit log.
type AuditEntry struct {
	ID     int64
	At     int64
	Actor  string
	Action string
	Target string
	Detail string
	IP     string
}

// AddAudit appends an audit entry.
func (db *DB) AddAudit(ctx context.Context, e AuditEntry) error {
	_, err := db.w.ExecContext(ctx, `INSERT INTO audit(at, actor, action, target, detail, ip) VALUES (?, ?, ?, ?, ?, ?)`,
		e.At, e.Actor, e.Action, nullStr(e.Target), nullStr(e.Detail), nullStr(e.IP))
	return err
}

func nullStr(s string) any {
	if s == "" {
		return nil
	}
	return s
}

// AuditFilter narrows an audit listing. Action keeps exactly that action string; Exact and Prefixes,
// when either is non-empty, keep actions equal to one of Exact or starting with one of Prefixes.
type AuditFilter struct {
	Action   string
	Exact    []string
	Prefixes []string
}

// ListAudit returns up to limit entries with id < before (0 = from the newest), newest first, that
// match f.
func (db *DB) ListAudit(ctx context.Context, before int64, f AuditFilter, limit int) ([]AuditEntry, error) {
	if before <= 0 {
		before = 1<<63 - 1
	}
	q := `SELECT id, at, actor, action, COALESCE(target, ''), COALESCE(detail, ''), COALESCE(ip, '') FROM audit WHERE id < ?`
	args := []any{before}
	if f.Action != "" {
		q += ` AND action = ?`
		args = append(args, f.Action)
	}
	if len(f.Exact)+len(f.Prefixes) > 0 {
		var or []string
		for _, a := range f.Exact {
			or = append(or, `action = ?`)
			args = append(args, a)
		}
		for _, p := range f.Prefixes {
			or = append(or, `substr(action, 1, ?) = ?`)
			args = append(args, len(p), p)
		}
		q += ` AND (` + strings.Join(or, ` OR `) + `)`
	}
	q += ` ORDER BY id DESC LIMIT ?`
	args = append(args, limit)
	rows, err := db.r.QueryContext(ctx, q, args...)
	if err != nil {
		return nil, err
	}
	defer rows.Close()
	var out []AuditEntry
	for rows.Next() {
		var e AuditEntry
		if err := rows.Scan(&e.ID, &e.At, &e.Actor, &e.Action, &e.Target, &e.Detail, &e.IP); err != nil {
			return nil, err
		}
		out = append(out, e)
	}
	return out, rows.Err()
}

// AuditActions returns the distinct action strings in the log, ascending.
func (db *DB) AuditActions(ctx context.Context) ([]string, error) {
	rows, err := db.r.QueryContext(ctx, `SELECT DISTINCT action FROM audit ORDER BY action`)
	if err != nil {
		return nil, err
	}
	defer rows.Close()
	var out []string
	for rows.Next() {
		var a string
		if err := rows.Scan(&a); err != nil {
			return nil, err
		}
		out = append(out, a)
	}
	return out, rows.Err()
}

// DeleteAuditBefore removes at most limit entries older than cutoff (unix ms), oldest first, and
// reports how many rows went away.
func (db *DB) DeleteAuditBefore(ctx context.Context, cutoff int64, limit int) (int64, error) {
	res, err := db.w.ExecContext(ctx,
		`DELETE FROM audit WHERE id IN (SELECT id FROM audit WHERE at < ? ORDER BY id LIMIT ?)`, cutoff, limit)
	if err != nil {
		return 0, err
	}
	return res.RowsAffected()
}

// AllSettings returns every stored setting as raw JSON text by key.
func (db *DB) AllSettings(ctx context.Context) (map[string]string, error) {
	rows, err := db.r.QueryContext(ctx, `SELECT key, value FROM settings`)
	if err != nil {
		return nil, err
	}
	defer rows.Close()
	out := map[string]string{}
	for rows.Next() {
		var k, v string
		if err := rows.Scan(&k, &v); err != nil {
			return nil, err
		}
		out[k] = v
	}
	return out, rows.Err()
}

// PutSettings upserts settings (key → JSON text) in one transaction.
func (db *DB) PutSettings(ctx context.Context, kv map[string]string) error {
	return db.WriteTx(ctx, func(tx *sql.Tx) error {
		for k, v := range kv {
			if _, err := tx.ExecContext(ctx,
				`INSERT INTO settings(key, value) VALUES (?, ?) ON CONFLICT(key) DO UPDATE SET value = excluded.value`, k, v); err != nil {
				return err
			}
		}
		return nil
	})
}

// KVGet reads a kv entry.
func (db *DB) KVGet(ctx context.Context, key string) (string, error) {
	var v string
	err := db.r.QueryRowContext(ctx, `SELECT value FROM kv WHERE key = ?`, key).Scan(&v)
	if errors.Is(err, sql.ErrNoRows) {
		return "", ErrNotFound
	}
	return v, err
}

// KVSet writes a kv entry.
func (db *DB) KVSet(ctx context.Context, key, value string) error {
	_, err := db.w.ExecContext(ctx,
		`INSERT INTO kv(key, value) VALUES (?, ?) ON CONFLICT(key) DO UPDATE SET value = excluded.value`, key, value)
	return err
}

// VacuumInto writes a consistent copy of the database to dest (which must not exist). It uses a
// short-lived dedicated connection: the reader pool is query_only (which forbids VACUUM INTO) and the
// writer connection must stay free for uploads.
func (db *DB) VacuumInto(ctx context.Context, dest string) error {
	d, err := sql.Open("sqlite", dsn(db.path, kindSnapshot))
	if err != nil {
		return err
	}
	defer d.Close()
	d.SetMaxOpenConns(1)
	_, err = d.ExecContext(ctx, `VACUUM INTO ?`, dest)
	return err
}
