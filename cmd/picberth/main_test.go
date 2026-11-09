package main

import (
	"bytes"
	"context"
	"io"
	"net/http"
	"os"
	"path/filepath"
	"strings"
	"sync"
	"testing"
	"time"

	"picberth/internal/config"
)

type result struct {
	code        int
	stdout, err string
}

func runCLI(t *testing.T, stdin string, args ...string) result {
	t.Helper()
	var out, errb bytes.Buffer
	st := streams{stdin: strings.NewReader(stdin), stdout: &out, stderr: &errb,
		env: config.Env{Getenv: func(string) string { return "" }, ReadFile: os.ReadFile}}
	code := run(context.Background(), args, st)
	return result{code, out.String(), errb.String()}
}

func TestVersionAndUsage(t *testing.T) {
	r := runCLI(t, "", "version")
	if r.code != 0 || !strings.Contains(r.stdout, "picberth 0.0.1-dev") {
		t.Fatalf("version: %+v", r)
	}
	if r := runCLI(t, "", "frobnicate"); r.code != 2 {
		t.Fatalf("unknown command: %+v", r)
	}
	d := t.TempDir()
	if r := runCLI(t, "pw", "admin", "create", "--data-dir", d, "--username", "x"); r.code != 2 || !strings.Contains(r.err, "password-stdin") {
		t.Fatalf("password must come from stdin: %+v", r)
	}
	if r := runCLI(t, "", "admin", "bogus"); r.code != 2 {
		t.Fatalf("admin bogus: %+v", r)
	}
}

func TestAdminCheckSnapshotRestoreOffline(t *testing.T) {
	data := filepath.Join(t.TempDir(), "data")
	r := runCLI(t, "short\n", "admin", "create", "--data-dir", data, "--username", "admin", "--password-stdin")
	if r.code != 1 {
		t.Fatalf("short password accepted: %+v", r)
	}
	r = runCLI(t, "correct horse battery\r\n", "admin", "create", "--data-dir", data, "--username", "admin", "--password-stdin")
	if r.code != 0 {
		t.Fatalf("create: %+v", r)
	}
	if r = runCLI(t, "another password 1\n", "admin", "create", "--data-dir", data, "--username", "other", "--password-stdin"); r.code != 1 {
		t.Fatalf("second admin create must fail: %+v", r)
	}
	if r = runCLI(t, "new password 12345\n", "admin", "reset-password", "--data-dir", data, "--username", "admin", "--password-stdin"); r.code != 0 {
		t.Fatalf("reset: %+v", r)
	}
	if r = runCLI(t, "new password 12345\n", "admin", "reset-password", "--data-dir", data, "--username", "ghost", "--password-stdin"); r.code != 1 {
		t.Fatalf("reset of unknown user: %+v", r)
	}
	if r = runCLI(t, "", "check", "--data-dir", data); r.code != 0 || !strings.Contains(r.stdout, "quick check") {
		t.Fatalf("check: %+v", r)
	}
	if r = runCLI(t, "", "check", "--full", "--data-dir", data); r.code != 0 || !strings.Contains(r.stdout, "full check") {
		t.Fatalf("full check: %+v", r)
	}
	if r = runCLI(t, "", "check", "--data-dir", filepath.Join(t.TempDir(), "missing")); r.code != 1 {
		t.Fatalf("check on a missing dir must fail without creating it: %+v", r)
	}

	snapRoot := filepath.Join(t.TempDir(), "snaps")
	r = runCLI(t, "", "snapshot", "--data-dir", data, "--out", snapRoot)
	if r.code != 0 {
		t.Fatalf("snapshot: %+v", r)
	}
	entries, _ := os.ReadDir(snapRoot)
	if len(entries) != 1 {
		t.Fatalf("snapshot dir entries: %d", len(entries))
	}
	snap := filepath.Join(snapRoot, entries[0].Name())

	target := filepath.Join(t.TempDir(), "restored")
	if r = runCLI(t, "", "restore", "--from", snap, "--data-dir", target); r.code != 0 || !strings.Contains(r.stdout, "verification passed") {
		t.Fatalf("restore: %+v", r)
	}
	if r = runCLI(t, "", "restore", "--from", snap, "--data-dir", target); r.code != 1 {
		t.Fatalf("restore into populated dir: %+v", r)
	}
	if r = runCLI(t, "", "restore", "--data-dir", target); r.code != 2 {
		t.Fatalf("restore without --from: %+v", r)
	}
	// the restored directory is a working data dir with the same admin
	if r = runCLI(t, "yet another pw 1\n", "admin", "create", "--data-dir", target, "--username", "x", "--password-stdin"); r.code != 1 {
		t.Fatalf("restored dir lost its administrator: %+v", r)
	}
}

