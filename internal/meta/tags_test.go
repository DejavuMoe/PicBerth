package meta

import (
	"context"
	"fmt"
	"path/filepath"
	"reflect"
	"testing"
)

// A database written by the first release (schema 1) gains the tag table, keeps its images and is backed up first.
func TestMigrateSchema1ToTags(t *testing.T) {
	ctx := context.Background()
	path := filepath.Join(t.TempDir(), "picberth.db")
	ms, err := loadMigrations()
	if err != nil {
		t.Fatal(err)
	}
	if len(ms) < 2 {
		t.Fatalf("expected at least 2 migrations, got %d", len(ms))
	}
	db, err := openWith(ctx, path, ms[:1])
	if err != nil {
		t.Fatal(err)
	}
	uid, _, _ := db.CreateFirstUser(ctx, "a", "h", 1)
	if _, err := db.InsertImage(ctx, sampleImage("aaaaaaaaaaaaaaaaaaaa", uid, "s1", 10, 100), false); err != nil {
		t.Fatal(err)
	}
	var n int
	if err := db.w.QueryRowContext(ctx, `SELECT count(*) FROM sqlite_schema WHERE name = 'image_tags'`).Scan(&n); err != nil || n != 0 {
		t.Fatalf("schema 1 must not have image_tags (%d, %v)", n, err)
	}
	db.Close()

	db, err = Open(ctx, path)
	if err != nil {
		t.Fatal(err)
	}
	defer db.Close()
	if db.PreMigrationBackup == "" {
		t.Fatal("upgrade from schema 1 must write a backup")
	}
	var uv int
	if err := db.w.QueryRowContext(ctx, `PRAGMA user_version`).Scan(&uv); err != nil || uv != len(ms) {
		t.Fatalf("user_version = %d (%v), want %d", uv, err, len(ms))
	}
	if im, err := db.GetImage(ctx, "aaaaaaaaaaaaaaaaaaaa"); err != nil || im.SHA256 != "s1" {
		t.Fatalf("existing image lost: %v %v", im, err)
	}
	tags, err := db.TagsFor(ctx, []string{"aaaaaaaaaaaaaaaaaaaa"})
	if err != nil || len(tags["aaaaaaaaaaaaaaaaaaaa"]) != 0 {
		t.Fatalf("upgraded image has tags: %v %v", tags, err)
	}
	if ok, err := db.PatchImage(ctx, "aaaaaaaaaaaaaaaaaaaa", ImagePatch{Tags: &[]string{"x"}}); err != nil || !ok {
		t.Fatalf("tagging after upgrade: %v %v", ok, err)
	}
	// the foreign key is enforced: tags need an image
	if _, err := db.w.ExecContext(ctx, `INSERT INTO image_tags(image_id, tag) VALUES ('zzzzzzzzzzzzzzzzzzzz', 't')`); err == nil {
		t.Fatal("tag for a missing image accepted")
	}
	if problems, err := Integrity(ctx, db.w); err != nil || len(problems) != 0 {
		t.Fatalf("integrity: %v %v", problems, err)
	}
}

func insertTagged(t *testing.T, db *DB, i int, uid int64, mod func(*Image), tags ...string) Image {
	t.Helper()
	id := fmt.Sprintf("%020d", i)
	im := sampleImage(id, uid, fmt.Sprintf("sha%02d", i), int64(100+i), int64(10+i))
	if mod != nil {
		mod(&im)
	}
	if dup, err := db.InsertImageWithTags(context.Background(), im, false, tags); err != nil || dup != nil {
		t.Fatalf("insert %d: %v %v", i, dup, err)
	}
	return im
}

func TestTagsStorageAndPatch(t *testing.T) {
	db := openTemp(t)
	ctx := context.Background()
	uid, _, _ := db.CreateFirstUser(ctx, "a", "h", 1)
	a := insertTagged(t, db, 1, uid, nil, "b", "a")
	b := insertTagged(t, db, 2, uid, nil)

	got, err := db.TagsFor(ctx, []string{a.ID, b.ID, "nope"})
	if err != nil {
		t.Fatal(err)
	}
	if !reflect.DeepEqual(got[a.ID], []string{"a", "b"}) || got[b.ID] == nil || len(got[b.ID]) != 0 || got["nope"] == nil {
		t.Fatalf("TagsFor = %#v", got)
	}

	vis, name, tags := "private", "renamed.png", []string{"z"}
	ok, err := db.PatchImage(ctx, a.ID, ImagePatch{Visibility: &vis, Name: &name, Tags: &tags})
	if err != nil || !ok {
		t.Fatalf("patch: %v %v", ok, err)
	}
	im, _ := db.GetImage(ctx, a.ID)
	if im.Visibility != "private" || im.OriginalName != "renamed.png" {
		t.Fatalf("patched record: %+v", im)
	}
	if got, _ := db.TagsFor(ctx, []string{a.ID}); !reflect.DeepEqual(got[a.ID], []string{"z"}) {
		t.Fatalf("tags not replaced: %v", got)
	}
	empty := []string{}
	if ok, _ := db.PatchImage(ctx, a.ID, ImagePatch{Tags: &empty}); !ok {
		t.Fatal("clearing tags failed")
	}
	if got, _ := db.TagsFor(ctx, []string{a.ID}); len(got[a.ID]) != 0 {
		t.Fatalf("tags not cleared: %v", got)
	}
	if ok, _ := db.PatchImage(ctx, "00000000000000009999", ImagePatch{Name: &name}); ok {
		t.Fatal("patched a missing record")
	}
}

