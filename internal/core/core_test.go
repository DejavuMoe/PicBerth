package core

import (
	"bytes"
	"context"
	"crypto/sha256"
	"encoding/hex"
	"fmt"
	"io"
	"os"
	"path/filepath"
	"strings"
	"sync"
	"testing"
	"time"

	"picberth/internal/blob"
	"picberth/internal/datadir"
	"picberth/internal/imaging"
	"picberth/internal/meta"
	"picberth/internal/ratelimit"
)

// assertInvariants checks I1-I5 strictly (the state a successful Check must leave behind).
func assertInvariants(t *testing.T, e *env) {
	t.Helper()
	ctx := context.Background()
	live := map[string]*meta.Image{}
	after := ""
	for {
		batch, err := e.db.ImageIDsAfter(ctx, after, 100)
		if err != nil {
			t.Fatal(err)
		}
		if len(batch) == 0 {
			break
		}
		for _, id := range batch {
			im, _ := e.db.GetImage(ctx, id)
			live[im.ObjectKey] = im
			after = id
			area := areaFor(im.State)
			f, err := e.svc.blobs.Open(area, im.ObjectKey)
			if err != nil {
				t.Errorf("I1/I2: %s record %s: file not in %s: %v", im.State, im.ID, area, err)
				continue
			}
			h := sha256.New()
			n, _ := io.Copy(h, f)
			f.Close()
			if n != im.Size || hex.EncodeToString(h.Sum(nil)) != im.SHA256 {
				t.Errorf("I1: record %s: content differs from record", im.ID)
			}
		}
	}
	for _, area := range []blob.Area{blob.Objects, blob.Trash} {
		e.svc.blobs.Walk(area, func(f blob.File) error {
			im := live[f.Key]
			if im == nil || areaFor(im.State) != area {
				t.Errorf("I3: file %s has no matching live record", f.Rel)
			}
			return nil
		})
	}
	stale := UploadTimeout(e.svc.cfg.HardMaxUpload)
	for _, tf := range mustListTemp(e) {
		if time.Since(tf.ModTime) > stale {
			t.Errorf("I4: stale temp file %s", tf.Name)
		}
	}
}

func mustListTemp(e *env) []blob.TempFile {
	l, _ := e.svc.blobs.ListTemp()
	return l
}

func TestUploadDedupeAndLifecycle(t *testing.T) {
	e := newEnv(t)
	ctx := context.Background()
	data := makePNG(t, 10, 10, 1)

	r, err := e.upload(e.admin, "../../evil dir/My Shot.PNG", data)
	if err != nil {
		t.Fatal(err)
	}
	v := r.Image
	if r.Duplicate || v.Duplicate || v.State != "active" || v.Ext != "png" || v.Mime != "image/png" || v.Width != 10 {
		t.Fatalf("unexpected view: %+v", v)
	}
	if v.Filename != "My Shot.PNG" {
		t.Fatalf("filename not sanitized: %q", v.Filename)
	}
	if v.Visibility != "public" || v.Source.Kind != "web" {
		t.Fatalf("defaults: %+v", v)
	}
	wantURL := "https://img.example.com/i/" + v.ID + ".png"
	if v.URL != wantURL || v.Links.Markdown != "![My Shot]("+wantURL+")" || !strings.Contains(v.Links.HTML, `width="10" height="10"`) || v.Links.BBCode != "[img]"+wantURL+"[/img]" {
		t.Fatalf("links: %+v", v.Links)
	}
	sum := sha256.Sum256(data)
	if v.SHA256 != hex.EncodeToString(sum[:]) || v.Size != int64(len(data)) {
		t.Fatal("hash/size mismatch")
	}
	if !e.fileExists("objects/" + v.ID[0:2] + "/" + v.ID[2:4] + "/" + v.ID + ".png") {
		t.Fatal("object file missing")
	}
	if len(e.tmpFiles()) != 0 {
		t.Fatalf("temp files left: %v", e.tmpFiles())
	}

	d, err := e.upload(e.admin, "again.png", data)
	if err != nil || !d.Duplicate || d.Image.ID != v.ID || !d.Image.Duplicate {
		t.Fatalf("duplicate: %+v %v", d, err)
	}
	if len(e.tmpFiles()) != 0 {
		t.Fatal("duplicate left a temp file")
	}

	// trash -> invisible to public lookups, file moved
	if _, err := e.svc.LookupPublic(ctx, v.ID, "png"); err != nil {
		t.Fatal(err)
	}
	if err := e.svc.Trash(ctx, e.admin, v.ID); err != nil {
		t.Fatal(err)
	}
	if err := e.svc.Trash(ctx, e.admin, v.ID); err != nil {
		t.Fatalf("trash must be idempotent: %v", err)
	}
	_, err = e.svc.LookupPublic(ctx, v.ID, "png")
	wantCode(t, err, CodeNotFound)
	if e.fileExists("objects/"+v.ID[0:2]+"/"+v.ID[2:4]+"/"+v.ID+".png") || !e.fileExists("trash/"+v.ID[0:2]+"/"+v.ID[2:4]+"/"+v.ID+".png") {
		t.Fatal("file not moved to trash")
	}
	wantCode(t, mustErr(e.svc.Purge(ctx, e.admin, "00000000000000000000")), CodeNotFound)

	// a duplicate of a trashed image is a new image
	d2 := e.mustUpload(e.admin, "new.png", data)
	if d2.ID == v.ID || d2.Duplicate {
		t.Fatal("upload after trash must create a new image")
	}
	// restore of the old one works, state conflict otherwise
	if _, err := e.svc.Restore(ctx, e.admin, d2.ID); err == nil {
		t.Fatal("restoring an active image must conflict")
	} else {
		wantCode(t, err, CodeConflict)
	}
	rv, err := e.svc.Restore(ctx, e.admin, v.ID)
	if err != nil || rv.State != "active" {
		t.Fatalf("restore: %+v %v", rv, err)
	}
	wantCode(t, e.svc.Purge(ctx, e.admin, v.ID), CodeConflict) // purge only from trash

	if err := e.svc.Trash(ctx, e.admin, v.ID); err != nil {
		t.Fatal(err)
	}
	if err := e.svc.Purge(ctx, e.admin, v.ID); err != nil {
		t.Fatal(err)
	}
	if e.fileExists("trash/" + v.ID[0:2] + "/" + v.ID[2:4] + "/" + v.ID + ".png") {
		t.Fatal("purged file still present")
	}
	im, err := e.db.GetImage(ctx, v.ID)
	if err != nil || im.State != "purged" || im.SHA256 != v.SHA256 {
		t.Fatalf("tombstone: %+v %v", im, err)
	}
	_, err = e.svc.GetImage(ctx, e.admin, v.ID)
	wantCode(t, err, CodeNotFound)
	wantCode(t, e.svc.Purge(ctx, e.admin, v.ID), CodeNotFound)
	assertInvariants(t, e)
}

func mustErr(err error) error { return err }

