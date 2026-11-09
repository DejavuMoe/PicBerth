package httpapi

import (
	"bytes"
	"context"
	"fmt"
	"net/http"
	"os"
	"path/filepath"
	"strings"
	"testing"
	"time"
)

func TestSetupFlow(t *testing.T) {
	h := newHarness(t, harnessOpts{skipAdmin: true})
	ctx := context.Background()

	resp, body := h.do("GET", "/api/v1/session", nil)
	wantStatus(t, resp, body, 200)
	sess := decode[map[string]any](t, body)
	if sess["authenticated"] != false || sess["setupRequired"] != true || sess["user"] != nil || sess["limits"] != nil || sess["version"] == "" {
		t.Fatalf("session before setup: %s", body)
	}
	resp, body = h.jsonReq("POST", "/api/v1/session", map[string]string{"username": "admin", "password": "whatever-1234"})
	wantProblem(t, resp, body, 409, "setup-required")

	code, err := h.svc.EnsureSetupCode(ctx)
	if err != nil || code == "" {
		t.Fatal(err)
	}
	creds := func(c string) map[string]string {
		return map[string]string{"setupCode": c, "username": "admin", "password": "correct horse battery"}
	}
	resp, body = h.jsonReq("POST", "/api/v1/setup", creds("AAAA-AAAA-AAAA"))
	wantProblem(t, resp, body, 403, "forbidden")
	resp, body = h.jsonReq("POST", "/api/v1/setup", map[string]string{"setupCode": code, "username": "admin", "password": "short"})
	wantProblem(t, resp, body, 400, "validation")
	resp, body = h.jsonReq("POST", "/api/v1/setup", map[string]string{"setupCode": code, "username": "ad min", "password": "correct horse battery"})
	wantProblem(t, resp, body, 400, "validation")
	resp, body = h.jsonReq("POST", "/api/v1/setup", map[string]any{"setupCode": code, "username": "admin", "password": "correct horse battery", "extra": 1})
	wantProblem(t, resp, body, 400, "validation") // unknown fields are rejected

	resp, body = h.jsonReq("POST", "/api/v1/setup", creds(code))
	wantStatus(t, resp, body, 204)
	var ck *http.Cookie
	for _, c := range resp.Cookies() {
		ck = c
	}
	if ck == nil || ck.Name != "__Host-pb_session" || !ck.HttpOnly || !ck.Secure || ck.SameSite != http.SameSiteStrictMode || ck.Path != "/" || ck.Domain != "" || ck.MaxAge <= 0 {
		t.Fatalf("session cookie: %+v", ck)
	}
	h.cookie = ck.Name + "=" + ck.Value

	resp, body = h.jsonReq("POST", "/api/v1/setup", creds(code))
	wantProblem(t, resp, body, 409, "setup-done")

	resp, body = h.do("GET", "/api/v1/session", nil, withHeader("Cookie", h.cookie))
	wantStatus(t, resp, body, 200)
	sess = decode[map[string]any](t, body)
	user, _ := sess["user"].(map[string]any)
	limits, _ := sess["limits"].(map[string]any)
	if sess["authenticated"] != true || sess["setupRequired"] != false || user["name"] != "admin" || limits["maxUploadBytes"] != float64(20<<20) {
		t.Fatalf("session after setup: %s", body)
	}
	if _, ok := limits["defaultVisibility"]; ok {
		t.Fatalf("defaultVisibility must be gone: %s", body)
	}
	if fmt.Sprint(limits["allowedFormats"]) != "[jpeg png gif webp avif]" {
		t.Fatalf("allowedFormats: %v", limits["allowedFormats"])
	}
	if resp.Header.Get("Cache-Control") != "no-store" {
		t.Fatal("API responses must be no-store")
	}
}

func TestSetupPlainHTTPCookieName(t *testing.T) {
	h := newHarness(t, harnessOpts{publicURL: "http://127.0.0.1:12433", imageURL: "http://127.0.0.1:12433"})
	if !strings.HasPrefix(h.cookie, "pb_session=") {
		t.Fatalf("cookie %q", h.cookie)
	}
}

func TestLoginLogout(t *testing.T) {
	h := newHarness(t, harnessOpts{})
	resp, body := h.jsonReq("POST", "/api/v1/session", map[string]string{"username": "admin", "password": "wrong password!"})
	wantProblem(t, resp, body, 401, "unauthorized")
	resp2, body2 := h.jsonReq("POST", "/api/v1/session", map[string]string{"username": "nobody", "password": "wrong password!"})
	wantProblem(t, resp2, body2, 401, "unauthorized")
	if !bytes.Equal(body, body2) {
		t.Fatalf("login errors reveal user existence:\n%s\n%s", body, body2)
	}
	resp, body = h.do("POST", "/api/v1/session", strings.NewReader("{"), withHeader("Content-Type", "application/json"))
	wantProblem(t, resp, body, 400, "validation")
	resp, body = h.do("POST", "/api/v1/session", strings.NewReader(`{"username":"admin","password":"x"}`), withHeader("Content-Type", "text/plain"))
	wantProblem(t, resp, body, 400, "validation")

	resp, body = h.do("GET", "/api/v1/images", nil, h.asWeb()...)
	wantStatus(t, resp, body, 200)
	// logout needs the CSRF header like any session-authenticated unsafe request
	resp, body = h.do("DELETE", "/api/v1/session", nil, withHeader("Cookie", h.cookie))
	wantProblem(t, resp, body, 403, "forbidden")
	resp, body = h.do("DELETE", "/api/v1/session", nil, h.asWeb()...)
	wantStatus(t, resp, body, 204)
	var cleared bool
	for _, c := range resp.Cookies() {
		if strings.HasSuffix(c.Name, "pb_session") && c.MaxAge < 0 {
			cleared = true
		}
	}
	if !cleared {
		t.Fatal("cookie not cleared")
	}
	resp, body = h.do("GET", "/api/v1/images", nil, h.asWeb()...)
	wantProblem(t, resp, body, 401, "unauthorized")
	resp, body = h.do("GET", "/api/v1/session", nil, withHeader("Cookie", h.cookie))
	if decode[map[string]any](t, body)["authenticated"] != false {
		t.Fatalf("logged-out cookie still authenticates: %s", body)
	}
}

func TestSessionRequiredEndpoints(t *testing.T) {
	h := newHarness(t, harnessOpts{})
	id := h.mustUploadWeb(1).ID
	for _, c := range []struct{ method, path string }{
		{"GET", "/api/v1/images"}, {"GET", "/api/v1/images/" + id}, {"PATCH", "/api/v1/images/" + id}, {"DELETE", "/api/v1/images/" + id},
		{"POST", "/api/v1/images/" + id + "/restore"}, {"POST", "/api/v1/images/" + id + "/purge"}, {"POST", "/api/v1/images/batch"},
		{"GET", "/api/v1/images/" + id + "/raw"}, {"GET", "/api/v1/images/" + id + "/thumb"}, {"POST", "/api/v1/images"},
		{"GET", "/api/v1/tokens"}, {"POST", "/api/v1/tokens"}, {"DELETE", "/api/v1/tokens/1"},
		{"GET", "/api/v1/settings"}, {"PATCH", "/api/v1/settings"}, {"GET", "/api/v1/status"},
		{"POST", "/api/v1/checks"}, {"POST", "/api/v1/snapshots"}, {"GET", "/api/v1/audit"},
	} {
		resp, body := h.jsonReq(c.method, c.path, map[string]any{})
		if resp.StatusCode != 401 {
			t.Errorf("%s %s without credentials: %d %s", c.method, c.path, resp.StatusCode, body)
			continue
		}
		wantProblem(t, resp, body, 401, "unauthorized")
	}
	// garbage and wrong-scheme credentials
	resp, body := h.do("GET", "/api/v1/images", nil, withBearer("pb_00000000000000000000000000000000"))
	wantProblem(t, resp, body, 401, "unauthorized")
	resp, body = h.do("GET", "/api/v1/images", nil, withHeader("Authorization", "Basic YWRtaW46eA=="))
	wantProblem(t, resp, body, 401, "unauthorized")
	resp, body = h.do("GET", "/api/v1/images", nil, withHeader("Cookie", "__Host-pb_session=garbage"))
	wantProblem(t, resp, body, 401, "unauthorized")
	// tokens are not accepted in the query string
	_, secret := h.createToken("manage")
	resp, body = h.do("GET", "/api/v1/images?token="+secret+"&access_token="+secret, nil)
	wantProblem(t, resp, body, 401, "unauthorized")
}

