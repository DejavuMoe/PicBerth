// Package meta is the SQLite metadata store: connection pools, embedded migrations and repositories.
package meta

import (
	"context"
	"database/sql"
	"embed"
	"errors"
	"fmt"
	"net/url"
	"os"
	"path/filepath"
	"runtime"
	"sort"
	"strconv"
	"strings"
	"time"

	_ "modernc.org/sqlite" // registers the "sqlite" driver
)

//go:embed migrations/*.sql
var migrationFS embed.FS

// ApplicationID is "PICB".
const ApplicationID = 0x50494342

// MinSQLiteVersion is the lowest acceptable embedded SQLite (WAL-reset fix).
const MinSQLiteVersion = "3.51.3"

var (
	// ErrNotFound is returned when a row does not exist.
	ErrNotFound = errors.New("meta: not found")
	// ErrSchemaTooNew is returned when the database was written by a newer PicBerth.
	ErrSchemaTooNew = errors.New("meta: database schema is newer than this build")
)

type migration struct {
	version int
	name    string
	sql     string
}

func loadMigrations() ([]migration, error) {
	entries, err := migrationFS.ReadDir("migrations")
	if err != nil {
		return nil, err
	}
	var ms []migration
	for _, e := range entries {
		numStr, _, _ := strings.Cut(e.Name(), "_")
		n, err := strconv.Atoi(numStr)
		if err != nil {
			return nil, fmt.Errorf("meta: bad migration name %q", e.Name())
		}
		b, err := migrationFS.ReadFile("migrations/" + e.Name())
		if err != nil {
			return nil, err
		}
		ms = append(ms, migration{version: n, name: e.Name(), sql: string(b)})
	}
	sort.Slice(ms, func(i, j int) bool { return ms[i].version < ms[j].version })
	for i, m := range ms {
		if m.version != i+1 {
			return nil, fmt.Errorf("meta: migrations must be numbered 1..N without gaps (found %s at position %d)", m.name, i+1)
		}
	}
	return ms, nil
}

// SchemaVersion is the schema version this build produces.
func SchemaVersion() int {
	ms, err := loadMigrations()
	if err != nil {
		panic(err)
	}
	return len(ms)
}

// DB holds the writer and reader pools.
type DB struct {
	w, r          *sql.DB
	path          string
	sqliteVersion string

	// PreMigrationBackup is the copy written before migrating an existing schema during Open ("" if none).
	PreMigrationBackup string
}

// dsn builds a connection string. Every pragma goes through the DSN so it applies to each pooled connection.
type poolKind int

const (
	kindWriter poolKind = iota
	kindReader
	kindSnapshot // read-write capable (VACUUM INTO needs it) but deferred transactions: never blocks the writer
)

func dsn(path string, kind poolKind) string {
	q := url.Values{}
	q.Add("_pragma", "journal_mode(WAL)")
	q.Add("_pragma", "synchronous(FULL)")
	q.Add("_pragma", "foreign_keys(1)")
	q.Add("_pragma", "busy_timeout(5000)")
	switch kind {
	case kindWriter:
		q.Set("_txlock", "immediate")
	case kindReader:
		q.Add("_pragma", "query_only(1)")
	}
	// The driver treats everything before '?' as the file name when it does not start with "file:",
	// so Windows drive letters and backslashes need no escaping.
	return path + "?" + q.Encode()
}

// Open opens (creating if needed) the database at path, verifies the SQLite version, sets the
// application id and applies pending migrations. It refuses databases from the future.
func Open(ctx context.Context, path string) (*DB, error) {
	ms, err := loadMigrations()
	if err != nil {
		return nil, err
	}
	return openWith(ctx, path, ms)
}

