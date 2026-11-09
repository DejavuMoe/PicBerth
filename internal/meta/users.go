package meta

import (
	"context"
	"database/sql"
	"errors"
)

// User is the (single) administrator account.
type User struct {
	ID           int64
	Username     string
	PasswordHash string
	CreatedAt    int64
}

// Session is a browser login. Expiry is derived from created_at/last_seen_at and fixed lifetimes.
type Session struct {
	IDHash     string
	UserID     int64
	CreatedAt  int64
	LastSeenAt int64
	Username   string // joined, read-only
}

// Token is an API token. The secret itself is never stored.
type Token struct {
	ID         int64
	OwnerID    int64
	Name       string
	Prefix     string
	SecretHash string
	Scopes     string // comma-separated
	CreatedAt  int64
	ExpiresAt  *int64
	LastUsedAt *int64
	RevokedAt  *int64
}

// CountUsers returns the number of user rows.
func (db *DB) CountUsers(ctx context.Context) (int, error) {
	var n int
	err := db.r.QueryRowContext(ctx, `SELECT count(*) FROM users`).Scan(&n)
	return n, err
}

// CreateFirstUser inserts the administrator only if no user exists yet, atomically. created is false when
// another user already exists.
func (db *DB) CreateFirstUser(ctx context.Context, username, hash string, now int64) (id int64, created bool, err error) {
	res, err := db.w.ExecContext(ctx,
		`INSERT INTO users(username, password_hash, created_at)
		 SELECT ?, ?, ? WHERE NOT EXISTS (SELECT 1 FROM users)`, username, hash, now)
	if err != nil {
		return 0, false, err
	}
	n, err := res.RowsAffected()
	if err != nil || n == 0 {
		return 0, false, err
	}
	id, err = res.LastInsertId()
	return id, true, err
}

// GetUserByName finds a user by name (case-insensitive).
func (db *DB) GetUserByName(ctx context.Context, name string) (*User, error) {
	return scanUser(db.r.QueryRowContext(ctx,
		`SELECT id, username, password_hash, created_at FROM users WHERE username = ?`, name))
}

// GetUser finds a user by id.
func (db *DB) GetUser(ctx context.Context, id int64) (*User, error) {
	return scanUser(db.r.QueryRowContext(ctx,
		`SELECT id, username, password_hash, created_at FROM users WHERE id = ?`, id))
}

func scanUser(row *sql.Row) (*User, error) {
	var u User
	if err := row.Scan(&u.ID, &u.Username, &u.PasswordHash, &u.CreatedAt); err != nil {
		if errors.Is(err, sql.ErrNoRows) {
			return nil, ErrNotFound
		}
		return nil, err
	}
	return &u, nil
}

// SetPassword replaces a user's password hash and ends all of their sessions.
func (db *DB) SetPassword(ctx context.Context, userID int64, hash string) error {
	return db.WriteTx(ctx, func(tx *sql.Tx) error {
		res, err := tx.ExecContext(ctx, `UPDATE users SET password_hash = ? WHERE id = ?`, hash, userID)
		if err != nil {
			return err
		}
		if n, _ := res.RowsAffected(); n == 0 {
			return ErrNotFound
		}
		_, err = tx.ExecContext(ctx, `DELETE FROM sessions WHERE user_id = ?`, userID)
		return err
	})
}

// CreateSession stores a new session.
func (db *DB) CreateSession(ctx context.Context, s Session) error {
	_, err := db.w.ExecContext(ctx,
		`INSERT INTO sessions(id_hash, user_id, created_at, last_seen_at) VALUES (?, ?, ?, ?)`,
		s.IDHash, s.UserID, s.CreatedAt, s.LastSeenAt)
	return err
}

// GetSession looks a session up by the SHA-256 of its id.
func (db *DB) GetSession(ctx context.Context, idHash string) (*Session, error) {
	var s Session
	err := db.r.QueryRowContext(ctx,
		`SELECT s.id_hash, s.user_id, s.created_at, s.last_seen_at, u.username
		 FROM sessions s JOIN users u ON u.id = s.user_id WHERE s.id_hash = ?`, idHash).
		Scan(&s.IDHash, &s.UserID, &s.CreatedAt, &s.LastSeenAt, &s.Username)
	if errors.Is(err, sql.ErrNoRows) {
		return nil, ErrNotFound
	}
	return &s, err
}

