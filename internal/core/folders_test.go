package core

import (
	"context"
	"fmt"
	"reflect"
	"slices"
	"sort"
	"strings"
	"sync"
	"testing"
	"time"

	"picberth/internal/meta"
)

func (e *env) auditActions() []string {
	e.t.Helper()
	items, _, err := e.svc.ListAudit(context.Background(), e.admin, AuditQuery{}, 100, "")
	if err != nil {
		e.t.Fatal(err)
	}
	var out []string
	for i := len(items) - 1; i >= 0; i-- {
		out = append(out, items[i].Action+"|"+items[i].Target+"|"+items[i].Detail)
	}
	return out
}

func (e *env) folder(p Principal, name string, parentID *string) FolderView {
	e.t.Helper()
	f, err := e.svc.CreateFolder(context.Background(), p, name, parentID)
	if err != nil {
		e.t.Fatalf("create folder %q: %v", name, err)
	}
	return f
}

func TestFolderRules(t *testing.T) {
	e := newEnv(t)
	ctx := context.Background()
	admin := e.admin

	a := e.folder(admin, "旅行", nil)
	if a.Depth != 1 || a.ParentID != nil || a.ImageCount != 0 || a.Name != "旅行" {
		t.Fatalf("root folder: %+v", a)
	}
	b := e.folder(admin, "2026", &a.ID)
	c := e.folder(admin, "云南", &b.ID)
	if b.Depth != 2 || c.Depth != 3 {
		t.Fatalf("depths: %+v %+v", b, c)
	}
	// the name is unique among siblings, case-insensitively
	if _, err := e.svc.CreateFolder(ctx, admin, " 旅行 ", nil); ErrorCode(err) != CodeFolderExists {
		t.Fatalf("duplicate sibling: %v", err)
	}
	d := e.folder(admin, "2027", &a.ID)
	if d.Depth != 2 {
		t.Fatalf("sibling depth: %+v", d)
	}
	// the fourth level cannot be created
	_, err := e.svc.CreateFolder(ctx, admin, "第四层", &c.ID)
	wantCode(t, err, CodeFolderDepth)

	// names
	for name, bad := range map[string]string{
		"empty": "  ", "too long": strings.Repeat("字", 65), "slash": "a/b", "backslash": `a\b`,
		"control": "a\x00b", "newline": "a\nb", "utf8": "a\xffb",
	} {
		if _, err := e.svc.CreateFolder(ctx, admin, bad, nil); err == nil {
			t.Errorf("%s accepted", name)
			continue
		} else {
			wantCode(t, err, CodeValidation)
		}
	}
	if _, err := e.svc.CreateFolder(ctx, admin, strings.Repeat("字", 64), nil); err != nil {
		t.Fatalf("64 runes must be accepted: %v", err)
	}

	// create/patch/delete require manage; upload-only tokens may only list
	up := e.tokenPrincipal(ScopeUpload)
	if _, err := e.svc.ListFolders(ctx, up); err != nil {
		t.Fatalf("upload token list: %v", err)
	}
	_, err = e.svc.CreateFolder(ctx, up, "x", nil)
	wantCode(t, err, CodeForbidden)
	_, err = e.svc.UpdateFolder(ctx, up, a.ID, FolderPatch{Name: ptr("y")})
	wantCode(t, err, CodeForbidden)
	wantCode(t, e.svc.DeleteFolder(ctx, up, a.ID), CodeForbidden)
	_, err = e.svc.ListFolders(ctx, Principal{})
	wantCode(t, err, CodeUnauthorized)

	// a folder of another owner is invisible
	other := e.admin
	other.UserID = admin.UserID + 99
	_, err = e.svc.UpdateFolder(ctx, other, a.ID, FolderPatch{Name: ptr("y")})
	wantCode(t, err, CodeNotFound)
	_, err = e.svc.CreateFolder(ctx, admin, "x", ptr("0000000000000000"))
	wantCode(t, err, CodeNotFound) // unknown parent

	// rename and move
	got, err := e.svc.UpdateFolder(ctx, admin, d.ID, FolderPatch{Name: ptr("2028")})
	if err != nil || got.Name != "2028" || got.Depth != 2 {
		t.Fatalf("rename: %+v %v", got, err)
	}
	// a non-cyclic move beyond three levels is rejected
	d1 := e.folder(admin, "d1", nil)
	d2 := e.folder(admin, "d2", &d1.ID)
	if _, err := e.svc.UpdateFolder(ctx, admin, a.ID, FolderPatch{ParentID: &d2.ID, Move: true}); ErrorCode(err) != CodeFolderDepth {
		t.Fatalf("move beyond depth: %v", err)
	}
	// a cycle (a into its own grandchild c) is a plain validation error
	if _, err := e.svc.UpdateFolder(ctx, admin, a.ID, FolderPatch{ParentID: &c.ID, Move: true}); err == nil {
		t.Fatal("cycle accepted")
	} else {
		wantCode(t, err, CodeValidation)
	}
	if _, err := e.svc.UpdateFolder(ctx, admin, b.ID, FolderPatch{ParentID: &b.ID, Move: true}); err == nil {
		t.Fatal("self-parent accepted")
	}
	if _, err := e.svc.UpdateFolder(ctx, admin, b.ID, FolderPatch{ParentID: nil, Move: true}); err != nil {
		t.Fatalf("move to root: %v", err)
	}
	folders, err := e.svc.ListFolders(ctx, admin)
	if err != nil {
		t.Fatal(err)
	}
	byID := map[string]FolderView{}
	for _, f := range folders {
		byID[f.ID] = f
	}
	if byID[a.ID].Depth != 1 || byID[b.ID].Depth != 1 || byID[c.ID].Depth != 2 || byID[d.ID].Depth != 2 {
		t.Fatalf("depths after move: %+v", byID)
	}
	// a same-name conflict in the target is refused, case-insensitively
	sib := e.folder(admin, "Sib", &a.ID)
	if _, err := e.svc.UpdateFolder(ctx, admin, d.ID, FolderPatch{Name: ptr("sib")}); ErrorCode(err) != CodeFolderExists {
		t.Fatalf("rename conflict: %v", err)
	}
	_ = sib
	if _, err := e.svc.UpdateFolder(ctx, admin, d.ID, FolderPatch{}); err == nil {
		t.Fatal("empty patch accepted")
	}

	// delete: blocked while it has subfolders or active images
	wantCode(t, e.svc.DeleteFolder(ctx, admin, a.ID), CodeFolderNotEmpty) // has children
	img := e.mustUpload(admin, "x.png", makePNG(t, 5, 5, 900))
	if _, err := e.svc.PatchImage(ctx, admin, img.ID, ImagePatch{FolderID: &d.ID, MoveFolder: true}); err != nil {
		t.Fatal(err)
	}
	wantCode(t, e.svc.DeleteFolder(ctx, admin, d.ID), CodeFolderNotEmpty)
	// trashed images do not block; they return to the root on delete
	if err := e.svc.Trash(ctx, admin, img.ID); err != nil {
		t.Fatal(err)
	}
	if err := e.svc.DeleteFolder(ctx, admin, d.ID); err != nil {
		t.Fatalf("delete with trashed image: %v", err)
	}
	stored, _ := e.db.GetImage(ctx, img.ID)
	if stored.FolderID != nil {
		t.Fatalf("trashed image still in a deleted folder: %+v", stored)
	}
	if err := e.svc.DeleteFolder(ctx, admin, d.ID); ErrorCode(err) != CodeNotFound {
		t.Fatalf("deleting twice: %v", err)
	}
	// purge clears the folder
	active := e.mustUpload(admin, "y.png", makePNG(t, 5, 5, 901))
	moved, err := e.svc.PatchImage(ctx, admin, active.ID, ImagePatch{FolderID: &b.ID, MoveFolder: true})
	if err != nil || moved.FolderID == nil || *moved.FolderID != b.ID {
		t.Fatalf("move: %+v %v", moved, err)
	}
	if err := e.svc.Trash(ctx, admin, active.ID); err != nil {
		t.Fatal(err)
	}
	if err := e.svc.Purge(ctx, admin, active.ID); err != nil {
		t.Fatal(err)
	}
	if stored, _ := e.db.GetImage(ctx, active.ID); stored.FolderID != nil {
		t.Fatalf("purge kept a folder reference: %+v", stored)
	}

	// folder imageCount counts active direct images only
	if err := e.svc.DeleteFolder(ctx, admin, c.ID); err != nil { // c has no images
		t.Fatalf("delete empty leaf: %v", err)
	}
}

