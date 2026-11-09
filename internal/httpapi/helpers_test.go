package httpapi

import (
	"bytes"
	"context"
	"encoding/binary"
	"encoding/json"
	"fmt"
	"hash/crc32"
	"image"
	"image/color"
	"image/png"
	"io"
	"log/slog"
	"mime/multipart"
	"net/http"
	"net/http/httptest"
	"net/textproto"
	"path/filepath"
	"strings"
	"sync"
	"testing"
	"testing/fstest"

	"picberth/internal/core"
	"picberth/internal/datadir"
	"picberth/internal/imaging"
	"picberth/internal/meta"
	"picberth/internal/netx"
	"picberth/internal/ratelimit"
)

func fakeSniff(head []byte) (imaging.Format, bool) {
	switch {
	case bytes.HasPrefix(head, []byte("\x89PNG\r\n\x1a\n")):
		return imaging.PNG, true
	case bytes.HasPrefix(head, []byte("\xff\xd8\xff")):
		return imaging.JPEG, true
	case bytes.HasPrefix(head, []byte("GIF8")):
		return imaging.GIF, true
	}
	return "", false
}

func fakeInspect(r io.ReaderAt, size int64, lim imaging.Limits) (imaging.Info, error) {
	cfg, name, err := image.DecodeConfig(io.NewSectionReader(r, 0, size))
	if err != nil {
		return imaging.Info{}, fmt.Errorf("%w: %v", imaging.ErrInvalid, err)
	}
	if int64(cfg.Width)*int64(cfg.Height) > lim.MaxPixels {
		return imaging.Info{}, imaging.ErrTooManyPixels
	}
	f, _ := imaging.ParseFormat(name)
	return imaging.Info{Format: f, Width: cfg.Width, Height: cfg.Height, Frames: 1, Orientation: 1}, nil
}

func fakeThumbnail(r io.ReaderAt, size int64, info imaging.Info, maxDim int, maxPixels int64) ([]byte, string, error) {
	var buf bytes.Buffer
	if err := png.Encode(&buf, image.NewNRGBA(image.Rect(0, 0, 2, 2))); err != nil {
		return nil, "", err
	}
	return buf.Bytes(), "png", nil
}

func makePNG(t testing.TB, seed int) []byte {
	t.Helper()
	img := image.NewNRGBA(image.Rect(0, 0, 6, 4))
	img.SetNRGBA(0, 0, color.NRGBA{R: uint8(seed), G: uint8(seed >> 8), B: 9, A: 255})
	var buf bytes.Buffer
	if err := png.Encode(&buf, img); err != nil {
		t.Fatal(err)
	}
	return buf.Bytes()
}

type harness struct {
	t      *testing.T
	ts     *httptest.Server
	srv    *Server
	svc    *core.Service
	db     *meta.DB
	dir    *datadir.Dir
	logBuf *syncBuffer
	client *http.Client
	cookie string // session cookie "name=value"
}

type syncBuffer struct {
	mu sync.Mutex
	b  bytes.Buffer
}

func (s *syncBuffer) Write(p []byte) (int, error) {
	s.mu.Lock()
	defer s.mu.Unlock()
	return s.b.Write(p)
}

func (s *syncBuffer) String() string {
	s.mu.Lock()
	defer s.mu.Unlock()
	return s.b.String()
}

type harnessOpts struct {
	trusted    []string
	keepLimits bool // keep the real login rate limits
	skipAdmin  bool // leave the instance in setup mode
	publicURL  string
	imageURL   string
}

var testWeb = fstest.MapFS{
	"index.html":              {Data: []byte("<!doctype html><title>PicBerth</title><script src=\"/assets/app.abc123.js\"></script>")},
	"assets/app.abc123.js":    {Data: []byte("console.log('hi')")},
	"assets/style.def456.css": {Data: []byte("body{}")},
	"favicon.svg":             {Data: []byte("<svg xmlns='http://www.w3.org/2000/svg'/>")},
}

