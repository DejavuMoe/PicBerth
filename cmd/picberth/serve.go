package main

import (
	"context"
	"errors"
	"fmt"
	"io/fs"
	"log/slog"
	"net"
	"net/http"
	"os"
	"time"

	"picberth/internal/buildinfo"
	"picberth/internal/config"
	"picberth/internal/core"
	"picberth/internal/datadir"
	"picberth/internal/httpapi"
	"picberth/internal/meta"
	"picberth/internal/netx"
	"picberth/internal/webui"
)

const shutdownTimeout = 8 * time.Second

func cmdServe(ctx context.Context, args []string, st streams) error {
	fs_, flags := newFlagSet("serve", st)
	if err := fs_.Parse(args); err != nil {
		return err
	}
	if fs_.NArg() > 0 {
		return usagef("unexpected argument %q", fs_.Arg(0))
	}
	cfg, err := flags.Resolve(st.env)
	if err != nil {
		return err
	}
	log := newLogger(cfg, st.stderr)
	return serve(ctx, cfg, log, nil)
}

// serve runs the server until ctx is cancelled. If ready is non-nil it receives the bound address once the
// listener is up (tests).
func serve(ctx context.Context, cfg *config.Config, log *slog.Logger, ready chan<- string) error {
	log.Info("starting PicBerth", "version", buildinfo.Version, "commit", buildinfo.Commit, "data_dir", cfg.DataDir)

	dir, err := datadir.Open(cfg.DataDir, datadir.Options{Create: true})
	if err != nil {
		return err
	}
	defer dir.Close()
	for _, w := range dir.Warnings() {
		log.Warn(w)
	}
	db, err := meta.Open(ctx, dir.DBPath())
	if err != nil {
		return err
	}
	defer db.Close()
	if db.PreMigrationBackup != "" {
		log.Info("database schema migrated; the previous metadata was saved first", "backup", db.PreMigrationBackup)
	}

	nonLoopback := !netx.IsLoopbackListen(cfg.Listen)
	cc := coreConfig(cfg)
	cc.ListenNonLoopback = nonLoopback
	svc, err := core.New(db, dir, cc, log)
	if err != nil {
		return err
	}
	if nonLoopback {
		log.Warn("listening on a non-loopback address: put a TLS-terminating reverse proxy in front", "listen", cfg.Listen)
	}
	if !cfg.PublicURLSet {
		log.Warn("no public URL configured; image links use the listen address (set PICBERTH_PUBLIC_URL)", "public_url", cfg.PublicURL)
	}

	var web fs.FS
	if cfg.DevWebDir != "" {
		web = os.DirFS(cfg.DevWebDir)
		log.Warn("serving the web UI from a directory (development mode)", "dir", cfg.DevWebDir)
	} else {
		web = webui.FS()
	}
	api := httpapi.New(httpapi.Options{Core: svc, PublicURL: cfg.PublicURL, ImageURL: cfg.ImageURL, Trusted: cfg.Trusted, Web: web, Logger: log})

	srv := &http.Server{
		Handler:           api.Handler(),
		ReadHeaderTimeout: 10 * time.Second,
		IdleTimeout:       120 * time.Second,
		MaxHeaderBytes:    64 << 10,
		ErrorLog:          slog.NewLogLogger(log.Handler(), slog.LevelWarn),
	}
	ln, err := net.Listen("tcp", cfg.Listen)
	if err != nil {
		svc.Close()
		return fmt.Errorf("listening on %s: %w", cfg.Listen, err)
	}

	svc.StartJobs()
	if code, err := svc.EnsureSetupCode(ctx); err != nil {
		log.Error("cannot determine setup state", "err", err)
	} else if code != "" {
		log.Warn("no administrator exists yet. Open "+cfg.PublicURL+"/setup and enter this one-time setup code, or run `picberth admin create`",
			"setup_code", code)
	}
	log.Info("listening", "addr", ln.Addr().String(), "public_url", cfg.PublicURL)
	if ready != nil {
		ready <- ln.Addr().String()
	}

	errCh := make(chan error, 1)
	go func() { errCh <- srv.Serve(ln) }()

	var serveErr error
	select {
	case <-ctx.Done():
		log.Info("shutting down")
	case serveErr = <-errCh:
	}
	api.SetReady(false)
	sctx, cancel := context.WithTimeout(context.Background(), shutdownTimeout)
	defer cancel()
	if err := srv.Shutdown(sctx); err != nil {
		log.Warn("graceful shutdown timed out; closing connections", "err", err)
		srv.Close()
	}
	svc.Close()
	if serveErr != nil && !errors.Is(serveErr, http.ErrServerClosed) {
		return serveErr
	}
	log.Info("stopped")
	return nil
}