func TestFolderListShape(t *testing.T) {
	e := newEnv(t)
	ctx := context.Background()
	f := e.folder(e.admin, "a", nil)
	e.mustUpload(e.admin, "one.png", makePNG(t, 5, 5, 910))
	img := e.mustUpload(e.admin, "two.png", makePNG(t, 5, 5, 911))
	if _, err := e.svc.PatchImage(ctx, e.admin, img.ID, ImagePatch{FolderID: &f.ID, MoveFolder: true}); err != nil {
		t.Fatal(err)
	}
	items, err := e.svc.ListFolders(ctx, e.admin)
	if err != nil || len(items) != 1 || items[0].ImageCount != 1 || items[0].ParentID != nil {
		t.Fatalf("folders: %+v %v", items, err)
	}
}

func TestUploadToFolderAndPerFolderDedupe(t *testing.T) {
	e := newEnv(t)
	now := time.Now()
	e.svc.Now = func() time.Time { return now }
	ctx := context.Background()
	data := makePNG(t, 10, 10, 920)

	root := e.mustUpload(e.admin, "root.png", data)
	if root.FolderID != nil || root.Duplicate {
		t.Fatalf("root upload: %+v", root)
	}
	now = now.Add(time.Millisecond) // Keep the root copy strictly older for dedupe ordering.
	f := e.folder(e.admin, "f", nil)

	// the same bytes in another folder are a new image
	u, err := e.svc.BeginUpload(ctx, e.admin)
	if err != nil {
		t.Fatal(err)
	}
	defer u.Close()
	u.SetFolder(f.ID)
	if err := u.WriteFile(ctx, strings.NewReader(string(data)), "in-folder.png"); err != nil {
		t.Fatal(err)
	}
	r, err := u.Commit(ctx)
	if err != nil || r.Duplicate || r.Image.FolderID == nil || *r.Image.FolderID != f.ID {
		t.Fatalf("folder upload: %+v %v", r, err)
	}
	// inside the same folder the upload deduplicates
	u2, err := e.svc.BeginUpload(ctx, e.admin)
	if err != nil {
		t.Fatal(err)
	}
	defer u2.Close()
	u2.SetFolder(f.ID)
	if err := u2.WriteFile(ctx, strings.NewReader(string(data)), "again.png"); err != nil {
		t.Fatal(err)
	}
	d, err := u2.Commit(ctx)
	if err != nil || !d.Duplicate || d.Image.ID != r.Image.ID {
		t.Fatalf("folder dedupe: %+v %v", d, err)
	}
	// the root still deduplicates against the root copy only
	again, err := e.upload(e.admin, "root-again.png", data)
	if err != nil || !again.Duplicate || again.Image.ID != root.ID {
		t.Fatalf("root dedupe: %+v %v", again, err)
	}

	// an upload-only token may upload into a folder and deduplicates against its own uploads there
	up := e.tokenPrincipal(ScopeUpload)
	u3, _ := e.svc.BeginUpload(ctx, up)
	defer u3.Close()
	u3.SetFolder(f.ID)
	if err := u3.WriteFile(ctx, strings.NewReader(string(makePNG(t, 10, 10, 921))), "t.png"); err != nil {
		t.Fatal(err)
	}
	tr, err := u3.Commit(ctx)
	if err != nil || tr.Image.FolderID == nil || *tr.Image.FolderID != f.ID {
		t.Fatalf("upload-only token folder upload: %+v %v", tr, err)
	}
	// an unknown folder is a 404, and folders are never created implicitly
	u4, _ := e.svc.BeginUpload(ctx, up)
	u4.SetFolder("0000000000000000")
	if err := u4.WriteFile(ctx, strings.NewReader(string(makePNG(t, 10, 10, 922))), "t2.png"); err != nil {
		t.Fatal(err)
	}
	_, err = u4.Commit(ctx)
	wantCode(t, err, CodeNotFound)
	u2.Close()
	u3.Close()
	u4.Close()
	if n := len(e.tmpFiles()); n != 0 {
		t.Fatalf("%d temp files left", n)
	}

	// moving the image changes the dedupe bucket
	if _, err := e.svc.PatchImage(ctx, e.admin, r.Image.ID, ImagePatch{FolderID: nil, MoveFolder: true}); err != nil {
		t.Fatal(err)
	}
	if m, err := e.upload(e.admin, "moved.png", data); err != nil || !m.Duplicate || m.Image.ID != root.ID {
		t.Fatalf("dedupe after move: %+v %v", m, err)
	}
}

