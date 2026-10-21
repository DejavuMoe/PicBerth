// Package config resolves infrastructure configuration: flags > environment > optional KEY=value file > defaults.
// Business settings (upload limits, formats, ...) live in the database, not here.
package config

import (
	"bufio"
	"bytes"
	"errors"
	"flag"
	"fmt"
	"log/slog"
	"net"
	"net/netip"
	"net/url"
	"path/filepath"
	"runtime"
	"strconv"
	"strings"

	"picberth/internal/netx"
)

// Config is the validated infrastructure configuration.
type Config struct {
	Listen         string
	DataDir        string // absolute
	PublicURL      string // normalized, no trailing slash
	PublicURLSet   bool   // false when derived from Listen
	ImageURL       string // normalized, no trailing slash
	TrustedProxies []string
	Trusted        *netx.Trusted
	HardMaxUpload  int64
	SnapshotDir    string // absolute
	LogLevel       slog.Level
	LogFormat      string // "text" | "json"
	DevWebDir      string
	ConfigFile     string
}

// Env/file key names.
const (
	KeyListen         = "PICBERTH_LISTEN"
	KeyDataDir        = "PICBERTH_DATA_DIR"
	KeyPublicURL      = "PICBERTH_PUBLIC_URL"
	KeyImageURL       = "PICBERTH_IMAGE_URL"
	KeyTrustedProxies = "PICBERTH_TRUSTED_PROXIES"
	KeyHardMaxUpload  = "PICBERTH_HARD_MAX_UPLOAD"
	KeySnapshotDir    = "PICBERTH_SNAPSHOT_DIR"
	KeyLogLevel       = "PICBERTH_LOG_LEVEL"
	KeyLogFormat      = "PICBERTH_LOG_FORMAT"
	KeyDevWebDir      = "PICBERTH_DEV_WEB_DIR"
	KeyConfig         = "PICBERTH_CONFIG"
)

// DefaultHardMaxUpload is 100 MiB.
const DefaultHardMaxUpload = 100 << 20

// MinHardMaxUpload keeps the hard maximum above the smallest allowed maxUploadBytes setting (1 KiB).
const MinHardMaxUpload = 1 << 10

type option struct {
	key  string
	flag string
	help string
}

var options = []option{
	{KeyListen, "listen", "listen address (host:port)"},
	{KeyDataDir, "data-dir", "data directory"},
	{KeyPublicURL, "public-url", "public base URL of the web UI, e.g. https://pics.example.com"},
	{KeyImageURL, "image-url", "base URL for image links (defaults to the public URL)"},
	{KeyTrustedProxies, "trusted-proxies", "comma-separated IPs/CIDRs of trusted reverse proxies"},
	{KeyHardMaxUpload, "hard-max-upload", "hard upper bound for the max upload size setting, in bytes"},
	{KeySnapshotDir, "snapshot-dir", "directory snapshots are written to (default <data-dir>/snapshots)"},
	{KeyLogLevel, "log-level", "debug | info | warn | error"},
	{KeyLogFormat, "log-format", "text | json"},
	{KeyDevWebDir, "dev-web-dir", "serve the SPA from this directory instead of the embedded build (development)"},
	{KeyConfig, "config", "optional file of KEY=value lines using the environment variable names"},
}

// Flags binds the configuration flags to a FlagSet so subcommands can add their own next to them.
type Flags struct {
	fs   *flag.FlagSet
	vals map[string]*string // by key
}

// RegisterFlags registers every configuration flag on fs.
func RegisterFlags(fs *flag.FlagSet) *Flags {
	f := &Flags{fs: fs, vals: map[string]*string{}}
	for _, o := range options {
		f.vals[o.key] = fs.String(o.flag, "", o.help)
	}
	return f
}

// Env abstracts os.Getenv / os.ReadFile so tests need no real environment.
type Env struct {
	Getenv   func(string) string
	ReadFile func(string) ([]byte, error)
	GOOS     string
}

// Resolve computes the final Config after fs.Parse has run.
func (f *Flags) Resolve(env Env) (*Config, error) {
	if env.GOOS == "" {
		env.GOOS = runtime.GOOS
	}
	explicit := map[string]string{}
	f.fs.Visit(func(fl *flag.Flag) {
		for _, o := range options {
			if o.flag == fl.Name {
				explicit[o.key] = fl.Value.String()
			}
		}
	})
	return Resolve(explicit, env)
}

