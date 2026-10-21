package meta

import (
	"context"
	"os"
	"path/filepath"
	"testing"
)

// An upgrade that adds a migration must back up the existing database before touching the schema,
// and a failing migration must leave the database at its previous schema version.
func TestPreMigrationBackupAndFailedMigration(t *testing.T) {
	ctx := context.Background()
	path := filepath.Join(t.TempDir(), "picberth.db")
	ms, err := loadMigrations()
	if err != nil {
		t.Fatal(err)
	}

	db, err := openWith(ctx, path, ms)
	if err != nil {
		t.Fatal(err)
	}
	if db.PreMigrationBackup != "" {
		t.Fatalf("fresh database must not produce a backup, got %q", db.PreMigrationBackup)
	}
	db.Close()

	broken := append(append([]migration(nil), ms...), migration{version: len(ms) + 1, name: "broken.sql", sql: "CREATE TABLE extra(x INTEGER) STRICT; SELECT nonsense FROM missing_table;"})
	if _, err := openWith(ctx, path, broken); err == nil {
		t.Fatal("a failing migration must fail Open")
	}
	db, err = openWith(ctx, path, ms)
	if err != nil {
		t.Fatalf("database must stay usable at the old schema after a failed migration: %v", err)
	}
	var v int
	if err := db.w.QueryRowContext(ctx, `PRAGMA user_version`).Scan(&v); err != nil || v != len(ms) {
		t.Fatalf("user_version = %d (%v), want %d", v, err, len(ms))
	}
	db.Close()

	next := append(append([]migration(nil), ms...), migration{version: len(ms) + 1, name: "extra.sql", sql: "CREATE TABLE extra(x INTEGER) STRICT;"})
	db, err = openWith(ctx, path, next)
	if err != nil {
		t.Fatal(err)
	}
	defer db.Close()
	if db.PreMigrationBackup == "" {
		t.Fatal("migrating an existing schema must write a backup first")
	}
	if _, err := os.Stat(db.PreMigrationBackup); err != nil {
		t.Fatalf("backup missing: %v", err)
	}
	old, err := OpenReadOnly(db.PreMigrationBackup)
	if err != nil {
		t.Fatal(err)
	}
	defer old.Close()
	if err := old.QueryRowContext(ctx, `PRAGMA user_version`).Scan(&v); err != nil || v != len(ms) {
		t.Fatalf("backup user_version = %d (%v), want %d", v, err, len(ms))
	}
}