func TestListFolderScopes(t *testing.T) {
	e := newEnv(t)
	ctx := context.Background()
	a := e.folder(e.admin, "a", nil)
	b := e.folder(e.admin, "b", &a.ID)
	rootImg := e.mustUpload(e.admin, "root.png", makePNG(t, 5, 5, 930))
	aImg := e.mustUpload(e.admin, "a.png", makePNG(t, 5, 5, 931))
	bImg := e.mustUpload(e.admin, "b.png", makePNG(t, 5, 5, 932))
	for id, fid := range map[string]*string{aImg.ID: &a.ID, bImg.ID: &b.ID} {
		if _, err := e.svc.PatchImage(ctx, e.admin, id, ImagePatch{FolderID: fid, MoveFolder: true}); err != nil {
			t.Fatal(err)
		}
	}
	list := func(lp ListParams) []string {
		t.Helper()
		items, _, err := e.svc.ListImages(ctx, e.admin, lp)
		if err != nil {
			t.Fatalf("%+v: %v", lp, err)
		}
		var ids []string
		for _, it := range items {
			ids = append(ids, it.ID)
		}
		return ids
	}
	eq := func(name string, got []string, want ...string) {
		t.Helper()
		g := append([]string(nil), got...)
		w := append([]string(nil), want...)
		sort.Strings(g)
		sort.Strings(w)
		if !reflect.DeepEqual(g, w) {
			t.Fatalf("%s: got %v, want %v", name, got, want)
		}
	}
	eq("no folder = all", list(ListParams{}), rootImg.ID, aImg.ID, bImg.ID)
	eq("root", list(ListParams{Folder: "root"}), rootImg.ID)
	eq("root recursive", list(ListParams{Folder: "root", Recursive: true}), rootImg.ID, aImg.ID, bImg.ID)
	eq("direct", list(ListParams{Folder: a.ID}), aImg.ID)
	eq("recursive", list(ListParams{Folder: a.ID, Recursive: true}), aImg.ID, bImg.ID)
	if _, _, err := e.svc.ListImages(ctx, e.admin, ListParams{Folder: "0000000000000000"}); ErrorCode(err) != CodeNotFound {
		t.Fatalf("unknown folder: %v", err)
	}
	// the cursor is bound to the folder scope
	items, next, err := e.svc.ListImages(ctx, e.admin, ListParams{Folder: a.ID, Recursive: true, Limit: 1})
	if err != nil || len(items) != 1 || next == "" {
		t.Fatalf("page: %v %q %v", items, next, err)
	}
	if _, _, err := e.svc.ListImages(ctx, e.admin, ListParams{Folder: b.ID, Limit: 1, Cursor: next}); err == nil {
		t.Fatal("cursor accepted across folder scopes")
	}
	if _, _, err := e.svc.ListImages(ctx, e.admin, ListParams{Folder: a.ID, Recursive: true, Limit: 1, Cursor: next}); err != nil {
		t.Fatalf("cursor bound incorrectly: %v", err)
	}

	// upload-only tokens see only their own uploads in a folder, but may scope the list by it
	up := e.tokenPrincipal(ScopeUpload)
	own := e.mustUpload(up, "own.png", makePNG(t, 5, 5, 933))
	if _, err := e.svc.PatchImage(ctx, e.admin, own.ID, ImagePatch{FolderID: &b.ID, MoveFolder: true}); err != nil {
		t.Fatal(err)
	}
	items, _, err = e.svc.ListImages(ctx, up, ListParams{Folder: b.ID})
	if err != nil || len(items) != 1 || items[0].ID != own.ID {
		t.Fatalf("upload token folder list: %v %v", items, err)
	}
	// an unknown folder id is 404 for the upload-only token too
	if _, _, err := e.svc.ListImages(ctx, up, ListParams{Folder: "zzzzzzzzzzzzzzzz"}); ErrorCode(err) != CodeNotFound {
		t.Fatalf("invalid folder id: %v", err)
	}
}