func TestUploadValidation(t *testing.T) {
	e := newEnv(t)
	ctx := context.Background()

	_, err := e.upload(e.admin, "x.png", []byte("<svg xmlns='http://www.w3.org/2000/svg'><script>alert(1)</script></svg>"))
	wantCode(t, err, CodeUnsupportedFormat)
	_, err = e.upload(e.admin, "x.png", []byte("<html><body>hi</body></html>"))
	wantCode(t, err, CodeUnsupportedFormat)
	_, err = e.upload(e.admin, "x.png", nil)
	wantCode(t, err, CodeUnsupportedFormat)

	// signature ok, structure broken
	_, err = e.upload(e.admin, "x.png", append([]byte("\x89PNG\r\n\x1a\n"), bytes.Repeat([]byte{1}, 40)...))
	wantCode(t, err, CodeInvalidImage)

	// too many pixels
	if _, err := e.svc.UpdateSettings(ctx, e.admin, SettingsPatch{MaxPixels: ptr(int64(1_000_000))}); err != nil {
		t.Fatal(err)
	}
	_, err = e.upload(e.admin, "big.png", makePNG(t, 1100, 1100, 3))
	wantCode(t, err, CodeTooManyPixels)

	// format not allowed
	if _, err := e.svc.UpdateSettings(ctx, e.admin, SettingsPatch{AllowedFormats: &[]string{"jpeg"}}); err != nil {
		t.Fatal(err)
	}
	_, err = e.upload(e.admin, "p.png", makePNG(t, 5, 5, 4))
	wantCode(t, err, CodeUnsupportedFormat)
	if _, err := e.upload(e.admin, "p.jpg", makeJPEG(t, 4)); err != nil {
		t.Fatal(err)
	}

	// size limit: streamed, aborted, temp file removed
	if _, err := e.svc.UpdateSettings(ctx, e.admin, SettingsPatch{MaxUploadBytes: ptr(int64(2048)), AllowedFormats: &[]string{"jpeg", "png"}}); err != nil {
		t.Fatal(err)
	}
	_, err = e.upload(e.admin, "huge.png", append(makePNG(t, 4, 4, 5), bytes.Repeat([]byte{0}, 5000)...))
	wantCode(t, err, CodeTooLarge)
	if len(e.tmpFiles()) != 0 {
		t.Fatalf("temp file left after 413: %v", e.tmpFiles())
	}
	assertInvariants(t, e)
}

func ptr[T any](v T) *T { return &v }

func TestUploadConcurrencyAndDiskWatermark(t *testing.T) {
	e := newEnv(t)
	ctx := context.Background()
	var started []*Upload
	for i := 0; i < maxConcurrentUploads; i++ {
		u, err := e.svc.BeginUpload(ctx, e.admin)
		if err != nil {
			t.Fatalf("upload %d: %v", i, err)
		}
		started = append(started, u)
	}
	_, err := e.svc.BeginUpload(ctx, e.admin)
	wantCode(t, err, CodeRateLimited)
	if ce := err.(*Error); ce.RetryAfter != 2*time.Second {
		t.Fatalf("Retry-After = %v", ce.RetryAfter)
	}
	for _, u := range started {
		u.Close()
	}
	u, err := e.svc.BeginUpload(ctx, e.admin)
	if err != nil {
		t.Fatalf("slot not released: %v", err)
	}
	u.Close()

	e.svc.DiskFree = func() (uint64, uint64, error) { return 100 << 20, 100 << 30, nil }
	_, err = e.svc.BeginUpload(ctx, e.admin)
	wantCode(t, err, CodeInsufficientStorage)
	if w := watermark(10 << 40); w != (10<<40)/50 {
		t.Fatalf("watermark(10 TiB) = %d", w)
	}
	if w := watermark(10 << 30); w != 1<<30 {
		t.Fatalf("watermark(10 GiB) = %d", w)
	}
}

func TestUploadRateLimit(t *testing.T) {
	e := newEnv(t)
	if _, err := e.svc.UpdateSettings(context.Background(), e.admin, SettingsPatch{UploadsPerMinute: ptr(2)}); err != nil {
		t.Fatal(err)
	}
	for i := 0; i < 2; i++ {
		u, err := e.svc.BeginUpload(context.Background(), e.admin)
		if err != nil {
			t.Fatal(err)
		}
		u.Close()
	}
	_, err := e.svc.BeginUpload(context.Background(), e.admin)
	wantCode(t, err, CodeRateLimited)
	if err.(*Error).RetryAfter < time.Second {
		t.Fatal("missing Retry-After")
	}
}

