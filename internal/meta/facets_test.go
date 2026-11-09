package meta

import (
	"context"
	"fmt"
	"reflect"
	"testing"
)

func insertImage(t *testing.T, db *DB, i int, uid int64, mod func(*Image)) Image {
	t.Helper()
	id := fmt.Sprintf("%020d", i)
	im := sampleImage(id, uid, fmt.Sprintf("sha%02d", i), int64(100+i), int64(10+i))
	if mod != nil {
		mod(&im)
	}
	if dup, err := db.InsertImage(context.Background(), im, nil); err != nil || dup != nil {
		t.Fatalf("insert %d: %v %v", i, dup, err)
	}
	return im
}

func idsOf(rows []Image) []string {
	var out []string
	for _, r := range rows {
		out = append(out, r.ID)
	}
	return out
}

func TestFolderStorage(t *testing.T) {
	db := openTemp(t)
	ctx := context.Background()
	uid, _, _ := db.CreateFirstUser(ctx, "a", "h", 1)

	root, err := db.CreateFolder(ctx, Folder{ID: "root000000000001", OwnerID: uid, Name: "旅行", NameKey: "旅行", Depth: 1, CreatedAt: 10})
	if err != nil {
		t.Fatal(err)
	}
	child, err := db.CreateFolder(ctx, Folder{ID: "child00000000001", OwnerID: uid, ParentID: &root.ID, Name: "2026", NameKey: "2026", Depth: 2, CreatedAt: 11})
	if err != nil {
		t.Fatal(err)
	}
	// the unique sibling index is the backstop for a lost race
	if _, err := db.CreateFolder(ctx, Folder{ID: "dup00000000000001", OwnerID: uid, Name: "旅行", NameKey: "旅行", Depth: 1, CreatedAt: 12}); err != ErrFolderExists {
		t.Fatalf("duplicate sibling: %v", err)
	}
	// same name under another parent is fine
	if _, err := db.CreateFolder(ctx, Folder{ID: "other00000000001", OwnerID: uid, ParentID: &root.ID, Name: "2027", NameKey: "2027", Depth: 2, CreatedAt: 13}); err != nil {
		t.Fatal(err)
	}

	folders, err := db.ListFolders(ctx, &uid)
	if err != nil || len(folders) != 3 {
		t.Fatalf("list: %v %v", folders, err)
	}
	if taken, _ := db.FolderNameTaken(ctx, uid, &root.ID, "旅行", ""); taken {
		t.Fatal("root-level name reported taken among children")
	}
	if taken, _ := db.FolderNameTaken(ctx, uid, nil, "旅行", ""); !taken {
		t.Fatal("existing sibling not detected")
	}
	if taken, _ := db.FolderNameTaken(ctx, uid, nil, "旅行", root.ID); taken {
		t.Fatal("a folder's own name counts as taken")
	}

	// subtree ids and depth
	sub, maxDepth, err := db.FolderSubtree(ctx, root.ID)
	if err != nil || len(sub) != 3 || maxDepth != 2 {
		t.Fatalf("subtree: %v %d %v", sub, maxDepth, err)
	}

	// counts only active images directly in the folder
	insertImage(t, db, 1, uid, func(im *Image) { im.FolderID = &root.ID })
	active := insertImage(t, db, 2, uid, func(im *Image) { im.FolderID = &child.ID })
	trashed := insertImage(t, db, 3, uid, func(im *Image) { im.FolderID = &child.ID })
	db.MarkTrashed(ctx, trashed.ID, 20)
	counts, err := db.FolderActiveCounts(ctx, &uid)
	if err != nil || counts[root.ID] != 1 || counts[child.ID] != 1 {
		t.Fatalf("counts: %v %v", counts, err)
	}
	if n, _ := db.FolderActiveImageCount(ctx, root.ID); n != 1 {
		t.Fatalf("active count: %d", n)
	}

	// moving a subtree shifts every descendant's depth
	name := "2026"
	key := "2026"
	newParent := root.ID
	if _, err := db.UpdateFolder(ctx, child.ID, FolderUpdate{OwnerID: uid, Name: &name, NameKey: &key, Move: true, ParentID: &newParent}); err != nil {
		t.Fatalf("update: %v", err)
	}

	// an active image keeps a delete from succeeding
	if err := db.DeleteFolder(ctx, child.ID, uid); err != ErrFolderNotEmpty {
		t.Fatalf("delete with active image: %v", err)
	}
	db.MarkTrashed(ctx, active.ID, 21)

	// deleting the root would orphan the children, which the caller must prevent; deleting the child
	// returns its trashed image to the root.
	if err := db.DeleteFolder(ctx, child.ID, uid); err != nil {
		t.Fatal(err)
	}
	got, err := db.GetImage(ctx, trashed.ID)
	if err != nil || got.FolderID != nil {
		t.Fatalf("trashed image must return to the root: %+v %v", got, err)
	}
	if _, err := db.GetFolder(ctx, child.ID); err != ErrNotFound {
		t.Fatalf("folder not deleted: %v", err)
	}
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
	root, err := db.CreateFolder(ctx, Folder{ID: "root000000000001", OwnerID: uid, Name: "a", NameKey: "a", Depth: 1, CreatedAt: 1})
	if err != nil {
		t.Fatal(err)
	}
	child, err := db.CreateFolder(ctx, Folder{ID: "child00000000001", OwnerID: uid, ParentID: &root.ID, Name: "b", NameKey: "b", Depth: 2, CreatedAt: 2})
	if err != nil {
		t.Fatal(err)
	}
	i1 := insertImage(t, db, 1, uid, func(im *Image) {
		im.CreatedAt = now
		im.Ext = "png"
		im.FolderID = &root.ID
	})
	i2 := insertImage(t, db, 2, uid, func(im *Image) {
		im.CreatedAt = now + 10
		im.Ext = "jpg"
		im.Source = "cli"
		im.FolderID = &child.ID
	})
	i3 := insertImage(t, db, 3, uid, func(im *Image) {
		tok(1)(im)
		im.CreatedAt = now + 20
		im.Ext = "webp"
	})
	i4 := insertImage(t, db, 4, uid, func(im *Image) {
		tok(2)(im)
		im.CreatedAt = now + 30
		im.Ext = "png"
		im.OriginalName = "a%co.png"
	})

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
		return idsOf(rows)
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
	eq("source web", list(ListQuery{Sources: []SourceFilter{{Kind: "web"}}}), i1)
	eq("source any-of", list(ListQuery{Sources: []SourceFilter{{Kind: "cli"}, {Kind: "token", TokenID: 2}}}), i2, i4)
	eq("source token", list(ListQuery{Sources: []SourceFilter{{Kind: "token", TokenID: 1}}}), i3)
	eq("root only", list(ListQuery{FolderRoot: true}), i3, i4)
	eq("one folder", list(ListQuery{FolderID: &root.ID}), i1)
	eq("subtree", list(ListQuery{FolderIDs: []string{root.ID, child.ID}}), i1, i2)
	since, until := now+10, now+30
	eq("since/until", list(ListQuery{Since: &since, Until: &until}), i2, i3)
	eq("combined AND", list(ListQuery{Exts: []string{"png"}, Sources: []SourceFilter{{Kind: "token", TokenID: 2}}, Since: &since}), i4)
	eq("q filename", list(ListQuery{Q: "PIC 00000000000000000002"}), i2)
	eq("q wildcard escaped", list(ListQuery{Q: "0%"}))
	eq("q wildcard literal", list(ListQuery{Q: "%"}), i4)
	eq("q no match", list(ListQuery{Q: "bird"}))
	// q never searches folder names or anything but name/id/hash
	eq("q does not search folders", list(ListQuery{Q: "child00000000001"}))
}