func TestFolderAuditActions(t *testing.T) {
	e := newEnv(t)
	ctx := context.Background()
	f := e.folder(e.admin, "a", nil)
	if _, err := e.svc.UpdateFolder(ctx, e.admin, f.ID, FolderPatch{Name: ptr("b")}); err != nil {
		t.Fatal(err)
	}
	if err := e.svc.DeleteFolder(ctx, e.admin, f.ID); err != nil {
		t.Fatal(err)
	}
	items, _, err := e.svc.ListAudit(ctx, e.admin, AuditQuery{}, 100, "")
	if err != nil {
		t.Fatal(err)
	}
	seen := map[string]string{}
	for _, it := range items {
		seen[it.Action] = it.Detail
	}
	if seen["folder.create"] != "a" || seen["folder.rename"] != "b" || seen["folder.delete"] != "b" {
		t.Fatalf("folder audit: %+v", seen)
	}
}

func TestImagePatchViaService(t *testing.T) {
	e := newEnv(t)
	ctx := context.Background()
	img := e.mustUpload(e.admin, "Shot [1].png", makePNG(t, 5, 5, 940))
	f := e.folder(e.admin, "f", nil)

	v, err := e.svc.PatchImage(ctx, e.admin, img.ID, ImagePatch{Filename: ptr("  Renamed *x*.png "), FolderID: &f.ID, MoveFolder: true})
	if err != nil {
		t.Fatal(err)
	}
	if v.Filename != "Renamed *x*.png" || v.FolderID == nil || *v.FolderID != f.ID {
		t.Fatalf("patch: %+v", v)
	}
	if v.ID != img.ID || v.URL != img.URL || !strings.Contains(v.Links.Markdown, `![Renamed \*x\*](`) || !strings.Contains(v.Links.HTML, `alt="Renamed *x*"`) {
		t.Fatalf("id/url must stay, alt text follows the name: %+v", v)
	}
	got, _ := e.svc.GetImage(ctx, e.admin, img.ID)
	if got.Filename != "Renamed *x*.png" || got.FolderID == nil || *got.FolderID != f.ID {
		t.Fatalf("not persisted: %+v", got)
	}
	// moving to the root with folderId null
	v, err = e.svc.PatchImage(ctx, e.admin, img.ID, ImagePatch{FolderID: nil, MoveFolder: true})
	if err != nil || v.FolderID != nil || v.Filename != "Renamed *x*.png" {
		t.Fatalf("move to root: %+v %v", v, err)
	}
	// no-op patch writes no audit row
	before := len(e.auditActions())
	if _, err := e.svc.PatchImage(ctx, e.admin, img.ID, ImagePatch{Filename: ptr("Renamed *x*.png")}); err != nil {
		t.Fatal(err)
	}
	if len(e.auditActions()) != before {
		t.Fatal("no-op patch wrote an audit row")
	}
	// validation and authorization
	for name, bad := range map[string]ImagePatch{
		"empty":  {},
		"name":   {Filename: ptr("a/b")},
		"folder": {FolderID: ptr("0000000000000000"), MoveFolder: true},
	} {
		_, err := e.svc.PatchImage(ctx, e.admin, img.ID, bad)
		if err == nil {
			t.Fatalf("%s accepted", name)
		}
		if name == "folder" {
			wantCode(t, err, CodeNotFound)
		} else {
			wantCode(t, err, CodeValidation)
		}
	}
	up, mg := e.tokenPrincipal(ScopeUpload), e.tokenPrincipal(ScopeManage)
	_, err = e.svc.PatchImage(ctx, up, img.ID, ImagePatch{Filename: ptr("x.png")})
	wantCode(t, err, CodeForbidden)
	_, err = e.svc.PatchImage(ctx, Principal{}, img.ID, ImagePatch{Filename: ptr("x.png")})
	wantCode(t, err, CodeUnauthorized)
	if _, err := e.svc.PatchImage(ctx, mg, img.ID, ImagePatch{FolderID: &f.ID, MoveFolder: true}); err != nil {
		t.Fatalf("manage token patch: %v", err)
	}
	// trashed images can still be moved; purged ones cannot
	if err := e.svc.Trash(ctx, e.admin, img.ID); err != nil {
		t.Fatal(err)
	}
	if _, err := e.svc.PatchImage(ctx, e.admin, img.ID, ImagePatch{FolderID: nil, MoveFolder: true}); err != nil {
		t.Fatalf("patch trashed: %v", err)
	}
	if err := e.svc.Purge(ctx, e.admin, img.ID); err != nil {
		t.Fatal(err)
	}
	_, err = e.svc.PatchImage(ctx, e.admin, img.ID, ImagePatch{Filename: ptr("x.png")})
	wantCode(t, err, CodeNotFound)
}

