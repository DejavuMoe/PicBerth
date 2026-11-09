package meta

import (
	"context"
	"database/sql"
	"errors"
	"fmt"
	"os"
	"path/filepath"
	"sync"
	"sync/atomic"
	"testing"
)

func openTemp(t *testing.T) *DB {
	t.Helper()
	dir := filepath.Join(t.TempDir(), "with space")
	if err := os.MkdirAll(dir, 0o700); err != nil {
		t.Fatal(err)
	}
	db, err := Open(context.Background(), filepath.Join(dir, "picberth.db"))
	if err != nil {
		t.Fatal(err)
	}
	t.Cleanup(func() { db.Close() })
	return db
}

func TestOpenMigratesAndSetsIdentity(t *testing.T) {
	db := openTemp(t)
	ctx := context.Background()
	var uv, appID int
	db.w.QueryRowContext(ctx, `PRAGMA user_version`).Scan(&uv)
	db.w.QueryRowContext(ctx, `PRAGMA application_id`).Scan(&appID)
	if uv != SchemaVersion() || uv < 1 {
		t.Fatalf("user_version = %d, SchemaVersion = %d", uv, SchemaVersion())
	}
	if appID != 0x50494342 {
		t.Fatalf("application_id = %#x", appID)
	}
	if compareVersions(db.SQLiteVersion(), MinSQLiteVersion) < 0 {
		t.Fatalf("sqlite %s too old", db.SQLiteVersion())
	}
	t.Logf("SQLite %s, schema %d", db.SQLiteVersion(), uv)
}

func TestPragmasApplyToEveryConnection(t *testing.T) {
	db := openTemp(t)
	ctx := context.Background()
	for name, pool := range map[string]*sql.DB{"writer": db.w, "reader": db.r} {
		// hold several connections open at once so the pool really creates more than one
		var conns []*sql.Conn
		n := 1
		if name == "reader" {
			n = 3
		}
		for i := 0; i < n; i++ {
			c, err := pool.Conn(ctx)
			if err != nil {
				t.Fatal(err)
			}
			conns = append(conns, c)
		}
		for i, c := range conns {
			var fk, sync, bt int
			var jm string
			c.QueryRowContext(ctx, `PRAGMA foreign_keys`).Scan(&fk)
			c.QueryRowContext(ctx, `PRAGMA synchronous`).Scan(&sync)
			c.QueryRowContext(ctx, `PRAGMA busy_timeout`).Scan(&bt)
			c.QueryRowContext(ctx, `PRAGMA journal_mode`).Scan(&jm)
			if fk != 1 || sync != 2 || bt != 5000 || jm != "wal" {
				t.Errorf("%s conn %d: foreign_keys=%d synchronous=%d busy_timeout=%d journal_mode=%s", name, i, fk, sync, bt, jm)
			}
			if name == "reader" {
				if _, err := c.ExecContext(ctx, `INSERT INTO kv(key, value) VALUES ('x', 'y')`); err == nil {
					t.Errorf("reader connection %d accepted a write", i)
				}
			}
		}
		for _, c := range conns {
			c.Close()
		}
	}
}

func TestReopenAndVersionGuard(t *testing.T) {
	path := filepath.Join(t.TempDir(), "p.db")
	ctx := context.Background()
	db, err := Open(ctx, path)
	if err != nil {
		t.Fatal(err)
	}
	if _, err := db.w.ExecContext(ctx, fmt.Sprintf(`PRAGMA user_version = %d`, SchemaVersion()+5)); err != nil {
		t.Fatal(err)
	}
	db.Close()
	if _, err := Open(ctx, path); !errors.Is(err, ErrSchemaTooNew) {
		t.Fatalf("want ErrSchemaTooNew, got %v", err)
	}

	// restore the right version: reopening works and is a no-op
	raw, _ := sql.Open("sqlite", path)
	raw.Exec(fmt.Sprintf(`PRAGMA user_version = %d`, SchemaVersion()))
	raw.Close()
	db, err = Open(ctx, path)
	if err != nil {
		t.Fatal(err)
	}
	db.Close()
}

func TestRefusesForeignDatabase(t *testing.T) {
	path := filepath.Join(t.TempDir(), "other.db")
	raw, _ := sql.Open("sqlite", path)
	if _, err := raw.Exec(`CREATE TABLE foo(x)`); err != nil {
		t.Fatal(err)
	}
	raw.Close()
	if _, err := Open(context.Background(), path); err == nil {
		t.Fatal("opened a non-PicBerth database")
	}
	if _, err := Open(context.Background(), path+"?x=1"); err == nil {
		t.Fatal("accepted a path with '?'")
	}
}