func TestAuthorizationMatrix(t *testing.T) {
	e := newEnv(t)
	ctx := context.Background()
	anon := Principal{}
	upTok := e.tokenPrincipal(ScopeUpload)
	rdTok := e.tokenPrincipal(ScopeRead)
	mgTok := e.tokenPrincipal(ScopeManage)
	upTok2 := e.tokenPrincipal(ScopeUpload)

	own := e.mustUpload(upTok, "own.png", makePNG(t, 5, 5, 10))       // uploaded by upTok
	other := e.mustUpload(e.admin, "other.png", makePNG(t, 5, 5, 11)) // uploaded by session
	if own.Source.Kind != "token" || own.Source.TokenID == nil || own.Source.TokenName == "" {
		t.Fatalf("token source: %+v", own.Source)
	}
	if _, err := e.upload(rdTok, "x.png", makePNG(t, 5, 5, 12)); err == nil {
		t.Fatal("read token uploaded")
	} else {
		wantCode(t, err, CodeForbidden)
	}
	if _, err := e.upload(anon, "x.png", makePNG(t, 5, 5, 12)); err == nil {
		t.Fatal("anonymous uploaded")
	} else {
		wantCode(t, err, CodeUnauthorized)
	}
	if _, err := e.upload(mgTok, "mg.png", makePNG(t, 5, 5, 13)); err != nil {
		t.Fatalf("manage token upload: %v", err)
	}

	// visibility of records
	_, err := e.svc.GetImage(ctx, upTok, other.ID)
	wantCode(t, err, CodeNotFound)
	_, err = e.svc.GetImage(ctx, upTok2, own.ID)
	wantCode(t, err, CodeNotFound)
	if _, err := e.svc.GetImage(ctx, upTok, own.ID); err != nil {
		t.Fatal(err)
	}
	for _, p := range []Principal{rdTok, mgTok, e.admin} {
		if _, err := e.svc.GetImage(ctx, p, other.ID); err != nil {
			t.Fatalf("%v cannot read: %v", p.Actor(), err)
		}
	}
	_, err = e.svc.GetImage(ctx, anon, own.ID)
	wantCode(t, err, CodeUnauthorized)

	items, _, err := e.svc.ListImages(ctx, upTok, ListParams{})
	if err != nil || len(items) != 1 || items[0].ID != own.ID {
		t.Fatalf("upload token list: %v %v", items, err)
	}
	items, _, _ = e.svc.ListImages(ctx, rdTok, ListParams{})
	if len(items) != 3 {
		t.Fatalf("read token list: %d", len(items))
	}
	// upload token may not widen the view with tokenId
	items, _, _ = e.svc.ListImages(ctx, upTok, ListParams{TokenID: ptr(int64(mgTok.TokenID))})
	if len(items) != 1 || items[0].ID != own.ID {
		t.Fatalf("upload token escaped its scope: %v", items)
	}
	_, _, err = e.svc.ListImages(ctx, upTok, ListParams{State: "trashed"})
	wantCode(t, err, CodeForbidden)
	_, _, err = e.svc.ListImages(ctx, rdTok, ListParams{State: "trashed"})
	wantCode(t, err, CodeForbidden)

	// mutations
	for _, p := range []Principal{upTok, rdTok} {
		_, err := e.svc.SetVisibility(ctx, p, own.ID, "private")
		wantCode(t, err, CodeForbidden)
		wantCode(t, e.svc.Trash(ctx, p, own.ID), CodeForbidden)
		_, err = e.svc.Restore(ctx, p, own.ID)
		wantCode(t, err, CodeForbidden)
		wantCode(t, e.svc.Purge(ctx, p, own.ID), CodeForbidden)
		_, err = e.svc.Batch(ctx, p, "trash", []string{own.ID}, "")
		wantCode(t, err, CodeForbidden)
	}
	wantCode(t, e.svc.Trash(ctx, anon, own.ID), CodeUnauthorized)
	if _, err := e.svc.SetVisibility(ctx, mgTok, other.ID, "private"); err != nil {
		t.Fatalf("manage token: %v", err)
	}

	// raw bytes of private images
	if _, err := e.svc.OpenRaw(ctx, upTok, other.ID); err == nil {
		t.Fatal("upload token read someone else's bytes")
	}
	o, err := e.svc.OpenRaw(ctx, rdTok, other.ID)
	if err != nil {
		t.Fatalf("read token raw of private: %v", err)
	}
	o.File.Close()
	o, err = e.svc.OpenRaw(ctx, upTok, own.ID)
	if err != nil {
		t.Fatalf("upload token raw of own: %v", err)
	}
	o.File.Close()
	if _, err := e.svc.OpenRaw(ctx, anon, own.ID); err == nil {
		t.Fatal("anonymous raw")
	}

	// admin-only surfaces
	for _, p := range []Principal{upTok, rdTok, mgTok} {
		_, _, err := e.svc.CreateToken(ctx, p, "x", []string{"upload"}, nil)
		wantCode(t, err, CodeForbidden)
		_, err = e.svc.ListTokens(ctx, p)
		wantCode(t, err, CodeForbidden)
		wantCode(t, e.svc.RevokeToken(ctx, p, 1), CodeForbidden)
		_, err = e.svc.GetSettings(p)
		wantCode(t, err, CodeForbidden)
		_, err = e.svc.UpdateSettings(ctx, p, SettingsPatch{Dedupe: ptr(false)})
		wantCode(t, err, CodeForbidden)
		_, err = e.svc.Status(ctx, p)
		wantCode(t, err, CodeForbidden)
		_, err = e.svc.Check(ctx, p, "quick")
		wantCode(t, err, CodeForbidden)
		_, err = e.svc.Snapshot(ctx, p, "")
		wantCode(t, err, CodeForbidden)
		_, _, err = e.svc.ListAudit(ctx, p, 10, "")
		wantCode(t, err, CodeForbidden)
	}
	_, err = e.svc.GetSettings(anon)
	wantCode(t, err, CodeUnauthorized)

	// trashed images are invisible to non-manage tokens
	if err := e.svc.Trash(ctx, mgTok, own.ID); err != nil {
		t.Fatal(err)
	}
	_, err = e.svc.GetImage(ctx, upTok, own.ID)
	wantCode(t, err, CodeNotFound)
	if _, err := e.svc.GetImage(ctx, mgTok, own.ID); err != nil {
		t.Fatal(err)
	}

	// revoked and expired tokens stop working
	if err := e.svc.RevokeToken(ctx, e.admin, rdTok.TokenID); err != nil {
		t.Fatal(err)
	}
	if err := e.svc.RevokeToken(ctx, e.admin, rdTok.TokenID); err != nil {
		t.Fatalf("revoke must be idempotent: %v", err)
	}
	wantCode(t, e.svc.RevokeToken(ctx, e.admin, 9999), CodeNotFound)
}

func TestTokenLifecycle(t *testing.T) {
	e := newEnv(t)
	ctx := context.Background()
	days := 1
	tv, secret, err := e.svc.CreateToken(ctx, e.admin, "  laptop  ", []string{"manage", "upload", "upload"}, &days)
	if err != nil {
		t.Fatal(err)
	}
	if tv.Name != "laptop" || !strings.HasPrefix(secret, "pb_") || tv.Prefix != secret[:9] || fmt.Sprint(tv.Scopes) != "[upload manage]" || tv.ExpiresAt == nil {
		t.Fatalf("token: %+v secret=%q", tv, secret)
	}
	if _, err := e.svc.AuthenticateToken(ctx, secret, "ip"); err != nil {
		t.Fatal(err)
	}
	last := "0"
	if strings.HasSuffix(secret, "0") {
		last = "1"
	}
	if _, err := e.svc.AuthenticateToken(ctx, secret[:len(secret)-1]+last, "ip"); err == nil {
		t.Fatal("wrong secret accepted")
	}
	// expiry
	e.svc.Now = func() time.Time { return time.Now().Add(25 * time.Hour) }
	_, err = e.svc.AuthenticateToken(ctx, secret, "ip")
	wantCode(t, err, CodeUnauthorized)
	e.svc.Now = time.Now

	for _, bad := range []struct {
		name   string
		scopes []string
		days   *int
	}{
		{"", []string{"upload"}, nil}, {strings.Repeat("x", 81), []string{"upload"}, nil}, {"ok", nil, nil}, {"ok", []string{"root"}, nil},
		{"ok", []string{"upload"}, ptr(0)}, {"ok", []string{"upload"}, ptr(3651)}, {"bad\nname", []string{"upload"}, nil},
	} {
		_, _, err := e.svc.CreateToken(ctx, e.admin, bad.name, bad.scopes, bad.days)
		wantCode(t, err, CodeValidation)
	}
	list, _ := e.svc.ListTokens(ctx, e.admin)
	if len(list) != 1 {
		t.Fatalf("token list: %d", len(list))
	}
	if err := e.svc.RevokeToken(ctx, e.admin, tv.ID); err != nil {
		t.Fatal(err)
	}
	list, _ = e.svc.ListTokens(ctx, e.admin)
	if len(list) != 1 || list[0].RevokedAt == nil {
		t.Fatalf("revoked token must stay listed: %+v", list)
	}
}

