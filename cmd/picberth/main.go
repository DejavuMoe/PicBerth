// Command picberth is the PicBerth image host: `serve` runs the server; the other subcommands are offline
// operator tools that share the data directory (and therefore its instance lock) with the server.
package main

import (
	"context"
	"errors"
	"flag"
	"fmt"
	"io"
	"log/slog"
	"os"
	"os/signal"
	"runtime"
	"strings"
	"syscall"

	"picberth/internal/buildinfo"
	"picberth/internal/config"
	"picberth/internal/core"
	"picberth/internal/datadir"
	"picberth/internal/meta"
)

// streams bundles the process I/O so commands are testable.
type streams struct {
	stdin  io.Reader
	stdout io.Writer
	stderr io.Writer
	env    config.Env
}

func main() {
	st := streams{stdin: os.Stdin, stdout: os.Stdout, stderr: os.Stderr,
		env: config.Env{Getenv: os.Getenv, ReadFile: os.ReadFile}}
	ctx, stop := signal.NotifyContext(context.Background(), os.Interrupt, syscall.SIGTERM)
	code := run(ctx, os.Args[1:], st)
	stop()
	os.Exit(code)
}

const usage = `usage: picberth <command> [flags]

commands:
  serve                 run the server (default)
  admin create          create the first administrator   --username U --password-stdin
  admin reset-password  set a new password               --username U --password-stdin
  check [--full]        verify data consistency (offline)
  snapshot [--out DIR]  write a verified snapshot (offline)
  restore               rebuild a data dir from a snapshot --from SNAP --data-dir DIR
  healthcheck           probe /readyz of the configured listen address
  version               print version information

Configuration: flags > PICBERTH_* environment > --config file > defaults. Run "picberth serve -h" for the list.
`

func run(ctx context.Context, args []string, st streams) int {
	cmd := "serve"
	if len(args) > 0 && !strings.HasPrefix(args[0], "-") {
		cmd, args = args[0], args[1:]
	}
	var err error
	switch cmd {
	case "serve":
		err = cmdServe(ctx, args, st)
	case "admin":
		err = cmdAdmin(ctx, args, st)
	case "check":
		err = cmdCheck(ctx, args, st)
	case "snapshot":
		err = cmdSnapshot(ctx, args, st)
	case "restore":
		err = cmdRestore(ctx, args, st)
	case "healthcheck":
		err = cmdHealthcheck(ctx, args, st)
	case "version", "--version":
		fmt.Fprintf(st.stdout, "picberth %s (commit %s, %s, %s/%s)\n", buildinfo.Version, buildinfo.Commit, runtime.Version(), runtime.GOOS, runtime.GOARCH)
	case "help":
		fmt.Fprint(st.stdout, usage)
	default:
		fmt.Fprintf(st.stderr, "picberth: unknown command %q\n\n%s", cmd, usage)
		return 2
	}
	if err == nil {
		return 0
	}
	if errors.Is(err, flag.ErrHelp) {
		return 0
	}
	var ue *usageError
	if errors.As(err, &ue) {
		fmt.Fprintf(st.stderr, "picberth %s: %v\n", cmd, err)
		return 2
	}
	fmt.Fprintf(st.stderr, "picberth %s: %v\n", cmd, err)
	return 1
}

type usageError struct{ msg string }

func (e *usageError) Error() string { return e.msg }

func usagef(format string, a ...any) error { return &usageError{fmt.Sprintf(format, a...)} }

// newFlagSet creates a flagset with the shared configuration flags registered.
func newFlagSet(name string, st streams) (*flag.FlagSet, *config.Flags) {
	fs := flag.NewFlagSet("picberth "+name, flag.ContinueOnError)
	fs.SetOutput(st.stderr)
	return fs, config.RegisterFlags(fs)
}

func newLogger(cfg *config.Config, w io.Writer) *slog.Logger {
	opts := &slog.HandlerOptions{Level: cfg.LogLevel}
	if cfg.LogFormat == "json" {
		return slog.New(slog.NewJSONHandler(w, opts))
	}
	return slog.New(slog.NewTextHandler(w, opts))
}

func coreConfig(cfg *config.Config) core.Config {
	return core.Config{
		PublicURL: cfg.PublicURL, ImageURL: cfg.ImageURL, PublicURLSet: cfg.PublicURLSet,
		HardMaxUpload: cfg.HardMaxUpload, SnapshotDir: cfg.SnapshotDir,
	}
}

// offline is an opened data directory + database + service for operator commands.
type offline struct {
	dir *datadir.Dir
	db  *meta.DB
	svc *core.Service
	log *slog.Logger
}

func (o *offline) Close() {
	o.svc.Close()
	o.db.Close()
	o.dir.Close()
}

// openOffline takes the instance lock (failing clearly if the server is running) and opens the stores.
func openOffline(ctx context.Context, cfg *config.Config, st streams, create bool) (*offline, error) {
	log := newLogger(cfg, st.stderr)
	dir, err := datadir.Open(cfg.DataDir, datadir.Options{Create: create})
	if err != nil {
		return nil, err
	}
	db, err := meta.Open(ctx, dir.DBPath())
	if err != nil {
		dir.Close()
		return nil, err
	}
	svc, err := core.New(db, dir, coreConfig(cfg), log)
	if err != nil {
		db.Close()
		dir.Close()
		return nil, err
	}
	return &offline{dir: dir, db: db, svc: svc, log: log}, nil
}