func TestServeLifecycleLockHealthcheckAndSetupCode(t *testing.T) {
	data := filepath.Join(t.TempDir(), "data")
	var logBuf syncBuf
	cfg, err := config.Resolve(map[string]string{config.KeyDataDir: data, config.KeyListen: "127.0.0.1:0", config.KeyPublicURL: "http://127.0.0.1:1"},
		config.Env{Getenv: func(string) string { return "" }})
	if err != nil {
		t.Fatal(err)
	}
	log := newLogger(cfg, &logBuf)
	ctx, cancel := context.WithCancel(context.Background())
	ready := make(chan string, 1)
	done := make(chan error, 1)
	go func() { done <- serve(ctx, cfg, log, ready) }()
	var addr string
	select {
	case addr = <-ready:
	case err := <-done:
		t.Fatalf("serve exited early: %v", err)
	case <-time.After(10 * time.Second):
		t.Fatal("server did not start")
	}

	logs := logBuf.String()
	if !strings.Contains(logs, "level=WARN") || !strings.Contains(logs, "setup_code=") || !strings.Contains(logs, "/setup") {
		t.Fatalf("setup code not logged at WARN:\n%s", logs)
	}
	resp, err := http.Get("http://" + addr + "/readyz")
	if err != nil || resp.StatusCode != 200 {
		t.Fatalf("readyz: %v %v", resp, err)
	}
	resp.Body.Close()
	resp, err = http.Get("http://" + addr + "/")
	if err != nil {
		t.Fatal(err)
	}
	body, _ := io.ReadAll(resp.Body)
	resp.Body.Close()
	if resp.StatusCode != 200 || !strings.Contains(string(body), "<") {
		t.Fatalf("embedded SPA: %d %q", resp.StatusCode, body)
	}

	// healthcheck against the running server
	if r := runCLI(t, "", "healthcheck", "--listen", addr); r.code != 0 {
		t.Fatalf("healthcheck: %+v", r)
	}
	// a second process on the same data dir (offline command) fails clearly
	r := runCLI(t, "", "check", "--data-dir", data)
	if r.code != 1 || !strings.Contains(r.err, "in use by another PicBerth process") {
		t.Fatalf("offline command while serving: %+v", r)
	}
	// and a second server fails too
	cfg2 := *cfg
	cfg2.Listen = "127.0.0.1:0"
	if err := serve(context.Background(), &cfg2, log, nil); err == nil {
		t.Fatal("second server started on the same data dir")
	}

	cancel()
	select {
	case err := <-done:
		if err != nil {
			t.Fatalf("serve returned %v", err)
		}
	case <-time.After(12 * time.Second):
		t.Fatal("graceful shutdown took too long")
	}
	// the lock is released: offline commands work again
	if r := runCLI(t, "", "check", "--data-dir", data); r.code != 0 {
		t.Fatalf("check after shutdown: %+v", r)
	}
	if r := runCLI(t, "", "healthcheck", "--listen", addr); r.code != 1 {
		t.Fatalf("healthcheck of a stopped server must fail: %+v", r)
	}
}

func TestHealthcheckMapsWildcard(t *testing.T) {
	// nothing listens on this port: the probe must fail (exit 1) rather than panic or hang
	start := time.Now()
	if r := runCLI(t, "", "healthcheck", "--listen", "0.0.0.0:1"); r.code != 1 {
		t.Fatalf("%+v", r)
	}
	if time.Since(start) > 5*time.Second {
		t.Fatal("healthcheck exceeded its timeout")
	}
	if r := runCLI(t, "", "healthcheck", "--listen", "nonsense"); r.code != 1 {
		t.Fatalf("bad listen: %+v", r)
	}
}

type syncBuf struct {
	mu sync.Mutex
	b  bytes.Buffer
}

func (s *syncBuf) Write(p []byte) (int, error) {
	s.mu.Lock()
	defer s.mu.Unlock()
	return s.b.Write(p)
}

func (s *syncBuf) String() string {
	s.mu.Lock()
	defer s.mu.Unlock()
	return s.b.String()
}