func TestSetupLoginSessions(t *testing.T) {
	root := t.TempDir()
	dir, _ := datadir.Open(filepath.Join(root, "d"), datadir.Options{Create: true})
	db, _ := meta.Open(context.Background(), dir.DBPath())
	svc, err := New(db, dir, testConfig(root), quietLogger())
	if err != nil {
		t.Fatal(err)
	}
	t.Cleanup(func() { svc.Close(); db.Close(); dir.Close() })
	ctx := context.Background()
	svc.SetLoginLimits(ratelimit.PerMinute(1000), ratelimit.PerMinute(1000))

	if req, _ := svc.SetupRequired(ctx); !req {
		t.Fatal("setup should be required")
	}
	_, err = svc.Login(ctx, "1.1.1.1", "ua", "admin", "whatever1234")
	wantCode(t, err, CodeSetupRequired)
	code, err := svc.EnsureSetupCode(ctx)
	if err != nil || len(code) != 14 {
		t.Fatalf("setup code %q %v", code, err)
	}
	if again, _ := svc.EnsureSetupCode(ctx); again != code {
		t.Fatal("setup code changed between calls")
	}
	_, err = svc.Setup(ctx, "1.1.1.1", "ua", "AAAA-AAAA-AAAA", "admin", "correct horse battery")
	wantCode(t, err, CodeForbidden)
	_, err = svc.Setup(ctx, "1.1.1.1", "ua", code, "bad name!", "correct horse battery")
	wantCode(t, err, CodeValidation)
	_, err = svc.Setup(ctx, "1.1.1.1", "ua", code, "admin", "short")
	wantCode(t, err, CodeValidation)
	sess, err := svc.Setup(ctx, "1.1.1.1", "ua", strings.ToLower(code), "admin", "correct horse battery")
	if err != nil || sess.ID == "" {
		t.Fatalf("setup: %v", err)
	}
	_, err = svc.Setup(ctx, "1.1.1.1", "ua", code, "admin2", "correct horse battery")
	wantCode(t, err, CodeSetupDone)
	if c, _ := svc.EnsureSetupCode(ctx); c != "" {
		t.Fatal("setup code still offered after setup")
	}

	p, err := svc.AuthenticateSession(ctx, sess.ID, "1.1.1.1")
	if err != nil || p.Username != "admin" || p.Kind != KindSession {
		t.Fatalf("session auth: %+v %v", p, err)
	}
	if _, err := svc.AuthenticateSession(ctx, "garbage", "ip"); err == nil {
		t.Fatal("garbage session accepted")
	}

	// stored hash is hashed, not plaintext
	var raw string
	db.Reader().QueryRow(`SELECT id_hash FROM sessions`).Scan(&raw)
	if raw == sess.ID || len(raw) != 64 {
		t.Fatalf("session id stored unhashed: %q", raw)
	}

	_, err = svc.Login(ctx, "1.1.1.1", "ua", "admin", "wrong password!!")
	wantCode(t, err, CodeUnauthorized)
	_, err = svc.Login(ctx, "1.1.1.1", "ua", "nobody", "wrong password!!")
	wantCode(t, err, CodeUnauthorized)
	login, err := svc.Login(ctx, "1.1.1.1", "ua", "ADMIN", "correct horse battery")
	if err != nil {
		t.Fatal(err)
	}
	if err := svc.Logout(ctx, p, login.ID); err != nil {
		t.Fatal(err)
	}
	if _, err := svc.AuthenticateSession(ctx, login.ID, "ip"); err == nil {
		t.Fatal("logged-out session still valid")
	}

	// idle and absolute expiry
	svc.Now = func() time.Time { return time.Now().Add(31 * time.Minute) }
	if _, err := svc.AuthenticateSession(ctx, sess.ID, "ip"); err == nil {
		t.Fatal("idle-expired session accepted")
	}
	svc.Now = time.Now
	s2, _ := svc.Login(ctx, "1.1.1.1", "ua", "admin", "correct horse battery")
	for i := 1; i <= 13; i++ { // keep it active every 20 minutes, but cross the 12 h absolute limit
		at := time.Now().Add(time.Duration(i) * 55 * time.Minute)
		svc.Now = func() time.Time { return at }
		_, err = svc.AuthenticateSession(ctx, s2.ID, "ip")
		if err != nil {
			break
		}
	}
	if err == nil {
		t.Fatal("absolute expiry not enforced")
	}

	// audit trail
	items, _, err := svc.ListAudit(ctx, p, 100, "")
	if err != nil {
		t.Fatal(err)
	}
	var actions []string
	for _, it := range items {
		actions = append(actions, it.Action)
	}
	joined := strings.Join(actions, ",")
	for _, want := range []string{"setup", "setup.failure", "login.success", "login.failure", "logout"} {
		if !strings.Contains(joined, want) {
			t.Errorf("audit lacks %q: %s", want, joined)
		}
	}
	for _, it := range items {
		if strings.Contains(it.Detail, "wrong password") || strings.Contains(it.Detail, "correct horse") {
			t.Fatalf("password leaked into audit: %+v", it)
		}
	}
}

func TestLoginRateLimits(t *testing.T) {
	e := newEnv(t)
	ctx := context.Background()
	e.svc.SetLoginLimits(ratelimit.PerMinute(5), ratelimit.New(10.0/600, 10))
	for i := 0; i < 5; i++ {
		_, err := e.svc.Login(ctx, "9.9.9.9", "ua", "admin", "wrong password!!")
		wantCode(t, err, CodeUnauthorized)
	}
	_, err := e.svc.Login(ctx, "9.9.9.9", "ua", "admin", "correct horse battery")
	wantCode(t, err, CodeRateLimited)
	if err.(*Error).RetryAfter < time.Second {
		t.Fatal("missing retry-after")
	}
	// another IP is not affected by the per-IP bucket
	if _, err := e.svc.Login(ctx, "9.9.9.8", "ua", "admin", "correct horse battery"); err != nil {
		t.Fatalf("other ip: %v", err)
	}
	// the per-username bucket (10 / 10 min) still applies across IPs: 1 success above refunded, so burn it with failures
	for i := 0; i < 12; i++ {
		e.svc.Login(ctx, fmt.Sprintf("8.8.8.%d", i), "ua", "admin", "wrong password!!")
	}
	_, err = e.svc.Login(ctx, "7.7.7.7", "ua", "admin", "correct horse battery")
	wantCode(t, err, CodeRateLimited)
}

func TestSettingsValidationAndAudit(t *testing.T) {
	e := newEnv(t)
	ctx := context.Background()
	v, err := e.svc.GetSettings(e.admin)
	if err != nil || v.MaxUploadBytes != 20<<20 || v.HardMaxUploadBytes != 100<<20 || v.PublicURL != "https://pics.example.com" || len(v.AllowedFormats) != 5 {
		t.Fatalf("defaults: %+v %v", v, err)
	}
	bad := []SettingsPatch{
		{MaxUploadBytes: ptr(int64(10))}, {MaxUploadBytes: ptr(int64(200 << 20))}, {MaxPixels: ptr(int64(5))}, {MaxPixels: ptr(int64(3e8))},
		{AllowedFormats: &[]string{}}, {AllowedFormats: &[]string{"svg"}}, {AllowedFormats: &[]string{"png", "png"}},
		{DefaultVisibility: ptr("secret")}, {TrashRetentionDays: ptr(-1)}, {TrashRetentionDays: ptr(366)},
		{UploadsPerMinute: ptr(0)}, {UploadsPerMinute: ptr(10001)}, {SessionIdleMinutes: ptr(4)}, {SessionAbsoluteHours: ptr(0)},
		{PublicURL: ptr("https://evil.example")}, {HardMaxUploadBytes: ptr(int64(1))},
	}
	for i, p := range bad {
		if _, err := e.svc.UpdateSettings(ctx, e.admin, p); err == nil {
			t.Errorf("bad patch %d accepted: %+v", i, p)
		} else {
			wantCode(t, err, CodeValidation)
		}
	}
	// echoing the read-only fields back unchanged is fine
	v2, err := e.svc.UpdateSettings(ctx, e.admin, SettingsPatch{PublicURL: &v.PublicURL, ImageURL: &v.ImageURL, HardMaxUploadBytes: &v.HardMaxUploadBytes, Dedupe: ptr(false), TrashRetentionDays: ptr(0)})
	if err != nil || v2.Dedupe || v2.TrashRetentionDays != 0 {
		t.Fatalf("update: %+v %v", v2, err)
	}
	// persisted across a restart of the service
	svc2, err := New(e.db, e.dir, testConfig(t.TempDir()), quietLogger())
	if err != nil {
		t.Fatal(err)
	}
	defer svc2.Close()
	if st := svc2.Settings(); st.Dedupe || st.TrashRetentionDays != 0 {
		t.Fatalf("settings not persisted: %+v", st)
	}
	items, _, _ := e.svc.ListAudit(ctx, e.admin, 10, "")
	found := false
	for _, it := range items {
		if it.Action == "settings.update" && strings.Contains(it.Detail, "dedupe") {
			found = true
		}
	}
	if !found {
		t.Fatalf("settings change not audited: %+v", items)
	}
}

