package httpapi

import (
	"fmt"
	"net/url"
	"reflect"
	"strings"
	"testing"
)

type taggedImage struct {
	imageJSON
	Filename string   `json:"filename"`
	Tags     []string `json:"tags"`
}

func TestUploadTagParts(t *testing.T) {
	h := newHarness(t, harnessOpts{})
	resp, body := h.uploadWeb(makePNG(t, 1), part{name: "tag", data: []byte(" cat ")}, part{name: "tag", data: []byte("狗")}, part{name: "tag", data: []byte("cat")})
	wantStatus(t, resp, body, 201)
	img := decode[taggedImage](t, body)
	if !reflect.DeepEqual(img.Tags, []string{"cat", "狗"}) {
		t.Fatalf("upload tags: %s", body)
	}
	if strings.Contains(string(body), `"tags":null`) {
		t.Fatal("tags must never be null")
	}

	// duplicate: 200, existing tags unchanged
	resp, body = h.uploadWeb(makePNG(t, 1), part{name: "tag", data: []byte("other")})
	wantStatus(t, resp, body, 200)
	if d := decode[taggedImage](t, body); !d.Duplicate || d.ID != img.ID || !reflect.DeepEqual(d.Tags, []string{"cat", "狗"}) {
		t.Fatalf("duplicate: %s", body)
	}

	// no tag parts -> empty array
	resp, body = h.uploadWeb(makePNG(t, 2))
	wantStatus(t, resp, body, 201)
	if !strings.Contains(string(body), `"tags":[]`) {
		t.Fatalf("expected an empty tags array: %s", body)
	}

	for name, bad := range map[string]part{
		"comma": {name: "tag", data: []byte("a,b")}, "hash": {name: "tag", data: []byte("#a")}, "empty": {name: "tag", data: []byte(" ")},
		"long": {name: "tag", data: []byte(strings.Repeat("x", 33))}, "huge": {name: "tag", data: []byte(strings.Repeat("x", 4000))},
	} {
		resp, body = h.uploadWeb(makePNG(t, 3), bad)
		if resp.StatusCode != 400 {
			t.Fatalf("%s: %d %s", name, resp.StatusCode, body)
		}
		wantProblem(t, resp, body, 400, "validation")
	}
	// 20 tags are fine and do not count against the 8-part limit; 21 are rejected
	var many []part
	for i := 0; i < 21; i++ {
		many = append(many, part{name: "tag", data: []byte(fmt.Sprint("t", i))})
	}
	resp, body = h.uploadWeb(makePNG(t, 4), many[:20]...)
	wantStatus(t, resp, body, 201)
	resp, body = h.uploadWeb(makePNG(t, 5), many...)
	wantProblem(t, resp, body, 400, "validation")
}

func TestPatchImageHTTP(t *testing.T) {
	h := newHarness(t, harnessOpts{})
	img := h.mustUploadWeb(10)
	path := "/api/v1/images/" + img.ID

	resp, body := h.web("PATCH", path, map[string]any{"filename": " 新名字 .png", "tags": []string{"b", "a", "b"}, "visibility": "private"})
	wantStatus(t, resp, body, 200)
	v := decode[taggedImage](t, body)
	if v.Filename != "新名字 .png" || v.Visibility != "private" || !reflect.DeepEqual(v.Tags, []string{"a", "b"}) || v.ID != img.ID || v.URL != img.URL {
		t.Fatalf("patch: %s", body)
	}
	if !strings.Contains(string(body), "![新名字 ](") {
		t.Fatalf("markdown alt follows the name: %s", body)
	}
	resp, body = h.web("PATCH", path, map[string]any{"tags": []string{}})
	wantStatus(t, resp, body, 200)
	if !strings.Contains(string(body), `"tags":[]`) {
		t.Fatalf("cleared tags: %s", body)
	}
	resp, body = h.web("GET", path, nil)
	if g := decode[taggedImage](t, body); g.Filename != "新名字 .png" || len(g.Tags) != 0 {
		t.Fatalf("get after patch: %s", body)
	}

	for name, bad := range map[string]any{
		"empty object": map[string]any{}, "null fields": map[string]any{"tags": nil, "filename": nil},
		"slash": map[string]any{"filename": "a/b.png"}, "backslash": map[string]any{"filename": `a\b`}, "blank name": map[string]any{"filename": "  "},
		"long name": map[string]any{"filename": strings.Repeat("x", 201)}, "bad tag": map[string]any{"tags": []string{"a,b"}},
		"hash tag": map[string]any{"tags": []string{"#x"}}, "tags type": map[string]any{"tags": "x"}, "unknown": map[string]any{"note": "x"},
		"vis": map[string]any{"visibility": "secret"},
	} {
		resp, body = h.web("PATCH", path, bad)
		if resp.StatusCode != 400 {
			t.Fatalf("%s: %d %s", name, resp.StatusCode, body)
		}
		wantProblem(t, resp, body, 400, "validation")
	}
	resp, body = h.do("PATCH", path, strings.NewReader(""), append(h.asWeb(), withHeader("Content-Type", "application/json"))...)
	wantProblem(t, resp, body, 400, "validation")

	// auth: same as visibility
	_, upSecret := h.createToken("upload")
	_, mgSecret := h.createToken("manage")
	resp, body = h.jsonReq("PATCH", path, map[string]any{"tags": []string{"x"}}, withBearer(upSecret))
	wantProblem(t, resp, body, 403, "forbidden")
	resp, body = h.jsonReq("PATCH", path, map[string]any{"filename": "by-token.png"}, withBearer(mgSecret))
	wantStatus(t, resp, body, 200)
	resp, body = h.jsonReq("PATCH", path, map[string]any{"tags": []string{"x"}})
	wantProblem(t, resp, body, 401, "unauthorized")

	// purged images are gone
	resp, body = h.web("DELETE", path, nil)
	wantStatus(t, resp, body, 204)
	resp, body = h.web("POST", path+"/purge", nil)
	wantStatus(t, resp, body, 204)
	resp, body = h.web("PATCH", path, map[string]any{"tags": []string{"x"}})
	wantProblem(t, resp, body, 404, "not-found")
}