func TestListNameSortPagination(t *testing.T) {
	db := openTemp(t)
	ctx := context.Background()
	uid, _, _ := db.CreateFirstUser(ctx, "a", "h", 1)
	names := []string{"banana.png", "Apple.png", "cherry.png", "apple.png", "Banana.png", "date.png", "Äpfel.png"}
	for i, n := range names {
		n := n
		insertImage(t, db, i, uid, func(im *Image) { im.OriginalName = n })
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

func TestImagePatchNameAndFolder(t *testing.T) {
	db := openTemp(t)
	ctx := context.Background()
	uid, _, _ := db.CreateFirstUser(ctx, "a", "h", 1)
	folder, err := db.CreateFolder(ctx, Folder{ID: "root000000000001", OwnerID: uid, Name: "f", NameKey: "f", Depth: 1, CreatedAt: 1})
	if err != nil {
		t.Fatal(err)
	}
	a := insertImage(t, db, 1, uid, nil)
	name := "renamed.png"
	if ok, err := db.PatchImage(ctx, a.ID, ImagePatch{Name: &name, FolderID: &folder.ID, MoveFolder: true}); err != nil || !ok {
		t.Fatalf("patch: %v %v", ok, err)
	}
	im, _ := db.GetImage(ctx, a.ID)
	if im.OriginalName != name || im.FolderID == nil || *im.FolderID != folder.ID {
		t.Fatalf("patched: %+v", im)
	}
	if ok, _ := db.PatchImage(ctx, a.ID, ImagePatch{FolderID: nil, MoveFolder: true}); !ok {
		t.Fatal("move to root failed")
	}
	if im, _ := db.GetImage(ctx, a.ID); im.FolderID != nil {
		t.Fatalf("not moved to root: %+v", im)
	}
	if ok, _ := db.PatchImage(ctx, "00000000000000009999", ImagePatch{Name: &name}); ok {
		t.Fatal("patched a missing record")
	}
	// tombstones cannot be patched
	db.MarkTrashed(ctx, a.ID, 5)
	db.MarkPurged(ctx, a.ID, 6)
	if ok, _ := db.PatchImage(ctx, a.ID, ImagePatch{Name: &name}); ok {
		t.Fatal("tombstone accepted a patch")
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
	insertImage(t, db, 1, uid, func(im *Image) { im.Size = 100; im.HasGPS = true })
	insertImage(t, db, 2, uid, func(im *Image) { im.Size = 50; im.Ext = "jpg"; im.Source = "cli" })
	insertImage(t, db, 3, uid, func(im *Image) {
		im.Size = 25
		im.Source = "token"
		seven := int64(7)
		im.TokenID = &seven
	})
	trashed := insertImage(t, db, 4, uid, func(im *Image) { im.Size = 1000 })
	db.MarkTrashed(ctx, trashed.ID, 9)
	insertImage(t, db, 5, oid, func(im *Image) { im.Size = 7 })

	f, err := db.ImageFacets(ctx, FacetQuery{State: "active", OwnerID: &uid})
	if err != nil {
		t.Fatal(err)
	}
	if f.Total != 3 || f.Bytes != 175 {
		t.Fatalf("totals: %+v", f)
	}
	if f.Exts["png"] != 2 || f.Exts["jpg"] != 1 {
		t.Fatalf("exts: %+v", f.Exts)
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

	tf, _ := db.ImageFacets(ctx, FacetQuery{State: "trashed", OwnerID: &uid})
	if tf.Total != 1 || tf.Bytes != 1000 {
		t.Fatalf("trashed facets: %+v", tf)
	}
	seven := int64(7)
	pf, _ := db.ImageFacets(ctx, FacetQuery{State: "active", OwnerID: &uid, TokenID: &seven})
	if pf.Total != 1 || len(pf.Sources) != 1 {
		t.Fatalf("token-scoped facets: %+v", pf)
	}
}