// TouchSession records activity.
func (db *DB) TouchSession(ctx context.Context, idHash string, now int64) error {
	_, err := db.w.ExecContext(ctx, `UPDATE sessions SET last_seen_at = ? WHERE id_hash = ?`, now, idHash)
	return err
}

// DeleteSession ends a session.
func (db *DB) DeleteSession(ctx context.Context, idHash string) error {
	_, err := db.w.ExecContext(ctx, `DELETE FROM sessions WHERE id_hash = ?`, idHash)
	return err
}

// DeleteExpiredSessions removes sessions past their idle or absolute lifetime and returns how many.
func (db *DB) DeleteExpiredSessions(ctx context.Context, now, idleMs, absoluteMs int64) (int64, error) {
	res, err := db.w.ExecContext(ctx,
		`DELETE FROM sessions WHERE last_seen_at < ? OR created_at < ?`, now-idleMs, now-absoluteMs)
	if err != nil {
		return 0, err
	}
	return res.RowsAffected()
}

const tokenCols = `t.id, t.owner_id, t.name, t.prefix, t.secret_hash, t.scopes, t.created_at, t.expires_at, t.last_used_at, t.revoked_at`

func scanToken(sc interface{ Scan(...any) error }) (*Token, error) {
	var t Token
	if err := sc.Scan(&t.ID, &t.OwnerID, &t.Name, &t.Prefix, &t.SecretHash, &t.Scopes, &t.CreatedAt, &t.ExpiresAt, &t.LastUsedAt, &t.RevokedAt); err != nil {
		if errors.Is(err, sql.ErrNoRows) {
			return nil, ErrNotFound
		}
		return nil, err
	}
	return &t, nil
}

// CreateToken inserts a token and returns it with its id.
func (db *DB) CreateToken(ctx context.Context, t Token) (*Token, error) {
	res, err := db.w.ExecContext(ctx,
		`INSERT INTO tokens(owner_id, name, prefix, secret_hash, scopes, created_at, expires_at) VALUES (?, ?, ?, ?, ?, ?, ?)`,
		t.OwnerID, t.Name, t.Prefix, t.SecretHash, t.Scopes, t.CreatedAt, t.ExpiresAt)
	if err != nil {
		return nil, err
	}
	t.ID, err = res.LastInsertId()
	return &t, err
}

// GetTokenBySecretHash finds a token by the SHA-256 of its secret.
func (db *DB) GetTokenBySecretHash(ctx context.Context, hash string) (*Token, error) {
	return scanToken(db.r.QueryRowContext(ctx, `SELECT `+tokenCols+` FROM tokens t WHERE t.secret_hash = ?`, hash))
}

// GetToken finds a token by id.
func (db *DB) GetToken(ctx context.Context, id int64) (*Token, error) {
	return scanToken(db.r.QueryRowContext(ctx, `SELECT `+tokenCols+` FROM tokens t WHERE t.id = ?`, id))
}

// ListTokens returns an owner's tokens, newest first, revoked included.
func (db *DB) ListTokens(ctx context.Context, ownerID int64) ([]Token, error) {
	rows, err := db.r.QueryContext(ctx, `SELECT `+tokenCols+` FROM tokens t WHERE t.owner_id = ? ORDER BY t.created_at DESC, t.id DESC`, ownerID)
	if err != nil {
		return nil, err
	}
	defer rows.Close()
	var out []Token
	for rows.Next() {
		t, err := scanToken(rows)
		if err != nil {
			return nil, err
		}
		out = append(out, *t)
	}
	return out, rows.Err()
}

// RevokeToken marks a token revoked; revoking twice keeps the first timestamp.
func (db *DB) RevokeToken(ctx context.Context, id int64, now int64) error {
	_, err := db.w.ExecContext(ctx, `UPDATE tokens SET revoked_at = ? WHERE id = ? AND revoked_at IS NULL`, now, id)
	return err
}

// TouchToken records token use.
func (db *DB) TouchToken(ctx context.Context, id int64, now int64) error {
	_, err := db.w.ExecContext(ctx, `UPDATE tokens SET last_used_at = ? WHERE id = ?`, now, id)
	return err
}