func TestListQueryParamsHTTP(t *testing.T) {
	h := newHarness(t, harnessOpts{})
	tokID, tokSecret := h.createToken("manage")
	for i, name := range []string{"beta", "Alpha", "gamma"} {
		img := h.mustUploadWeb(20+i, part{name: "tag", data: []byte("t" + fmt.Sprint(i%2))}, part{name: "tag", data: []byte("all")})
		h.web("PATCH", "/api/v1/images/"+img.ID, map[string]string{"filename": name + ".png"})
	}
	resp, body := h.upload(makePNG(t, 30), []reqOpt{withBearer(tokSecret)}, part{name: "tag", data: []byte("tok")})
	wantStatus(t, resp, body, 201)

	names := func(query string) []string {
		t.Helper()
		resp, body := h.web("GET", "/api/v1/images?"+query, nil)
		wantStatus(t, resp, body, 200)
		var out []string
		for _, it := range decode[struct{ Items []taggedImage }](t, body).Items {
			out = append(out, it.Filename)
		}
		return out
	}
	if got := names("tag=all&tag=t0&sort=name"); !reflect.DeepEqual(got, []string{"beta.png", "gamma.png"}) {
		t.Fatalf("repeated tag (AND): %v", got)
	}
	if got := names("format=png&format=jpeg"); len(got) != 4 {
		t.Fatalf("repeated format: %v", got)
	}
	if got := names("format=gif,jpeg"); len(got) != 0 {
		t.Fatalf("comma format: %v", got)
	}
	if got := names("format=jpeg,png"); len(got) != 4 {
		t.Fatalf("comma format any-of: %v", got)
	}
	if got := names("source=web&sort=name"); !reflect.DeepEqual(got, []string{"Alpha.png", "beta.png", "gamma.png"}) {
		t.Fatalf("source=web: %v", got)
	}
	if got := names(fmt.Sprintf("source=token:%d", tokID)); len(got) != 1 {
		t.Fatalf("source=token: %v", got)
	}
	if got := names("source=cli&source=token:" + fmt.Sprint(tokID)); len(got) != 1 {
		t.Fatalf("source any-of: %v", got)
	}
	if got := names("q=TOK"); len(got) != 1 {
		t.Fatalf("q matches tags: %v", got)
	}
	future := url.QueryEscape("2999-01-01T00:00:00+01:00")
	past := url.QueryEscape("2000-01-01T00:00:00Z")
	if got := names("since=" + past + "&until=" + future); len(got) != 4 {
		t.Fatalf("since/until: %v", got)
	}
	if got := names("since=" + future); len(got) != 0 {
		t.Fatalf("since in the future: %v", got)
	}
	if got := names("until=" + past); len(got) != 0 {
		t.Fatalf("until in the past: %v", got)
	}

	// name sort across pages with the cursor
	var seen []string
	cursor := ""
	for page := 0; page < 5; page++ {
		path := "/api/v1/images?sort=name&limit=2"
		if cursor != "" {
			path += "&cursor=" + url.QueryEscape(cursor)
		}
		resp, body = h.web("GET", path, nil)
		wantStatus(t, resp, body, 200)
		l := decode[struct {
			Items      []taggedImage
			NextCursor *string
		}](t, body)
		for _, it := range l.Items {
			seen = append(seen, it.Filename)
		}
		if l.NextCursor == nil {
			break
		}
		cursor = *l.NextCursor
	}
	if len(seen) != 4 || seen[0] != "Alpha.png" || seen[1] != "beta.png" || seen[2] != "gamma.png" {
		t.Fatalf("name sort pages: %v", seen)
	}

	for _, bad := range []string{"since=yesterday", "until=2026-01-01", "since=1700000000", "source=ftp", "source=token:x", "source=token:0", "format=png,bmp", "format=bmp&format=png", "sort=nope"} {
		resp, body = h.web("GET", "/api/v1/images?"+bad, nil)
		wantProblem(t, resp, body, 400, "validation")
	}
}