func TestRetentionZeroPurgesImmediately(t *testing.T) {
	e := newEnv(t)
	ctx := context.Background()
	if _, err := e.svc.UpdateSettings(ctx, e.admin, SettingsPatch{TrashRetentionDays: ptr(0)}); err != nil {
		t.Fatal(err)
	}
	v := e.mustUpload(e.admin, "a.png", makePNG(t, 5, 5, 20))
	if err := e.svc.Trash(ctx, e.admin, v.ID); err != nil {
		t.Fatal(err)
	}
	im, _ := e.db.GetImage(ctx, v.ID)
	if im.State != "purged" || e.fileExists("trash/"+v.ID[0:2]+"/"+v.ID[2:4]+"/"+v.ID+".png") || e.fileExists("objects/"+v.ID[0:2]+"/"+v.ID[2:4]+"/"+v.ID+".png") {
		t.Fatalf("not purged: %+v", im)
	}
}

func TestPurgeExpired(t *testing.T) {
	e := newEnv(t)
	ctx := context.Background()
	old := e.mustUpload(e.admin, "old.png", makePNG(t, 5, 5, 30))
	fresh := e.mustUpload(e.admin, "fresh.png", makePNG(t, 5, 5, 31))
	e.svc.Now = func() time.Time { return time.Now().Add(-40 * 24 * time.Hour) }
	if err := e.svc.Trash(ctx, e.admin, old.ID); err != nil {
		t.Fatal(err)
	}
	e.svc.Now = time.Now
	if err := e.svc.Trash(ctx, e.admin, fresh.ID); err != nil {
		t.Fatal(err)
	}
	n, err := e.svc.PurgeExpired(ctx)
	if err != nil || n != 1 {
		t.Fatalf("PurgeExpired = %d, %v", n, err)
	}
	if im, _ := e.db.GetImage(ctx, old.ID); im.State != "purged" {
		t.Fatal("old image not purged")
	}
	if im, _ := e.db.GetImage(ctx, fresh.ID); im.State != "trashed" {
		t.Fatal("fresh image purged")
	}
	assertInvariants(t, e)
}

func TestThumbnailWorker(t *testing.T) {
	e := newEnv(t)
	ctx := context.Background()
	a := e.mustUpload(e.admin, "a.png", makePNG(t, 5, 5, 40))
	b := e.mustUpload(e.admin, "b.gif", makeGIF(t))
	e.svc.ProcessThumbnails(ctx)
	ia, _ := e.db.GetImage(ctx, a.ID)
	ib, _ := e.db.GetImage(ctx, b.ID)
	if ia.Thumb != "png" {
		t.Fatalf("thumb state %q", ia.Thumb)
	}
	if ib.Thumb != "png" && ib.Thumb != "none" {
		t.Fatalf("gif thumb state %q", ib.Thumb)
	}
	o, err := e.svc.OpenThumb(ctx, e.admin, a.ID)
	if err != nil {
		t.Fatal(err)
	}
	data, _ := io.ReadAll(o.File)
	o.File.Close()
	if o.ContentType != "image/png" || len(data) == 0 {
		t.Fatalf("thumb: %s %d", o.ContentType, len(data))
	}
	if v, _ := e.svc.GetImage(ctx, e.admin, a.ID); !v.HasThumb {
		t.Fatal("hasThumb false")
	}
	// purge removes the thumbnail
	e.svc.Trash(ctx, e.admin, a.ID)
	e.svc.Purge(ctx, e.admin, a.ID)
	if e.fileExists("cache/thumbs/" + a.ID[0:2] + "/" + a.ID + ".png") {
		t.Fatal("thumbnail survived purge")
	}
	// failing thumbnailer marks 'none' and never panics
	e.svc.Thumbnail = func(io.ReaderAt, int64, imaging.Info, int, int64) ([]byte, string, error) { panic("decoder bug") }
	c := e.mustUpload(e.admin, "c.png", makePNG(t, 6, 6, 41))
	e.svc.ProcessThumbnails(ctx)
	if ic, _ := e.db.GetImage(ctx, c.ID); ic.Thumb != "none" {
		t.Fatalf("panicking thumbnailer: state %q", ic.Thumb)
	}
}

func TestSanitizeFilename(t *testing.T) {
	long := strings.Repeat("é", 200) + ".png"
	cases := map[string]string{
		"photo.png":                   "photo.png",
		"../../etc/passwd":            "passwd",
		`C:\Users\x\evil.png`:         "evil.png",
		"a\x00b\r\n.png":              "ab.png",
		"":                            "image.png",
		"..":                          "image.png",
		"   ":                         "image.png",
		"dir/":                        "image.png",
		"x\u202egnp.exe":              "xgnp.exe",
		"bad\xffutf8.png":             "badutf8.png",
		"trailing dots...":            "trailing dots",
		"name with  spaces .png":      "name with  spaces .png",
		"emoji \U0001F600 \u2028.gif": "emoji \U0001F600 .gif",
	}
	for in, want := range cases {
		if got := SanitizeFilename(in, "png"); got != want {
			t.Errorf("SanitizeFilename(%q) = %q, want %q", in, got, want)
		}
	}
	got := SanitizeFilename(long, "png")
	if len(got) > 255 || !strings.HasPrefix(got, "é") || strings.ContainsRune(got, '\uFFFD') {
		t.Errorf("long name: len=%d", len(got))
	}
}

// --- crash consistency -------------------------------------------------------------------------

func TestCrashAfterRenameBeforeInsert(t *testing.T) {
	e := newEnv(t)
	ctx := context.Background()
	keep := e.mustUpload(e.admin, "keep.png", makePNG(t, 5, 5, 50))
	e.svc.Fault = crashAt(FaultUploadAfterRename)
	_, err := e.upload(e.admin, "lost.png", makePNG(t, 5, 5, 51))
	if err != errCrash {
		t.Fatalf("expected injected crash, got %v", err)
	}
	e.svc.Fault = nil
	// the orphan sits in objects/ without a record
	orphans := 0
	e.svc.blobs.Walk(blob.Objects, func(f blob.File) error {
		if _, err := e.db.GetImageByKey(ctx, f.Key); err != nil {
			orphans++
		}
		return nil
	})
	if orphans != 1 {
		t.Fatalf("expected 1 orphan, found %d", orphans)
	}
	if rep := e.mustCheck(CheckQuick); hasIssue(rep, IssueOrphanFile) {
		t.Fatal("quick check does not walk objects/")
	}
	rep := e.mustCheck(CheckFull)
	if !hasIssue(rep, IssueOrphanFile) || !rep.OK {
		t.Fatalf("full check: %+v", rep)
	}
	if q, _ := os.ReadDir(filepath.Join(e.dir.Path(), "quarantine")); len(q) != 1 {
		t.Fatalf("quarantine holds %d files", len(q))
	}
	assertInvariants(t, e)
	if rep := e.mustCheck(CheckFull); len(rep.Issues) != 0 {
		t.Fatalf("second check not clean: %+v", rep.Issues)
	}
	// the surviving image is unaffected and uploads still work
	if _, err := e.svc.GetImage(ctx, e.admin, keep.ID); err != nil {
		t.Fatal(err)
	}
	e.mustUpload(e.admin, "retry.png", makePNG(t, 5, 5, 51))
	assertInvariants(t, e)
}