// lookup builds the key → value resolver (flags > environment > config file) and returns the config file path.
func lookup(flags map[string]string, env Env) (val func(string) string, cfgPath string, err error) {
	get := func(key string) string {
		if v, ok := flags[key]; ok {
			return v
		}
		return env.Getenv(key)
	}
	fileVals := map[string]string{}
	cfgPath = get(KeyConfig)
	if cfgPath != "" {
		if env.ReadFile == nil {
			return nil, "", errors.New("config: no file reader available")
		}
		data, err := env.ReadFile(cfgPath)
		if err != nil {
			return nil, "", fmt.Errorf("config: reading %s: %w", cfgPath, err)
		}
		if fileVals, err = ParseFile(data); err != nil {
			return nil, "", fmt.Errorf("config: %s: %w", cfgPath, err)
		}
	}
	return func(key string) string {
		if v := get(key); v != "" {
			return v
		}
		return fileVals[key]
	}, cfgPath, nil
}

// ResolveListen resolves only the listen address (used by the healthcheck, which must work even when no
// data directory can be determined).
func ResolveListen(flags map[string]string, env Env) (string, error) {
	val, _, err := lookup(flags, env)
	if err != nil {
		return "", err
	}
	listen := orDefault(val(KeyListen), "127.0.0.1:12433")
	return listen, validateListen(listen)
}

// Resolve merges explicit flag values (by key), the environment, the config file and defaults.
func Resolve(flags map[string]string, env Env) (*Config, error) {
	if env.GOOS == "" {
		env.GOOS = runtime.GOOS
	}
	val, cfgPath, err := lookup(flags, env)
	if err != nil {
		return nil, err
	}

	c := &Config{ConfigFile: cfgPath}
	c.Listen = orDefault(val(KeyListen), "127.0.0.1:12433")
	if err := validateListen(c.Listen); err != nil {
		return nil, fmt.Errorf("%s: %w", KeyListen, err)
	}

	dd := val(KeyDataDir)
	if dd == "" {
		var err error
		if dd, err = DefaultDataDir(env.GOOS, env.Getenv); err != nil {
			return nil, err
		}
	}
	abs, err := filepath.Abs(dd)
	if err != nil {
		return nil, fmt.Errorf("%s: %w", KeyDataDir, err)
	}
	c.DataDir = abs

	if pu := val(KeyPublicURL); pu != "" {
		if c.PublicURL, err = NormalizeBaseURL(pu); err != nil {
			return nil, fmt.Errorf("%s: %w", KeyPublicURL, err)
		}
		c.PublicURLSet = true
	} else {
		c.PublicURL = "http://" + displayHost(c.Listen)
	}
	if iu := val(KeyImageURL); iu != "" {
		if c.ImageURL, err = NormalizeBaseURL(iu); err != nil {
			return nil, fmt.Errorf("%s: %w", KeyImageURL, err)
		}
	} else {
		c.ImageURL = c.PublicURL
	}

	for _, e := range strings.Split(val(KeyTrustedProxies), ",") {
		if e = strings.TrimSpace(e); e != "" {
			c.TrustedProxies = append(c.TrustedProxies, e)
		}
	}
	if c.Trusted, err = netx.ParseTrusted(c.TrustedProxies); err != nil {
		return nil, fmt.Errorf("%s: %w", KeyTrustedProxies, err)
	}

	c.HardMaxUpload = DefaultHardMaxUpload
	if v := val(KeyHardMaxUpload); v != "" {
		n, err := strconv.ParseInt(v, 10, 64)
		if err != nil || n < MinHardMaxUpload {
			return nil, fmt.Errorf("%s: must be an integer number of bytes >= %d", KeyHardMaxUpload, MinHardMaxUpload)
		}
		c.HardMaxUpload = n
	}

	if sd := val(KeySnapshotDir); sd != "" {
		if c.SnapshotDir, err = filepath.Abs(sd); err != nil {
			return nil, fmt.Errorf("%s: %w", KeySnapshotDir, err)
		}
	} else {
		c.SnapshotDir = filepath.Join(c.DataDir, "snapshots")
	}

	switch strings.ToLower(orDefault(val(KeyLogLevel), "info")) {
	case "debug":
		c.LogLevel = slog.LevelDebug
	case "info":
		c.LogLevel = slog.LevelInfo
	case "warn", "warning":
		c.LogLevel = slog.LevelWarn
	case "error":
		c.LogLevel = slog.LevelError
	default:
		return nil, fmt.Errorf("%s: must be debug, info, warn or error", KeyLogLevel)
	}
	c.LogFormat = strings.ToLower(orDefault(val(KeyLogFormat), "text"))
	if c.LogFormat != "text" && c.LogFormat != "json" {
		return nil, fmt.Errorf("%s: must be text or json", KeyLogFormat)
	}
	c.DevWebDir = val(KeyDevWebDir)
	return c, nil
}