func TestNormalizeFilename(t *testing.T) {
	if got, err := normalizeFilename("  新 名字.png "); err != nil || got != "新 名字.png" {
		t.Fatalf("%q %v", got, err)
	}
	if _, err := normalizeFilename(strings.Repeat("字", 200)); err != nil {
		t.Fatalf("200 runes: %v", err)
	}
	for name, bad := range map[string]string{
		"empty": "", "blank": "  ", "long": strings.Repeat("字", 201), "slash": "a/b.png", "backslash": `a\b.png`,
		"control": "a\x07b", "newline": "a\nb", "utf8": "a\xffb",
	} {
		if _, err := normalizeFilename(bad); err == nil {
			t.Errorf("%s accepted", name)
		} else {
			wantCode(t, err, CodeValidation)
		}
	}
}

func TestDedupeScopeIsPerFolderInMeta(t *testing.T) {
	e := newEnv(t)
	ctx := context.Background()
	_ = ctx
	f := e.folder(e.admin, "f", nil)
	data := makePNG(t, 6, 6, 950)
	inFolder := e.mustUpload(e.admin, "a.png", data)
	if _, err := e.svc.PatchImage(ctx, e.admin, inFolder.ID, ImagePatch{FolderID: &f.ID, MoveFolder: true}); err != nil {
		t.Fatal(err)
	}
	scope := meta.DupScope{OwnerID: e.admin.UserID, FolderID: &f.ID}
	if d, err := e.db.FindActiveDuplicate(ctx, scope, inFolder.SHA256); err != nil || d.ID != inFolder.ID {
		t.Fatalf("folder duplicate: %v %v", d, err)
	}
	rootScope := meta.DupScope{OwnerID: e.admin.UserID}
	if _, err := e.db.FindActiveDuplicate(ctx, rootScope, inFolder.SHA256); err == nil {
		t.Fatal("root scope found a folder record")
	}
}