func TestCrashAfterTrashDBBeforeMove(t *testing.T) {
	e := newEnv(t)
	ctx := context.Background()
	v := e.mustUpload(e.admin, "a.png", makePNG(t, 5, 5, 60))
	e.svc.Fault = crashAt(FaultTrashAfterDB)
	if err := e.svc.Trash(ctx, e.admin, v.ID); err != errCrash {
		t.Fatalf("got %v", err)
	}
	e.svc.Fault = nil
	im, _ := e.db.GetImage(ctx, v.ID)
	if im.State != "trashed" || !e.fileExists("objects/"+im.ObjectKey) {
		t.Fatalf("setup of crash state failed: %+v", im)
	}
	// already invisible publicly although the file is still in objects/
	_, err := e.svc.LookupPublic(ctx, v.ID, "png")
	wantCode(t, err, CodeNotFound)
	rep := e.mustCheck(CheckQuick)
	if !hasIssue(rep, IssuePendingMove) || !rep.OK {
		t.Fatalf("quick check should complete the move: %+v", rep)
	}
	if e.fileExists("objects/"+im.ObjectKey) || !e.fileExists("trash/"+im.ObjectKey) {
		t.Fatal("move not completed")
	}
	assertInvariants(t, e)
	if rep := e.mustCheck(CheckFull); len(rep.Issues) != 0 {
		t.Fatalf("not clean: %+v", rep.Issues)
	}
}

func TestCrashRestoreAfterMoveBeforeDB(t *testing.T) {
	e := newEnv(t)
	ctx := context.Background()
	v := e.mustUpload(e.admin, "a.png", makePNG(t, 5, 5, 70))
	if err := e.svc.Trash(ctx, e.admin, v.ID); err != nil {
		t.Fatal(err)
	}
	e.svc.Fault = crashAt(FaultRestoreAfterMove)
	_, err := e.svc.Restore(ctx, e.admin, v.ID)
	if err != errCrash {
		t.Fatalf("got %v", err)
	}
	e.svc.Fault = nil
	im, _ := e.db.GetImage(ctx, v.ID)
	if im.State != "trashed" || !e.fileExists("objects/"+im.ObjectKey) {
		t.Fatalf("crash state wrong: %+v", im)
	}
	rep := e.mustCheck(CheckQuick)
	if !hasIssue(rep, IssuePendingMove) || !rep.OK {
		t.Fatalf("check: %+v", rep)
	}
	assertInvariants(t, e) // record is still trashed, file back in trash
	// and the restore can simply be retried
	if _, err := e.svc.Restore(ctx, e.admin, v.ID); err != nil {
		t.Fatal(err)
	}
	assertInvariants(t, e)
}

func TestCrashAfterPurgeDBBeforeDelete(t *testing.T) {
	e := newEnv(t)
	ctx := context.Background()
	v := e.mustUpload(e.admin, "a.png", makePNG(t, 5, 5, 80))
	other := e.mustUpload(e.admin, "b.png", makePNG(t, 5, 5, 81))
	e.svc.Trash(ctx, e.admin, v.ID)
	e.svc.Fault = crashAt(FaultPurgeAfterDB)
	if err := e.svc.Purge(ctx, e.admin, v.ID); err != errCrash {
		t.Fatalf("got %v", err)
	}
	e.svc.Fault = nil
	im, _ := e.db.GetImage(ctx, v.ID)
	if im.State != "purged" || !e.fileExists("trash/"+im.ObjectKey) {
		t.Fatalf("crash state wrong: %+v", im)
	}
	rep := e.mustCheck(CheckQuick)
	if !hasIssue(rep, IssueOrphanFile) || !rep.OK {
		t.Fatalf("check: %+v", rep)
	}
	if e.fileExists("trash/" + im.ObjectKey) {
		t.Fatal("leftover of purged image not deleted")
	}
	assertInvariants(t, e)
	if _, err := e.svc.GetImage(ctx, e.admin, other.ID); err != nil {
		t.Fatal(err)
	}
	// quarantine stays empty: purged leftovers are deleted, not quarantined
	if q, _ := os.ReadDir(filepath.Join(e.dir.Path(), "quarantine")); len(q) != 0 {
		t.Fatalf("quarantine: %d", len(q))
	}
}

func TestStaleTempCleanup(t *testing.T) {
	e := newEnv(t)
	fresh, _ := e.svc.blobs.CreateTemp()
	fresh.Close()
	stale, _ := e.svc.blobs.CreateTemp()
	stale.Close()
	old := time.Now().Add(-3 * time.Hour)
	if err := os.Chtimes(e.dir.Abs(stale.Name()), old, old); err != nil {
		t.Fatal(err)
	}
	rep := e.mustCheck(CheckQuick)
	if !hasIssue(rep, IssueStaleTemp) {
		t.Fatalf("stale temp not reported: %+v", rep)
	}
	if e.fileExists(stale.Name()) || !e.fileExists(fresh.Name()) {
		t.Fatal("wrong temp files removed")
	}
	assertInvariants(t, e)
}

func TestFullCheckDetectsBitFlipAndRecovers(t *testing.T) {
	e := newEnv(t)
	ctx := context.Background()
	v := e.mustUpload(e.admin, "a.png", makePNG(t, 5, 5, 90))
	good := e.mustUpload(e.admin, "b.png", makePNG(t, 5, 5, 91))
	path := e.dir.Abs("objects/" + v.ID[0:2] + "/" + v.ID[2:4] + "/" + v.ID + ".png")
	orig, _ := os.ReadFile(path)
	mod := bytes.Clone(orig)
	mod[len(mod)-5] ^= 0x01 // one bit, same size
	if err := os.WriteFile(path, mod, 0o600); err != nil {
		t.Fatal(err)
	}
	if rep := e.mustCheck(CheckQuick); len(rep.Issues) != 0 {
		t.Fatalf("quick check cannot see a same-size flip: %+v", rep.Issues)
	}
	rep := e.mustCheck(CheckFull)
	if rep.OK || !hasIssue(rep, IssueHashMismatch) || rep.Issues[0].ImageID == nil || *rep.Issues[0].ImageID != v.ID {
		t.Fatalf("full check: %+v", rep)
	}
	if im, _ := e.db.GetImage(ctx, v.ID); im.BrokenAt == nil {
		t.Fatal("record not marked broken")
	}
	if view, _ := e.svc.GetImage(ctx, e.admin, v.ID); !view.Broken {
		t.Fatal("view does not report broken")
	}
	// broken images are not served and not deduplicated against
	_, err := e.svc.LookupPublic(ctx, v.ID, "png")
	wantCode(t, err, CodeNotFound)
	if r, err := e.upload(e.admin, "same.png", orig); err != nil || r.Duplicate {
		t.Fatalf("upload of identical bytes against a broken record: %+v %v", r, err)
	}
	if st, _ := e.svc.Status(ctx, e.admin); st.Images.Broken != 1 || !hasWarning(st, "check-failed") {
		t.Fatalf("status: %+v", st)
	}
	// repair from backup, quick check leaves the marker, full check clears it
	os.WriteFile(path, orig, 0o600)
	e.mustCheck(CheckQuick)
	if im, _ := e.db.GetImage(ctx, v.ID); im.BrokenAt == nil {
		t.Fatal("quick check must not clear hash-based breakage")
	}
	if rep := e.mustCheck(CheckFull); !rep.OK {
		t.Fatalf("after repair: %+v", rep)
	}
	if im, _ := e.db.GetImage(ctx, v.ID); im.BrokenAt != nil {
		t.Fatal("broken marker not cleared")
	}
	if _, err := e.svc.LookupPublic(ctx, v.ID, "png"); err != nil {
		t.Fatalf("not served after repair: %v", err)
	}
	_ = good
}

