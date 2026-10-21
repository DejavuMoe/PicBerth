package core

import (
	"bytes"
	"context"
	"errors"
	"fmt"
	"image"
	"image/color"
	"image/gif"
	"image/jpeg"
	"image/png"
	"io"
	"log/slog"
	"os"
	"path/filepath"
	"strings"
	"testing"
	"time"

	"picberth/internal/datadir"
	"picberth/internal/imaging"
	"picberth/internal/meta"
	"picberth/internal/ratelimit"
)

// fakeSniff / fakeInspect / fakeThumbnail let these tests run without the real imaging package: they only
// rely on the standard library decoders.
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
	f, _ := imaging.ParseFormat(map[string]string{"jpeg": "jpeg", "png": "png", "gif": "gif"}[name])
	return imaging.Info{Format: f, Width: cfg.Width, Height: cfg.Height, Frames: 1, Orientation: 1}, nil
}

func fakeThumbnail(r io.ReaderAt, size int64, info imaging.Info, maxDim int, maxPixels int64) ([]byte, string, error) {
	img, _, err := image.Decode(io.NewSectionReader(r, 0, size))
	if err != nil {
		return nil, "", imaging.ErrUnsupported
	}
	_ = img
	var buf bytes.Buffer
	if err := png.Encode(&buf, image.NewNRGBA(image.Rect(0, 0, 2, 2))); err != nil {
		return nil, "", err
	}
	return buf.Bytes(), "png", nil
}

func makePNG(t testing.TB, w, h int, seed int) []byte {
	t.Helper()
	img := image.NewNRGBA(image.Rect(0, 0, w, h))
	c := color.NRGBA{R: uint8(seed), G: uint8(seed >> 8), B: uint8(seed * 7), A: 255}
	for y := 0; y < h && y < 4; y++ {
		for x := 0; x < w && x < 4; x++ {
			img.SetNRGBA(x, y, c)
		}
	}
	img.SetNRGBA(0, 0, color.NRGBA{R: uint8(seed), G: 1, B: uint8(seed >> 8), A: 255})
	var buf bytes.Buffer
	if err := png.Encode(&buf, img); err != nil {
		t.Fatal(err)
	}
	return buf.Bytes()
}

func makeJPEG(t testing.TB, seed int) []byte {
	t.Helper()
	img := image.NewRGBA(image.Rect(0, 0, 8, 8))
	img.Set(1, 1, color.RGBA{R: uint8(seed), A: 255})
	var buf bytes.Buffer
	if err := jpeg.Encode(&buf, img, nil); err != nil {
		t.Fatal(err)
	}
	return buf.Bytes()
}

func makeGIF(t testing.TB) []byte {
	t.Helper()
	var buf bytes.Buffer
	if err := gif.Encode(&buf, image.NewPaletted(image.Rect(0, 0, 3, 3), color.Palette{color.Black, color.White}), nil); err != nil {
		t.Fatal(err)
	}
	return buf.Bytes()
}

type env struct {
	svc   *Service
	db    *meta.DB
	dir   *datadir.Dir
	admin Principal
	t     *testing.T
}

func quietLogger() *slog.Logger { return slog.New(slog.NewTextHandler(io.Discard, nil)) }

func testConfig(dir string) Config {
	return Config{PublicURL: "https://pics.example.com", ImageURL: "https://img.example.com", PublicURLSet: true,
		HardMaxUpload: 100 << 20, SnapshotDir: filepath.Join(dir, "snapshots-out")}
}