func TestAuditPruneAndActions(t *testing.T) {
	e := newEnv(t)
	ctx := context.Background()
	now := e.svc.now()
	for i := 0; i < 5; i++ {
		e.svc.auditActor(ctx, "user:admin", "", "test.action", fmt.Sprint(i), "")
	}
	e.svc.Now = func() time.Time { return now.Add(-200 * 24 * time.Hour) }
	e.svc.auditActor(ctx, "user:admin", "", "test.old", "", "")
	e.svc.Now = func() time.Time { return now }
	if n, err := e.svc.PruneExpiredAudit(ctx); err != nil || n != 1 {
		t.Fatalf("prune: %d %v", n, err)
	}
	items, _, err := e.svc.ListAudit(ctx, e.admin, AuditQuery{Action: "test.action"}, 100, "")
	if err != nil || len(items) != 5 {
		t.Fatalf("filtered audit: %v %v", items, err)
	}
	actions, err := e.svc.ListAuditActions(ctx, e.admin)
	if err != nil {
		t.Fatal(err)
	}
	foundAction := false
	for _, a := range actions {
		if a == "test.action" {
			foundAction = true
		}
	}
	if !foundAction {
		t.Fatalf("actions: %v %v", actions, err)
	}
	// default page size is 10 and the maximum 100
	if items, _, err := e.svc.ListAudit(ctx, e.admin, AuditQuery{Action: "test.action"}, 0, ""); err != nil || len(items) != 5 {
		t.Fatalf("default limit: %v %v", items, err)
	}
	if _, _, err := e.svc.ListAudit(ctx, e.admin, AuditQuery{}, 101, ""); err == nil {
		t.Fatal("limit 101 accepted")
	}
}

func TestAuditKindFilter(t *testing.T) {
	e := newEnv(t)
	ctx := context.Background()
	for _, a := range []string{"image.trash", "imagex", "folder.create", "token.revoke", "settings.update",
		"check.quick", "snapshot", "admin.create", "login.success", "logout", "setup"} {
		e.svc.auditActor(ctx, "user:admin", "", a, "", "")
	}
	want := map[string][]string{
		"image":    {"image.trash"},
		"folder":   {"folder.create"},
		"token":    {"token.revoke"},
		"settings": {"settings.update"},
		"system":   {"admin.create", "snapshot", "check.quick", "admin.create"}, // newEnv creates the admin from the CLI
		"session":  {"setup", "logout", "login.success"},
	}
	for kind, actions := range want {
		items, _, err := e.svc.ListAudit(ctx, e.admin, AuditQuery{Kind: kind}, 100, "")
		if err != nil {
			t.Fatalf("%s: %v", kind, err)
		}
		var got []string
		for _, it := range items {
			got = append(got, it.Action)
		}
		if !slices.Equal(got, actions) {
			t.Fatalf("kind %s: got %v want %v", kind, got, actions)
		}
	}
	if items, _, err := e.svc.ListAudit(ctx, e.admin, AuditQuery{Kind: "image", Action: "folder.create"}, 100, ""); err != nil || len(items) != 0 {
		t.Fatalf("kind and action combine with AND: %v %v", items, err)
	}
	_, _, err := e.svc.ListAudit(ctx, e.admin, AuditQuery{Kind: "tag"}, 100, "")
	wantCode(t, err, CodeValidation)
}