func hasWarning(st *Status, code string) bool {
	for _, w := range st.Warnings {
		if w.Code == code {
			return true
		}
	}
	return false
}

func TestQuickCheckMissingAndSizeMismatch(t *testing.T) {
	e := newEnv(t)
	ctx := context.Background()
	a := e.mustUpload(e.admin, "a.png", makePNG(t, 5, 5, 100))
	b := e.mustUpload(e.admin, "b.png", makePNG(t, 5, 5, 101))
	os.Remove(e.dir.Abs("objects/" + a.ID[0:2] + "/" + a.ID[2:4] + "/" + a.ID + ".png"))
	os.WriteFile(e.dir.Abs("objects/"+b.ID[0:2]+"/"+b.ID[2:4]+"/"+b.ID+".png"), []byte("short"), 0o600)
	rep := e.mustCheck(CheckQuick)
	if rep.OK || !hasIssue(rep, IssueMissingFile) || !hasIssue(rep, IssueSizeMismatch) || rep.Checked != 2 {
		t.Fatalf("report: %+v", rep)
	}
	for _, id := range []string{a.ID, b.ID} {
		if im, _ := e.db.GetImage(ctx, id); im.BrokenAt == nil {
			t.Fatalf("%s not marked broken", id)
		}
	}
	// restore of a broken image whose file is gone is refused cleanly
	e.svc.Trash(ctx, e.admin, a.ID)
	_, err := e.svc.Restore(ctx, e.admin, a.ID)
	wantCode(t, err, CodeConflict)
}

func TestConcurrentCheckRefused(t *testing.T) {
	e := newEnv(t)
	e.svc.checkMu.Lock()
	_, err := e.svc.Check(context.Background(), e.admin, CheckQuick)
	e.svc.checkMu.Unlock()
	wantCode(t, err, CodeConflict)
}

// --- snapshot / restore ------------------------------------------------------------------------

func TestSnapshotDuringUploadsAndRestore(t *testing.T) {
	e := newEnv(t)
	ctx := context.Background()
	for i := 0; i < 6; i++ {
		e.mustUpload(e.admin, fmt.Sprintf("seed%d.png", i), makePNG(t, 6, 6, 200+i))
	}
	trashed := e.mustUpload(e.admin, "trashed.png", makePNG(t, 6, 6, 250))
	e.svc.Trash(ctx, e.admin, trashed.ID)
	e.mustUpload(e.admin, "j.jpg", makeJPEG(t, 3))

	stop := make(chan struct{})
	var wg sync.WaitGroup
	wg.Add(1)
	go func() {
		defer wg.Done()
		for i := 0; ; i++ {
			select {
			case <-stop:
				return
			default:
			}
			r, err := e.upload(e.admin, "live.png", makePNG(t, 6, 6, 1000+i))
			if err != nil {
				t.Error(err)
				return
			}
			if i%3 == 0 {
				e.svc.Trash(ctx, e.admin, r.Image.ID)
			}
			if i%6 == 0 {
				e.svc.Restore(ctx, e.admin, r.Image.ID)
			}
		}
	}()
	var info SnapshotInfo
	var err error
	for i := 0; i < 3; i++ {
		info, err = e.svc.Snapshot(ctx, e.admin, "")
		if err != nil {
			break
		}
	}
	close(stop)
	wg.Wait()
	if err != nil {
		t.Fatal(err)
	}
	if info.Images < 8 {
		t.Fatalf("snapshot images = %d", info.Images)
	}
	raw, err := os.ReadFile(filepath.Join(info.Path, "MANIFEST.json"))
	if err != nil || !strings.Contains(string(raw), `"images": `) || !strings.Contains(string(raw), "picberth-snapshot 1") {
		t.Fatalf("manifest: %s %v", raw, err)
	}
	if st, _ := e.svc.Status(ctx, e.admin); st.LastSnapshot == nil || st.LastSnapshot.Path != info.Path {
		t.Fatalf("last snapshot not recorded: %+v", st.LastSnapshot)
	}

	target := filepath.Join(t.TempDir(), "restored")
	rep, err := Restore(ctx, info.Path, target, testConfig(t.TempDir()), quietLogger())
	if err != nil {
		t.Fatalf("restore: %v (report %+v)", err, rep)
	}
	if rep.Checked != info.Images || len(rep.Issues) != 0 {
		t.Fatalf("restore report: %+v (snapshot has %d)", rep, info.Images)
	}
	// open the restored directory like a server would and run another full check
	dir2, err := datadir.Open(target, datadir.Options{})
	if err != nil {
		t.Fatal(err)
	}
	defer dir2.Close()
	db2, err := meta.Open(ctx, dir2.DBPath())
	if err != nil {
		t.Fatal(err)
	}
	defer db2.Close()
	svc2, err := New(db2, dir2, testConfig(t.TempDir()), quietLogger())
	if err != nil {
		t.Fatal(err)
	}
	defer svc2.Close()
	rep2, err := svc2.Check(ctx, CLIPrincipal(), CheckFull)
	if err != nil || !rep2.OK || len(rep2.Issues) != 0 {
		t.Fatalf("check after restore: %+v %v", rep2, err)
	}

	// restore refuses a non-empty target and a non-snapshot source
	if _, err := Restore(ctx, info.Path, target, testConfig(t.TempDir()), quietLogger()); err == nil {
		t.Fatal("restore into a populated directory succeeded")
	}
	if _, err := Restore(ctx, t.TempDir(), filepath.Join(t.TempDir(), "x"), testConfig(t.TempDir()), quietLogger()); err == nil {
		t.Fatal("restore from a directory without manifest succeeded")
	}
}