func TestFacetsEndpoint(t *testing.T) {
	h := newHarness(t, harnessOpts{})
	tokID, tokSecret := h.createToken("manage")
	_, upSecret := h.createToken("upload")
	a := h.mustUploadWeb(40, part{name: "tag", data: []byte("截图")})
	h.mustUploadWeb(41, part{name: "tag", data: []byte("截图")}, part{name: "tag", data: []byte("x")})
	resp, body := h.upload(makePNG(t, 42), []reqOpt{withBearer(tokSecret)})
	wantStatus(t, resp, body, 201)
	tokImg := decode[imageJSON](t, body)
	resp, body = h.upload(makePNG(t, 43), []reqOpt{withBearer(upSecret)}, part{name: "tag", data: []byte("mine")})
	wantStatus(t, resp, body, 201)
	h.web("PATCH", "/api/v1/images/"+a.ID, map[string]string{"visibility": "private"})
	h.web("DELETE", "/api/v1/images/"+tokImg.ID, nil)

	type facets struct {
		Total      int64            `json:"total"`
		Bytes      int64            `json:"bytes"`
		Formats    map[string]int64 `json:"formats"`
		Visibility map[string]int64 `json:"visibility"`
		Sources    []struct {
			Key       string `json:"key"`
			Kind      string `json:"kind"`
			TokenID   *int64 `json:"tokenId"`
			TokenName string `json:"tokenName"`
			Count     int64  `json:"count"`
		} `json:"sources"`
		Tags []struct {
			Tag   string `json:"tag"`
			Count int64  `json:"count"`
		} `json:"tags"`
	}

	// "facets" must not be taken for an image id
	resp, body = h.web("GET", "/api/v1/images/facets", nil)
	wantStatus(t, resp, body, 200)
	f := decode[facets](t, body)
	if f.Total != 3 || f.Bytes <= 0 {
		t.Fatalf("facets: %s", body)
	}
	if !reflect.DeepEqual(f.Formats, map[string]int64{"jpeg": 0, "png": 3, "gif": 0, "webp": 0, "avif": 0}) {
		t.Fatalf("formats: %s", body)
	}
	if !reflect.DeepEqual(f.Visibility, map[string]int64{"public": 2, "private": 1}) {
		t.Fatalf("visibility: %s", body)
	}
	if len(f.Sources) != 2 || f.Sources[0].Key != "web" || f.Sources[0].Kind != "web" || f.Sources[0].Count != 2 || f.Sources[0].TokenID != nil ||
		f.Sources[1].Kind != "token" || f.Sources[1].TokenID == nil || f.Sources[1].TokenName == "" || f.Sources[1].Count != 1 {
		t.Fatalf("sources: %s", body)
	}
	if len(f.Tags) != 3 || f.Tags[0].Tag != "截图" || f.Tags[0].Count != 2 || f.Tags[1].Tag != "mine" || f.Tags[2].Tag != "x" {
		t.Fatalf("tags: %s", body)
	}

	resp, body = h.web("GET", "/api/v1/images/facets?state=trashed", nil)
	wantStatus(t, resp, body, 200)
	tf := decode[facets](t, body)
	if tf.Total != 1 || len(tf.Sources) != 1 || tf.Sources[0].Key != fmt.Sprintf("token:%d", tokID) || len(tf.Tags) != 0 {
		t.Fatalf("trashed facets: %s", body)
	}
	if !strings.Contains(string(body), `"tags":[]`) {
		t.Fatalf("empty tags must be an array: %s", body)
	}

	// token principals
	resp, body = h.jsonReq("GET", "/api/v1/images/facets", nil, withBearer(upSecret))
	wantStatus(t, resp, body, 200)
	if uf := decode[facets](t, body); uf.Total != 1 || len(uf.Tags) != 1 || uf.Tags[0].Tag != "mine" {
		t.Fatalf("upload token sees only its own: %s", body)
	}
	resp, body = h.jsonReq("GET", "/api/v1/images/facets?state=trashed", nil, withBearer(upSecret))
	wantProblem(t, resp, body, 403, "forbidden")
	resp, body = h.web("GET", "/api/v1/images/facets?state=purged", nil)
	wantProblem(t, resp, body, 400, "validation")
	resp, body = h.do("GET", "/api/v1/images/facets", nil)
	wantProblem(t, resp, body, 401, "unauthorized")
}
