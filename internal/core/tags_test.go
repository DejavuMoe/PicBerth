package core

import (
	"context"
	"fmt"
	"reflect"
	"strings"
	"testing"
	"time"
)

func sp(s string) *string { return &s }

func tp(tags ...string) *[]string {
	if tags == nil {
		tags = []string{}
	}
	return &tags
}

func (e *env) auditActions() []string {
	e.t.Helper()
	items, _, err := e.svc.ListAudit(context.Background(), e.admin, 200, "")
	if err != nil {
		e.t.Fatal(err)
	}
	var out []string
	for i := len(items) - 1; i >= 0; i-- {
		out = append(out, items[i].Action+"|"+items[i].Target+"|"+items[i].Detail)
	}
	return out
}

func TestNormalizeTags(t *testing.T) {
	got, err := normalizeTags([]string{"  猫 ", "dog", "猫", "dog", "a b", "Dog"})
	if err != nil || !reflect.DeepEqual(got, []string{"Dog", "a b", "dog", "猫"}) {
		t.Fatalf("normalize: %v %v", got, err)
	}
	if got, err := normalizeTags(nil); err != nil || got == nil || len(got) != 0 {
		t.Fatalf("empty: %#v %v", got, err)
	}
	if _, err := normalizeTags([]string{strings.Repeat("字", 32)}); err != nil {
		t.Fatalf("32 runes must be accepted: %v", err)
	}
	var many []string
	for i := 0; i < 20; i++ {
		many = append(many, fmt.Sprint("t", i))
	}
	if _, err := normalizeTags(append(many, "t0")); err != nil {
		t.Fatalf("duplicates must not count towards the limit: %v", err)
	}
	for name, bad := range map[string][]string{
		"empty":      {""},
		"blank":      {"   "},
		"too long":   {strings.Repeat("字", 33)},
		"comma":      {"a,b"},
		"hash":       {"#tag"},
		"hash after": {" #tag"},
		"control":    {"a\x00b"},
		"tab inside": {"a\tb"},
		"newline":    {"a\nb"},
		"bad utf8":   {"a\xffb"},
		"21 tags":    append(append([]string(nil), many...), "t20"),
	} {
		_, err := normalizeTags(bad)
		if err == nil {
			t.Errorf("%s: accepted", name)
			continue
		}
		wantCode(t, err, CodeValidation)
		if err.(*Error).Detail == "" {
			t.Errorf("%s: no detail", name)
		}
	}
	if got, _ := normalizeTags([]string{"a#b"}); len(got) != 1 {
		t.Fatal("# is only forbidden at the start")
	}
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

func TestPatchImage(t *testing.T) {
	e := newEnv(t)
	ctx := context.Background()
	img := e.mustUpload(e.admin, "Shot [1].png", makePNG(t, 5, 5, 900))
	if img.Tags == nil || len(img.Tags) != 0 {
		t.Fatalf("new image tags must be an empty array: %#v", img.Tags)
	}

	// all three at once
	v, err := e.svc.PatchImage(ctx, e.admin, img.ID, ImagePatch{Visibility: sp("private"), Filename: sp("  Renamed *x*.png "), Tags: tp("b", " a ", "b")})
	if err != nil {
		t.Fatal(err)
	}
	if v.Visibility != "private" || v.Filename != "Renamed *x*.png" || !reflect.DeepEqual(v.Tags, []string{"a", "b"}) {
		t.Fatalf("combined patch: %+v", v)
	}
	if v.ID != img.ID || v.URL != img.URL || !strings.Contains(v.Links.Markdown, `![Renamed \*x\*](`) || !strings.Contains(v.Links.HTML, `alt="Renamed *x*"`) {
		t.Fatalf("id/url must stay, alt text follows the name: %+v", v)
	}
	got, _ := e.svc.GetImage(ctx, e.admin, img.ID)
	if !reflect.DeepEqual(got.Tags, []string{"a", "b"}) || got.Filename != "Renamed *x*.png" || got.Visibility != "private" {
		t.Fatalf("not persisted: %+v", got)
	}

	// subsets
	v, _ = e.svc.PatchImage(ctx, e.admin, img.ID, ImagePatch{Filename: sp("only-name.png")})
	if v.Filename != "only-name.png" || v.Visibility != "private" || len(v.Tags) != 2 {
		t.Fatalf("rename only touched other fields: %+v", v)
	}
	v, _ = e.svc.PatchImage(ctx, e.admin, img.ID, ImagePatch{Tags: tp()})
	if v.Tags == nil || len(v.Tags) != 0 || v.Filename != "only-name.png" {
		t.Fatalf("clearing tags: %+v", v)
	}
	v, _ = e.svc.PatchImage(ctx, e.admin, img.ID, ImagePatch{Visibility: sp("public"), Tags: tp("z")})
	if v.Visibility != "public" || !reflect.DeepEqual(v.Tags, []string{"z"}) {
		t.Fatalf("visibility+tags: %+v", v)
	}

	// audit: only real changes, naming like the existing visibility entry
	want := []string{
		"image.visibility|" + img.ID + "|public -> private",
		"image.rename|" + img.ID + "|Renamed *x*.png",
		"image.tags|" + img.ID + "|a,b",
		"image.rename|" + img.ID + "|only-name.png",
		"image.tags|" + img.ID + "|",
		"image.visibility|" + img.ID + "|private -> public",
		"image.tags|" + img.ID + "|z",
	}
	if a := e.auditActions(); !reflect.DeepEqual(a[len(a)-len(want):], want) {
		t.Fatalf("audit:\n got %v\nwant %v", a[len(a)-len(want):], want)
	}
	before := len(e.auditActions())
	e.svc.PatchImage(ctx, e.admin, img.ID, ImagePatch{Visibility: sp("public"), Filename: sp("only-name.png"), Tags: tp("z")})
	if len(e.auditActions()) != before {
		t.Fatal("no-op patch must not write audit rows")
	}

	// errors; none of them may change anything
	for name, bad := range map[string]ImagePatch{
		"empty":      {},
		"vis":        {Visibility: sp("secret")},
		"name":       {Filename: sp("a/b")},
		"tags":       {Tags: tp("a,b")},
		"valid+tags": {Filename: sp("changed.png"), Tags: tp("#x")},
	} {
		_, err := e.svc.PatchImage(ctx, e.admin, img.ID, bad)
		if err == nil {
			t.Fatalf("%s accepted", name)
		}
		wantCode(t, err, CodeValidation)
	}
	if got, _ := e.svc.GetImage(ctx, e.admin, img.ID); got.Filename != "only-name.png" {
		t.Fatalf("a rejected patch changed the name: %q", got.Filename)
	}

	// authorization and visibility rules
	up, rd, mg := e.tokenPrincipal(ScopeUpload), e.tokenPrincipal(ScopeRead), e.tokenPrincipal(ScopeManage)
	for _, p := range []Principal{up, rd} {
		_, err := e.svc.PatchImage(ctx, p, img.ID, ImagePatch{Tags: tp("x")})
		wantCode(t, err, CodeForbidden)
	}
	_, err = e.svc.PatchImage(ctx, Principal{}, img.ID, ImagePatch{Tags: tp("x")})
	wantCode(t, err, CodeUnauthorized)
	if _, err := e.svc.PatchImage(ctx, mg, img.ID, ImagePatch{Tags: tp("by-token")}); err != nil {
		t.Fatalf("manage token: %v", err)
	}
	_, err = e.svc.PatchImage(ctx, e.admin, "00000000000000000000", ImagePatch{Tags: tp("x")})
	wantCode(t, err, CodeNotFound)

	// trashed images can still be edited, purged ones cannot
	if err := e.svc.Trash(ctx, e.admin, img.ID); err != nil {
		t.Fatal(err)
	}
	if _, err := e.svc.PatchImage(ctx, e.admin, img.ID, ImagePatch{Tags: tp("in-trash")}); err != nil {
		t.Fatalf("patch trashed: %v", err)
	}
	if err := e.svc.Purge(ctx, e.admin, img.ID); err != nil {
		t.Fatal(err)
	}
	_, err = e.svc.PatchImage(ctx, e.admin, img.ID, ImagePatch{Tags: tp("x")})
	wantCode(t, err, CodeNotFound)
	if rows := mustTagRows(t, e, img.ID); rows != 0 {
		t.Fatalf("%d tag rows survive the purge", rows)
	}
}

func mustTagRows(t *testing.T, e *env, id string) int {
	t.Helper()
	var n int
	if err := e.db.Reader().QueryRow(`SELECT count(*) FROM image_tags WHERE image_id = ?`, id).Scan(&n); err != nil {
		t.Fatal(err)
	}
	return n
}

func TestTagsSurviveTrashRestoreAndGoOnPurge(t *testing.T) {
	e := newEnv(t)
	ctx := context.Background()
	img := e.mustUpload(e.admin, "a.png", makePNG(t, 5, 5, 901))
	e.svc.PatchImage(ctx, e.admin, img.ID, ImagePatch{Tags: tp("x", "y")})
	if mustTagRows(t, e, img.ID) != 2 {
		t.Fatal("tags missing")
	}
	if err := e.svc.Trash(ctx, e.admin, img.ID); err != nil {
		t.Fatal(err)
	}
	if mustTagRows(t, e, img.ID) != 2 {
		t.Fatal("trashing must keep tags (restore needs them)")
	}
	r, err := e.svc.Restore(ctx, e.admin, img.ID)
	if err != nil || !reflect.DeepEqual(r.Tags, []string{"x", "y"}) {
		t.Fatalf("restore: %+v %v", r, err)
	}
	// the batch purge path removes them too
	if res, err := e.svc.Batch(ctx, e.admin, "trash", []string{img.ID}, ""); err != nil || !res[0].OK {
		t.Fatalf("batch trash: %v %v", res, err)
	}
	if res, err := e.svc.Batch(ctx, e.admin, "purge", []string{img.ID}, ""); err != nil || !res[0].OK {
		t.Fatalf("batch purge: %v %v", res, err)
	}
	if n := mustTagRows(t, e, img.ID); n != 0 {
		t.Fatalf("%d tag rows survive the purge", n)
	}
}

func TestUploadWithTags(t *testing.T) {
	e := newEnv(t)
	ctx := context.Background()
	data := makePNG(t, 5, 5, 902)

	up := func(tags ...string) (UploadResult, error) {
		u, err := e.svc.BeginUpload(ctx, e.admin)
		if err != nil {
			return UploadResult{}, err
		}
		defer u.Close()
		for _, tg := range tags {
			if err := u.AddTag(tg); err != nil {
				return UploadResult{}, err
			}
		}
		if err := u.WriteFile(ctx, strings.NewReader(string(data)), "t.png"); err != nil {
			return UploadResult{}, err
		}
		return u.Commit(ctx)
	}
	r, err := up(" b ", "a", "b")
	if err != nil || !reflect.DeepEqual(r.Image.Tags, []string{"a", "b"}) {
		t.Fatalf("upload tags: %+v %v", r.Image, err)
	}
	got, _ := e.svc.GetImage(ctx, e.admin, r.Image.ID)
	if !reflect.DeepEqual(got.Tags, []string{"a", "b"}) {
		t.Fatalf("stored tags: %v", got.Tags)
	}

	// duplicate: existing tags stay untouched, and the response shows them
	d, err := up("other")
	if err != nil || !d.Duplicate || d.Image.ID != r.Image.ID || !reflect.DeepEqual(d.Image.Tags, []string{"a", "b"}) {
		t.Fatalf("duplicate: %+v %v", d, err)
	}
	if got, _ := e.svc.GetImage(ctx, e.admin, r.Image.ID); !reflect.DeepEqual(got.Tags, []string{"a", "b"}) {
		t.Fatalf("duplicate upload changed tags: %v", got.Tags)
	}

	u, _ := e.svc.BeginUpload(ctx, e.admin)
	defer u.Close()
	wantCode(t, u.AddTag("#bad"), CodeValidation)
	for i := 0; i < 20; i++ {
		if err := u.AddTag(fmt.Sprint("t", i)); err != nil {
			t.Fatal(err)
		}
	}
	wantCode(t, u.AddTag("one-too-many"), CodeValidation)
}

func TestListFiltersViaService(t *testing.T) {
	e := newEnv(t)
	ctx := context.Background()
	pg := e.tokenPrincipal(ScopeManage)
	mk := func(p Principal, name string, seed int, tags ...string) ImageView {
		v := e.mustUpload(p, name, makePNG(t, 5, 5, 1000+seed))
		if len(tags) > 0 {
			if _, err := e.svc.PatchImage(ctx, e.admin, v.ID, ImagePatch{Tags: tp(tags...)}); err != nil {
				t.Fatal(err)
			}
		}
		return v
	}
	base := time.Date(2026, 3, 1, 12, 0, 0, 0, time.UTC)
	clock := base
	e.svc.Now = func() time.Time { return clock }
	at := func(hours int) { clock = base.Add(time.Duration(hours) * time.Hour) }

	at(1)
	mk(e.admin, "alpha.png", 1, "cat", "red") // web
	at(2)
	mk(pg, "Bravo.png", 2, "cat") // token
	at(3)
	mk(e.admin, "charlie.png", 3, "dog") // web
	at(4)
	mk(e.cliUploader(), "delta.png", 4) // cli
	at(5)
	if _, err := e.upload(e.admin, "echo.jpg", makeJPEG(t, 5)); err != nil { // the only jpeg
		t.Fatal(err)
	}
	ids := func(p ListParams) []string {
		t.Helper()
		items, _, err := e.svc.ListImages(ctx, e.admin, p)
		if err != nil {
			t.Fatalf("%+v: %v", p, err)
		}
		var out []string
		for _, it := range items {
			out = append(out, it.Filename)
		}
		return out
	}
	eq := func(what string, got []string, want ...string) {
		t.Helper()
		if !reflect.DeepEqual(got, want) {
			t.Fatalf("%s: got %v, want %v", what, got, want)
		}
	}
	eq("tag", ids(ListParams{Tags: []string{"cat"}, Sort: "old"}), "alpha.png", "Bravo.png")
	eq("tag all", ids(ListParams{Tags: []string{"cat", "red"}}), "alpha.png")
	eq("repeated+comma format", ids(ListParams{Formats: []string{"png,gif", "jpeg"}, Sort: "name", Limit: 2}), "alpha.png", "Bravo.png")
	eq("format any-of", ids(ListParams{Formats: []string{"gif", "jpeg"}}), "echo.jpg")
	eq("format compat", ids(ListParams{Format: "jpeg"}), "echo.jpg")
	eq("format comma in Format", ids(ListParams{Format: "jpeg,webp"}), "echo.jpg")
	eq("web source", ids(ListParams{Sources: []string{"web"}, Sort: "old"}), "alpha.png", "charlie.png", "echo.jpg")
	eq("cli+token", ids(ListParams{Sources: []string{"cli", fmt.Sprintf("token:%d", pg.TokenID)}, Sort: "old"}), "Bravo.png", "delta.png")
	since, until := base.Add(2*time.Hour), base.Add(4*time.Hour)
	eq("since/until", ids(ListParams{Since: &since, Until: &until, Sort: "old"}), "Bravo.png", "charlie.png")
	eq("q matches tag", ids(ListParams{Q: "DOG"}), "charlie.png")
	eq("filters combine with AND", ids(ListParams{Tags: []string{"cat"}, Sources: []string{"web"}, Since: &since}))

	for name, bad := range map[string]ListParams{
		"format": {Format: "bmp"}, "formats": {Formats: []string{"png,bmp"}}, "source": {Sources: []string{"ftp"}},
		"token0": {Sources: []string{"token:0"}}, "tokenx": {Sources: []string{"token:x"}}, "token-": {Sources: []string{"token:-1"}},
		"sort": {Sort: "random"},
	} {
		_, _, err := e.svc.ListImages(ctx, e.admin, bad)
		if err == nil {
			t.Errorf("%s accepted", name)
			continue
		}
		wantCode(t, err, CodeValidation)
	}

	// name sort: case-insensitive, stable across pages, cursor bound to the sort
	var names []string
	cursor := ""
	for page := 0; page < 6; page++ {
		items, next, err := e.svc.ListImages(ctx, e.admin, ListParams{Sort: "name", Limit: 2, Cursor: cursor})
		if err != nil {
			t.Fatal(err)
		}
		for _, it := range items {
			names = append(names, it.Filename)
		}
		if next == "" {
			break
		}
		cursor = next
	}
	eq("name order", names, "alpha.png", "Bravo.png", "charlie.png", "delta.png", "echo.jpg")
	_, _, err := e.svc.ListImages(ctx, e.admin, ListParams{Sort: "new", Cursor: cursor})
	wantCode(t, err, CodeValidation)
	// an old-style cursor (no name key) is still valid for the other sorts
	items, next, _ := e.svc.ListImages(ctx, e.admin, ListParams{Sort: "new", Limit: 2})
	items2, _, err := e.svc.ListImages(ctx, e.admin, ListParams{Sort: "new", Limit: 2, Cursor: next})
	if err != nil || len(items) != 2 || len(items2) != 2 || items[1].ID == items2[0].ID {
		t.Fatalf("cursor for sort new: %v %v", items2, err)
	}

	// upload-only tokens keep their restrictions with the new filters
	upTok := e.tokenPrincipal(ScopeUpload)
	own := e.mustUpload(upTok, "mine.png", makePNG(t, 5, 5, 1100))
	e.svc.PatchImage(ctx, e.admin, own.ID, ImagePatch{Tags: tp("cat")})
	items, _, err = e.svc.ListImages(ctx, upTok, ListParams{Tags: []string{"cat"}, Sources: []string{"web", "cli"}})
	if err != nil || len(items) != 0 {
		t.Fatalf("upload token escaped via source filter: %v %v", items, err)
	}
	items, _, _ = e.svc.ListImages(ctx, upTok, ListParams{Tags: []string{"cat"}})
	if len(items) != 1 || items[0].ID != own.ID {
		t.Fatalf("upload token tag filter: %v", items)
	}
}

func TestListTagsLoadedOncePerPage(t *testing.T) {
	e := newEnv(t)
	ctx := context.Background()
	for i := 0; i < 5; i++ {
		v := e.mustUpload(e.admin, fmt.Sprintf("p%d.png", i), makePNG(t, 5, 5, 1200+i))
		e.svc.PatchImage(ctx, e.admin, v.ID, ImagePatch{Tags: tp(fmt.Sprint("t", i), "common")})
	}
	items, _, err := e.svc.ListImages(ctx, e.admin, ListParams{})
	if err != nil || len(items) != 5 {
		t.Fatal(err)
	}
	for _, it := range items {
		if len(it.Tags) != 2 || it.Tags[0] != "common" {
			t.Fatalf("tags on list item: %+v", it.Tags)
		}
	}
}

func TestImageFacetsService(t *testing.T) {
	e := newEnv(t)
	ctx := context.Background()
	tokView, secret, err := e.svc.CreateToken(ctx, e.admin, "PicGo · 笔记本", []string{ScopeManage}, nil)
	if err != nil {
		t.Fatal(err)
	}
	mg, err := e.svc.AuthenticateToken(ctx, secret, "127.0.0.1")
	if err != nil {
		t.Fatal(err)
	}
	up := e.tokenPrincipal(ScopeUpload)
	rd := e.tokenPrincipal(ScopeRead)

	w1 := e.mustUpload(e.admin, "w1.png", makePNG(t, 5, 5, 1300))
	w2 := e.mustUpload(e.admin, "w2.png", makePNG(t, 5, 5, 1301))
	t1 := e.mustUpload(mg, "t1.png", makePNG(t, 5, 5, 1302))
	t2 := e.mustUpload(mg, "t2.png", makePNG(t, 5, 5, 1303))
	own := e.mustUpload(up, "own.png", makePNG(t, 5, 5, 1304))
	c1 := e.mustUpload(e.cliUploader(), "c1.jpg", makeJPEG(t, 1))
	e.svc.PatchImage(ctx, e.admin, w1.ID, ImagePatch{Tags: tp("截图", "x"), Visibility: sp("private")})
	e.svc.PatchImage(ctx, e.admin, w2.ID, ImagePatch{Tags: tp("截图")})
	e.svc.PatchImage(ctx, e.admin, t1.ID, ImagePatch{Tags: tp("截图", "y")})
	e.svc.PatchImage(ctx, e.admin, own.ID, ImagePatch{Tags: tp("x")})
	if err := e.svc.Trash(ctx, e.admin, t2.ID); err != nil {
		t.Fatal(err)
	}
	// revoke the manage token: the facet still names it
	if err := e.svc.RevokeToken(ctx, e.admin, tokView.ID); err != nil {
		t.Fatal(err)
	}

	f, err := e.svc.ImageFacets(ctx, e.admin, "")
	if err != nil {
		t.Fatal(err)
	}
	var bytes int64
	for _, v := range []ImageView{w1, w2, t1, own, c1} {
		bytes += v.Size
	}
	if f.Total != 5 || f.Bytes != bytes || f.GPS != 0 {
		t.Fatalf("totals: %+v (bytes want %d)", f, bytes)
	}
	if !reflect.DeepEqual(f.Formats, map[string]int64{"jpeg": 1, "png": 4, "gif": 0, "webp": 0, "avif": 0}) {
		t.Fatalf("formats: %v", f.Formats)
	}
	if !reflect.DeepEqual(f.Visibility, map[string]int64{"public": 4, "private": 1}) {
		t.Fatalf("visibility: %v", f.Visibility)
	}
	if len(f.Sources) != 4 || f.Sources[0].Key != "web" || f.Sources[0].Count != 2 || f.Sources[3].Key != "cli" || f.Sources[3].Count != 1 {
		t.Fatalf("sources: %+v", f.Sources)
	}
	if s := f.Sources[1]; s.Kind != "token" || s.TokenID == nil || *s.TokenID != tokView.ID || s.TokenName != "PicGo · 笔记本" || s.Count != 1 || s.Key != fmt.Sprintf("token:%d", tokView.ID) {
		t.Fatalf("revoked token source: %+v", s)
	}
	if s := f.Sources[2]; s.TokenID == nil || *s.TokenID != int64(up.TokenID) || s.Count != 1 {
		t.Fatalf("second token source: %+v", f.Sources[2])
	}
	if want := []FacetTag{{"截图", 3}, {"x", 2}, {"y", 1}}; !reflect.DeepEqual(f.Tags, want) {
		t.Fatalf("tags: %v, want %v", f.Tags, want)
	}

	// trashed set
	tf, err := e.svc.ImageFacets(ctx, e.admin, "trashed")
	if err != nil || tf.Total != 1 || tf.Bytes != t2.Size || len(tf.Sources) != 1 || tf.Sources[0].Count != 1 || len(tf.Tags) != 0 || tf.Formats["png"] != 1 {
		t.Fatalf("trashed: %+v %v", tf, err)
	}

	// principals
	uf, err := e.svc.ImageFacets(ctx, up, "")
	if err != nil || uf.Total != 1 || len(uf.Sources) != 1 || uf.Sources[0].Count != 1 || len(uf.Tags) != 1 || uf.Tags[0].Tag != "x" || uf.Tags[0].Count != 1 {
		t.Fatalf("upload token facets must cover its own uploads only: %+v %v", uf, err)
	}
	if rf, err := e.svc.ImageFacets(ctx, rd, ""); err != nil || rf.Total != 5 {
		t.Fatalf("read token: %+v %v", rf, err)
	}
	_, err = e.svc.ImageFacets(ctx, up, "trashed")
	wantCode(t, err, CodeForbidden)
	_, err = e.svc.ImageFacets(ctx, rd, "trashed")
	wantCode(t, err, CodeForbidden)
	_, err = e.svc.ImageFacets(ctx, Principal{}, "")
	wantCode(t, err, CodeUnauthorized)
	_, err = e.svc.ImageFacets(ctx, e.admin, "purged")
	wantCode(t, err, CodeValidation)
	mg2 := e.tokenPrincipal(ScopeManage)
	if mf, err := e.svc.ImageFacets(ctx, mg2, "trashed"); err != nil || mf.Total != 1 {
		t.Fatalf("manage token trashed: %+v %v", mf, err)
	}
}

func TestFacetsEmptyShape(t *testing.T) {
	e := newEnv(t)
	f, err := e.svc.ImageFacets(context.Background(), e.admin, "")
	if err != nil {
		t.Fatal(err)
	}
	if f.Total != 0 || f.Sources == nil || f.Tags == nil || len(f.Formats) != 5 || len(f.Visibility) != 2 {
		t.Fatalf("empty facets: %#v", f)
	}
}

// cliUploader is the administrator acting through the local command line (source "cli").
func (e *env) cliUploader() Principal {
	p := e.admin
	p.Kind = KindCLI
	return p
}