func openWith(ctx context.Context, path string, ms []migration) (*DB, error) {
	if strings.ContainsAny(path, "?#") {
		return nil, fmt.Errorf("meta: database path %q must not contain '?' or '#'", path)
	}
	w, err := sql.Open("sqlite", dsn(path, kindWriter))
	if err != nil {
		return nil, err
	}
	w.SetMaxOpenConns(1)
	w.SetMaxIdleConns(1)
	w.SetConnMaxLifetime(0)

	db := &DB{w: w, path: path}
	if err := db.init(ctx, ms); err != nil {
		w.Close()
		return nil, err
	}

	r, err := sql.Open("sqlite", dsn(path, kindReader))
	if err != nil {
		w.Close()
		return nil, err
	}
	n := runtime.GOMAXPROCS(0)
	if n < 4 {
		n = 4
	}
	r.SetMaxOpenConns(n)
	r.SetMaxIdleConns(n)
	db.r = r
	return db, nil
}

func (db *DB) init(ctx context.Context, ms []migration) error {
	if err := db.w.QueryRowContext(ctx, `SELECT sqlite_version()`).Scan(&db.sqliteVersion); err != nil {
		return fmt.Errorf("meta: opening database: %w", err)
	}
	if compareVersions(db.sqliteVersion, MinSQLiteVersion) < 0 {
		return fmt.Errorf("meta: embedded SQLite %s is older than the required %s", db.sqliteVersion, MinSQLiteVersion)
	}
	var appID, userVersion int
	if err := db.w.QueryRowContext(ctx, `PRAGMA application_id`).Scan(&appID); err != nil {
		return err
	}
	if err := db.w.QueryRowContext(ctx, `PRAGMA user_version`).Scan(&userVersion); err != nil {
		return err
	}
	switch {
	case appID == ApplicationID:
	case appID == 0 && userVersion == 0:
		var n int
		if err := db.w.QueryRowContext(ctx, `SELECT count(*) FROM sqlite_schema`).Scan(&n); err != nil {
			return err
		}
		if n != 0 {
			return errors.New("meta: file is an SQLite database but not a PicBerth one")
		}
		if _, err := db.w.ExecContext(ctx, fmt.Sprintf(`PRAGMA application_id = %d`, ApplicationID)); err != nil {
			return err
		}
	default:
		return fmt.Errorf("meta: file is not a PicBerth database (application_id %#x)", appID)
	}
	if userVersion > len(ms) {
		return fmt.Errorf("%w (schema %d, this build knows %d): use a newer PicBerth or restore the snapshot taken before the upgrade",
			ErrSchemaTooNew, userVersion, len(ms))
	}
	if userVersion > 0 && userVersion < len(ms) {
		// Baseline §12.4: snapshot the metadata before changing an existing schema. Image files are
		// immutable and untouched by migrations, so the database copy is what a rollback needs.
		backup, err := db.backupBeforeMigration(ctx, userVersion)
		if err != nil {
			return fmt.Errorf("meta: pre-migration backup failed, not migrating: %w", err)
		}
		db.PreMigrationBackup = backup
	}
	for _, m := range ms[userVersion:] {
		if err := db.apply(ctx, m); err != nil {
			return fmt.Errorf("meta: migration %s failed (database left at schema %d): %w", m.name, m.version-1, err)
		}
	}
	return nil
}

// backupBeforeMigration writes a consistent copy of the database to backups/ next to it.
func (db *DB) backupBeforeMigration(ctx context.Context, from int) (string, error) {
	dir := filepath.Join(filepath.Dir(db.path), "backups")
	if err := os.MkdirAll(dir, 0o700); err != nil {
		return "", err
	}
	base := filepath.Join(dir, fmt.Sprintf("picberth-schema%d-%s", from, time.Now().UTC().Format("20060102T150405Z")))
	name := base + ".db"
	for i := 2; ; i++ { // VACUUM INTO refuses existing files; a retry within the same second must not fail
		if _, err := os.Lstat(name); errors.Is(err, os.ErrNotExist) {
			break
		}
		name = fmt.Sprintf("%s-%d.db", base, i)
	}
	if _, err := db.w.ExecContext(ctx, `VACUUM INTO ?`, name); err != nil {
		return "", err
	}
	return name, nil
}

