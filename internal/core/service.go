package core

import (
	"context"
	"hash/fnv"
	"io"
	"log/slog"
	"sync"
	"sync/atomic"
	"time"

	"picberth/internal/blob"
	"picberth/internal/datadir"
	"picberth/internal/imaging"
	"picberth/internal/meta"
	"picberth/internal/password"
	"picberth/internal/ratelimit"
)

// Config is the infrastructure configuration core needs (all validated by internal/config).
type Config struct {
	PublicURL         string // no trailing slash
	ImageURL          string // no trailing slash
	PublicURLSet      bool
	ListenNonLoopback bool
	HardMaxUpload     int64
	SnapshotDir       string // absolute
}

// Fault injection points (tests only): Service.Fault is called at these moments and a non-nil error
// aborts the operation right there, leaving state exactly as a crash would.
const (
	FaultUploadAfterRename = "upload:after-rename"
	FaultTrashAfterDB      = "trash:after-db"
	FaultRestoreAfterMove  = "restore:after-move"
	FaultPurgeAfterDB      = "purge:after-db"
)

const (
	maxConcurrentUploads = 4
	thumbMaxDim          = 480
	thumbMaxPixels       = 25_000_000
	touchInterval        = time.Minute
)

// Service is the application service layer.
type Service struct {
	cfg    Config
	db     *meta.DB
	blobs  *blob.Store
	dir    *datadir.Dir
	log    *slog.Logger
	hasher *password.Hasher

	// Injectable collaborators; New wires the real implementations. Set before use.
	Now       func() time.Time
	Sniff     func(head []byte) (imaging.Format, bool)
	Inspect   func(r io.ReaderAt, size int64, lim imaging.Limits) (imaging.Info, error)
	Thumbnail func(r io.ReaderAt, size int64, info imaging.Info, maxDim int, maxPixels int64) ([]byte, string, error)
	DiskFree  func() (free, total uint64, err error)
	Fault     func(point string) error

	settings   atomic.Pointer[Settings]
	settingsMu sync.Mutex

	uploadSem      chan struct{}
	uploadLimit    *ratelimit.Limiter
	loginIPLimit   *ratelimit.Limiter
	loginUserLimit *ratelimit.Limiter
	setupLimit     *ratelimit.Limiter

	setupMu   sync.Mutex
	setupCode string

	// purgeMu: purge operations hold RLock; a snapshot holds Lock (pauses purging, not uploads).
	purgeMu sync.RWMutex
	// commitMu: uploads hold RLock between the rename into objects/ and the INSERT; orphan detection
	// takes Lock so it can never quarantine a file whose record is about to be written.
	commitMu sync.RWMutex
	idLocks  [64]sync.Mutex
	checkMu  sync.Mutex

	thumbQ     chan string
	thumbDirty atomic.Bool
	jobCtx     context.Context
	jobCancel  context.CancelFunc
	wg         sync.WaitGroup

	startedAt          time.Time
	untrustedForwarded atomic.Int64 // unix ms of the last request carrying X-Forwarded-For from an untrusted peer
	readyProbe         atomic.Int64
}

// New builds the service on an opened database and data directory.
func New(db *meta.DB, dir *datadir.Dir, cfg Config, log *slog.Logger) (*Service, error) {
	if log == nil {
		log = slog.Default()
	}
	s := &Service{
		cfg: cfg, db: db, dir: dir, blobs: blob.New(dir.Root()), log: log,
		hasher:         password.NewHasher(2),
		Now:            time.Now,
		Sniff:          imaging.Sniff,
		Inspect:        imaging.Inspect,
		Thumbnail:      imaging.Thumbnail,
		DiskFree:       dir.FreeSpace,
		uploadSem:      make(chan struct{}, maxConcurrentUploads),
		uploadLimit:    ratelimit.PerMinute(60),
		loginIPLimit:   ratelimit.PerMinute(5),
		loginUserLimit: ratelimit.New(10.0/600, 10),
		setupLimit:     ratelimit.New(20.0/600, 20),
		thumbQ:         make(chan string, 256),
		startedAt:      time.Now(),
	}
	s.jobCtx, s.jobCancel = context.WithCancel(context.Background())
	if err := s.loadSettings(context.Background()); err != nil {
		return nil, err
	}
	return s, nil
}

// Close stops background jobs and waits for them. It does not close the database or data directory.
func (s *Service) Close() {
	s.jobCancel()
	s.wg.Wait()
}

func (s *Service) now() time.Time { return s.Now() }

func (s *Service) nowMs() int64 { return s.now().UnixMilli() }

// SetLoginLimits replaces the login rate limits (tests).
func (s *Service) SetLoginLimits(ip, user *ratelimit.Limiter) {
	s.loginIPLimit, s.loginUserLimit = ip, user
}

func (s *Service) fault(point string) error {
	if s.Fault == nil {
		return nil
	}
	return s.Fault(point)
}

func (s *Service) denyAdmin(p Principal) error {
	if p.Kind == KindAnonymous {
		return unauthorized()
	}
	return forbidden("This operation requires an interactive administrator session.")
}

// lockImage serializes state changes and consistency repairs for one image.
func (s *Service) lockImage(id string) *sync.Mutex {
	h := fnv.New32a()
	h.Write([]byte(id))
	m := &s.idLocks[h.Sum32()%uint32(len(s.idLocks))]
	m.Lock()
	return m
}

// NoteUntrustedForwarded records that a peer outside trusted-proxies sent X-Forwarded-For.
func (s *Service) NoteUntrustedForwarded() { s.untrustedForwarded.Store(s.nowMs()) }

// ImageURL builds the public link for an image. It only ever uses the configured image URL.
func (s *Service) ImageURL(id, ext string) string { return s.cfg.ImageURL + "/i/" + id + "." + ext }

// PublicURL is the configured public base URL.
func (s *Service) PublicURL() string { return s.cfg.PublicURL }

// Ready reports whether the database is readable and the data directory writable.
func (s *Service) Ready(ctx context.Context) error {
	if err := s.db.Ping(ctx); err != nil {
		return err
	}
	// Probe writability at most every 15 s (readiness is polled frequently).
	now := s.nowMs()
	if last := s.readyProbe.Load(); now-last < 15_000 && last != 0 {
		return nil
	}
	tmp, err := s.blobs.CreateTemp()
	if err != nil {
		if blob.IsDiskFull(err) {
			return nil // a full disk is reported by the status page, not by failing readiness
		}
		return err
	}
	name := tmp.Name()
	tmp.Close()
	s.blobs.RemoveTemp(name)
	s.readyProbe.Store(now)
	return nil
}