// newEnv opens a fresh data dir + DB, wires fake imaging and creates the administrator.
func newEnv(t *testing.T) *env {
	t.Helper()
	root := t.TempDir()
	dir, err := datadir.Open(filepath.Join(root, "data"), datadir.Options{Create: true})
	if err != nil {
		t.Fatal(err)
	}
	db, err := meta.Open(context.Background(), dir.DBPath())
	if err != nil {
		dir.Close()
		t.Fatal(err)
	}
	svc, err := New(db, dir, testConfig(root), quietLogger())
	if err != nil {
		t.Fatal(err)
	}
	svc.Sniff, svc.Inspect, svc.Thumbnail = fakeSniff, fakeInspect, fakeThumbnail
	svc.DiskFree = func() (uint64, uint64, error) { return 500 << 30, 1000 << 30, nil }
	svc.SetLoginLimits(ratelimit.PerMinute(1000), ratelimit.PerMinute(1000))
	e := &env{svc: svc, db: db, dir: dir, t: t}
	t.Cleanup(func() {
		svc.Close()
		db.Close()
		dir.Close()
	})
	ctx := context.Background()
	if err := svc.CreateAdmin(ctx, "admin", "correct horse battery"); err != nil {
		t.Fatal(err)
	}
	u, err := db.GetUserByName(ctx, "admin")
	if err != nil {
		t.Fatal(err)
	}
	e.admin = Principal{Kind: KindSession, UserID: u.ID, Username: "admin", IP: "127.0.0.1"}
	return e
}

func (e *env) tokenPrincipal(scopes ...string) Principal {
	e.t.Helper()
	_, secret, err := e.svc.CreateToken(context.Background(), e.admin, "t-"+strings.Join(scopes, "+"), scopes, nil)
	if err != nil {
		e.t.Fatal(err)
	}
	p, err := e.svc.AuthenticateToken(context.Background(), secret, "127.0.0.1")
	if err != nil {
		e.t.Fatal(err)
	}
	return p
}

// upload runs the full upload sequence for p.
func (e *env) upload(p Principal, name string, data []byte) (UploadResult, error) {
	e.t.Helper()
	ctx := context.Background()
	u, err := e.svc.BeginUpload(ctx, p)
	if err != nil {
		return UploadResult{}, err
	}
	defer u.Close()
	if err := u.WriteFile(ctx, bytes.NewReader(data), name); err != nil {
		return UploadResult{}, err
	}
	return u.Commit(ctx)
}

func (e *env) mustUpload(p Principal, name string, data []byte) ImageView {
	e.t.Helper()
	r, err := e.upload(p, name, data)
	if err != nil {
		e.t.Fatalf("upload %s: %v", name, err)
	}
	return r.Image
}

func (e *env) fileExists(rel string) bool {
	_, err := e.dir.Root().Lstat(rel)
	return err == nil
}

func (e *env) tmpFiles() []string {
	list, _ := e.svc.blobs.ListTemp()
	var out []string
	for _, f := range list {
		out = append(out, f.Name)
	}
	return out
}

func wantCode(t *testing.T, err error, code string) {
	t.Helper()
	if err == nil {
		t.Fatalf("expected error %q, got nil", code)
	}
	if got := ErrorCode(err); got != code {
		t.Fatalf("expected error code %q, got %q (%v)", code, got, err)
	}
}

func isNotExist(err error) bool { return errors.Is(err, os.ErrNotExist) }

var errCrash = errors.New("injected crash")

func crashAt(point string) func(string) error {
	return func(p string) error {
		if p == point {
			return errCrash
		}
		return nil
	}
}

func (e *env) mustCheck(mode string) *CheckReport {
	e.t.Helper()
	rep, err := e.svc.Check(context.Background(), e.admin, mode)
	if err != nil {
		e.t.Fatal(err)
	}
	return rep
}

func hasIssue(rep *CheckReport, kind string) bool {
	for _, is := range rep.Issues {
		if is.Kind == kind {
			return true
		}
	}
	return false
}

func waitFor(t *testing.T, d time.Duration, cond func() bool) {
	t.Helper()
	deadline := time.Now().Add(d)
	for time.Now().Before(deadline) {
		if cond() {
			return
		}
		time.Sleep(10 * time.Millisecond)
	}
	t.Fatal("condition not met in time")
}