func TestMigrationsAreSequential(t *testing.T) {
	ms, err := loadMigrations()
	if err != nil || len(ms) == 0 {
		t.Fatalf("loadMigrations: %v %v", ms, err)
	}
	for i, m := range ms {
		if m.version != i+1 {
			t.Fatalf("migration %s has version %d at index %d", m.name, m.version, i)
		}
	}
}

func TestCreateFirstUserIsAtomic(t *testing.T) {
	db := openTemp(t)
	ctx := context.Background()
	var created atomic.Int32
	var wg sync.WaitGroup
	for i := 0; i < 20; i++ {
		wg.Add(1)
		go func(i int) {
			defer wg.Done()
			_, ok, err := db.CreateFirstUser(ctx, fmt.Sprintf("admin%d", i), "hash", 1)
			if err != nil {
				t.Error(err)
			}
			if ok {
				created.Add(1)
			}
		}(i)
	}
	wg.Wait()
	if created.Load() != 1 {
		t.Fatalf("%d users created concurrently, want exactly 1", created.Load())
	}
	if n, _ := db.CountUsers(ctx); n != 1 {
		t.Fatalf("user count %d", n)
	}
	if u, err := db.GetUserByName(ctx, "ADMIN0"); err == nil && u == nil {
		t.Fatal("unexpected")
	}
}

func sampleImage(id string, owner int64, sha string, created int64, size int64) Image {
	return Image{ID: id, OwnerID: owner, Source: "web", ObjectKey: id[0:2] + "/" + id[2:4] + "/" + id + ".png",
		OriginalName: "pic " + id + ".png", Mime: "image/png", Ext: "png", Size: size, Width: 2, Height: 2, Frames: 1,
		SHA256: sha, CreatedAt: created}
}

func TestInsertDedupeAndLifecycle(t *testing.T) {
	db := openTemp(t)
	ctx := context.Background()
	uid, _, _ := db.CreateFirstUser(ctx, "a", "h", 1)
	root := DupScope{OwnerID: uid}
	a := sampleImage("aaaaaaaaaaaaaaaaaaaa", uid, "s1", 10, 100)
	if dup, err := db.InsertImage(ctx, a, &root); err != nil || dup != nil {
		t.Fatalf("first insert: %v %v", dup, err)
	}
	b := sampleImage("bbbbbbbbbbbbbbbbbbbb", uid, "s1", 11, 100)
	dup, err := db.InsertImage(ctx, b, &root)
	if err != nil || dup == nil || dup.ID != a.ID {
		t.Fatalf("expected duplicate of a, got %v %v", dup, err)
	}
	if _, err := db.GetImage(ctx, b.ID); !errors.Is(err, ErrNotFound) {
		t.Fatal("duplicate was inserted anyway")
	}
	// dedupe off inserts anyway
	if dup, err := db.InsertImage(ctx, b, nil); err != nil || dup != nil {
		t.Fatalf("insert without dedupe: %v %v", dup, err)
	}
	// duplicate object key violates the unique constraint (I5)
	c := sampleImage("cccccccccccccccccccc", uid, "s2", 12, 5)
	c.ObjectKey = a.ObjectKey
	if _, err := db.InsertImage(ctx, c, nil); err == nil {
		t.Fatal("two records share one object key")
	}

	if ok, _ := db.MarkTrashed(ctx, a.ID, 50); !ok {
		t.Fatal("trash failed")
	}
	if ok, _ := db.MarkTrashed(ctx, a.ID, 51); ok {
		t.Fatal("trashing twice should report no change")
	}
	// after trashing, the hash is no longer a duplicate
	if d, err := db.FindActiveDuplicate(ctx, root, "s1"); err != nil || d.ID != b.ID {
		t.Fatalf("active duplicate after trash: %v %v", d, err)
	}
	if ok, _ := db.MarkPurged(ctx, a.ID, 60); !ok {
		t.Fatal("purge failed")
	}
	got, _ := db.GetImage(ctx, a.ID)
	if got.State != StatePurged || got.PurgedAt == nil || *got.PurgedAt != 60 {
		t.Fatalf("tombstone: %+v", got)
	}
	st, _ := db.ImageStats(ctx)
	if st.Active != 1 || st.Trashed != 0 || st.Bytes != 100 {
		t.Fatalf("stats %+v", st)
	}
}

