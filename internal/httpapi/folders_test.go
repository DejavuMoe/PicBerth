package httpapi

import (
	"net/http"
	"reflect"
	"strings"
	"testing"
)

type folderJSON struct {
	ID         string  `json:"id"`
	Name       string  `json:"name"`
	ParentID   *string `json:"parentId"`
	Depth      int     `json:"depth"`
	ImageCount int     `json:"imageCount"`
	CreatedAt  string  `json:"createdAt"`
}

func (h *harness) createFolder(name string, parentID *string) folderJSON {
	h.t.Helper()
	body := map[string]any{"name": name}
	if parentID != nil {
		body["parentId"] = *parentID
	}
	resp, out := h.web("POST", "/api/v1/folders", body)
	wantStatus(h.t, resp, out, 201)
	return decode[folderJSON](h.t, out)
}

func TestFolderEndpoints(t *testing.T) {
	h := newHarness(t, harnessOpts{})
	a := h.createFolder("旅行", nil)
	b := h.createFolder("2026", &a.ID)
	if a.Depth != 1 || a.ParentID != nil || a.ImageCount != 0 || a.CreatedAt == "" || len(a.ID) != 16 {
		t.Fatalf("root folder: %+v", a)
	}
	if b.Depth != 2 || b.ParentID == nil || *b.ParentID != a.ID {
		t.Fatalf("child folder: %+v", b)
	}
	c := h.createFolder("云南", &b.ID)
	if c.Depth != 3 {
		t.Fatalf("depth 3: %+v", c)
	}
	// errors
	for name, tc := range map[string]struct {
		body   map[string]any
		status int
		code   string
	}{
		"duplicate":   {map[string]any{"name": "旅行"}, 409, "folder-exists"},
		"depth":       {map[string]any{"name": "第四层", "parentId": c.ID}, 422, "folder-depth"},
		"unknown":     {map[string]any{"name": "x", "parentId": "0000000000000000"}, 404, "not-found"},
		"slash":       {map[string]any{"name": "a/b"}, 400, "validation"},
		"empty":       {map[string]any{"name": "  "}, 400, "validation"},
		"long":        {map[string]any{"name": strings.Repeat("x", 65)}, 400, "validation"},
		"unknown key": {map[string]any{"name": "x", "path": "y"}, 400, "validation"},
	} {
		resp, body := h.web("POST", "/api/v1/folders", tc.body)
		wantProblem(t, resp, body, tc.status, tc.code)
		_ = name
	}
	// flat list with active direct counts
	img := h.mustUploadWeb(601, part{name: "folder", data: []byte(b.ID)})
	if img.FolderID == nil || *img.FolderID != b.ID {
		t.Fatalf("upload folder: %+v", img)
	}
	resp, body := h.web("GET", "/api/v1/folders", nil)
	wantStatus(t, resp, body, 200)
	items := decode[struct{ Items []folderJSON }](t, body).Items
	if len(items) != 3 {
		t.Fatalf("folders: %s", body)
	}
	for _, f := range items {
		if f.ID == b.ID && f.ImageCount != 1 {
			t.Fatalf("imageCount: %+v", f)
		}
		if f.ID != b.ID && f.ImageCount != 0 {
			t.Fatalf("imageCount: %+v", f)
		}
	}
	// rename and move
	resp, body = h.web("PATCH", "/api/v1/folders/"+c.ID, map[string]any{"name": "大理"})
	wantStatus(t, resp, body, 200)
	if f := decode[folderJSON](t, body); f.Name != "大理" || f.Depth != 3 {
		t.Fatalf("rename: %s", body)
	}
	resp, body = h.web("PATCH", "/api/v1/folders/"+c.ID, map[string]any{"parentId": nil})
	wantStatus(t, resp, body, 200)
	if f := decode[folderJSON](t, body); f.ParentID != nil || f.Depth != 1 {
		t.Fatalf("move to root: %s", body)
	}
	// cycles and depth on move
	resp, body = h.web("PATCH", "/api/v1/folders/"+a.ID, map[string]any{"parentId": b.ID})
	wantProblem(t, resp, body, 400, "validation") // b is inside a
	resp, body = h.web("PATCH", "/api/v1/folders/"+a.ID, map[string]any{"parentId": a.ID})
	wantProblem(t, resp, body, 400, "validation")
	// a non-cyclic move that would exceed three levels is 422
	d := h.createFolder("d", nil)
	e := h.createFolder("e", &d.ID)
	resp, body = h.web("PATCH", "/api/v1/folders/"+a.ID, map[string]any{"parentId": e.ID})
	wantProblem(t, resp, body, 422, "folder-depth")
	// a move into a parent that already has a same-named child is refused
	dupTarget := h.createFolder("2026", nil)
	resp, body = h.web("PATCH", "/api/v1/folders/"+b.ID, map[string]any{"parentId": nil})
	wantProblem(t, resp, body, 409, "folder-exists")
	_ = dupTarget
	// same-name conflict in the target
	resp, body = h.web("PATCH", "/api/v1/folders/"+c.ID, map[string]any{"name": "旅行"})
	wantProblem(t, resp, body, 409, "folder-exists")
	// empty patch
	resp, body = h.web("PATCH", "/api/v1/folders/"+a.ID, map[string]any{})
	wantProblem(t, resp, body, 400, "validation")
	// delete is blocked by active images
	resp, body = h.web("DELETE", "/api/v1/folders/"+b.ID, nil)
	wantProblem(t, resp, body, 409, "folder-not-empty")
	// and by subfolders
	resp, body = h.web("DELETE", "/api/v1/folders/"+a.ID, nil)
	wantProblem(t, resp, body, 409, "folder-not-empty")
	// a trashed image no longer blocks; it returns to the root
	h.web("DELETE", "/api/v1/images/"+img.ID, nil)
	resp, body = h.web("DELETE", "/api/v1/folders/"+b.ID, nil)
	wantStatus(t, resp, body, 204)
	resp, body = h.web("GET", "/api/v1/images/"+img.ID, nil)
	if decode[imageJSON](t, body).FolderID != nil {
		t.Fatalf("trashed image kept the deleted folder: %s", body)
	}
	resp, body = h.web("DELETE", "/api/v1/folders/"+b.ID, nil)
	wantProblem(t, resp, body, 404, "not-found")
	// auth: upload-only tokens list but cannot write
	_, upSecret := h.createToken("upload")
	resp, body = h.do("GET", "/api/v1/folders", nil, withBearer(upSecret))
	wantStatus(t, resp, body, 200)
	resp, body = h.jsonReq("POST", "/api/v1/folders", map[string]any{"name": "x"}, withBearer(upSecret))
	wantProblem(t, resp, body, 403, "forbidden")
	resp, body = h.do("GET", "/api/v1/folders", nil)
	wantProblem(t, resp, body, 401, "unauthorized")
}