func newHarness(t *testing.T, o harnessOpts) *harness {
	t.Helper()
	root := t.TempDir()
	dir, err := datadir.Open(filepath.Join(root, "data"), datadir.Options{Create: true})
	if err != nil {
		t.Fatal(err)
	}
	db, err := meta.Open(context.Background(), dir.DBPath())
	if err != nil {
		t.Fatal(err)
	}
	if o.publicURL == "" {
		o.publicURL = "https://pics.example.com"
	}
	if o.imageURL == "" {
		o.imageURL = "https://img.example.com"
	}
	logBuf := &syncBuffer{}
	log := slog.New(slog.NewTextHandler(logBuf, &slog.HandlerOptions{Level: slog.LevelDebug}))
	svc, err := core.New(db, dir, core.Config{PublicURL: o.publicURL, ImageURL: o.imageURL, PublicURLSet: true,
		HardMaxUpload: 100 << 20, SnapshotDir: filepath.Join(root, "snaps")}, log)
	if err != nil {
		t.Fatal(err)
	}
	svc.Sniff, svc.Inspect, svc.Thumbnail = fakeSniff, fakeInspect, fakeThumbnail
	svc.DiskFree = func() (uint64, uint64, error) { return 500 << 30, 1000 << 30, nil }
	if !o.keepLimits {
		svc.SetLoginLimits(ratelimit.PerMinute(10000), ratelimit.PerMinute(10000))
	}
	trusted, err := netx.ParseTrusted(o.trusted)
	if err != nil {
		t.Fatal(err)
	}
	srv := New(Options{Core: svc, PublicURL: o.publicURL, ImageURL: o.imageURL, Trusted: trusted, Web: testWeb, Logger: log})
	ts := httptest.NewServer(srv.Handler())
	h := &harness{t: t, ts: ts, srv: srv, svc: svc, db: db, dir: dir, logBuf: logBuf,
		client: &http.Client{CheckRedirect: func(*http.Request, []*http.Request) error { return http.ErrUseLastResponse }}}
	t.Cleanup(func() {
		ts.Close()
		svc.Close()
		db.Close()
		dir.Close()
	})
	if !o.skipAdmin {
		if err := svc.CreateAdmin(context.Background(), "admin", "correct horse battery"); err != nil {
			t.Fatal(err)
		}
		h.login()
	}
	return h
}

type reqOpt func(*http.Request)

func withHeader(k, v string) reqOpt { return func(r *http.Request) { r.Header.Set(k, v) } }
func withBearer(tok string) reqOpt  { return withHeader("Authorization", "Bearer "+tok) }

// asWeb adds the cookie and the CSRF client header, like the SPA does.
func (h *harness) asWeb() []reqOpt {
	return []reqOpt{withHeader("Cookie", h.cookie), withHeader("X-PicBerth-Client", "web")}
}

func (h *harness) do(method, path string, body io.Reader, opts ...reqOpt) (*http.Response, []byte) {
	h.t.Helper()
	req, err := http.NewRequest(method, h.ts.URL+path, body)
	if err != nil {
		h.t.Fatal(err)
	}
	for _, o := range opts {
		o(req)
	}
	resp, err := h.client.Do(req)
	if err != nil {
		h.t.Fatalf("%s %s: %v", method, path, err)
	}
	defer resp.Body.Close()
	data, _ := io.ReadAll(resp.Body)
	return resp, data
}

func (h *harness) jsonReq(method, path string, v any, opts ...reqOpt) (*http.Response, []byte) {
	h.t.Helper()
	var body io.Reader
	if v != nil {
		b, _ := json.Marshal(v)
		body = bytes.NewReader(b)
	}
	opts = append([]reqOpt{withHeader("Content-Type", "application/json")}, opts...)
	return h.do(method, path, body, opts...)
}

// web performs a JSON request as the logged-in browser.
func (h *harness) web(method, path string, v any) (*http.Response, []byte) {
	h.t.Helper()
	return h.jsonReq(method, path, v, h.asWeb()...)
}

func (h *harness) login() {
	h.t.Helper()
	resp, _ := h.jsonReq("POST", "/api/v1/session", map[string]string{"username": "admin", "password": "correct horse battery"})
	if resp.StatusCode != 204 {
		h.t.Fatalf("login status %d", resp.StatusCode)
	}
	for _, c := range resp.Cookies() {
		h.cookie = c.Name + "=" + c.Value
	}
	if h.cookie == "" {
		h.t.Fatal("no session cookie issued")
	}
}

type part struct {
	name     string
	filename string
	data     []byte
}

func multipartBody(parts ...part) (io.Reader, string) {
	var buf bytes.Buffer
	mw := multipart.NewWriter(&buf)
	for _, p := range parts {
		hdr := textproto.MIMEHeader{}
		disp := fmt.Sprintf(`form-data; name="%s"`, p.name)
		if p.filename != "" {
			disp += fmt.Sprintf(`; filename="%s"`, p.filename)
			hdr.Set("Content-Type", "application/octet-stream")
		}
		hdr.Set("Content-Disposition", disp)
		w, _ := mw.CreatePart(hdr)
		w.Write(p.data)
	}
	mw.Close()
	return &buf, mw.FormDataContentType()
}