func TestPurgeRemovesTags(t *testing.T) {
	db := openTemp(t)
	ctx := context.Background()
	uid, _, _ := db.CreateFirstUser(ctx, "a", "h", 1)
	a := insertTagged(t, db, 1, uid, nil, "keep")
	b := insertTagged(t, db, 2, uid, nil, "gone")
	db.MarkTrashed(ctx, b.ID, 5)
	if ok, err := db.MarkPurged(ctx, a.ID, 6); err != nil || ok {
		t.Fatalf("purging an active image: %v %v", ok, err)
	}
	if got, _ := db.TagsFor(ctx, []string{a.ID}); len(got[a.ID]) != 1 {
		t.Fatal("failed purge must keep tags")
	}
	if ok, err := db.MarkPurged(ctx, b.ID, 6); err != nil || !ok {
		t.Fatalf("purge: %v %v", ok, err)
	}
	var n int
	db.r.QueryRowContext(ctx, `SELECT count(*) FROM image_tags WHERE image_id = ?`, b.ID).Scan(&n)
	if n != 0 {
		t.Fatalf("%d tag rows left after purge", n)
	}
	if ok, _ := db.PatchImage(ctx, b.ID, ImagePatch{Tags: &[]string{"x"}}); ok {
		t.Fatal("tombstone accepted tags")
	}
}

func ids(rows []Image) []string {
	var out []string
	for _, r := range rows {
		out = append(out, r.ID)
	}
	return out
}

func TestListFilters(t *testing.T) {
	db := openTemp(t)
	ctx := context.Background()
	uid, _, _ := db.CreateFirstUser(ctx, "a", "h", 1)
	now := int64(1_000_000)
	tok := func(id int64) func(*Image) {
		return func(im *Image) { im.Source, im.TokenID = "token", &id }
	}
	// tokens must exist for the foreign key
	for i := 1; i <= 2; i++ {
		if _, err := db.w.ExecContext(ctx, `INSERT INTO tokens(id, owner_id, name, prefix, secret_hash, scopes, created_at) VALUES (?, ?, ?, 'pb_', ?, 'upload', 1)`,
			i, uid, fmt.Sprintf("tok%d", i), fmt.Sprintf("h%d", i)); err != nil {
			t.Fatal(err)
		}
	}
	i1 := insertTagged(t, db, 1, uid, func(im *Image) { im.CreatedAt = now; im.Ext = "png" }, "cat", "red")
	i2 := insertTagged(t, db, 2, uid, func(im *Image) { im.CreatedAt = now + 10; im.Ext = "jpg"; im.Source = "cli" }, "cat")
	i3 := insertTagged(t, db, 3, uid, func(im *Image) { tok(1)(im); im.CreatedAt = now + 20; im.Ext = "webp" }, "dog", "100%")
	i4 := insertTagged(t, db, 4, uid, func(im *Image) { tok(2)(im); im.CreatedAt = now + 30; im.Ext = "png" })

	list := func(q ListQuery) []string {
		t.Helper()
		q.State, q.Limit = "active", 50
		if q.Sort == "" {
			q.Sort = "old"
		}
		rows, err := db.ListImages(ctx, q)
		if err != nil {
			t.Fatal(err)
		}
		return ids(rows)
	}
	eq := func(what string, got []string, want ...Image) {
		t.Helper()
		var w []string
		for _, im := range want {
			w = append(w, im.ID)
		}
		if !reflect.DeepEqual(got, w) {
			t.Fatalf("%s: got %v, want %v", what, got, w)
		}
	}
	eq("multi format", list(ListQuery{Exts: []string{"png", "jpg"}}), i1, i2, i4)
	eq("tag all", list(ListQuery{Tags: []string{"cat", "red"}}), i1)
	eq("tag single", list(ListQuery{Tags: []string{"cat"}}), i1, i2)
	eq("tag none", list(ListQuery{Tags: []string{"cat", "dog"}}))
	eq("source web", list(ListQuery{Sources: []SourceFilter{{Kind: "web"}}}), i1)
	eq("source any-of", list(ListQuery{Sources: []SourceFilter{{Kind: "cli"}, {Kind: "token", TokenID: 2}}}), i2, i4)
	eq("source token", list(ListQuery{Sources: []SourceFilter{{Kind: "token", TokenID: 1}}}), i3)
	since, until := now+10, now+30
	eq("since/until", list(ListQuery{Since: &since, Until: &until}), i2, i3)
	eq("combined AND", list(ListQuery{Exts: []string{"png"}, Sources: []SourceFilter{{Kind: "token", TokenID: 2}}, Since: &since}), i4)
	eq("q matches tag", list(ListQuery{Q: "DO"}), i3)
	eq("q tag wildcard escaped", list(ListQuery{Q: "0%"}), i3)
	eq("q tag wildcard literal", list(ListQuery{Q: "%"}), i3)
	eq("q no match", list(ListQuery{Q: "bird"}))
}