func orDefault(v, d string) string {
	if v == "" {
		return d
	}
	return v
}

// ParseFile parses KEY=value lines; "#" starts a comment line. Unknown keys are errors (typo protection).
func ParseFile(data []byte) (map[string]string, error) {
	known := map[string]bool{}
	for _, o := range options {
		known[o.key] = true
	}
	out := map[string]string{}
	sc := bufio.NewScanner(bytes.NewReader(bytes.TrimPrefix(data, []byte("\xef\xbb\xbf"))))
	for n := 1; sc.Scan(); n++ {
		line := strings.TrimSpace(sc.Text())
		if line == "" || strings.HasPrefix(line, "#") {
			continue
		}
		k, v, ok := strings.Cut(line, "=")
		if !ok {
			return nil, fmt.Errorf("line %d: expected KEY=value", n)
		}
		k, v = strings.TrimSpace(k), strings.TrimSpace(v)
		if !known[k] {
			return nil, fmt.Errorf("line %d: unknown key %q", n, k)
		}
		if k == KeyConfig {
			return nil, fmt.Errorf("line %d: %s cannot be set inside the config file", n, k)
		}
		if len(v) >= 2 && (v[0] == '"' && v[len(v)-1] == '"' || v[0] == '\'' && v[len(v)-1] == '\'') {
			v = v[1 : len(v)-1]
		}
		out[k] = v
	}
	return out, sc.Err()
}

// DefaultDataDir returns the platform convention for the data directory.
func DefaultDataDir(goos string, getenv func(string) string) (string, error) {
	switch goos {
	case "windows":
		if d := getenv("LOCALAPPDATA"); d != "" {
			return filepath.Join(d, "PicBerth"), nil
		}
		if d := getenv("USERPROFILE"); d != "" {
			return filepath.Join(d, "AppData", "Local", "PicBerth"), nil
		}
	case "darwin":
		if h := getenv("HOME"); h != "" {
			return filepath.Join(h, "Library", "Application Support", "PicBerth"), nil
		}
	default:
		if d := getenv("XDG_DATA_HOME"); d != "" && filepath.IsAbs(d) {
			return filepath.Join(d, "picberth"), nil
		}
		if h := getenv("HOME"); h != "" {
			return filepath.Join(h, ".local", "share", "picberth"), nil
		}
	}
	return "", errors.New("cannot determine a default data directory; set " + KeyDataDir)
}

func validateListen(addr string) error {
	host, port, err := net.SplitHostPort(addr)
	if err != nil {
		return fmt.Errorf("invalid listen address %q (want host:port): %w", addr, err)
	}
	p, err := strconv.Atoi(port)
	if err != nil || p < 0 || p > 65535 {
		return fmt.Errorf("invalid port %q", port)
	}
	if host != "" && host != "localhost" {
		if _, err := netip.ParseAddr(host); err != nil {
			// A DNS name is allowed, but reject obviously broken hosts.
			if strings.ContainsAny(host, " /\\") {
				return fmt.Errorf("invalid host %q", host)
			}
		}
	}
	return nil
}

// displayHost turns a listen address into something a browser can open (wildcards become loopback).
func displayHost(listen string) string {
	host, port, err := net.SplitHostPort(listen)
	if err != nil {
		return listen
	}
	if host == "" || host == "0.0.0.0" || host == "::" {
		host = "127.0.0.1"
	}
	return net.JoinHostPort(host, port)
}

// NormalizeBaseURL validates an http(s) base URL and returns it without a trailing slash.
// Only an origin is accepted: no path, query, fragment or credentials.
func NormalizeBaseURL(s string) (string, error) {
	u, err := url.Parse(strings.TrimSpace(s))
	if err != nil {
		return "", fmt.Errorf("invalid URL: %w", err)
	}
	if u.Scheme != "http" && u.Scheme != "https" {
		return "", errors.New("URL scheme must be http or https")
	}
	if u.Host == "" || u.Hostname() == "" {
		return "", errors.New("URL has no host")
	}
	if u.User != nil {
		return "", errors.New("URL must not contain credentials")
	}
	if u.RawQuery != "" || u.ForceQuery || u.Fragment != "" {
		return "", errors.New("URL must not contain a query or fragment")
	}
	if u.Path != "" && u.Path != "/" {
		return "", errors.New("URL must not contain a path (sub-path hosting is not supported)")
	}
	return u.Scheme + "://" + strings.ToLower(u.Host), nil
}