func TestDupScopeLimitsDedupe(t *testing.T) {
	db := openTemp(t)
	ctx := context.Background()
	uid, _, _ := db.CreateFirstUser(ctx, "a", "h", 1)
	tokA, err := db.CreateToken(ctx, Token{OwnerID: uid, Name: "a", Prefix: "pa", SecretHash: "ha", Scopes: "upload", CreatedAt: 1})
	if err != nil {
		t.Fatal(err)
	}
	tokB, err := db.CreateToken(ctx, Token{OwnerID: uid, Name: "b", Prefix: "pb", SecretHash: "hb", Scopes: "upload", CreatedAt: 2})
	if err != nil {
		t.Fatal(err)
	}
	a, b := tokA.ID, tokB.ID
	scopeA := DupScope{OwnerID: uid, TokenID: &a}
	scopeB := DupScope{OwnerID: uid, TokenID: &b}

	insert := func(im Image, scope *DupScope) *Image {
		t.Helper()
		dup, err := db.InsertImage(ctx, im, scope)
		if err != nil {
			t.Fatalf("insert %s: %v", im.ID, err)
		}
		return dup
	}
	active := func(scope DupScope) (string, error) {
		d, err := db.FindActiveDuplicate(ctx, scope, "shared")
		if err != nil {
			return "", err
		}
		return d.ID, nil
	}

	admin := sampleImage("aaaaaaaaaaaaaaaaaaaa", uid, "shared", 10, 100)
	if dup := insert(admin, nil); dup != nil {
		t.Fatalf("first insert deduped: %+v", dup)
	}

	// The root scope still finds the earliest record.
	if id, err := active(DupScope{OwnerID: uid}); err != nil || id != admin.ID {
		t.Fatalf("root duplicate: %q %v", id, err)
	}

	// A token that cannot see the admin record must not deduplicate against it, in the pre-check
	// or in the INSERT transaction.
	bImg := sampleImage("bbbbbbbbbbbbbbbbbbbb", uid, "shared", 11, 100)
	bImg.TokenID, bImg.Source = &b, "token"
	if dup := insert(bImg, &scopeB); dup != nil {
		t.Fatalf("token B reused an invisible record: %+v", dup)
	}
	bDup := sampleImage("cccccccccccccccccccc", uid, "shared", 12, 100)
	bDup.TokenID, bDup.Source = &b, "token"
	if dup := insert(bDup, &scopeB); dup == nil || dup.ID != bImg.ID {
		t.Fatalf("token B transaction dedupe: %+v", dup)
	}
	if _, err := db.GetImage(ctx, bDup.ID); !errors.Is(err, ErrNotFound) {
		t.Fatal("token B duplicate was inserted")
	}

	if _, err := active(scopeA); !errors.Is(err, ErrNotFound) {
		t.Fatalf("token A saw another scope's record: %v", err)
	}
	aImg := sampleImage("dddddddddddddddddddd", uid, "shared", 13, 100)
	aImg.TokenID, aImg.Source = &a, "token"
	if dup := insert(aImg, &scopeA); dup != nil {
		t.Fatalf("token A reused an invisible record: %+v", dup)
	}
	// Even with earlier matching rows, token A dedupes to its own record.
	if id, err := active(scopeA); err != nil || id != aImg.ID {
		t.Fatalf("token A duplicate: %q %v", id, err)
	}
	aDup := sampleImage("eeeeeeeeeeeeeeeeeeee", uid, "shared", 14, 100)
	aDup.TokenID, aDup.Source = &a, "token"
	if dup := insert(aDup, &scopeA); dup == nil || dup.ID != aImg.ID {
		t.Fatalf("token A transaction dedupe: %+v", dup)
	}
	if id, err := active(DupScope{OwnerID: uid}); err != nil || id != admin.ID {
		t.Fatalf("root duplicate changed: %q %v", id, err)
	}
}

func TestListPagination(t *testing.T) {
	db := openTemp(t)
	ctx := context.Background()
	uid, _, _ := db.CreateFirstUser(ctx, "a", "h", 1)
	for i := 0; i < 7; i++ {
		id := fmt.Sprintf("%020d", i)
		im := sampleImage(id, uid, fmt.Sprintf("sha%02d", i), int64(100+i/2), int64(10-i%3)) // equal keys exercise the id tiebreak
		if _, err := db.InsertImage(ctx, im, nil); err != nil {
			t.Fatal(err)
		}
	}
	for _, sortBy := range []string{"new", "old", "large", "small"} {
		var all []string
		var cur *Cursor
		for page := 0; page < 10; page++ {
			rows, err := db.ListImages(ctx, ListQuery{State: "active", Sort: sortBy, Limit: 3, After: cur})
			if err != nil {
				t.Fatal(err)
			}
			n := len(rows)
			if n > 3 {
				n = 3
			}
			for _, r := range rows[:n] {
				all = append(all, r.ID)
			}
			if len(rows) <= 3 {
				break
			}
			last := rows[2]
			key := last.CreatedAt
			if sortBy == "large" || sortBy == "small" {
				key = last.Size
			}
			cur = &Cursor{Key: key, ID: last.ID}
		}
		seen := map[string]bool{}
		for _, id := range all {
			if seen[id] {
				t.Fatalf("sort %s: duplicate %s across pages", sortBy, id)
			}
			seen[id] = true
		}
		if len(all) != 7 {
			t.Fatalf("sort %s: got %d items, want 7: %v", sortBy, len(all), all)
		}
	}
	rows, _ := db.ListImages(ctx, ListQuery{State: "active", Q: "sha03", Limit: 10})
	if len(rows) != 0 {
		t.Fatalf("hash prefix shorter than 8 must not match: %d", len(rows))
	}
	rows, _ = db.ListImages(ctx, ListQuery{State: "active", Q: "pic 00000000000000000003", Limit: 10})
	if len(rows) != 1 {
		t.Fatalf("filename substring: %d", len(rows))
	}
	rows, _ = db.ListImages(ctx, ListQuery{State: "active", Q: "100%_", Limit: 10})
	if len(rows) != 0 {
		t.Fatalf("LIKE wildcards must be escaped: %d", len(rows))
	}
}