func TestUploadNewThenDuplicate(t *testing.T) {
	h := newHarness(t, harnessOpts{})
	data := makePNG(t, 7)
	resp, body := h.uploadWeb(data)
	wantStatus(t, resp, body, 201)
	img := decode[map[string]any](t, body)
	id := img["id"].(string)
	if len(id) != 20 || img["duplicate"] != false || img["state"] != "active" || img["folderId"] != nil || img["ext"] != "png" ||
		img["mime"] != "image/png" || img["width"] != float64(6) || img["height"] != float64(4) || img["frames"] != float64(1) || img["filename"] != "shot.png" {
		t.Fatalf("image: %s", body)
	}
	if _, ok := img["visibility"]; ok {
		t.Fatalf("visibility must be gone: %s", body)
	}
	if _, ok := img["tags"]; ok {
		t.Fatalf("tags must be gone: %s", body)
	}
	if img["url"] != "https://img.example.com/i/"+id+".png" {
		t.Fatalf("url: %v", img["url"])
	}
	links := img["links"].(map[string]any)
	if links["markdown"] != "![shot](https://img.example.com/i/"+id+".png)" || links["bbcode"] != "[img]https://img.example.com/i/"+id+".png[/img]" || links["url"] != img["url"] {
		t.Fatalf("links: %v", links)
	}
	if src := img["source"].(map[string]any); src["kind"] != "web" {
		t.Fatalf("source: %v", src)
	}
	if resp.Header.Get("Location") != "/api/v1/images/"+id {
		t.Fatalf("Location: %q", resp.Header.Get("Location"))
	}
	for _, k := range []string{"createdAt", "sha256", "hasGps", "hasThumb", "broken", "trashedAt"} {
		if _, ok := img[k]; !ok {
			t.Errorf("missing field %q", k)
		}
	}
	if _, err := time.Parse(time.RFC3339, img["createdAt"].(string)); err != nil || !strings.HasSuffix(img["createdAt"].(string), "Z") {
		t.Fatalf("createdAt %v", img["createdAt"])
	}

	resp, body = h.uploadWeb(data)
	wantStatus(t, resp, body, 200)
	dup := decode[imageJSON](t, body)
	if !dup.Duplicate || dup.ID != id {
		t.Fatalf("duplicate: %s", body)
	}
	// fetch and list
	resp, body = h.web("GET", "/api/v1/images/"+id, nil)
	wantStatus(t, resp, body, 200)
	resp, body = h.web("GET", "/api/v1/images", nil)
	wantStatus(t, resp, body, 200)
	list := decode[struct {
		Items      []imageJSON `json:"items"`
		NextCursor *string     `json:"nextCursor"`
	}](t, body)
	if len(list.Items) != 1 || list.NextCursor != nil {
		t.Fatalf("list: %s", body)
	}
	if tmp, _ := os.ReadDir(filepath.Join(h.dir.Path(), "tmp")); len(tmp) != 0 {
		t.Fatalf("temp files left: %d", len(tmp))
	}
}

func TestUploadRejections(t *testing.T) {
	h := newHarness(t, harnessOpts{})
	for name, payload := range map[string]string{
		"svg":  `<svg xmlns="http://www.w3.org/2000/svg" onload="alert(1)"><script>alert(1)</script></svg>`,
		"html": `<!doctype html><script>alert(1)</script>`,
		"text": "hello",
	} {
		resp, body := h.uploadWeb([]byte(payload))
		if resp.StatusCode != 415 {
			t.Errorf("%s: %d %s", name, resp.StatusCode, body)
			continue
		}
		wantProblem(t, resp, body, 415, "unsupported-format")
	}
	resp, body := h.uploadWeb(append([]byte("\x89PNG\r\n\x1a\n"), bytes.Repeat([]byte{7}, 50)...))
	wantProblem(t, resp, body, 422, "invalid-image")

	// 50 MP is fixed and no longer a setting
	resp, body = h.jsonReq("PATCH", "/api/v1/settings", map[string]any{"maxPixels": 1000000}, h.asWeb()...)
	wantProblem(t, resp, body, 400, "validation")
	resp, body = h.uploadWeb(pngHeader(t, 8000, 6251))
	wantProblem(t, resp, body, 422, "too-many-pixels")
}

func TestUploadTooLarge(t *testing.T) {
	h := newHarness(t, harnessOpts{})
	resp, body := h.web("PATCH", "/api/v1/settings", map[string]any{"maxUploadBytes": 2048})
	wantStatus(t, resp, body, 200)

	// slightly over the limit: the core limit triggers mid-stream
	resp, body = h.uploadWeb(append(makePNG(t, 3), bytes.Repeat([]byte{0}, 3000)...))
	wantProblem(t, resp, body, 413, "too-large")
	// way over limit + framing slack: the body reader cap triggers
	resp, body = h.uploadWeb(append(makePNG(t, 3), bytes.Repeat([]byte{0}, 300<<10)...))
	wantProblem(t, resp, body, 413, "too-large")
	if tmp, _ := os.ReadDir(filepath.Join(h.dir.Path(), "tmp")); len(tmp) != 0 {
		t.Fatalf("temp files left after 413: %d", len(tmp))
	}
	if st, _ := h.svc.Status(context.Background(), coreAdmin(h)); st.Images.Active != 0 {
		t.Fatal("rejected upload created a record")
	}
}

func TestMultipartRules(t *testing.T) {
	h := newHarness(t, harnessOpts{})
	png := makePNG(t, 11)
	do := func(parts ...part) (*http.Response, []byte) {
		body, ct := multipartBody(parts...)
		return h.do("POST", "/api/v1/images", body, append(h.asWeb(), withHeader("Content-Type", ct))...)
	}
	file := part{name: "file", filename: "a.png", data: png}
	_, folderBody := h.web("POST", "/api/v1/folders", map[string]any{"name": "uploads"})
	folder := decode[map[string]any](t, folderBody)["id"].(string)

	// no file part
	resp, body := do(part{name: "folder", data: []byte(folder)})
	wantProblem(t, resp, body, 400, "validation")
	// misnamed file field
	resp, body = do(part{name: "image", filename: "a.png", data: png})
	wantProblem(t, resp, body, 400, "validation")
	// two file parts
	resp, body = do(file, part{name: "file", filename: "b.png", data: makePNG(t, 12)})
	wantProblem(t, resp, body, 400, "validation")
	// too many parts
	many := []part{file}
	for i := 0; i < 8; i++ {
		many = append(many, part{name: fmt.Sprintf("x%d", i), data: []byte("1")})
	}
	resp, body = do(many...)
	wantProblem(t, resp, body, 400, "validation")
	// an unknown folder is 404, never created implicitly
	resp, body = do(file, part{name: "folder", data: []byte("0000000000000000")})
	wantProblem(t, resp, body, 404, "not-found")
	// not multipart at all
	resp, body = h.do("POST", "/api/v1/images", bytes.NewReader(png), append(h.asWeb(), withHeader("Content-Type", "image/png"))...)
	wantProblem(t, resp, body, 400, "validation")
	resp, body = h.do("POST", "/api/v1/images", strings.NewReader("--x\r\ngarbage"), append(h.asWeb(), withHeader("Content-Type", "multipart/form-data; boundary=x"))...)
	if resp.StatusCode != 400 {
		t.Fatalf("garbage multipart: %d %s", resp.StatusCode, body)
	}

	// folder before the file, unknown text parts ignored
	resp, body = do(part{name: "folder", data: []byte(folder)}, part{name: "note", data: []byte("ignored")}, file)
	wantStatus(t, resp, body, 201)
	if v := decode[imageJSON](t, body); v.FolderID == nil || *v.FolderID != folder {
		t.Fatalf("folder before file: %s", body)
	}
	// folder after the file; blanks mean the root
	resp, body = do(part{name: "file", filename: "b.png", data: makePNG(t, 13)}, part{name: "folder", data: []byte(folder)})
	wantStatus(t, resp, body, 201)
	if v := decode[imageJSON](t, body); v.FolderID == nil || *v.FolderID != folder {
		t.Fatalf("folder after file: %s", body)
	}
	resp, body = do(part{name: "file", filename: "c.png", data: makePNG(t, 14)}, part{name: "folder", data: []byte("  ")})
	wantStatus(t, resp, body, 201)
	if v := decode[imageJSON](t, body); v.FolderID != nil {
		t.Fatalf("blank folder must mean root: %s", body)
	}
	// legacy tag parts are ignored and do not count against the part limit
	tagged := []part{file}
	for i := 0; i < 20; i++ {
		tagged = append(tagged, part{name: "tag", data: []byte(fmt.Sprint("t", i))})
	}
	resp, body = do(tagged...)
	wantStatus(t, resp, body, 201)
	if strings.Contains(string(body), `"tags"`) {
		t.Fatalf("tags must be gone: %s", body)
	}
	// an over-long folder value is a rejection
	resp, body = do(part{name: "file", filename: "d.png", data: makePNG(t, 15)}, part{name: "folder", data: []byte(strings.Repeat("x", 100))})
	wantProblem(t, resp, body, 400, "validation")
	if tmp, _ := os.ReadDir(filepath.Join(h.dir.Path(), "tmp")); len(tmp) != 0 {
		t.Fatalf("temp files left: %d", len(tmp))
	}
}