func TestUploadToFolderWithUploadOnlyToken(t *testing.T) {
	h := newHarness(t, harnessOpts{})
	f := h.createFolder("uploads", nil)
	_, upSecret := h.createToken("upload")
	_, mgSecret := h.createToken("manage")

	// upload-only tokens may upload into any folder of the owner
	resp, body := h.upload(makePNG(t, 700), []reqOpt{withBearer(upSecret)}, part{name: "folder", data: []byte(f.ID)})
	wantStatus(t, resp, body, 201)
	own := decode[imageJSON](t, body)
	if own.FolderID == nil || *own.FolderID != f.ID {
		t.Fatalf("upload-only folder upload: %s", body)
	}
	// dedupe stays inside the token's own uploads in that folder
	resp, body = h.upload(makePNG(t, 700), []reqOpt{withBearer(upSecret)}, part{name: "folder", data: []byte(f.ID)})
	wantStatus(t, resp, body, 200)
	if d := decode[imageJSON](t, body); !d.Duplicate || d.ID != own.ID {
		t.Fatalf("upload-only dedupe: %s", body)
	}
	// a manage token deduplicates against the same folder's owner-wide set, including the token's image
	resp, body = h.upload(makePNG(t, 700), []reqOpt{withBearer(mgSecret)}, part{name: "folder", data: []byte(f.ID)})
	wantStatus(t, resp, body, 200)
	if d := decode[imageJSON](t, body); !d.Duplicate || d.ID != own.ID {
		t.Fatalf("manage token folder dedupe: %s", body)
	}
	// an unknown folder is 404 for every token
	resp, body = h.upload(makePNG(t, 701), []reqOpt{withBearer(upSecret)}, part{name: "folder", data: []byte("0000000000000000")})
	wantProblem(t, resp, body, 404, "not-found")
	// listing scoped by the folder keeps the token's own-upload rule
	resp, body = h.do("GET", "/api/v1/images?folder="+f.ID, nil, withBearer(upSecret))
	wantStatus(t, resp, body, 200)
	if l := decode[struct{ Items []imageJSON }](t, body); len(l.Items) != 1 || l.Items[0].ID != own.ID {
		t.Fatalf("upload-only folder list: %s", body)
	}
}

