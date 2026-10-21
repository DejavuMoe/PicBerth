package meta

import (
	"context"
	"database/sql"
	"errors"
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

// ListAudit returns up to limit entries with id < before (0 = from the newest), newest first.
func (db *DB) ListAudit(ctx context.Context, before int64, limit int) ([]AuditEntry, error) {
	if before <= 0 {
		before = 1<<63 - 1
	}
	rows, err := db.r.QueryContext(ctx,
		`SELECT id, at, actor, action, COALESCE(target, ''), COALESCE(detail, ''), COALESCE(ip, '') FROM audit WHERE id < ? ORDER BY id DESC LIMIT ?`,
		before, limit)
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