func TestFilenameSanitizedInResponse(t *testing.T) {
	h := newHarness(t, harnessOpts{})
	body, ct := multipartBody(part{name: "file", filename: `..\..\evil/<img src=x onerror=alert(1)>.png`, data: makePNG(t, 21)})
	resp, out := h.do("POST", "/api/v1/images", body, append(h.asWeb(), withHeader("Content-Type", ct))...)
	wantStatus(t, resp, out, 201)
	m := decode[map[string]any](t, out)
	if strings.ContainsAny(m["filename"].(string), `/\`) {
		t.Fatalf("path separators survived: %v", m["filename"])
	}
	html := m["links"].(map[string]any)["html"].(string)
	if strings.Contains(html, "<img src=x") {
		t.Fatalf("html link not escaped: %s", html)
	}
}

func TestTokenScopeMatrix(t *testing.T) {
	h := newHarness(t, harnessOpts{})
	_, upSecret := h.createToken("upload")
	_, upSecret2 := h.createToken("upload")
	_, mgSecret := h.createToken("manage")

	webImg := h.mustUploadWeb(31)
	resp, body := h.upload(makePNG(t, 32), []reqOpt{withBearer(upSecret)})
	wantStatus(t, resp, body, 201)
	own := decode[imageJSON](t, body)
	if own.Source.Kind != "token" || !strings.HasPrefix(own.Source.TokenName, "t upload") {
		t.Fatalf("source: %+v", own.Source)
	}
	resp, body = h.upload(makePNG(t, 33), []reqOpt{withBearer(mgSecret)})
	wantStatus(t, resp, body, 201)
	mgImg := decode[imageJSON](t, body)

	get := func(secret, path string) (*http.Response, []byte) { return h.do("GET", path, nil, withBearer(secret)) }
	send := func(secret, method, path string, v any) (*http.Response, []byte) {
		return h.jsonReq(method, path, v, withBearer(secret))
	}

	// upload token: own image only
	resp, body = get(upSecret, "/api/v1/images/"+own.ID)
	wantStatus(t, resp, body, 200)
	resp, body = get(upSecret, "/api/v1/images/"+webImg.ID)
	wantProblem(t, resp, body, 404, "not-found")
	resp, body = get(upSecret2, "/api/v1/images/"+own.ID)
	wantProblem(t, resp, body, 404, "not-found")
	resp, body = get(upSecret, "/api/v1/images")
	wantStatus(t, resp, body, 200)
	if l := decode[struct{ Items []imageJSON }](t, body); len(l.Items) != 1 || l.Items[0].ID != own.ID {
		t.Fatalf("upload token sees: %s", body)
	}
	resp, body = get(upSecret, "/api/v1/images/"+own.ID+"/raw")
	wantStatus(t, resp, body, 200)
	resp, body = get(upSecret, "/api/v1/images/"+webImg.ID+"/raw")
	wantProblem(t, resp, body, 404, "not-found")

	// upload token cannot mutate images or folders, but may list folders
	resp, body = send(upSecret, "PATCH", "/api/v1/images/"+own.ID, map[string]any{"filename": "x.png"})
	wantProblem(t, resp, body, 403, "forbidden")
	resp, body = send(upSecret, "DELETE", "/api/v1/images/"+own.ID, nil)
	wantProblem(t, resp, body, 403, "forbidden")
	resp, body = send(upSecret, "POST", "/api/v1/images/"+own.ID+"/restore", nil)
	wantProblem(t, resp, body, 403, "forbidden")
	resp, body = send(upSecret, "POST", "/api/v1/images/"+own.ID+"/purge", nil)
	wantProblem(t, resp, body, 403, "forbidden")
	resp, body = send(upSecret, "POST", "/api/v1/images/batch", map[string]any{"action": "trash", "ids": []string{own.ID}})
	wantProblem(t, resp, body, 403, "forbidden")
	resp, body = send(upSecret, "POST", "/api/v1/folders", map[string]any{"name": "x"})
	wantProblem(t, resp, body, 403, "forbidden")
	resp, body = get(upSecret, "/api/v1/folders")
	wantStatus(t, resp, body, 200)

	// no token can touch tokens, settings, status, operations, audit
	for _, secret := range []string{upSecret, mgSecret} {
		for _, c := range []struct{ method, path string }{
			{"GET", "/api/v1/tokens"}, {"POST", "/api/v1/tokens"}, {"DELETE", "/api/v1/tokens/1"},
			{"GET", "/api/v1/settings"}, {"PATCH", "/api/v1/settings"}, {"GET", "/api/v1/status"},
			{"POST", "/api/v1/checks"}, {"POST", "/api/v1/snapshots"}, {"GET", "/api/v1/audit"}, {"GET", "/api/v1/audit/actions"},
		} {
			var payload any
			if c.method == "POST" || c.method == "PATCH" {
				payload = map[string]any{"name": "x", "scopes": []string{"upload"}, "mode": "quick", "dedupe": false}
			}
			resp, body = send(secret, c.method, c.path, payload)
			if resp.StatusCode != 403 {
				t.Errorf("%s %s with token: %d %s", c.method, c.path, resp.StatusCode, body)
			}
		}
	}
	if resp, body = h.web("GET", "/api/v1/settings", nil); decode[map[string]any](t, body)["dedupe"] != true {
		t.Fatal("a token changed the settings")
	}

	// manage token: full image control, including moves
	resp, body = send(mgSecret, "PATCH", "/api/v1/images/"+mgImg.ID, map[string]any{"filename": "managed.png"})
	wantStatus(t, resp, body, 200)
	if v := decode[imageJSON](t, body); v.URL == "" {
		t.Fatalf("patch: %s", body)
	}
	resp, body = send(mgSecret, "DELETE", "/api/v1/images/"+own.ID, nil)
	wantStatus(t, resp, body, 204)
	resp, body = get(upSecret, "/api/v1/images/"+own.ID)
	wantProblem(t, resp, body, 404, "not-found") // trashed: gone for the upload token
	resp, body = send(mgSecret, "POST", "/api/v1/images/"+own.ID+"/restore", nil)
	wantStatus(t, resp, body, 200)
	resp, body = send(mgSecret, "DELETE", "/api/v1/images/"+own.ID, nil)
	wantStatus(t, resp, body, 204)
	resp, body = send(mgSecret, "POST", "/api/v1/images/"+own.ID+"/purge", nil)
	wantStatus(t, resp, body, 204)

	// revoked tokens stop working at once
	list := func() []struct {
		ID        int64   `json:"id"`
		RevokedAt *string `json:"revokedAt"`
	} {
		_, b := h.web("GET", "/api/v1/tokens", nil)
		return decode[struct {
			Items []struct {
				ID        int64   `json:"id"`
				RevokedAt *string `json:"revokedAt"`
			}
		}](t, b).Items
	}
	toks := list()
	if len(toks) != 3 {
		t.Fatalf("tokens: %d", len(toks))
	}
	for _, tk := range toks {
		resp, body = h.web("DELETE", fmt.Sprintf("/api/v1/tokens/%d", tk.ID), nil)
		wantStatus(t, resp, body, 204)
		resp, body = h.web("DELETE", fmt.Sprintf("/api/v1/tokens/%d", tk.ID), nil) // idempotent
		wantStatus(t, resp, body, 204)
	}
	resp, body = get(mgSecret, "/api/v1/images")
	wantProblem(t, resp, body, 401, "unauthorized")
	for _, tk := range list() {
		if tk.RevokedAt == nil {
			t.Fatal("revoked token lost its revokedAt")
		}
	}
	resp, body = h.web("DELETE", "/api/v1/tokens/99999", nil)
	wantProblem(t, resp, body, 404, "not-found")
	resp, body = h.web("DELETE", "/api/v1/tokens/abc", nil)
	wantProblem(t, resp, body, 404, "not-found")
}

func TestTokenCreateValidation(t *testing.T) {
	h := newHarness(t, harnessOpts{})
	for _, bad := range []map[string]any{
		{"name": "", "scopes": []string{"upload"}},
		{"name": "x", "scopes": []string{}},
		{"name": "x", "scopes": []string{"upload"}, "expiresInDays": 0},
		{"name": "x", "scopes": []string{"upload"}, "unknownField": true},
	} {
		resp, body := h.web("POST", "/api/v1/tokens", bad)
		wantProblem(t, resp, body, 400, "validation")
	}
	// unknown and removed scopes are 422; read no longer exists
	for _, scopes := range [][]string{{"admin"}, {"read"}, {"upload", "read"}} {
		resp, body := h.web("POST", "/api/v1/tokens", map[string]any{"name": "x", "scopes": scopes})
		wantProblem(t, resp, body, 422, "invalid-scope")
	}
	resp, body := h.web("POST", "/api/v1/tokens", map[string]any{"name": "PicGo", "scopes": []string{"upload"}, "expiresInDays": 30})
	wantStatus(t, resp, body, 201)
	out := decode[struct {
		Token  map[string]any `json:"token"`
		Secret string         `json:"secret"`
	}](t, body)
	if !strings.HasPrefix(out.Secret, "pb_") || len(out.Secret) != 35 || out.Token["prefix"] != out.Secret[:9] || out.Token["expiresAt"] == nil {
		t.Fatalf("created: %s", body)
	}
	resp, body = h.web("GET", "/api/v1/tokens", nil)
	if strings.Contains(string(body), out.Secret) || strings.Contains(string(body), "secret") {
		t.Fatalf("token listing leaks the secret: %s", body)
	}
}

func TestCSRF(t *testing.T) {
	h := newHarness(t, harnessOpts{})
	img := h.mustUploadWeb(41)
	_, secret := h.createToken("manage")
	cookie := withHeader("Cookie", h.cookie)

	// session POST without the client header
	resp, body := h.jsonReq("POST", "/api/v1/tokens", map[string]any{"name": "x", "scopes": []string{"upload"}}, cookie)
	wantProblem(t, resp, body, 403, "forbidden")
	// cross-site fetch metadata
	for _, hdr := range [][2]string{{"Sec-Fetch-Site", "cross-site"}, {"Sec-Fetch-Site", "same-site"}} {
		resp, body = h.jsonReq("POST", "/api/v1/tokens", map[string]any{"name": "x", "scopes": []string{"upload"}},
			append(h.asWeb(), withHeader(hdr[0], hdr[1]))...)
		wantProblem(t, resp, body, 403, "forbidden")
	}
	// cross-origin Origin header
	resp, body = h.jsonReq("POST", "/api/v1/tokens", map[string]any{"name": "x", "scopes": []string{"upload"}},
		append(h.asWeb(), withHeader("Origin", "https://evil.example"))...)
	wantProblem(t, resp, body, 403, "forbidden")
	resp, body = h.jsonReq("DELETE", "/api/v1/images/"+img.ID, nil, append(h.asWeb(), withHeader("Sec-Fetch-Site", "cross-site"))...)
	wantProblem(t, resp, body, 403, "forbidden")
	if _, b := h.web("GET", "/api/v1/images/"+img.ID, nil); decode[imageJSON](t, b).State != "active" {
		t.Fatal("cross-site delete took effect")
	}
	// same-origin browser request works
	resp, body = h.jsonReq("POST", "/api/v1/tokens", map[string]any{"name": "ok", "scopes": []string{"upload"}},
		append(h.asWeb(), withHeader("Sec-Fetch-Site", "same-origin"), withHeader("Origin", h.ts.URL))...)
	wantStatus(t, resp, body, 201)
	// the configured public origin is trusted even when Host differs
	resp, body = h.jsonReq("POST", "/api/v1/tokens", map[string]any{"name": "ok2", "scopes": []string{"upload"}},
		append(h.asWeb(), withHeader("Origin", "https://pics.example.com"))...)
	wantStatus(t, resp, body, 201)
	// safe methods are not subject to the header
	resp, body = h.do("GET", "/api/v1/tokens", nil, cookie)
	wantStatus(t, resp, body, 200)
	// token requests need neither header
	resp, body = h.jsonReq("PATCH", "/api/v1/images/"+img.ID, map[string]any{"filename": "csrf.png"}, withBearer(secret))
	wantStatus(t, resp, body, 200)
	resp, body = h.upload(makePNG(t, 42), []reqOpt{withBearer(secret)})
	wantStatus(t, resp, body, 201)
	// a bearer token wins over a cookie, so no CSRF header is demanded when both are present
	resp, body = h.jsonReq("DELETE", "/api/v1/images/"+img.ID, nil, withBearer(secret), cookie)
	wantStatus(t, resp, body, 204)
	// login/setup are not session-authenticated and need only the cross-origin protection
	resp, body = h.jsonReq("POST", "/api/v1/session", map[string]string{"username": "admin", "password": "correct horse battery"},
		withHeader("Sec-Fetch-Site", "cross-site"))
	wantProblem(t, resp, body, 403, "forbidden")
}

func TestPublicImageHeadersAnd304(t *testing.T) {
	h := newHarness(t, harnessOpts{})
	data := makePNG(t, 51)
	resp, body := h.uploadWeb(data)
	wantStatus(t, resp, body, 201)
	img := decode[imageJSON](t, body)
	path := "/i/" + img.ID + ".png"

	resp, body = h.do("GET", path, nil)
	wantStatus(t, resp, body, 200)
	if !bytes.Equal(body, data) {
		t.Fatal("body differs from upload")
	}
	hd := resp.Header
	check := map[string]string{
		"Content-Type":                 "image/png",
		"Cache-Control":                "public, max-age=31536000, immutable",
		"ETag":                         `"` + img.SHA256 + `"`,
		"Content-Security-Policy":      "default-src 'none'; sandbox",
		"X-Content-Type-Options":       "nosniff",
		"Cross-Origin-Resource-Policy": "cross-origin",
		"Content-Disposition":          `inline; filename="` + img.ID + `.png"`,
	}
	for k, want := range check {
		if got := hd.Get(k); got != want {
			t.Errorf("%s = %q, want %q", k, got, want)
		}
	}
	if hd.Get("Last-Modified") == "" || hd.Get("X-PicBerth-Version") == "" || hd.Get("Accept-Ranges") != "bytes" {
		t.Errorf("headers: %v", hd)
	}
	if strings.Contains(hd.Get("Content-Disposition"), "shot") {
		t.Error("original filename exposed publicly")
	}
	if len(hd.Values("Set-Cookie")) != 0 {
		t.Error("public images must not set cookies")
	}

	// conditional requests
	etag := hd.Get("ETag")
	for _, inm := range []string{etag, "W/" + etag, `"nope", ` + etag, "*"} {
		resp, body = h.do("GET", path, nil, withHeader("If-None-Match", inm))
		if resp.StatusCode != 304 || len(body) != 0 {
			t.Errorf("If-None-Match %q: %d, %d body bytes", inm, resp.StatusCode, len(body))
		}
		if resp.Header.Get("ETag") != etag || resp.Header.Get("Cache-Control") != check["Cache-Control"] {
			t.Errorf("304 headers: %v", resp.Header)
		}
	}
	resp, body = h.do("GET", path, nil, withHeader("If-None-Match", `"other"`))
	wantStatus(t, resp, body, 200)
	// 304 is answered without opening the file: delete the file and it still works...
	os.Remove(filepath.Join(h.dir.Path(), "objects", img.ID[0:2], img.ID[2:4], img.ID+".png"))
	resp, _ = h.do("GET", path, nil, withHeader("If-None-Match", etag))
	if resp.StatusCode != 304 {
		t.Fatalf("304 needed the file: %d", resp.StatusCode)
	}
	// ...while a full request for the missing file is a clean 404
	resp, _ = h.do("GET", path, nil)
	if resp.StatusCode != 404 {
		t.Fatalf("missing file: %d", resp.StatusCode)
	}
}

func TestPublicImageRangeHeadAndWrongExt(t *testing.T) {
	h := newHarness(t, harnessOpts{})
	data := makePNG(t, 52)
	img := h.mustUploadWeb(52)
	_ = data
	resp, body := h.do("GET", "/i/"+img.ID+".png", nil, withHeader("Range", "bytes=0-7"))
	wantStatus(t, resp, body, 206)
	if len(body) != 8 || !bytes.HasPrefix(body, []byte("\x89PNG")) {
		t.Fatalf("range body %q", body)
	}
	resp, body = h.do("HEAD", "/i/"+img.ID+".png", nil)
	wantStatus(t, resp, body, 200)
	if len(body) != 0 || resp.Header.Get("Content-Length") == "" || resp.Header.Get("Content-Type") != "image/png" {
		t.Fatalf("HEAD: %v", resp.Header)
	}
	for _, p := range []string{"/i/" + img.ID + ".jpg", "/i/" + img.ID + ".PNG", "/i/" + img.ID, "/i/" + strings.ToUpper(img.ID) + ".png", "/i/" + img.ID + ".", "/i/.png", "/i/", "/i/" + img.ID + ".png/extra"} {
		resp, _ := h.do("GET", p, nil)
		if resp.StatusCode != 404 {
			t.Errorf("GET %s: %d, want 404", p, resp.StatusCode)
		}
	}
	resp, _ = h.do("POST", "/i/"+img.ID+".png", nil)
	if resp.StatusCode == 200 {
		t.Error("POST to an image URL succeeded")
	}
}

func TestPublicImageAlwaysPublic(t *testing.T) {
	h := newHarness(t, harnessOpts{})
	img := h.mustUploadWeb(61)
	resp, body := h.do("GET", "/i/"+img.ID+".png", nil)
	wantStatus(t, resp, body, 200)
	// a session cookie changes nothing on the public route
	resp, body = h.do("GET", "/i/"+img.ID+".png", nil, withHeader("Cookie", h.cookie))
	wantStatus(t, resp, body, 200)
	resp, body = h.do("GET", "/api/v1/images/"+img.ID+"/raw", nil, withHeader("Cookie", h.cookie))
	wantStatus(t, resp, body, 200)
	if resp.Header.Get("Cache-Control") != "private, no-store" || resp.Header.Get("Cross-Origin-Resource-Policy") != "same-origin" ||
		resp.Header.Get("Content-Type") != "image/png" || resp.Header.Get("X-Content-Type-Options") != "nosniff" ||
		!strings.Contains(resp.Header.Get("Content-Security-Policy"), "sandbox") {
		t.Fatalf("raw headers: %v", resp.Header)
	}
	if !bytes.HasPrefix(body, []byte("\x89PNG")) {
		t.Fatal("raw body")
	}
	resp, body = h.do("GET", "/api/v1/images/"+img.ID+"/raw", nil)
	wantProblem(t, resp, body, 401, "unauthorized")

	// trashed and purged images are 404 on the public route, and an unknown id answers identically
	h.web("DELETE", "/api/v1/images/"+img.ID, nil)
	resp, _ = h.do("GET", "/i/"+img.ID+".png", nil)
	if resp.StatusCode != 404 {
		t.Fatalf("trashed image still public: %d", resp.StatusCode)
	}
	h.web("POST", "/api/v1/images/"+img.ID+"/restore", nil)
	resp, _ = h.do("GET", "/i/"+img.ID+".png", nil)
	if resp.StatusCode != 200 {
		t.Fatalf("restored image not public: %d", resp.StatusCode)
	}
	// unknown JSON fields on PATCH are rejected
	resp, body = h.web("PATCH", "/api/v1/images/"+img.ID, map[string]any{"visibility": "private"})
	wantProblem(t, resp, body, 400, "validation")
	resp, body = h.web("PATCH", "/api/v1/images/"+img.ID, map[string]any{"tags": []string{"x"}})
	wantProblem(t, resp, body, 400, "validation")
	resp, body = h.web("PATCH", "/api/v1/images/"+img.ID, map[string]any{})
	wantProblem(t, resp, body, 400, "validation")
}

func TestThumbnailEndpoint(t *testing.T) {
	h := newHarness(t, harnessOpts{})
	img := h.mustUploadWeb(71)
	resp, body := h.web("GET", "/api/v1/images/"+img.ID+"/thumb", nil)
	wantProblem(t, resp, body, 404, "not-found") // not generated yet: client falls back to raw
	h.svc.ProcessThumbnails(context.Background())
	resp, body = h.do("GET", "/api/v1/images/"+img.ID+"/thumb", nil, withHeader("Cookie", h.cookie))
	wantStatus(t, resp, body, 200)
	if resp.Header.Get("Content-Type") != "image/png" || resp.Header.Get("Cache-Control") != "private, no-store" || !bytes.HasPrefix(body, []byte("\x89PNG")) {
		t.Fatalf("thumb: %v", resp.Header)
	}
	_, body = h.web("GET", "/api/v1/images/"+img.ID, nil)
	if !decode[imageJSON](t, body).HasThumb {
		t.Fatal("hasThumb false")
	}
}

func TestTrashRestorePurgeFlow(t *testing.T) {
	h := newHarness(t, harnessOpts{})
	img := h.mustUploadWeb(81)
	pub := "/i/" + img.ID + ".png"

	resp, body := h.web("POST", "/api/v1/images/"+img.ID+"/purge", nil)
	wantProblem(t, resp, body, 409, "conflict") // only from the trash
	resp, body = h.web("POST", "/api/v1/images/"+img.ID+"/restore", nil)
	wantProblem(t, resp, body, 409, "conflict")

	resp, body = h.web("DELETE", "/api/v1/images/"+img.ID, nil)
	wantStatus(t, resp, body, 204)
	resp, body = h.web("DELETE", "/api/v1/images/"+img.ID, nil)
	wantStatus(t, resp, body, 204) // idempotent
	if resp, _ = h.do("GET", pub, nil); resp.StatusCode != 404 {
		t.Fatalf("trashed image still public: %d", resp.StatusCode)
	}
	resp, body = h.web("GET", "/api/v1/images", nil)
	if l := decode[struct{ Items []imageJSON }](t, body); len(l.Items) != 0 {
		t.Fatalf("trashed image in active list: %s", body)
	}
	resp, body = h.web("GET", "/api/v1/images?state=trashed", nil)
	l := decode[struct{ Items []imageJSON }](t, body)
	if len(l.Items) != 1 || l.Items[0].State != "trashed" {
		t.Fatalf("trash list: %s", body)
	}
	resp, body = h.web("GET", "/api/v1/images/"+img.ID+"/raw", nil)
	wantStatus(t, resp, body, 200) // the admin can still see trashed bytes

	resp, body = h.web("POST", "/api/v1/images/"+img.ID+"/restore", nil)
	wantStatus(t, resp, body, 200)
	if decode[imageJSON](t, body).State != "active" {
		t.Fatal("restore did not activate")
	}
	if resp, _ = h.do("GET", pub, nil); resp.StatusCode != 200 {
		t.Fatalf("restored image not public: %d", resp.StatusCode)
	}

	h.web("DELETE", "/api/v1/images/"+img.ID, nil)
	resp, body = h.web("POST", "/api/v1/images/"+img.ID+"/purge", nil)
	wantStatus(t, resp, body, 204)
	resp, body = h.web("GET", "/api/v1/images/"+img.ID, nil)
	wantProblem(t, resp, body, 404, "not-found")
	resp, body = h.web("POST", "/api/v1/images/"+img.ID+"/purge", nil)
	wantProblem(t, resp, body, 404, "not-found")
	resp, body = h.web("DELETE", "/api/v1/images/"+img.ID, nil)
	wantProblem(t, resp, body, 404, "not-found")
	if resp, _ = h.do("GET", pub, nil); resp.StatusCode != 404 {
		t.Fatal("purged image reachable")
	}

	// batch
	a, b := h.mustUploadWeb(82), h.mustUploadWeb(83)
	resp, body = h.web("POST", "/api/v1/images/batch", map[string]any{"action": "trash", "ids": []string{a.ID, b.ID, "00000000000000000000"}})
	wantStatus(t, resp, body, 200)
	res := decode[struct {
		Results []struct {
			ID   string `json:"id"`
			OK   bool   `json:"ok"`
			Code string `json:"code"`
		}
	}](t, body).Results
	if len(res) != 3 || !res[0].OK || !res[1].OK || res[2].OK || res[2].Code != "not-found" {
		t.Fatalf("batch: %s", body)
	}
	resp, body = h.web("POST", "/api/v1/images/batch", map[string]any{"action": "bogus", "ids": []string{a.ID}})
	wantProblem(t, resp, body, 400, "validation")
}

func TestPathTraversal(t *testing.T) {
	h := newHarness(t, harnessOpts{})
	img := h.mustUploadWeb(91)
	// plant a secret next to the data dir to prove nothing escapes
	secretFile := filepath.Join(filepath.Dir(h.dir.Path()), "secret.txt")
	os.WriteFile(secretFile, []byte("TOP SECRET"), 0o600)

	evil := []string{
		"/i/../../secret.txt", "/i/..%2f..%2fsecret.txt", "/i/%2e%2e/%2e%2e/secret.txt", "/i/..%5c..%5csecret.txt",
		"/i/" + img.ID + ".png/../../../secret.txt", "/i/" + img.ID + "%00.png", "/i/....//....//secret.txt",
		"/i/..%2F..%2Fdata%2Fpicberth.db", "/i/%2e%2e%2f%2e%2e%2fdata%2fFORMAT", "/i/c:%5cwindows%5cwin.ini",
		"/i/..;/..;/secret.txt", "/i/" + img.ID + ".png%20", "/i/" + img.ID + ".p%6eg%2f..%2f..%2fFORMAT",
	}
	for _, p := range evil {
		resp, body := h.do("GET", p, nil)
		if strings.Contains(string(body), "TOP SECRET") || strings.Contains(string(body), "picberth-data") || strings.Contains(string(body), "SQLite") {
			t.Fatalf("GET %s leaked file content", p)
		}
		if resp.StatusCode != 404 {
			t.Errorf("GET %s: %d, want 404", p, resp.StatusCode)
		}
	}
	apiEvil := []string{
		"/api/v1/images/..%2f..%2fsecret.txt", "/api/v1/images/%2e%2e%2f%2e%2e%2fsecret.txt", "/api/v1/images/..%5c..%5csecret.txt",
		"/api/v1/images/../../../secret.txt", "/api/v1/images/" + img.ID + "%2f..%2f..%2fsecret.txt",
		"/api/v1/images/..%2f..%2fdata%2fFORMAT/raw", "/api/v1/images/..%2f..%2fdata%2fFORMAT/thumb",
		"/api/v1/images/" + img.ID + "/../../secret.txt",
	}
	for _, p := range apiEvil {
		for _, m := range []string{"GET", "DELETE"} {
			resp, body := h.do(m, p, nil, h.asWeb()...)
			if strings.Contains(string(body), "TOP SECRET") || strings.Contains(string(body), "picberth-data") {
				t.Fatalf("%s %s leaked file content", m, p)
			}
			if resp.StatusCode != 404 {
				t.Errorf("%s %s: %d %s, want 404", m, p, resp.StatusCode, body)
			}
		}
	}
	// the image and the secret are untouched
	if b, err := os.ReadFile(secretFile); err != nil || string(b) != "TOP SECRET" {
		t.Fatal("secret file was modified")
	}
	if resp, _ := h.do("GET", "/i/"+img.ID+".png", nil); resp.StatusCode != 200 {
		t.Fatal("legit image no longer served")
	}
}

func TestHostHeaderInjectionDoesNotChangeURLs(t *testing.T) {
	h := newHarness(t, harnessOpts{})
	data := makePNG(t, 101)
	resp, body := h.upload(data, append(h.asWeb(),
		withHeader("X-Forwarded-Host", "evil.example"), withHeader("X-Forwarded-Proto", "http"),
		withHeader("X-Forwarded-For", "6.6.6.6"), withHeader("Forwarded", "host=evil.example;proto=http")))
	wantStatus(t, resp, body, 201)
	img := decode[map[string]any](t, body)
	id := img["id"].(string)
	want := "https://img.example.com/i/" + id + ".png"
	if img["url"] != want || img["links"].(map[string]any)["markdown"] != "![shot]("+want+")" {
		t.Fatalf("URLs follow request headers: %s", body)
	}
	req, _ := http.NewRequest("GET", h.ts.URL+"/api/v1/images/"+id, nil)
	req.Host = "evil.example"
	req.Header.Set("Cookie", h.cookie)
	req.Header.Set("X-Forwarded-Host", "evil.example")
	r2, err := h.client.Do(req)
	if err != nil {
		t.Fatal(err)
	}
	b2, _ := readAll(r2)
	if strings.Contains(string(b2), "evil.example") || !strings.Contains(string(b2), want) {
		t.Fatalf("Host header changed generated URLs: %s", b2)
	}
	// cookies keep their configured attributes (no Domain, Secure follows the configured URL)
	resp, _ = h.jsonReq("POST", "/api/v1/session", map[string]string{"username": "admin", "password": "correct horse battery"},
		withHeader("X-Forwarded-Proto", "http"), withHeader("X-Forwarded-Host", "evil.example"))
	for _, c := range resp.Cookies() {
		if c.Domain != "" || !c.Secure || c.Name != "__Host-pb_session" {
			t.Fatalf("cookie follows request headers: %+v", c)
		}
	}
}

func readAll(r *http.Response) ([]byte, error) {
	defer r.Body.Close()
	var buf bytes.Buffer
	_, err := buf.ReadFrom(r.Body)
	return buf.Bytes(), err
}

func TestLoginRateLimitHTTP(t *testing.T) {
	h := newHarness(t, harnessOpts{keepLimits: true, skipAdmin: true})
	h.svc.CreateAdmin(context.Background(), "admin", "correct horse battery")
	bad := map[string]string{"username": "admin", "password": "wrong password!"}
	for i := 0; i < 5; i++ {
		resp, body := h.jsonReq("POST", "/api/v1/session", bad)
		wantProblem(t, resp, body, 401, "unauthorized")
	}
	resp, body := h.jsonReq("POST", "/api/v1/session", map[string]string{"username": "admin", "password": "correct horse battery"})
	wantProblem(t, resp, body, 429, "rate-limited")
	ra := resp.Header.Get("Retry-After")
	if n := atoi(ra); n < 1 {
		t.Fatalf("Retry-After %q", ra)
	}
}

func atoi(s string) int {
	n := 0
	fmt.Sscanf(s, "%d", &n)
	return n
}

func TestUploadRateLimitHTTP(t *testing.T) {
	h := newHarness(t, harnessOpts{})
	now := time.Now()
	h.svc.Now = func() time.Time { return now }
	// the upload rate limit is a fixed 60 per minute and no longer a setting
	resp, body := h.web("PATCH", "/api/v1/settings", map[string]any{"uploadsPerMinute": 2})
	wantProblem(t, resp, body, 400, "validation")
	for i := 0; i < 60; i++ {
		resp, body = h.uploadWeb(makePNG(t, 200+i))
		wantStatus(t, resp, body, 201)
	}
	resp, body = h.uploadWeb(makePNG(t, 210))
	wantProblem(t, resp, body, 429, "rate-limited")
	if atoi(resp.Header.Get("Retry-After")) < 1 {
		t.Fatalf("Retry-After %q", resp.Header.Get("Retry-After"))
	}
	now = now.Add(time.Second)
	resp, body = h.uploadWeb(makePNG(t, 260))
	wantStatus(t, resp, body, 201)
	resp, body = h.uploadWeb(makePNG(t, 261))
	wantProblem(t, resp, body, 429, "rate-limited")
}

func TestSettingsEndpoints(t *testing.T) {
	h := newHarness(t, harnessOpts{})
	resp, body := h.web("GET", "/api/v1/settings", nil)
	wantStatus(t, resp, body, 200)
	cur := decode[map[string]any](t, body)
	for _, k := range []string{"maxUploadBytes", "allowedFormats", "dedupe", "trashRetentionDays", "auditRetentionDays", "publicUrl", "imageUrl", "hardMaxUploadBytes"} {
		if _, ok := cur[k]; !ok {
			t.Errorf("settings lack %q", k)
		}
	}
	for _, gone := range []string{"maxPixels", "defaultVisibility", "uploadsPerMinute", "sessionIdleMinutes", "sessionAbsoluteHours"} {
		if _, ok := cur[gone]; ok {
			t.Errorf("settings still expose %q", gone)
		}
	}
	if cur["publicUrl"] != "https://pics.example.com" || cur["imageUrl"] != "https://img.example.com" || cur["hardMaxUploadBytes"] != float64(100<<20) {
		t.Fatalf("read-only values: %v", cur)
	}
	if cur["auditRetentionDays"] != float64(180) || cur["trashRetentionDays"] != float64(30) || cur["dedupe"] != true {
		t.Fatalf("defaults: %v", cur)
	}
	resp, body = h.web("PATCH", "/api/v1/settings", map[string]any{"dedupe": false, "trashRetentionDays": 7, "auditRetentionDays": 30, "allowedFormats": []string{"png"}})
	wantStatus(t, resp, body, 200)
	upd := decode[map[string]any](t, body)
	if upd["dedupe"] != false || upd["trashRetentionDays"] != float64(7) || upd["auditRetentionDays"] != float64(30) || fmt.Sprint(upd["allowedFormats"]) != "[png]" {
		t.Fatalf("patched: %s", body)
	}
	// echoing a whole GET response back is accepted
	resp, body = h.web("PATCH", "/api/v1/settings", upd)
	wantStatus(t, resp, body, 200)
	// read-only changes, bad values and removed keys are rejected
	for _, bad := range []map[string]any{
		{"publicUrl": "https://evil.example"}, {"hardMaxUploadBytes": 1}, {"maxUploadBytes": 1}, {"maxUploadBytes": 1 << 40},
		{"maxPixels": 1000000}, {"allowedFormats": []string{"svg"}}, {"trashRetentionDays": 400}, {"trashRetentionDays": -1},
		{"auditRetentionDays": 6}, {"auditRetentionDays": 3651}, {"uploadsPerMinute": 10}, {"sessionIdleMinutes": 10},
		{"defaultVisibility": "private"}, {"nonsense": 1}, {"dedupe": "yes"},
	} {
		resp, body = h.web("PATCH", "/api/v1/settings", bad)
		wantProblem(t, resp, body, 400, "validation")
	}
	// format restriction applies
	resp, body = h.uploadWeb(makeJPEGish())
	wantProblem(t, resp, body, 415, "unsupported-format")
	// dedupe off: same bytes twice make two images in the same folder
	a := h.uploadAndDecode(makePNG(t, 301))
	b := h.uploadAndDecode(makePNG(t, 301))
	if a.ID == b.ID || b.Duplicate {
		t.Fatal("dedupe still active after being disabled")
	}
}

func makeJPEGish() []byte { return append([]byte("\xff\xd8\xff\xe0"), bytes.Repeat([]byte{0}, 40)...) }

func (h *harness) uploadAndDecode(data []byte) imageJSON {
	h.t.Helper()
	resp, body := h.uploadWeb(data)
	if resp.StatusCode != 201 && resp.StatusCode != 200 {
		h.t.Fatalf("upload: %d %s", resp.StatusCode, body)
	}
	return decode[imageJSON](h.t, body)
}

func TestStatusChecksSnapshotsAudit(t *testing.T) {
	h := newHarness(t, harnessOpts{})
	img := h.mustUploadWeb(401)
	h.web("PATCH", "/api/v1/images/"+img.ID, map[string]any{"filename": "renamed.png"})
	_, fbody := h.web("POST", "/api/v1/folders", map[string]any{"name": "ops"})
	folder := decode[map[string]any](t, fbody)["id"].(string)
	h.web("PATCH", "/api/v1/folders/"+folder, map[string]any{"name": "ops2"})
	h.web("DELETE", "/api/v1/folders/"+folder, nil)
	h.web("DELETE", "/api/v1/images/"+img.ID, nil)
	h.web("POST", "/api/v1/images/"+img.ID+"/restore", nil)
	h.web("DELETE", "/api/v1/images/"+img.ID, nil)
	h.web("POST", "/api/v1/images/"+img.ID+"/purge", nil)
	h.createToken("upload")

	resp, body := h.web("GET", "/api/v1/status", nil)
	wantStatus(t, resp, body, 200)
	st := decode[map[string]any](t, body)
	for _, k := range []string{"version", "commit", "goVersion", "sqliteVersion", "startedAt", "uptimeSeconds", "images", "database", "disk", "dataDir", "lastSnapshot", "lastCheck", "warnings"} {
		if _, ok := st[k]; !ok {
			t.Errorf("status lacks %q", k)
		}
	}
	if !strings.HasPrefix(st["goVersion"].(string), "go") || st["dataDir"] != h.dir.Path() {
		t.Fatalf("status: %s", body)
	}

	resp, body = h.web("POST", "/api/v1/checks", map[string]string{"mode": "full"})
	wantStatus(t, resp, body, 200)
	rep := decode[map[string]any](t, body)
	if rep["mode"] != "full" || rep["ok"] != true || rep["issues"] == nil {
		t.Fatalf("check: %s", body)
	}
	resp, body = h.web("POST", "/api/v1/checks", map[string]string{"mode": "weird"})
	wantProblem(t, resp, body, 400, "validation")

	keep := h.mustUploadWeb(402)
	resp, body = h.web("POST", "/api/v1/snapshots", nil)
	wantStatus(t, resp, body, 201)
	snap := decode[map[string]any](t, body)
	if snap["images"] != float64(1) || snap["bytes"].(float64) <= 0 || snap["path"] == "" || snap["createdAt"] == "" {
		t.Fatalf("snapshot: %s", body)
	}
	if _, err := os.Stat(filepath.Join(snap["path"].(string), "MANIFEST.json")); err != nil {
		t.Fatal(err)
	}
	_ = keep
	_, body = h.web("GET", "/api/v1/status", nil)
	st = decode[map[string]any](t, body)
	if st["lastSnapshot"] == nil || st["lastCheck"].(map[string]any)["mode"] != "full" {
		t.Fatalf("status after ops: %s", body)
	}

	// audit trail
	resp, body = h.web("GET", "/api/v1/audit?limit=100", nil)
	wantStatus(t, resp, body, 200)
	items := decode[struct {
		Items []struct {
			ID     int64  `json:"id"`
			At     string `json:"at"`
			Actor  string `json:"actor"`
			Action string `json:"action"`
			Target string `json:"target"`
			IP     string `json:"ip"`
		}
		NextCursor *string `json:"nextCursor"`
	}](t, body)
	seen := map[string]bool{}
	for _, it := range items.Items {
		seen[it.Action] = true
		if it.At == "" || it.Actor == "" {
			t.Fatalf("incomplete audit item: %+v", it)
		}
	}
	for _, want := range []string{"login.success", "image.rename", "image.trash", "image.restore", "image.purge",
		"folder.create", "folder.rename", "folder.delete", "token.create", "check.full", "snapshot"} {
		if !seen[want] {
			t.Errorf("audit lacks %q (have %v)", want, seen)
		}
	}
	// action filter: exactly that action, still newest first
	resp, body = h.web("GET", "/api/v1/audit?action=image.trash&limit=10", nil)
	wantStatus(t, resp, body, 200)
	filtered := decode[struct{ Items []struct{ Action string } }](t, body)
	if len(filtered.Items) == 0 {
		t.Fatalf("filtered audit empty: %s", body)
	}
	for _, it := range filtered.Items {
		if it.Action != "image.trash" {
			t.Fatalf("filter leaked %q", it.Action)
		}
	}
	// kind filter: one category of actions
	resp, body = h.web("GET", "/api/v1/audit?kind=image&limit=100", nil)
	wantStatus(t, resp, body, 200)
	byKind := decode[struct{ Items []struct{ Action string } }](t, body)
	if len(byKind.Items) == 0 {
		t.Fatalf("kind=image empty: %s", body)
	}
	for _, it := range byKind.Items {
		if !strings.HasPrefix(it.Action, "image.") {
			t.Fatalf("kind filter leaked %q", it.Action)
		}
	}
	resp, body = h.web("GET", "/api/v1/audit?kind=tag", nil)
	wantStatus(t, resp, body, 400)
	// distinct actions
	resp, body = h.web("GET", "/api/v1/audit/actions", nil)
	wantStatus(t, resp, body, 200)
	actions := decode[struct{ Items []string }](t, body).Items
	if len(actions) == 0 || actions[0] > actions[len(actions)-1] {
		t.Fatalf("actions: %v", actions)
	}
	found := false
	for _, a := range actions {
		if a == "image.trash" {
			found = true
		}
	}
	if !found {
		t.Fatalf("actions lack image.trash: %v", actions)
	}
	// paging
	resp, body = h.web("GET", "/api/v1/audit?limit=2", nil)
	p1 := decode[struct {
		Items      []struct{ ID int64 }
		NextCursor *string
	}](t, body)
	if len(p1.Items) != 2 || p1.NextCursor == nil {
		t.Fatalf("page 1: %s", body)
	}
	resp, body = h.web("GET", "/api/v1/audit?limit=100&cursor="+*p1.NextCursor, nil)
	p2 := decode[struct{ Items []struct{ ID int64 } }](t, body)
	if len(p2.Items) == 0 || p2.Items[0].ID >= p1.Items[1].ID {
		t.Fatalf("page 2 overlaps: %s", body)
	}
	for _, bad := range []string{"limit=0", "limit=101", "limit=x", "cursor=abc"} {
		resp, body = h.web("GET", "/api/v1/audit?"+bad, nil)
		wantProblem(t, resp, body, 400, "validation")
	}
}

func TestListFiltersAndPagination(t *testing.T) {
	h := newHarness(t, harnessOpts{})
	var ids []string
	for i := 0; i < 5; i++ {
		ids = append(ids, h.mustUploadWeb(500+i).ID)
	}
	var seen []string
	cursor := ""
	for page := 0; page < 5; page++ {
		path := "/api/v1/images?limit=2&sort=old"
		if cursor != "" {
			path += "&cursor=" + cursor
		}
		resp, body := h.web("GET", path, nil)
		wantStatus(t, resp, body, 200)
		l := decode[struct {
			Items      []imageJSON
			NextCursor *string
		}](t, body)
		for _, it := range l.Items {
			seen = append(seen, it.ID)
		}
		if l.NextCursor == nil {
			break
		}
		cursor = *l.NextCursor
	}
	if len(seen) != 5 {
		t.Fatalf("paged %d items", len(seen))
	}
	for _, bad := range []string{"limit=0", "limit=500", "limit=x", "sort=sideways", "state=purged", "format=bmp", "tokenId=x", "cursor=%25%25", "recursive=maybe"} {
		resp, body := h.web("GET", "/api/v1/images?"+bad, nil)
		wantProblem(t, resp, body, 400, "validation")
	}
	resp, body := h.web("GET", "/api/v1/images?folder=0000000000000000", nil)
	wantProblem(t, resp, body, 404, "not-found")
	resp, body = h.web("GET", "/api/v1/images?q="+ids[2]+"&format=png", nil)
	wantStatus(t, resp, body, 200)
	if l := decode[struct{ Items []imageJSON }](t, body); len(l.Items) != 1 || l.Items[0].ID != ids[2] {
		t.Fatalf("filtered: %s", body)
	}
	// The removed gps parameter is an unknown parameter: ignored like any other, never filtering.
	for _, gps := range []string{"gps=true", "gps=false"} {
		resp, body = h.web("GET", "/api/v1/images?limit=100&"+gps, nil)
		wantStatus(t, resp, body, 200)
		if l := decode[struct{ Items []imageJSON }](t, body); len(l.Items) != 5 {
			t.Fatalf("%s should be ignored, got %d items: %s", gps, len(l.Items), body)
		}
	}
}

func TestSPAAndNamespaces(t *testing.T) {
	h := newHarness(t, harnessOpts{imageURL: "https://img.example.com"})
	for _, p := range []string{"/", "/login", "/setup", "/library", "/library/旅行", "/library/旅行/2026/云南", "/library/", "/docs/api", "/trash", "/tokens", "/settings", "/status"} {
		resp, body := h.do("GET", p, nil)
		wantStatus(t, resp, body, 200)
		if !strings.Contains(string(body), "<title>PicBerth</title>") {
			t.Errorf("%s: not the SPA index", p)
		}
		csp := resp.Header.Get("Content-Security-Policy")
		for _, want := range []string{"default-src 'self'", "img-src 'self' blob: data: https://img.example.com", "script-src 'self'", "frame-ancestors 'none'", "object-src 'none'", "base-uri 'none'", "form-action 'self'", "connect-src 'self'"} {
			if !strings.Contains(csp, want) {
				t.Errorf("%s CSP lacks %q: %s", p, want, csp)
			}
		}
		if resp.Header.Get("Cache-Control") != "no-cache" || resp.Header.Get("Referrer-Policy") != "same-origin" ||
			resp.Header.Get("X-Frame-Options") != "DENY" || resp.Header.Get("Cross-Origin-Opener-Policy") != "same-origin" ||
			!strings.HasPrefix(resp.Header.Get("Content-Type"), "text/html") || resp.Header.Get("X-Content-Type-Options") != "nosniff" {
			t.Errorf("%s headers: %v", p, resp.Header)
		}
	}
	// the machine-readable contract needs no session
	resp, body := h.do("GET", "/api/v1/openapi.json", nil)
	wantStatus(t, resp, body, 200)
	if !strings.Contains(resp.Header.Get("Content-Type"), "application/json") || !strings.Contains(string(body), `"openapi": "3.1.0"`) {
		t.Fatalf("openapi.json: %v %.120s", resp.Header, body)
	}
	resp, body = h.do("GET", "/assets/app.abc123.js", nil)
	wantStatus(t, resp, body, 200)
	if resp.Header.Get("Cache-Control") != "public, max-age=31536000, immutable" || !strings.HasPrefix(resp.Header.Get("Content-Type"), "text/javascript") {
		t.Fatalf("asset headers: %v", resp.Header)
	}
	resp, body = h.do("GET", "/assets/style.def456.css", nil)
	if !strings.HasPrefix(resp.Header.Get("Content-Type"), "text/css") {
		t.Fatalf("css type: %v", resp.Header)
	}
	resp, body = h.do("GET", "/favicon.svg", nil)
	wantStatus(t, resp, body, 200)
	if resp.Header.Get("Cache-Control") != "public, max-age=86400" || resp.Header.Get("Content-Type") != "image/svg+xml" {
		t.Fatalf("favicon headers: %v", resp.Header)
	}
	// everything else is a 404
	for _, p := range []string{"/nope", "/assets/missing.js", "/assets/", "/assets/../index.html", "/index.html", "/.env", "/api", "/api/", "/api/v1", "/api/v2/images", "/login/x", "/i", "/i/x/y"} {
		resp, body = h.do("GET", p, nil)
		if resp.StatusCode != 404 && resp.StatusCode != 301 {
			t.Errorf("GET %s: %d, want 404", p, resp.StatusCode)
			continue
		}
		if resp.StatusCode == 301 { // mux slash-redirect; must lead to a 404 or the same SPA route
			continue
		}
		if strings.HasPrefix(p, "/api") {
			wantProblem(t, resp, body, 404, "not-found")
		} else if !strings.HasPrefix(resp.Header.Get("Content-Type"), "text/plain") {
			t.Errorf("GET %s: non-API 404 should be plain text, got %q", p, resp.Header.Get("Content-Type"))
		}
	}
	// SPA routes only answer GET/HEAD
	resp, _ = h.do("POST", "/library", strings.NewReader("x"))
	if resp.StatusCode == 200 {
		t.Error("POST /library returned the SPA")
	}
	// the version header is on every response
	if resp, _ := h.do("GET", "/nope", nil); resp.Header.Get("X-PicBerth-Version") == "" || resp.Header.Get("X-Content-Type-Options") != "nosniff" {
		t.Error("base headers missing on a 404")
	}
	// API: security headers
	resp, _ = h.do("GET", "/api/v1/session", nil)
	if resp.Header.Get("Content-Security-Policy") != "default-src 'none'; frame-ancestors 'none'" || resp.Header.Get("Cache-Control") != "no-store" {
		t.Errorf("API headers: %v", resp.Header)
	}
}

func TestSPAImgSrcWithoutSeparateImageOrigin(t *testing.T) {
	h := newHarness(t, harnessOpts{publicURL: "https://pics.example.com", imageURL: "https://pics.example.com"})
	resp, _ := h.do("GET", "/", nil)
	csp := resp.Header.Get("Content-Security-Policy")
	if !strings.Contains(csp, "img-src 'self' blob: data:;") {
		t.Fatalf("CSP: %s", csp)
	}
}

func TestHealthEndpoints(t *testing.T) {
	h := newHarness(t, harnessOpts{})
	resp, body := h.do("GET", "/healthz", nil)
	wantStatus(t, resp, body, 200)
	if strings.TrimSpace(string(body)) != "ok" {
		t.Fatalf("healthz: %q", body)
	}
	resp, body = h.do("GET", "/readyz", nil)
	wantStatus(t, resp, body, 200)
	h.srv.SetReady(false)
	resp, body = h.do("GET", "/readyz", nil)
	wantStatus(t, resp, body, 503)
	resp, body = h.do("GET", "/healthz", nil)
	wantStatus(t, resp, body, 200) // liveness is independent of readiness
	h.srv.SetReady(true)
	resp, body = h.do("GET", "/readyz", nil)
	wantStatus(t, resp, body, 200)
}

func TestRealIPAndUntrustedForwardedWarning(t *testing.T) {
	// no trusted proxies: X-Forwarded-For is ignored and raises the warning
	h := newHarness(t, harnessOpts{})
	h.do("GET", "/api/v1/session", nil, withHeader("X-Forwarded-For", "6.6.6.6"))
	resp, body := h.do("POST", "/api/v1/session", strings.NewReader(`{"username":"admin","password":"correct horse battery"}`),
		withHeader("Content-Type", "application/json"), withHeader("X-Forwarded-For", "6.6.6.6"))
	wantStatus(t, resp, body, 204)
	_, b := h.web("GET", "/api/v1/status", nil)
	if !strings.Contains(string(b), `"untrusted-forwarded"`) {
		t.Fatalf("missing untrusted-forwarded warning: %s", b)
	}
	_, b = h.web("GET", "/api/v1/audit?limit=5", nil)
	if strings.Contains(string(b), "6.6.6.6") || !strings.Contains(string(b), "127.0.0.1") {
		t.Fatalf("untrusted X-Forwarded-For was used: %s", b)
	}

	// trusted loopback proxy: the right-most untrusted hop is the client
	h2 := newHarness(t, harnessOpts{trusted: []string{"127.0.0.1", "::1"}})
	h2.do("POST", "/api/v1/session", strings.NewReader(`{"username":"admin","password":"correct horse battery"}`),
		withHeader("Content-Type", "application/json"), withHeader("X-Forwarded-For", "9.9.9.9, 198.51.100.7, 127.0.0.1"))
	_, b = h2.web("GET", "/api/v1/audit?limit=5", nil)
	if !strings.Contains(string(b), "198.51.100.7") || strings.Contains(string(b), "9.9.9.9") {
		t.Fatalf("trusted proxy chain mis-resolved: %s", b)
	}
	_, b = h2.web("GET", "/api/v1/status", nil)
	if strings.Contains(string(b), `"untrusted-forwarded"`) {
		t.Fatalf("warning raised for a trusted proxy: %s", b)
	}
}

func TestRateLimitIsPerRealIP(t *testing.T) {
	h := newHarness(t, harnessOpts{keepLimits: true, skipAdmin: true, trusted: []string{"127.0.0.1", "::1"}})
	h.svc.CreateAdmin(context.Background(), "admin", "correct horse battery")
	attempt := func(ip string) *http.Response {
		resp, _ := h.jsonReq("POST", "/api/v1/session", map[string]string{"username": "x" + ip, "password": "wrong password!"},
			withHeader("X-Forwarded-For", "203.0.113.50, "+ip))
		return resp
	}
	// spoofed left-most entries do not create new buckets: the right-most untrusted hop counts
	for i := 0; i < 5; i++ {
		if r := attempt("198.51.100.1"); r.StatusCode != 401 {
			t.Fatalf("attempt %d: %d", i, r.StatusCode)
		}
	}
	if r := attempt("198.51.100.1"); r.StatusCode != 429 {
		t.Fatalf("expected 429 for the exhausted client, got %d", r.StatusCode)
	}
	if r := attempt("198.51.100.2"); r.StatusCode != 401 {
		t.Fatalf("another client must not be limited: %d", r.StatusCode)
	}
}

func TestAccessLogNeverContainsSecrets(t *testing.T) {
	h := newHarness(t, harnessOpts{})
	_, secret := h.createToken("manage")
	cookieVal := h.cookie[strings.Index(h.cookie, "=")+1:]
	h.do("GET", "/api/v1/images?q=topsecretquery&token="+secret, nil, withBearer(secret))
	h.do("GET", "/api/v1/images?sneaky=1", nil, withHeader("Cookie", h.cookie))
	h.jsonReq("POST", "/api/v1/session", map[string]string{"username": "admin", "password": "correct horse battery"})
	h.do("GET", "/i/00000000000000000000.png?secret=abc", nil)
	logs := h.logBuf.String()
	if !strings.Contains(logs, "msg=request") || !strings.Contains(logs, "path=/api/v1/images") {
		t.Fatalf("access log missing:\n%s", logs)
	}
	for _, forbidden := range []string{secret, cookieVal, "topsecretquery", "sneaky", "secret=abc", "correct horse", "Authorization", "Bearer", "pb_session"} {
		if strings.Contains(logs, forbidden) {
			t.Errorf("log contains %q", forbidden)
		}
	}
}

func TestRequestBodyLimitsOnJSON(t *testing.T) {
	h := newHarness(t, harnessOpts{})
	huge := `{"name":"` + strings.Repeat("a", 300<<10) + `","scopes":["upload"]}`
	resp, body := h.do("POST", "/api/v1/tokens", strings.NewReader(huge), append(h.asWeb(), withHeader("Content-Type", "application/json"))...)
	wantProblem(t, resp, body, 413, "too-large")
	resp, body = h.do("POST", "/api/v1/tokens", strings.NewReader(`{"name":"x","scopes":["upload"]} {"x":1}`), append(h.asWeb(), withHeader("Content-Type", "application/json"))...)
	wantProblem(t, resp, body, 400, "validation")
	resp, body = h.do("POST", "/api/v1/tokens", strings.NewReader(``), append(h.asWeb(), withHeader("Content-Type", "application/json"))...)
	wantProblem(t, resp, body, 400, "validation")
}

func TestUnknownAPIRoutesAreJSON404(t *testing.T) {
	h := newHarness(t, harnessOpts{})
	for _, c := range []struct{ m, p string }{
		{"GET", "/api/v1/nothing"}, {"PUT", "/api/v1/images"}, {"DELETE", "/api/v1/settings"}, {"GET", "/api/"}, {"POST", "/api/v2/x"},
	} {
		resp, body := h.do(c.m, c.p, nil, h.asWeb()...)
		wantProblem(t, resp, body, 404, "not-found")
	}
}

func TestConcurrentUploadsHTTP(t *testing.T) {
	h := newHarness(t, harnessOpts{})
	errs := make(chan string, 20)
	for i := 0; i < 12; i++ {
		go func(i int) {
			resp, body := h.uploadWeb(makePNG(t, 600+i))
			if resp.StatusCode != 201 && resp.StatusCode != 429 {
				errs <- fmt.Sprintf("upload %d: %d %s", i, resp.StatusCode, body)
				return
			}
			errs <- ""
		}(i)
	}
	for i := 0; i < 12; i++ {
		if e := <-errs; e != "" {
			t.Error(e)
		}
	}
	if tmp, _ := os.ReadDir(filepath.Join(h.dir.Path(), "tmp")); len(tmp) != 0 {
		t.Fatalf("temp files left: %d", len(tmp))
	}
}
