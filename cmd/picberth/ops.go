package main

import (
	"context"
	"errors"
	"flag"
	"fmt"
	"io"
	"net"
	"net/http"
	"strings"
	"time"

	"picberth/internal/config"
	"picberth/internal/core"
)

func readPassword(r io.Reader) (string, error) {
	b, err := io.ReadAll(io.LimitReader(r, 4096))
	if err != nil {
		return "", err
	}
	pw := strings.TrimSuffix(strings.TrimSuffix(string(b), "\n"), "\r")
	if pw == "" {
		return "", errors.New("no password on stdin")
	}
	return pw, nil
}

func cmdAdmin(ctx context.Context, args []string, st streams) error {
	if len(args) == 0 || (args[0] != "create" && args[0] != "reset-password") {
		return usagef("expected \"create\" or \"reset-password\"")
	}
	sub := args[0]
	fs_, flags := newFlagSet("admin "+sub, st)
	username := fs_.String("username", "", "administrator user name")
	pwStdin := fs_.Bool("password-stdin", false, "read the password from standard input")
	if err := fs_.Parse(args[1:]); err != nil {
		return err
	}
	if *username == "" {
		return usagef("--username is required")
	}
	if !*pwStdin {
		return usagef("--password-stdin is required (passwords are never accepted as arguments)")
	}
	cfg, err := flags.Resolve(st.env)
	if err != nil {
		return err
	}
	pw, err := readPassword(st.stdin)
	if err != nil {
		return err
	}
	o, err := openOffline(ctx, cfg, st, sub == "create")
	if err != nil {
		return err
	}
	defer o.Close()
	if sub == "create" {
		if err := o.svc.CreateAdmin(ctx, *username, pw); err != nil {
			return err
		}
		fmt.Fprintf(st.stdout, "administrator %q created\n", *username)
		return nil
	}
	if err := o.svc.ResetPassword(ctx, *username, pw); err != nil {
		return err
	}
	fmt.Fprintf(st.stdout, "password for %q reset; existing sessions ended\n", *username)
	return nil
}

func cmdCheck(ctx context.Context, args []string, st streams) error {
	fs_, flags := newFlagSet("check", st)
	full := fs_.Bool("full", false, "also re-hash every file and look for orphan files")
	if err := fs_.Parse(args); err != nil {
		return err
	}
	cfg, err := flags.Resolve(st.env)
	if err != nil {
		return err
	}
	o, err := openOffline(ctx, cfg, st, false)
	if err != nil {
		return err
	}
	defer o.Close()
	mode := core.CheckQuick
	if *full {
		mode = core.CheckFull
	}
	rep, err := o.svc.Check(ctx, core.CLIPrincipal(), mode)
	if err != nil {
		return err
	}
	printReport(st.stdout, rep)
	if !rep.OK {
		return errors.New("check found problems that could not be repaired")
	}
	return nil
}

func printReport(w io.Writer, rep *core.CheckReport) {
	fmt.Fprintf(w, "%s check: %d images checked in %d ms, %d issue(s)\n", rep.Mode, rep.Checked, rep.DurationMs, len(rep.Issues))
	for _, is := range rep.Issues {
		id, path := "-", "-"
		if is.ImageID != nil {
			id = *is.ImageID
		}
		if is.Path != nil {
			path = *is.Path
		}
		state := "NOT REPAIRED"
		if is.Repaired {
			state = "repaired"
		}
		fmt.Fprintf(w, "  %-14s image=%s path=%s [%s]\n", is.Kind, id, path, state)
	}
}

func cmdSnapshot(ctx context.Context, args []string, st streams) error {
	fs_, flags := newFlagSet("snapshot", st)
	out := fs_.String("out", "", "directory to create the snapshot in (default: the configured snapshot directory)")
	if err := fs_.Parse(args); err != nil {
		return err
	}
	cfg, err := flags.Resolve(st.env)
	if err != nil {
		return err
	}
	o, err := openOffline(ctx, cfg, st, false)
	if err != nil {
		return err
	}
	defer o.Close()
	info, err := o.svc.Snapshot(ctx, core.CLIPrincipal(), *out)
	if err != nil {
		return err
	}
	fmt.Fprintf(st.stdout, "snapshot written to %s (%d images, %d bytes)\n", info.Path, info.Images, info.Bytes)
	return nil
}

func cmdRestore(ctx context.Context, args []string, st streams) error {
	fs_, flags := newFlagSet("restore", st)
	from := fs_.String("from", "", "snapshot directory to restore")
	if err := fs_.Parse(args); err != nil {
		return err
	}
	if *from == "" {
		return usagef("--from is required")
	}
	cfg, err := flags.Resolve(st.env)
	if err != nil {
		return err
	}
	log := newLogger(cfg, st.stderr)
	rep, err := core.Restore(ctx, *from, cfg.DataDir, coreConfig(cfg), log)
	if rep != nil {
		printReport(st.stdout, rep)
	}
	if err != nil {
		return err
	}
	fmt.Fprintf(st.stdout, "restored %d images into %s; verification passed\n", rep.Checked, cfg.DataDir)
	return nil
}

func cmdHealthcheck(ctx context.Context, args []string, st streams) error {
	fs_, flags := newFlagSet("healthcheck", st)
	if err := fs_.Parse(args); err != nil {
		return err
	}
	_ = flags
	explicit := map[string]string{}
	fs_.Visit(func(f *flag.Flag) {
		if f.Name == "listen" {
			explicit[config.KeyListen] = f.Value.String()
		}
		if f.Name == "config" {
			explicit[config.KeyConfig] = f.Value.String()
		}
	})
	listen, err := config.ResolveListen(explicit, st.env)
	if err != nil {
		return err
	}
	host, port, _ := net.SplitHostPort(listen)
	switch host {
	case "", "0.0.0.0":
		host = "127.0.0.1"
	case "::":
		host = "::1"
	}
	ctx, cancel := context.WithTimeout(ctx, 3*time.Second)
	defer cancel()
	req, err := http.NewRequestWithContext(ctx, http.MethodGet, "http://"+net.JoinHostPort(host, port)+"/readyz", nil)
	if err != nil {
		return err
	}
	resp, err := (&http.Client{CheckRedirect: func(*http.Request, []*http.Request) error { return http.ErrUseLastResponse }}).Do(req)
	if err != nil {
		return err
	}
	defer resp.Body.Close()
	io.Copy(io.Discard, io.LimitReader(resp.Body, 1<<10))
	if resp.StatusCode != http.StatusOK {
		return fmt.Errorf("readiness probe returned %s", resp.Status)
	}
	return nil
}
