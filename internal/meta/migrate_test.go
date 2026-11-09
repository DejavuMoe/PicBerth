package meta

import (
	"context"
	"os"
	"path/filepath"
	"testing"
	"time"
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

// A database written before folders (schema 2) keeps every image, loses tags/visibility/read scopes and
// gains the folders table; the pre-migration backup still holds the old schema.
func TestMigrateSchema2ToFolders(t *testing.T) {
	ctx := context.Background()
	path := filepath.Join(t.TempDir(), "picberth.db")
	ms, err := loadMigrations()
	if err != nil {
		t.Fatal(err)
	}
	if len(ms) < 3 {
		t.Fatalf("expected at least 3 migrations, got %d", len(ms))
	}
	db, err := openWith(ctx, path, ms[:2])
	if err != nil {
		t.Fatal(err)
	}
	uid, _, _ := db.CreateFirstUser(ctx, "a", "h", 1)
	now := int64(1_700_000_000_000)
	if _, err := db.w.ExecContext(ctx, `INSERT INTO images(id, owner_id, source, object_key, original_name, mime, ext, size, width, height,
			frames, sha256, has_gps, visibility, state, created_at)
		VALUES ('aaaaaaaaaaaaaaaaaaaa', ?, 'web', 'aa/aa/aaaaaaaaaaaaaaaaaaaa.png', 'old.png', 'image/png', 'png', 12, 2, 2, 1, 'abc', 0, 'private', 'active', ?)`, uid, now); err != nil {
		t.Fatal(err)
	}
	if _, err := db.w.ExecContext(ctx, `INSERT INTO image_tags(image_id, tag) VALUES ('aaaaaaaaaaaaaaaaaaaa', 'x')`); err != nil {
		t.Fatal(err)
	}
	for i, scopes := range []struct {
		scopes  string
		revoked *int64
	}{
		{"upload", nil}, {"read", nil}, {"upload,read", nil}, {"read,manage", nil}, {"upload,manage", nil}, {"upload,read,manage", nil}, {"read", &now},
	} {
		if _, err := db.w.ExecContext(ctx, `INSERT INTO tokens(owner_id, name, prefix, secret_hash, scopes, created_at, revoked_at) VALUES (?, ?, 'pb_', ?, ?, 1, ?)`,
			uid, "t"+string(rune('a'+i)), "hash"+string(rune('a'+i)), scopes.scopes, scopes.revoked); err != nil {
			t.Fatal(err)
		}
	}
	if _, err := db.w.ExecContext(ctx, `INSERT INTO sessions(id_hash, user_id, created_at, last_seen_at, expires_at, ip, user_agent) VALUES ('sid', ?, 1, 2, 3, '1.2.3.4', 'ua')`, uid); err != nil {
		t.Fatal(err)
	}
	for _, k := range []string{"maxPixels", "defaultVisibility", "uploadsPerMinute", "sessionIdleMinutes", "sessionAbsoluteHours"} {
		if _, err := db.w.ExecContext(ctx, `INSERT INTO settings(key, value) VALUES (?, '1')`, k); err != nil {
			t.Fatal(err)
		}
	}
	db.Close()

	db, err = Open(ctx, path)
	if err != nil {
		t.Fatal(err)
	}
	defer db.Close()
	if db.PreMigrationBackup == "" {
		t.Fatal("upgrade from schema 2 must write a backup")
	}
	var uv int
	if err := db.w.QueryRowContext(ctx, `PRAGMA user_version`).Scan(&uv); err != nil || uv != len(ms) {
		t.Fatalf("user_version = %d (%v), want %d", uv, err, len(ms))
	}

	// the image survives, with no folder and no visibility column any more
	im, err := db.GetImage(ctx, "aaaaaaaaaaaaaaaaaaaa")
	if err != nil || im.SHA256 != "abc" || im.FolderID != nil {
		t.Fatalf("existing image: %+v %v", im, err)
	}
	var table string
	if err := db.w.QueryRowContext(ctx, `SELECT name FROM sqlite_schema WHERE name = 'image_tags'`).Scan(&table); err == nil {
		t.Fatal("image_tags still exists")
	}
	var visibilityCols int
	if err := db.w.QueryRowContext(ctx, `SELECT count(*) FROM pragma_table_info('images') WHERE name = 'visibility'`).Scan(&visibilityCols); err != nil || visibilityCols != 0 {
		t.Fatalf("visibility column: %d %v", visibilityCols, err)
	}
	var folderCols int
	if err := db.w.QueryRowContext(ctx, `SELECT count(*) FROM pragma_table_info('images') WHERE name = 'folder_id'`).Scan(&folderCols); err != nil || folderCols != 1 {
		t.Fatalf("folder_id column: %d %v", folderCols, err)
	}
	var n int
	if err := db.w.QueryRowContext(ctx, `SELECT count(*) FROM sqlite_schema WHERE name = 'folders'`).Scan(&n); err != nil || n != 1 {
		t.Fatalf("folders table: %d %v", n, err)
	}

	// read is stripped from every token; a token left with no scope is revoked with an audit reason
	rows, err := db.w.QueryContext(ctx, `SELECT id, scopes, revoked_at FROM tokens ORDER BY id`)
	if err != nil {
		t.Fatal(err)
	}
	type tokRow struct {
		id      int64
		scopes  string
		revoked *int64
	}
	var toks []tokRow
	for rows.Next() {
		var tr tokRow
		if err := rows.Scan(&tr.id, &tr.scopes, &tr.revoked); err != nil {
			t.Fatal(err)
		}
		toks = append(toks, tr)
	}
	rows.Close()
	wantScopes := []string{"upload", "", "upload", "manage", "upload,manage", "upload,manage", ""}
	for i, tr := range toks {
		if tr.scopes != wantScopes[i] {
			t.Errorf("token %d scopes = %q, want %q", tr.id, tr.scopes, wantScopes[i])
		}
	}
	if toks[1].revoked == nil {
		t.Error("token with no scope left was not revoked")
	}
	if toks[0].revoked != nil {
		t.Error("token with upload scope was revoked")
	}
	if toks[6].revoked == nil {
		t.Error("already revoked token lost its revoked_at")
	}
	var audited int
	if err := db.w.QueryRowContext(ctx, `SELECT count(*) FROM audit WHERE action = 'token.revoke' AND actor = 'system'`).Scan(&audited); err != nil || audited != 1 {
		t.Fatalf("revocation audit rows: %d %v", audited, err)
	}

	// sessions lost the write-only columns but kept their rows
	var sessionCols int
	if err := db.w.QueryRowContext(ctx, `SELECT count(*) FROM pragma_table_info('sessions') WHERE name IN ('expires_at', 'ip', 'user_agent')`).Scan(&sessionCols); err != nil || sessionCols != 0 {
		t.Fatalf("session columns: %d %v", sessionCols, err)
	}
	var sessions int
	if err := db.w.QueryRowContext(ctx, `SELECT count(*) FROM sessions`).Scan(&sessions); err != nil || sessions != 1 {
		t.Fatalf("sessions preserved: %d %v", sessions, err)
	}
	if _, err := db.GetSession(ctx, "sid"); err != nil {
		t.Fatalf("session row unusable: %v", err)
	}

	// removed settings are gone; the audit prune index exists
	var staleSettings int
	if err := db.w.QueryRowContext(ctx, `SELECT count(*) FROM settings`).Scan(&staleSettings); err != nil || staleSettings != 0 {
		t.Fatalf("stale settings left: %d %v", staleSettings, err)
	}
	if err := db.w.QueryRowContext(ctx, `SELECT count(*) FROM sqlite_schema WHERE name = 'audit_at'`).Scan(&n); err != nil || n != 1 {
		t.Fatalf("audit_at index: %d %v", n, err)
	}
	if problems, err := Integrity(ctx, db.w); err != nil || len(problems) != 0 {
		t.Fatalf("integrity: %v %v", problems, err)
	}

	// the backup is the untouched schema 2 file
	old, err := OpenReadOnly(db.PreMigrationBackup)
	if err != nil {
		t.Fatal(err)
	}
	defer old.Close()
	if err := old.QueryRowContext(ctx, `PRAGMA user_version`).Scan(&uv); err != nil || uv != 2 {
		t.Fatalf("backup user_version = %d (%v), want 2", uv, err)
	}
	if err := old.QueryRowContext(ctx, `SELECT tag FROM image_tags WHERE image_id = 'aaaaaaaaaaaaaaaaaaaa'`).Scan(&table); err != nil || table != "x" {
		t.Fatalf("backup lost the tag: %q %v", table, err)
	}
}

func TestBackupRotationKeepsNewestFive(t *testing.T) {
	dir := t.TempDir()
	base := time.Now().Add(-time.Hour)
	for i := 0; i < 7; i++ {
		name := filepath.Join(dir, "picberth-schema1-2026010"+string(rune('0'+i))+"T000000Z.db")
		if err := os.WriteFile(name, []byte("x"), 0o600); err != nil {
			t.Fatal(err)
		}
		ts := base.Add(time.Duration(i) * time.Minute)
		if err := os.Chtimes(name, ts, ts); err != nil {
			t.Fatal(err)
		}
	}
	other := filepath.Join(dir, "keep-me.txt")
	if err := os.WriteFile(other, []byte("x"), 0o600); err != nil {
		t.Fatal(err)
	}
	rotateBackups(dir, 5)
	entries, err := os.ReadDir(dir)
	if err != nil {
		t.Fatal(err)
	}
	var backups, files int
	for _, e := range entries {
		if e.Name() == "keep-me.txt" {
			files++
			continue
		}
		backups++
		// the two oldest names must be gone
		if e.Name() == "picberth-schema1-20260100T000000Z.db" || e.Name() == "picberth-schema1-20260101T000000Z.db" {
			t.Errorf("old backup %s survived", e.Name())
		}
	}
	if backups != 5 || files != 1 {
		t.Fatalf("backups = %d, other files = %d", backups, files)
	}
}