func TestVacuumIntoViaReaderAndOpenReadOnly(t *testing.T) {
	db := openTemp(t)
	ctx := context.Background()
	uid, _, _ := db.CreateFirstUser(ctx, "a", "h", 1)
	db.InsertImage(ctx, sampleImage("aaaaaaaaaaaaaaaaaaaa", uid, "s", 1, 1), nil)
	dest := filepath.Join(t.TempDir(), "snap dir")
	os.MkdirAll(dest, 0o700)
	dest = filepath.Join(dest, "picberth.db")
	if err := db.VacuumInto(ctx, dest); err != nil {
		t.Fatalf("VACUUM INTO via reader: %v", err)
	}
	ro, err := OpenReadOnly(dest)
	if err != nil {
		t.Fatal(err)
	}
	defer ro.Close()
	probs, err := Integrity(ctx, ro)
	if err != nil || len(probs) != 0 {
		t.Fatalf("integrity: %v %v", probs, err)
	}
	var n int
	if err := ForEachLiveImage(ctx, ro, func(r ImageRef) error { n++; return nil }); err != nil || n != 1 {
		t.Fatalf("ForEachLiveImage: %d %v", n, err)
	}
}

func TestSettingsKVAudit(t *testing.T) {
	db := openTemp(t)
	ctx := context.Background()
	if err := db.PutSettings(ctx, map[string]string{"a": "1", "b": `"x"`}); err != nil {
		t.Fatal(err)
	}
	if err := db.PutSettings(ctx, map[string]string{"a": "2"}); err != nil {
		t.Fatal(err)
	}
	m, _ := db.AllSettings(ctx)
	if m["a"] != "2" || m["b"] != `"x"` {
		t.Fatalf("settings %v", m)
	}
	if _, err := db.KVGet(ctx, "nope"); !errors.Is(err, ErrNotFound) {
		t.Fatal(err)
	}
	db.KVSet(ctx, "k", "v1")
	db.KVSet(ctx, "k", "v2")
	if v, _ := db.KVGet(ctx, "k"); v != "v2" {
		t.Fatalf("kv %q", v)
	}
	for i := 0; i < 5; i++ {
		db.AddAudit(ctx, AuditEntry{At: int64(i), Actor: "x", Action: fmt.Sprint("a", i)})
	}
	page1, _ := db.ListAudit(ctx, 0, AuditFilter{}, 2)
	page2, _ := db.ListAudit(ctx, page1[len(page1)-1].ID, AuditFilter{}, 10)
	if len(page1) != 2 || len(page2) != 3 || page1[0].Action != "a4" || page2[2].Action != "a0" {
		t.Fatalf("audit paging: %v %v", page1, page2)
	}
	filtered, _ := db.ListAudit(ctx, 0, AuditFilter{Action: "a2"}, 10)
	if len(filtered) != 1 || filtered[0].Action != "a2" {
		t.Fatalf("audit action filter: %v", filtered)
	}
	actions, _ := db.AuditActions(ctx)
	if len(actions) != 5 || actions[0] != "a0" || actions[4] != "a4" {
		t.Fatalf("audit actions: %v", actions)
	}
	// pruning removes the oldest rows first and only below the cutoff
	n, err := db.DeleteAuditBefore(ctx, 3, 10)
	if err != nil || n != 3 {
		t.Fatalf("prune: %d %v", n, err)
	}
	if left, _ := db.ListAudit(ctx, 0, AuditFilter{}, 10); len(left) != 2 {
		t.Fatalf("rows left after prune: %v", left)
	}
}