func TestSnapshotFailsOnCorruptFileAndLeavesNothing(t *testing.T) {
	e := newEnv(t)
	ctx := context.Background()
	a := e.mustUpload(e.admin, "a.png", makePNG(t, 6, 6, 300))
	e.mustUpload(e.admin, "b.png", makePNG(t, 6, 6, 301))
	path := e.dir.Abs("objects/" + a.ID[0:2] + "/" + a.ID[2:4] + "/" + a.ID + ".png")
	data, _ := os.ReadFile(path)
	data[len(data)-6] ^= 0xff
	os.WriteFile(path, data, 0o600)

	parent := filepath.Join(t.TempDir(), "out")
	_, err := e.svc.Snapshot(ctx, e.admin, parent)
	if err == nil {
		t.Fatal("snapshot succeeded with a corrupted file")
	}
	if !strings.Contains(err.Error(), a.ID) {
		t.Fatalf("error does not name the image: %v", err)
	}
	entries, _ := os.ReadDir(parent)
	if len(entries) != 0 {
		t.Fatalf("failed snapshot left %d entries behind", len(entries))
	}
	if st, _ := e.svc.Status(ctx, e.admin); st.LastSnapshot != nil {
		t.Fatal("failed snapshot recorded as last snapshot")
	}

	// missing file fails too
	os.Remove(path)
	if _, err := e.svc.Snapshot(ctx, e.admin, parent); err == nil {
		t.Fatal("snapshot succeeded with a missing file")
	}
	if entries, _ := os.ReadDir(parent); len(entries) != 0 {
		t.Fatal("leftovers after second failure")
	}
}

func TestSnapshotIncludesTrashAndLinksOrCopies(t *testing.T) {
	e := newEnv(t)
	ctx := context.Background()
	a := e.mustUpload(e.admin, "a.png", makePNG(t, 6, 6, 400))
	b := e.mustUpload(e.admin, "b.png", makePNG(t, 6, 6, 401))
	e.svc.Trash(ctx, e.admin, b.ID)
	info, err := e.svc.Snapshot(ctx, e.admin, "")
	if err != nil {
		t.Fatal(err)
	}
	if info.Images != 2 {
		t.Fatalf("images = %d", info.Images)
	}
	if _, err := os.Stat(filepath.Join(info.Path, "objects", a.ID[0:2], a.ID[2:4], a.ID+".png")); err != nil {
		t.Fatal(err)
	}
	if _, err := os.Stat(filepath.Join(info.Path, "trash", b.ID[0:2], b.ID[2:4], b.ID+".png")); err != nil {
		t.Fatal(err)
	}
	if _, err := os.Stat(filepath.Join(info.Path, "picberth.db")); err != nil {
		t.Fatal(err)
	}
}

func TestSecondInstanceCannotOpenDataDir(t *testing.T) {
	e := newEnv(t)
	if _, err := datadir.Open(e.dir.Path(), datadir.Options{}); err == nil {
		t.Fatal("second instance acquired the lock")
	}
}

func TestListPaginationViaService(t *testing.T) {
	e := newEnv(t)
	ctx := context.Background()
	for i := 0; i < 7; i++ {
		e.mustUpload(e.admin, fmt.Sprintf("img%d.png", i), makePNG(t, 5, 5, 500+i))
	}
	for _, sortBy := range []string{"new", "old", "large", "small"} {
		seen := map[string]bool{}
		cursor := ""
		pages := 0
		for {
			items, next, err := e.svc.ListImages(ctx, e.admin, ListParams{Limit: 3, Sort: sortBy, Cursor: cursor})
			if err != nil {
				t.Fatal(err)
			}
			for _, it := range items {
				if seen[it.ID] {
					t.Fatalf("%s: duplicate across pages", sortBy)
				}
				seen[it.ID] = true
			}
			pages++
			if next == "" {
				break
			}
			cursor = next
		}
		if len(seen) != 7 || pages != 3 {
			t.Fatalf("%s: %d items in %d pages", sortBy, len(seen), pages)
		}
	}
	_, _, err := e.svc.ListImages(ctx, e.admin, ListParams{Limit: 3, Sort: "new", Cursor: "!!!"})
	wantCode(t, err, CodeValidation)
	items, _, _ := e.svc.ListImages(ctx, e.admin, ListParams{Q: "img3"})
	if len(items) != 1 {
		t.Fatalf("q filter: %d", len(items))
	}
	items, _, _ = e.svc.ListImages(ctx, e.admin, ListParams{Format: "jpeg"})
	if len(items) != 0 {
		t.Fatalf("format filter: %d", len(items))
	}
}

func TestBatch(t *testing.T) {
	e := newEnv(t)
	ctx := context.Background()
	a := e.mustUpload(e.admin, "a.png", makePNG(t, 5, 5, 600))
	b := e.mustUpload(e.admin, "b.png", makePNG(t, 5, 5, 601))
	res, err := e.svc.Batch(ctx, e.admin, "trash", []string{a.ID, b.ID, a.ID, "00000000000000000000", "nonsense"}, "")
	if err != nil {
		t.Fatal(err)
	}
	if len(res) != 4 || !res[0].OK || !res[1].OK || res[2].OK || res[2].Code != CodeNotFound || res[3].Code != CodeNotFound {
		t.Fatalf("results: %+v", res)
	}
	res, _ = e.svc.Batch(ctx, e.admin, "purge", []string{a.ID}, "")
	if !res[0].OK {
		t.Fatalf("purge: %+v", res)
	}
	res, _ = e.svc.Batch(ctx, e.admin, "visibility", []string{b.ID}, "private")
	if !res[0].OK {
		t.Fatalf("visibility on trashed: %+v", res)
	}
	res, _ = e.svc.Batch(ctx, e.admin, "restore", []string{b.ID}, "")
	if !res[0].OK {
		t.Fatalf("restore: %+v", res)
	}
	if _, err := e.svc.Batch(ctx, e.admin, "explode", []string{a.ID}, ""); err == nil {
		t.Fatal("unknown action accepted")
	}
	tooMany := make([]string, 501)
	if _, err := e.svc.Batch(ctx, e.admin, "trash", tooMany, ""); err == nil {
		t.Fatal("501 ids accepted")
	}
	if _, err := e.svc.Batch(ctx, e.admin, "visibility", []string{b.ID}, "x"); err == nil {
		t.Fatal("bad visibility accepted")
	}
}

func TestStatus(t *testing.T) {
	e := newEnv(t)
	ctx := context.Background()
	e.mustUpload(e.admin, "a.png", makePNG(t, 5, 5, 700))
	e.svc.NoteUntrustedForwarded()
	st, err := e.svc.Status(ctx, e.admin)
	if err != nil {
		t.Fatal(err)
	}
	if st.Images.Active != 1 || st.Database.Bytes == 0 || st.Disk.TotalBytes == 0 || st.DataDir != e.dir.Path() || st.SQLiteVersion == "" {
		t.Fatalf("status: %+v", st)
	}
	if !hasWarning(st, "untrusted-forwarded") || !hasWarning(st, "snapshot-stale") || hasWarning(st, "public-url-unset") {
		t.Fatalf("warnings: %+v", st.Warnings)
	}
}

func TestReadyAndBackgroundJobs(t *testing.T) {
	e := newEnv(t)
	if err := e.svc.Ready(context.Background()); err != nil {
		t.Fatal(err)
	}
	e.mustUpload(e.admin, "a.png", makePNG(t, 5, 5, 800))
	e.svc.StartJobs()
	waitFor(t, 5*time.Second, func() bool {
		im, _ := e.db.GetImageByKey(context.Background(), mustKey(e))
		return im != nil && im.Thumb != ""
	})
}

func mustKey(e *env) string {
	var key string
	e.svc.blobs.Walk(blob.Objects, func(f blob.File) error { key = f.Key; return nil })
	return key
}
