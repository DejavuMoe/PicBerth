package core

import (
	"context"
	"encoding/json"
	"errors"
	"runtime"
	"time"

	"picberth/internal/buildinfo"
	"picberth/internal/datadir"
	"picberth/internal/meta"
)

// Status is the data behind the status page.
type Status struct {
	Version       string          `json:"version"`
	Commit        string          `json:"commit"`
	GoVersion     string          `json:"goVersion"`
	SQLiteVersion string          `json:"sqliteVersion"`
	StartedAt     string          `json:"startedAt"`
	UptimeSeconds int64           `json:"uptimeSeconds"`
	Images        StatusImages    `json:"images"`
	Database      StatusDatabase  `json:"database"`
	Disk          StatusDisk      `json:"disk"`
	DataDir       string          `json:"dataDir"`
	LastSnapshot  *LastSnapshot   `json:"lastSnapshot"`
	LastCheck     *LastCheck      `json:"lastCheck"`
	Warnings      []StatusWarning `json:"warnings"`
}

type StatusImages struct {
	Active  int64 `json:"active"`
	Trashed int64 `json:"trashed"`
	Bytes   int64 `json:"bytes"`
	Broken  int64 `json:"broken"`
}

type StatusDatabase struct {
	Bytes    int64 `json:"bytes"`
	WALBytes int64 `json:"walBytes"`
}

type StatusDisk struct {
	FreeBytes      uint64 `json:"freeBytes"`
	TotalBytes     uint64 `json:"totalBytes"`
	WatermarkBytes uint64 `json:"watermarkBytes"`
}

type LastSnapshot struct {
	At   string `json:"at"`
	Path string `json:"path"`
}

type LastCheck struct {
	At     string `json:"at"`
	Mode   string `json:"mode"`
	OK     bool   `json:"ok"`
	Issues int    `json:"issues"`
}

// StatusWarning is a machine-readable configuration or health warning.
type StatusWarning struct {
	Code    string `json:"code"`
	Message string `json:"message"`
}

const (
	walLargeBytes     = 64 << 20
	snapshotStaleDays = 14
)

// Status collects the status page data.
func (s *Service) Status(ctx context.Context, p Principal) (*Status, error) {
	if !p.IsAdmin() {
		return nil, s.denyAdmin(p)
	}
	stats, err := s.db.ImageStats(ctx)
	if err != nil {
		return nil, err
	}
	now := s.now()
	st := &Status{
		Version: buildinfo.Version, Commit: buildinfo.Commit, GoVersion: runtime.Version(), SQLiteVersion: s.db.SQLiteVersion(),
		StartedAt: s.startedAt.UTC().Format(time.RFC3339), UptimeSeconds: int64(now.Sub(s.startedAt).Seconds()),
		Images:   StatusImages{Active: stats.Active, Trashed: stats.Trashed, Bytes: stats.Bytes, Broken: stats.Broken},
		DataDir:  s.dir.Path(),
		Warnings: []StatusWarning{},
	}
	if fi, err := s.dir.Root().Stat(datadir.DBFile); err == nil {
		st.Database.Bytes = fi.Size()
	}
	if fi, err := s.dir.Root().Stat(datadir.DBFile + "-wal"); err == nil {
		st.Database.WALBytes = fi.Size()
	}
	if free, total, err := s.DiskFree(); err == nil {
		st.Disk = StatusDisk{FreeBytes: free, TotalBytes: total, WatermarkBytes: watermark(total)}
	}
	if v, err := s.db.KVGet(ctx, "last_snapshot"); err == nil {
		var ls struct{ At, Path string }
		if json.Unmarshal([]byte(v), &ls) == nil {
			st.LastSnapshot = &LastSnapshot{At: ls.At, Path: ls.Path}
		}
	} else if !errors.Is(err, meta.ErrNotFound) {
		return nil, err
	}
	if v, err := s.db.KVGet(ctx, "last_check"); err == nil {
		var lc LastCheck
		if json.Unmarshal([]byte(v), &lc) == nil {
			st.LastCheck = &lc
		}
	} else if !errors.Is(err, meta.ErrNotFound) {
		return nil, err
	}

	warn := func(code, msg string) { st.Warnings = append(st.Warnings, StatusWarning{Code: code, Message: msg}) }
	if !s.cfg.PublicURLSet {
		warn("public-url-unset", "No public URL is configured; image links use the listen address. Set PICBERTH_PUBLIC_URL.")
	} else if len(s.cfg.PublicURL) >= 7 && s.cfg.PublicURL[:7] == "http://" && !isLoopbackURL(s.cfg.PublicURL) {
		warn("insecure-public-url", "The public URL uses plain HTTP; cookies cannot be marked Secure.")
	}
	if s.cfg.ListenNonLoopback {
		warn("non-loopback-listen", "The server listens on a non-loopback address. Put a TLS-terminating reverse proxy in front of it.")
	}
	if t := s.untrustedForwarded.Load(); t != 0 {
		warn("untrusted-forwarded", "Requests carrying X-Forwarded-For arrived from a peer that is not a trusted proxy; rate limits see the proxy address. Check PICBERTH_TRUSTED_PROXIES.")
	}
	if st.Disk.TotalBytes > 0 && st.Disk.FreeBytes < st.Disk.WatermarkBytes {
		warn("low-disk", "Free disk space is below the upload watermark; uploads are refused.")
	}
	if st.LastSnapshot == nil {
		if stats.Active+stats.Trashed > 0 {
			warn("snapshot-stale", "No snapshot has been taken yet.")
		}
	} else if at, err := time.Parse(time.RFC3339, st.LastSnapshot.At); err == nil && now.Sub(at) > snapshotStaleDays*24*time.Hour {
		warn("snapshot-stale", "The last snapshot is more than 14 days old.")
	}
	if st.LastCheck != nil && !st.LastCheck.OK {
		warn("check-failed", "The last consistency check found problems that were not repaired.")
	}
	if st.Database.WALBytes > walLargeBytes {
		warn("wal-large", "The database write-ahead log is unusually large.")
	}
	for _, w := range s.dir.Warnings() {
		warn("data-dir", w)
	}
	return st, nil
}

func isLoopbackURL(u string) bool {
	for _, h := range []string{"http://127.0.0.1", "http://localhost", "http://[::1]"} {
		if len(u) >= len(h) && u[:len(h)] == h && (len(u) == len(h) || u[len(h)] == ':') {
			return true
		}
	}
	return false
}