func (db *DB) apply(ctx context.Context, m migration) error {
	tx, err := db.w.BeginTx(ctx, nil)
	if err != nil {
		return err
	}
	defer tx.Rollback()
	if _, err := tx.ExecContext(ctx, m.sql); err != nil {
		return err
	}
	if _, err := tx.ExecContext(ctx, fmt.Sprintf(`PRAGMA user_version = %d`, m.version)); err != nil {
		return err
	}
	return tx.Commit()
}

// compareVersions compares dotted numeric versions ("3.53.4").
func compareVersions(a, b string) int {
	pa, pb := strings.Split(a, "."), strings.Split(b, ".")
	for i := 0; i < len(pa) || i < len(pb); i++ {
		var x, y int
		if i < len(pa) {
			x, _ = strconv.Atoi(pa[i])
		}
		if i < len(pb) {
			y, _ = strconv.Atoi(pb[i])
		}
		if x != y {
			if x < y {
				return -1
			}
			return 1
		}
	}
	return 0
}

// SQLiteVersion is the version of the embedded SQLite library.
func (db *DB) SQLiteVersion() string { return db.sqliteVersion }

// Path is the database file path.
func (db *DB) Path() string { return db.path }

// Reader is the read-only pool (queries only).
func (db *DB) Reader() *sql.DB { return db.r }

// Writer is the single-connection write pool.
func (db *DB) Writer() *sql.DB { return db.w }

// Close closes both pools. Closing the last connection checkpoints and removes the WAL.
func (db *DB) Close() error {
	return errors.Join(db.r.Close(), db.w.Close())
}

// WriteTx runs fn in a BEGIN IMMEDIATE transaction on the writer.
func (db *DB) WriteTx(ctx context.Context, fn func(tx *sql.Tx) error) error {
	tx, err := db.w.BeginTx(ctx, nil)
	if err != nil {
		return err
	}
	if err := fn(tx); err != nil {
		tx.Rollback()
		return err
	}
	return tx.Commit()
}

// Ping checks that the database can be read.
func (db *DB) Ping(ctx context.Context) error {
	var one int
	return db.r.QueryRowContext(ctx, `SELECT 1`).Scan(&one)
}

// OpenReadOnly opens an existing database file read-only without migrating it (snapshot verification).
// It does not force WAL mode, so rollback-journal files produced by VACUUM INTO open cleanly.
func OpenReadOnly(path string) (*sql.DB, error) {
	if strings.ContainsAny(path, "?#") {
		return nil, fmt.Errorf("meta: database path %q must not contain '?' or '#'", path)
	}
	q := url.Values{}
	q.Add("_pragma", "foreign_keys(1)")
	q.Add("_pragma", "busy_timeout(5000)")
	q.Add("_pragma", "query_only(1)")
	d, err := sql.Open("sqlite", path+"?"+q.Encode())
	if err != nil {
		return nil, err
	}
	d.SetMaxOpenConns(1)
	return d, nil
}

// Integrity runs PRAGMA integrity_check and foreign_key_check and returns the problems found.
func Integrity(ctx context.Context, d *sql.DB) ([]string, error) {
	var problems []string
	rows, err := d.QueryContext(ctx, `PRAGMA integrity_check`)
	if err != nil {
		return nil, err
	}
	for rows.Next() {
		var s string
		if err := rows.Scan(&s); err != nil {
			rows.Close()
			return nil, err
		}
		if s != "ok" {
			problems = append(problems, "integrity_check: "+s)
		}
	}
	if err := rows.Err(); err != nil {
		rows.Close()
		return nil, err
	}
	rows.Close()
	rows, err = d.QueryContext(ctx, `PRAGMA foreign_key_check`)
	if err != nil {
		return nil, err
	}
	defer rows.Close()
	for rows.Next() {
		var table string
		var rowid sql.NullInt64
		var parent string
		var fk sql.NullInt64
		if err := rows.Scan(&table, &rowid, &parent, &fk); err != nil {
			return nil, err
		}
		problems = append(problems, fmt.Sprintf("foreign_key_check: %s row %d references missing %s", table, rowid.Int64, parent))
	}
	return problems, rows.Err()
}