func TestListNameSortPagination(t *testing.T) {
	db := openTemp(t)
	ctx := context.Background()
	uid, _, _ := db.CreateFirstUser(ctx, "a", "h", 1)
	names := []string{"banana.png", "Apple.png", "cherry.png", "apple.png", "Banana.png", "date.png", "Äpfel.png"}
	for i, n := range names {
		n := n
		insertTagged(t, db, i, uid, func(im *Image) { im.OriginalName = n })
	}
	var all []string
	var cur *Cursor
	for page := 0; page < 10; page++ {
		rows, err := db.ListImages(ctx, ListQuery{State: "active", Sort: "name", Limit: 3, After: cur})
		if err != nil {
			t.Fatal(err)
		}
		n := min(len(rows), 3)
		for _, r := range rows[:n] {
			all = append(all, r.OriginalName)
		}
		if len(rows) <= 3 {
			break
		}
		cur = &Cursor{StrKey: rows[2].OriginalName, ID: rows[2].ID}
	}
	// NOCASE folds ASCII only; equal names fall back to id order ("Apple" id 1 before "apple" id 3).
	want := []string{"Apple.png", "apple.png", "banana.png", "Banana.png", "cherry.png", "date.png", "Äpfel.png"}
	if !reflect.DeepEqual(all, want) {
		t.Fatalf("name order across pages: %v, want %v", all, want)
	}
}

func TestImageFacets(t *testing.T) {
	db := openTemp(t)
	ctx := context.Background()
	uid, _, _ := db.CreateFirstUser(ctx, "a", "h", 1)
	other, _ := db.w.ExecContext(ctx, `INSERT INTO users(username, password_hash, created_at) VALUES ('b', 'h', 1)`)
	oid, _ := other.LastInsertId()
	if _, err := db.w.ExecContext(ctx, `INSERT INTO tokens(id, owner_id, name, prefix, secret_hash, scopes, created_at, revoked_at) VALUES (7, ?, 'PicGo · 笔记本', 'pb_', 'h7', 'upload', 1, 5)`, uid); err != nil {
		t.Fatal(err)
	}
	insertTagged(t, db, 1, uid, func(im *Image) { im.Size = 100; im.HasGPS = true }, "a", "b")
	insertTagged(t, db, 2, uid, func(im *Image) { im.Size = 50; im.Visibility = "private"; im.Ext = "jpg"; im.Source = "cli" }, "b")
	insertTagged(t, db, 3, uid, func(im *Image) {
		im.Size = 25
		im.Source = "token"
		seven := int64(7)
		im.TokenID = &seven
	}, "b", "c")
	trashed := insertTagged(t, db, 4, uid, func(im *Image) { im.Size = 1000 }, "t")
	db.MarkTrashed(ctx, trashed.ID, 9)
	insertTagged(t, db, 5, oid, func(im *Image) { im.Size = 7 }, "b")

	f, err := db.ImageFacets(ctx, FacetQuery{State: "active", OwnerID: &uid})
	if err != nil {
		t.Fatal(err)
	}
	if f.Total != 3 || f.Bytes != 175 {
		t.Fatalf("totals: %+v", f)
	}
	if f.Exts["png"] != 2 || f.Exts["jpg"] != 1 || f.Visibility["public"] != 2 || f.Visibility["private"] != 1 {
		t.Fatalf("exts/visibility: %+v", f)
	}
	srcs := map[string]SourceCount{}
	for _, s := range f.Sources {
		key := s.Source
		if s.TokenID != nil {
			key += fmt.Sprint(":", *s.TokenID)
		}
		srcs[key] = s
	}
	if len(srcs) != 3 || srcs["web"].Count != 1 || srcs["cli"].Count != 1 || srcs["token:7"].Count != 1 || srcs["token:7"].TokenName != "PicGo · 笔记本" {
		t.Fatalf("sources: %+v", f.Sources)
	}
	wantTags := []TagCount{{"b", 3}, {"a", 1}, {"c", 1}}
	if !reflect.DeepEqual(f.Tags, wantTags) {
		t.Fatalf("tags: %v, want %v", f.Tags, wantTags)
	}

	tf, _ := db.ImageFacets(ctx, FacetQuery{State: "trashed", OwnerID: &uid})
	if tf.Total != 1 || tf.Bytes != 1000 || len(tf.Tags) != 1 || tf.Tags[0].Tag != "t" {
		t.Fatalf("trashed facets: %+v", tf)
	}
	seven := int64(7)
	pf, _ := db.ImageFacets(ctx, FacetQuery{State: "active", OwnerID: &uid, TokenID: &seven})
	if pf.Total != 1 || len(pf.Sources) != 1 {
		t.Fatalf("token-scoped facets: %+v", pf)
	}
}
