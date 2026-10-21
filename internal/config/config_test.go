package config

import (
	"flag"
	"log/slog"
	"path/filepath"
	"strings"
	"testing"
)

func envOf(m map[string]string) func(string) string { return func(k string) string { return m[k] } }

func TestDefaults(t *testing.T) {
	c, err := Resolve(nil, Env{Getenv: envOf(map[string]string{"HOME": "/home/u"}), GOOS: "linux"})
	if err != nil {
		t.Fatal(err)
	}
	if c.Listen != "127.0.0.1:12433" || c.PublicURL != "http://127.0.0.1:12433" || c.PublicURLSet || c.ImageURL != c.PublicURL {
		t.Fatalf("defaults: %+v", c)
	}
	if !strings.HasSuffix(filepath.ToSlash(c.DataDir), "/home/u/.local/share/picberth") {
		t.Fatalf("data dir %q", c.DataDir)
	}
	if c.HardMaxUpload != 104857600 || c.LogLevel != slog.LevelInfo || c.LogFormat != "text" {
		t.Fatalf("defaults: %+v", c)
	}
	if filepath.Base(c.SnapshotDir) != "snapshots" {
		t.Fatalf("snapshot dir %q", c.SnapshotDir)
	}
}

func TestWildcardListenDerivesLoopbackURL(t *testing.T) {
	c, err := Resolve(map[string]string{KeyListen: "0.0.0.0:8080", KeyDataDir: "x"}, Env{Getenv: envOf(nil)})
	if err != nil {
		t.Fatal(err)
	}
	if c.PublicURL != "http://127.0.0.1:8080" {
		t.Fatalf("public url %q", c.PublicURL)
	}
}

func TestPrecedenceFlagsEnvFileDefaults(t *testing.T) {
	files := map[string]string{
		"/etc/pb.conf": "# comment\nPICBERTH_LISTEN=127.0.0.1:1111\nPICBERTH_LOG_LEVEL=\"warn\"\nPICBERTH_PUBLIC_URL=https://file.example\nPICBERTH_DATA_DIR=/file/data\nPICBERTH_LOG_FORMAT=json\n",
	}
	env := Env{
		Getenv: envOf(map[string]string{
			KeyConfig:    "/etc/pb.conf",
			KeyListen:    "127.0.0.1:2222",
			KeyPublicURL: "https://env.example/",
			KeyDataDir:   "/env/data",
		}),
		ReadFile: func(p string) ([]byte, error) { return []byte(files[p]), nil },
		GOOS:     "linux",
	}
	c, err := Resolve(map[string]string{KeyListen: "127.0.0.1:3333"}, env)
	if err != nil {
		t.Fatal(err)
	}
	if c.Listen != "127.0.0.1:3333" {
		t.Errorf("flag should beat env: %s", c.Listen)
	}
	if c.PublicURL != "https://env.example" {
		t.Errorf("env should beat file: %s", c.PublicURL)
	}
	if filepath.ToSlash(c.DataDir) != "/env/data" && !strings.HasSuffix(filepath.ToSlash(c.DataDir), "/env/data") {
		t.Errorf("env should beat file for data dir: %s", c.DataDir)
	}
	if c.LogLevel != slog.LevelWarn || c.LogFormat != "json" {
		t.Errorf("file values should apply when nothing overrides: %v %s", c.LogLevel, c.LogFormat)
	}
	if !c.PublicURLSet {
		t.Error("PublicURLSet should be true")
	}
}

func TestFlagSetIntegration(t *testing.T) {
	fs := flag.NewFlagSet("serve", flag.ContinueOnError)
	f := RegisterFlags(fs)
	if err := fs.Parse([]string{"--listen", "127.0.0.1:9", "--trusted-proxies", "10.0.0.0/8, ::1", "--data-dir", "d"}); err != nil {
		t.Fatal(err)
	}
	c, err := f.Resolve(Env{Getenv: envOf(map[string]string{KeyListen: "127.0.0.1:1", KeyHardMaxUpload: "2048"})})
	if err != nil {
		t.Fatal(err)
	}
	if c.Listen != "127.0.0.1:9" || c.HardMaxUpload != 2048 || len(c.TrustedProxies) != 2 {
		t.Fatalf("%+v", c)
	}
	if c.Trusted == nil || c.Trusted.Empty() {
		t.Fatal("trusted proxies not parsed")
	}
}

func TestValidation(t *testing.T) {
	bad := map[string]string{
		KeyListen:         "nonsense",
		KeyPublicURL:      "ftp://x.example",
		KeyImageURL:       "https://x.example/?q=1",
		KeyTrustedProxies: "10.0.0.0/40",
		KeyHardMaxUpload:  "10",
		KeyLogLevel:       "loud",
		KeyLogFormat:      "xml",
	}
	for k, v := range bad {
		if _, err := Resolve(map[string]string{k: v, KeyDataDir: "x"}, Env{Getenv: envOf(nil)}); err == nil {
			t.Errorf("%s=%q accepted", k, v)
		}
	}
	for _, u := range []string{"https://a.example/path", "https://u:p@a.example", "https://a.example#frag", "http://", "//a.example", "a.example"} {
		if _, err := NormalizeBaseURL(u); err == nil {
			t.Errorf("NormalizeBaseURL(%q) accepted", u)
		}
	}
	if got, err := NormalizeBaseURL("HTTPS://Pics.Example.com:8443/"); err != nil || got != "https://pics.example.com:8443" {
		t.Errorf("normalize: %q %v", got, err)
	}
}

func TestParseFileErrors(t *testing.T) {
	for _, s := range []string{"PICBERTH_NOPE=1", "just text", "PICBERTH_CONFIG=other"} {
		if _, err := ParseFile([]byte(s)); err == nil {
			t.Errorf("ParseFile(%q) accepted", s)
		}
	}
}

func TestDefaultDataDirPerOS(t *testing.T) {
	xdg := t.TempDir() // any absolute path for the host OS
	cases := []struct {
		goos string
		env  map[string]string
		want string
	}{
		{"windows", map[string]string{"LOCALAPPDATA": `C:\Users\x\AppData\Local`}, filepath.Join(`C:\Users\x\AppData\Local`, "PicBerth")},
		{"darwin", map[string]string{"HOME": "/Users/x"}, filepath.Join("/Users/x", "Library", "Application Support", "PicBerth")},
		{"linux", map[string]string{"XDG_DATA_HOME": xdg}, filepath.Join(xdg, "picberth")},
	}
	for _, tc := range cases {
		got, err := DefaultDataDir(tc.goos, envOf(tc.env))
		if err != nil || got != tc.want {
			t.Errorf("%s: %q %v, want %q", tc.goos, got, err, tc.want)
		}
	}
	if _, err := DefaultDataDir("linux", envOf(nil)); err == nil {
		t.Error("expected error without HOME")
	}
}