func (h *harness) upload(data []byte, opts []reqOpt, extra ...part) (*http.Response, []byte) {
	h.t.Helper()
	parts := append([]part{{name: "file", filename: "shot.png", data: data}}, extra...)
	body, ct := multipartBody(parts...)
	return h.do("POST", "/api/v1/images", body, append([]reqOpt{withHeader("Content-Type", ct)}, opts...)...)
}

func (h *harness) uploadWeb(data []byte, extra ...part) (*http.Response, []byte) {
	h.t.Helper()
	return h.upload(data, h.asWeb(), extra...)
}

func decode[T any](t *testing.T, data []byte) T {
	t.Helper()
	var v T
	if err := json.Unmarshal(data, &v); err != nil {
		t.Fatalf("bad JSON %q: %v", data, err)
	}
	return v
}

type imageJSON struct {
	ID        string  `json:"id"`
	URL       string  `json:"url"`
	FolderID  *string `json:"folderId"`
	State     string  `json:"state"`
	Duplicate bool    `json:"duplicate"`
	Ext       string  `json:"ext"`
	SHA256    string  `json:"sha256"`
	HasThumb  bool    `json:"hasThumb"`
	Source    struct {
		Kind      string `json:"kind"`
		TokenName string `json:"tokenName"`
	} `json:"source"`
}

// pngHeader builds a PNG with a valid IHDR chunk and no pixel data; image.DecodeConfig accepts it, so a
// test can declare more than 50 MP without allocating anything.
func pngHeader(t testing.TB, w, h int) []byte {
	t.Helper()
	var buf bytes.Buffer
	buf.WriteString("\x89PNG\r\n\x1a\n")
	data := make([]byte, 13)
	binary.BigEndian.PutUint32(data[0:], uint32(w))
	binary.BigEndian.PutUint32(data[4:], uint32(h))
	data[8] = 8
	data[9] = 6
	binary.Write(&buf, binary.BigEndian, uint32(len(data)))
	typ := []byte("IHDR")
	buf.Write(typ)
	buf.Write(data)
	binary.Write(&buf, binary.BigEndian, crc32.ChecksumIEEE(append(typ, data...)))
	return buf.Bytes()
}

func (h *harness) mustUploadWeb(seed int, extra ...part) imageJSON {
	h.t.Helper()
	resp, body := h.uploadWeb(makePNG(h.t, seed), extra...)
	if resp.StatusCode != 201 {
		h.t.Fatalf("upload: %d %s", resp.StatusCode, body)
	}
	return decode[imageJSON](h.t, body)
}

func wantStatus(t *testing.T, resp *http.Response, body []byte, status int) {
	t.Helper()
	if resp.StatusCode != status {
		t.Fatalf("status %d, want %d; body: %s", resp.StatusCode, status, body)
	}
}

func wantProblem(t *testing.T, resp *http.Response, body []byte, status int, code string) {
	t.Helper()
	wantStatus(t, resp, body, status)
	if ct := resp.Header.Get("Content-Type"); ct != "application/problem+json" {
		t.Fatalf("Content-Type %q, want application/problem+json", ct)
	}
	p := decode[problem](t, body)
	if p.Code != code || p.Status != status || p.Type != "urn:picberth:problem:"+code || p.Title == "" {
		t.Fatalf("problem: %+v (want code %s)", p, code)
	}
}

func (h *harness) createToken(scopes ...string) (id int64, secret string) {
	h.t.Helper()
	resp, body := h.web("POST", "/api/v1/tokens", map[string]any{"name": "t " + strings.Join(scopes, ","), "scopes": scopes, "expiresInDays": nil})
	wantStatus(h.t, resp, body, 201)
	var out struct {
		Token  struct{ ID int64 } `json:"token"`
		Secret string             `json:"secret"`
	}
	json.Unmarshal(body, &out)
	return out.Token.ID, out.Secret
}

func pngOfSize(t testing.TB, w, h int) []byte {
	t.Helper()
	var buf bytes.Buffer
	if err := png.Encode(&buf, image.NewNRGBA(image.Rect(0, 0, w, h))); err != nil {
		t.Fatal(err)
	}
	return buf.Bytes()
}

func coreAdmin(h *harness) core.Principal { return core.CLIPrincipal() }