func TestPatchImageFolderAndBatchMove(t *testing.T) {
	h := newHarness(t, harnessOpts{})
	a := h.createFolder("a", nil)
	b := h.createFolder("b", &a.ID)
	img := h.mustUploadWeb(720)
	path := "/api/v1/images/" + img.ID

	// folderId is an opaque id; null moves to the root
	resp, body := h.web("PATCH", path, map[string]any{"folderId": a.ID})
	wantStatus(t, resp, body, 200)
	if v := decode[imageJSON](t, body); v.FolderID == nil || *v.FolderID != a.ID {
		t.Fatalf("patch folder: %s", body)
	}
	resp, body = h.web("PATCH", path, map[string]any{"filename": "renamed.png", "folderId": b.ID})
	wantStatus(t, resp, body, 200)
	if v := decode[map[string]any](t, body); v["folderId"] != b.ID || v["filename"] != "renamed.png" {
		t.Fatalf("combined patch: %s", body)
	}
	resp, body = h.web("PATCH", path, map[string]any{"folderId": nil})
	wantStatus(t, resp, body, 200)
	if v := decode[imageJSON](t, body); v.FolderID != nil {
		t.Fatalf("move to root: %s", body)
	}
	// an omitted folderId leaves the folder alone
	resp, body = h.web("PATCH", path, map[string]any{"filename": "again.png"})
	wantStatus(t, resp, body, 200)
	if v := decode[imageJSON](t, body); v.FolderID != nil {
		t.Fatalf("omitted folderId moved the image: %s", body)
	}
	// unknown folders and unknown fields are rejected
	resp, body = h.web("PATCH", path, map[string]any{"folderId": "0000000000000000"})
	wantProblem(t, resp, body, 404, "not-found")
	resp, body = h.web("PATCH", path, map[string]any{"visibility": "public"})
	wantProblem(t, resp, body, 400, "validation")
	resp, body = h.web("PATCH", path, map[string]any{"tags": []string{"x"}})
	wantProblem(t, resp, body, 400, "validation")

	// batch move to a folder and back to the root
	other := h.mustUploadWeb(721)
	resp, body = h.web("POST", "/api/v1/images/batch", map[string]any{"action": "move", "ids": []string{img.ID, other.ID}, "folderId": a.ID})
	wantStatus(t, resp, body, 200)
	res := decode[struct {
		Results []struct {
			ID string `json:"id"`
			OK bool   `json:"ok"`
		}
	}](t, body).Results
	if len(res) != 2 || !res[0].OK || !res[1].OK {
		t.Fatalf("batch move: %s", body)
	}
	resp, body = h.web("GET", "/api/v1/images?folder="+a.ID, nil)
	wantStatus(t, resp, body, 200)
	if l := decode[struct{ Items []imageJSON }](t, body); len(l.Items) != 2 {
		t.Fatalf("folder listing: %s", body)
	}
	resp, body = h.web("POST", "/api/v1/images/batch", map[string]any{"action": "move", "ids": []string{img.ID}, "folderId": nil})
	wantStatus(t, resp, body, 200)
	resp, body = h.web("GET", "/api/v1/images?folder=root", nil)
	if l := decode[struct{ Items []imageJSON }](t, body); len(l.Items) != 1 {
		t.Fatalf("root listing: %s", body)
	}
	// unknown destination fails the whole batch
	resp, body = h.web("POST", "/api/v1/images/batch", map[string]any{"action": "move", "ids": []string{img.ID}, "folderId": "0000000000000000"})
	wantProblem(t, resp, body, 404, "not-found")
	// the removed visibility action is rejected
	resp, body = h.web("POST", "/api/v1/images/batch", map[string]any{"action": "visibility", "ids": []string{img.ID}, "visibility": "private"})
	wantProblem(t, resp, body, 400, "validation")
}