// assertFolderTreeValid checks the invariants a folder tree must keep under concurrent writes: no
// cycle, every parent exists, depth agrees with the parent chain, and no folder is deeper than 3.
func assertFolderTreeValid(t *testing.T, folders []FolderView) {
	t.Helper()
	byID := make(map[string]FolderView, len(folders))
	for _, f := range folders {
		byID[f.ID] = f
	}
	for _, f := range folders {
		depth := 1
		seen := map[string]bool{f.ID: true}
		cur := f
		for cur.ParentID != nil {
			p, ok := byID[*cur.ParentID]
			if !ok {
				t.Fatalf("folder %s references missing parent %s", f.ID, *cur.ParentID)
			}
			if seen[p.ID] {
				t.Fatalf("folder cycle at %s", f.ID)
			}
			seen[p.ID] = true
			depth++
			cur = p
		}
		if depth != f.Depth {
			t.Fatalf("folder %s has depth %d, parent chain says %d", f.ID, f.Depth, depth)
		}
		if depth > 3 {
			t.Fatalf("folder %s is %d levels deep", f.ID, depth)
		}
	}
}

func (e *env) uploadToFolder(p Principal, name string, data []byte, folderID string) (UploadResult, error) {
	e.t.Helper()
	ctx := context.Background()
	u, err := e.svc.BeginUpload(ctx, p)
	if err != nil {
		return UploadResult{}, err
	}
	defer u.Close()
	u.SetFolder(folderID)
	if err := u.WriteFile(ctx, strings.NewReader(string(data)), name); err != nil {
		return UploadResult{}, err
	}
	return u.Commit(ctx)
}

// TestConcurrentFolderMovesStayValid races moves that are individually valid but jointly would create
// a cycle or a fourth level. The serialized write transactions must reject the loser.
func TestConcurrentFolderMovesStayValid(t *testing.T) {
	e := newEnv(t)
	ctx := context.Background()
	admin := e.admin

	// A under B and B under A: at most one can win; the second must see a cycle.
	for i := 0; i < 20; i++ {
		a := e.folder(admin, fmt.Sprintf("cycle-a-%d", i), nil)
		b := e.folder(admin, fmt.Sprintf("cycle-b-%d", i), nil)
		start := make(chan struct{})
		var wg sync.WaitGroup
		wg.Add(2)
		go func() {
			defer wg.Done()
			<-start
			_, _ = e.svc.UpdateFolder(ctx, admin, a.ID, FolderPatch{ParentID: &b.ID, Move: true})
		}()
		go func() {
			defer wg.Done()
			<-start
			_, _ = e.svc.UpdateFolder(ctx, admin, b.ID, FolderPatch{ParentID: &a.ID, Move: true})
		}()
		close(start)
		wg.Wait()
	}

	// f2 under f1 and f1 under f3a: each is valid alone, together they would be four levels deep.
	for i := 0; i < 20; i++ {
		f1 := e.folder(admin, fmt.Sprintf("depth-1-%d", i), nil)
		f2 := e.folder(admin, fmt.Sprintf("depth-2-%d", i), nil)
		f3 := e.folder(admin, fmt.Sprintf("depth-3-%d", i), nil)
		f3a := e.folder(admin, fmt.Sprintf("depth-3a-%d", i), &f3.ID)
		start := make(chan struct{})
		var wg sync.WaitGroup
		wg.Add(2)
		go func() {
			defer wg.Done()
			<-start
			_, _ = e.svc.UpdateFolder(ctx, admin, f2.ID, FolderPatch{ParentID: &f1.ID, Move: true})
		}()
		go func() {
			defer wg.Done()
			<-start
			_, _ = e.svc.UpdateFolder(ctx, admin, f1.ID, FolderPatch{ParentID: &f3a.ID, Move: true})
		}()
		close(start)
		wg.Wait()
	}

	folders, err := e.svc.ListFolders(ctx, admin)
	if err != nil {
		t.Fatal(err)
	}
	assertFolderTreeValid(t, folders)
}