func TestListByFolderHTTP(t *testing.T) {
	h := newHarness(t, harnessOpts{})
	a := h.createFolder("a", nil)
	b := h.createFolder("b", &a.ID)
	root := h.mustUploadWeb(740)
	inA := h.mustUploadWeb(741, part{name: "folder", data: []byte(a.ID)})
	inB := h.mustUploadWeb(742, part{name: "folder", data: []byte(b.ID)})
	names := func(query string) []string {
		t.Helper()
		resp, body := h.web("GET", "/api/v1/images?"+query, nil)
		wantStatus(t, resp, body, 200)
		var out []string
		for _, it := range decode[struct{ Items []imageJSON }](t, body).Items {
			out = append(out, it.ID)
		}
		return out
	}
	if got := names("folder=root"); !reflect.DeepEqual(got, []string{root.ID}) {
		t.Fatalf("root: %v", got)
	}
	if got := names("folder=" + a.ID); !reflect.DeepEqual(got, []string{inA.ID}) {
		t.Fatalf("direct: %v", got)
	}
	if got := names("folder=" + a.ID + "&recursive=true"); len(got) != 2 {
		t.Fatalf("recursive: %v", got)
	}
	if got := names("folder=root&recursive=true"); len(got) != 3 {
		t.Fatalf("recursive root: %v", got)
	}
	_ = inB
	// pagination keeps the folder scope in the cursor
	resp, body := h.web("GET", "/api/v1/images?folder="+a.ID+"&recursive=true&limit=1", nil)
	p1 := decode[struct {
		Items      []imageJSON
		NextCursor *string
	}](t, body)
	if p1.NextCursor == nil {
		t.Fatalf("no cursor: %s", body)
	}
	resp, body = h.web("GET", "/api/v1/images?folder="+b.ID+"&limit=1&cursor="+*p1.NextCursor, nil)
	wantProblem(t, resp, body, 400, "validation")
}

func TestFacetsShapeAndRemovedFilters(t *testing.T) {
	h := newHarness(t, harnessOpts{})
	f := h.createFolder("f", nil)
	a := h.mustUploadWeb(760)
	c := h.mustUploadWeb(761, part{name: "folder", data: []byte(f.ID)})
	h.web("PATCH", "/api/v1/images/"+a.ID, map[string]any{"folderId": f.ID})
	resp, body := h.web("GET", "/api/v1/images/facets", nil)
	wantStatus(t, resp, body, 200)
	facets := decode[map[string]any](t, body)
	if facets["total"] != float64(2) {
		t.Fatalf("facets total: %s", body)
	}
	if _, ok := facets["tags"]; ok {
		t.Fatalf("facets still expose tags: %s", body)
	}
	if _, ok := facets["visibility"]; ok {
		t.Fatalf("facets still expose visibility: %s", body)
	}
	if fm, ok := facets["formats"].(map[string]any); !ok || len(fm) != 5 || fm["png"] != float64(2) {
		t.Fatalf("formats: %s", body)
	}
	// removed list parameters are unknown and therefore ignored
	resp, body = h.web("GET", "/api/v1/images?visibility=public&tag=cat", nil)
	wantStatus(t, resp, body, 200)
	if l := decode[struct{ Items []imageJSON }](t, body); len(l.Items) != 2 {
		t.Fatalf("removed filters changed the result: %s", body)
	}
	_ = c
	// facets never require a body and reject bad states
	resp, body = h.web("GET", "/api/v1/images/facets?state=purged", nil)
	wantProblem(t, resp, body, 400, "validation")
	// the openapi document is valid JSON
	resp, body = h.do("GET", "/api/v1/openapi.json", nil)
	wantStatus(t, resp, body, 200)
	if !strings.Contains(string(body), `"/api/v1/folders"`) {
		t.Fatalf("openapi lacks folders")
	}
	if resp.Header.Get("ETag") == "" {
		t.Fatal("openapi has no ETag")
	}
	resp, body = h.do("GET", "/api/v1/openapi.json", nil, withHeader("If-None-Match", resp.Header.Get("ETag")))
	if resp.StatusCode != http.StatusNotModified {
		t.Fatalf("openapi 304: %d", resp.StatusCode)
	}
}