// TestConcurrentFolderDeleteAndUpload races a delete against an upload into the same folder. The
// upload must either win (and block the delete as not-empty) or lose with a 404; an active image may
// never reference a missing folder nor be silently re-homed to the root.
func TestConcurrentFolderDeleteAndUpload(t *testing.T) {
	e := newEnv(t)
	ctx := context.Background()
	admin := e.admin
	for i := 0; i < 25; i++ {
		f := e.folder(admin, fmt.Sprintf("del-%d", i), nil)
		data := makePNG(t, 6, 6, 4000+i)
		start := make(chan struct{})
		var wg sync.WaitGroup
		var delErr, upErr error
		var up UploadResult
		wg.Add(2)
		go func() {
			defer wg.Done()
			<-start
			delErr = e.svc.DeleteFolder(ctx, admin, f.ID)
		}()
		go func() {
			defer wg.Done()
			<-start
			up, upErr = e.uploadToFolder(admin, "race.png", data, f.ID)
		}()
		close(start)
		wg.Wait()

		if upErr == nil {
			if ErrorCode(delErr) != CodeFolderNotEmpty {
				t.Fatalf("upload won but delete returned %v", delErr)
			}
			if up.Image.FolderID == nil || *up.Image.FolderID != f.ID {
				t.Fatalf("upload was re-homed: %+v", up.Image)
			}
			if _, err := e.db.GetFolder(ctx, f.ID); err != nil {
				t.Fatalf("folder gone although an active image is in it: %v", err)
			}
		} else {
			wantCode(t, upErr, CodeNotFound)
			if delErr != nil {
				t.Fatalf("delete lost to a failed upload: %v", delErr)
			}
		}

		var dangling int
		if err := e.db.Reader().QueryRowContext(ctx,
			`SELECT COUNT(*) FROM images i LEFT JOIN folders f ON f.id = i.folder_id
			 WHERE i.state = 'active' AND i.folder_id IS NOT NULL AND f.id IS NULL`).Scan(&dangling); err != nil {
			t.Fatal(err)
		}
		if dangling != 0 {
			t.Fatalf("%d active images reference a missing folder", dangling)
		}
		if upErr == nil {
			stored, err := e.db.GetImage(ctx, up.Image.ID)
			if err != nil || stored.FolderID == nil || *stored.FolderID != f.ID {
				t.Fatalf("active image silently left its folder: %+v %v", stored, err)
			}
		}
	}
}

// TestUploadAudit checks the image.upload row: one per stored image, none for a dedupe hit, target =
// image id, detail = folder id or empty, and the image kind filter includes it.
func TestUploadAudit(t *testing.T) {
	e := newEnv(t)
	ctx := context.Background()
	f := e.folder(e.admin, "f", nil)
	uploads := func() []AuditItem {
		t.Helper()
		items, _, err := e.svc.ListAudit(ctx, e.admin, AuditQuery{Action: "image.upload"}, 100, "")
		if err != nil {
			t.Fatal(err)
		}
		return items
	}

	img := e.mustUpload(e.admin, "a.png", makePNG(t, 5, 5, 5000))
	items := uploads()
	if len(items) != 1 || items[0].Target != img.ID || items[0].Detail != "" {
		t.Fatalf("upload audit: %+v", items)
	}

	// a dedupe hit stores nothing and writes no audit row
	dup, err := e.upload(e.admin, "a-again.png", makePNG(t, 5, 5, 5000))
	if err != nil || !dup.Duplicate {
		t.Fatalf("dedupe: %+v %v", dup, err)
	}
	if got := uploads(); len(got) != 1 {
		t.Fatalf("dedupe wrote %d upload rows", len(got))
	}

	// a folder upload records the destination folder id as detail
	inFolder, err := e.uploadToFolder(e.admin, "b.png", makePNG(t, 5, 5, 5001), f.ID)
	if err != nil || inFolder.Duplicate {
		t.Fatalf("folder upload: %+v %v", inFolder, err)
	}
	items = uploads()
	if len(items) != 2 || items[0].Target != inFolder.Image.ID || items[0].Detail != f.ID {
		t.Fatalf("folder upload audit: %+v", items)
	}

	// the image category includes uploads
	kindItems, _, err := e.svc.ListAudit(ctx, e.admin, AuditQuery{Kind: "image"}, 100, "")
	if err != nil {
		t.Fatal(err)
	}
	n := 0
	for _, it := range kindItems {
		if it.Action == "image.upload" {
			n++
		}
	}
	if n != 2 {
		t.Fatalf("kind=image returned %d upload rows", n)
	}
}
