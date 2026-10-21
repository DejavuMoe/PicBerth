package blob

import (
	"errors"
	"io"
	"io/fs"
	"os"
	"strings"
	"testing"
	"time"

	"picberth/internal/datadir"
	"picberth/internal/ids"
)

func newStore(t *testing.T) (*Store, *datadir.Dir) {
	t.Helper()
	d, err := datadir.Open(t.TempDir(), datadir.Options{Create: true})
	if err != nil {
		t.Fatal(err)
	}
	t.Cleanup(func() { d.Close() })
	return New(d.Root()), d
}

func putObject(t *testing.T, s *Store, content string) string {
	t.Helper()
	tmp, err := s.CreateTemp()
	if err != nil {
		t.Fatal(err)
	}
	if _, err := tmp.WriteString(content); err != nil {
		t.Fatal(err)
	}
	if err := tmp.Sync(); err != nil {
		t.Fatal(err)
	}
	if err := tmp.Close(); err != nil {
		t.Fatal(err)
	}
	key := ids.ObjectKey(ids.NewImageID(), "png")
	if err := s.CommitTemp(tmp.Name(), key); err != nil {
		t.Fatal(err)
	}
	return key
}

func readAll(t *testing.T, s *Store, a Area, key string) string {
	t.Helper()
	f, err := s.Open(a, key)
	if err != nil {
		t.Fatal(err)
	}
	defer f.Close()
	b, _ := io.ReadAll(f)
	return string(b)
}

func TestCommitMoveRestoreRemove(t *testing.T) {
	s, _ := newStore(t)
	key := putObject(t, s, "hello")
	if got := readAll(t, s, Objects, key); got != "hello" {
		t.Fatalf("content %q", got)
	}
	if tl, _ := s.ListTemp(); len(tl) != 0 {
		t.Fatalf("temp not consumed: %v", tl)
	}

	if err := s.Move(Objects, Trash, key); err != nil {
		t.Fatal(err)
	}
	if _, err := s.Stat(Objects, key); !errors.Is(err, fs.ErrNotExist) {
		t.Fatalf("still in objects: %v", err)
	}
	if got := readAll(t, s, Trash, key); got != "hello" {
		t.Fatalf("trash content %q", got)
	}
	// idempotent: already moved
	if err := s.Move(Objects, Trash, key); err != nil {
		t.Fatalf("repeat move: %v", err)
	}
	if err := s.Move(Trash, Objects, key); err != nil {
		t.Fatal(err)
	}
	if got := readAll(t, s, Objects, key); got != "hello" {
		t.Fatalf("restored content %q", got)
	}
	if err := s.Remove(Objects, key); err != nil {
		t.Fatal(err)
	}
	if err := s.Remove(Objects, key); err != nil {
		t.Fatalf("removing a missing object must succeed: %v", err)
	}
	if err := s.Move(Objects, Trash, key); !errors.Is(err, fs.ErrNotExist) {
		t.Fatalf("moving a missing object: %v", err)
	}
}

func TestCommitNeverOverwrites(t *testing.T) {
	s, _ := newStore(t)
	key := putObject(t, s, "first")
	tmp, _ := s.CreateTemp()
	tmp.WriteString("second")
	tmp.Close()
	if err := s.CommitTemp(tmp.Name(), key); !errors.Is(err, ErrExists) {
		t.Fatalf("want ErrExists, got %v", err)
	}
	if got := readAll(t, s, Objects, key); got != "first" {
		t.Fatalf("object overwritten: %q", got)
	}
	if err := s.RemoveTemp(tmp.Name()); err != nil {
		t.Fatal(err)
	}
}

func TestInvalidKeysRejected(t *testing.T) {
	s, _ := newStore(t)
	good := ids.ObjectKey(ids.NewImageID(), "jpg")
	if !ValidKey(good) {
		t.Fatalf("generated key %q rejected", good)
	}
	bad := []string{
		"", "../../etc/passwd", "ab/cd/abcdefghjkmnpqrstvwx.png/../../x", "AB/CD/ABCDEFGHJKMNPQRSTVWX.PNG",
		`ab\cd\abcdefghjkmnpqrstvwx.png`, "ab/cd/abcdefghjkmnpqrstvwx", "xx/yy/abcdefghjkmnpqrstvwx.png",
		strings.Repeat("a", 300), "ab/cd/abcdefghjkmnpqrstvwx.png\x00",
	}
	for _, k := range bad {
		if ValidKey(k) {
			t.Errorf("ValidKey(%q) = true", k)
		}
		if _, err := s.Open(Objects, k); !errors.Is(err, ErrInvalidKey) {
			t.Errorf("Open(%q): %v", k, err)
		}
		if err := s.Remove(Trash, k); !errors.Is(err, ErrInvalidKey) {
			t.Errorf("Remove(%q): %v", k, err)
		}
		if err := s.Move(Objects, Trash, k); !errors.Is(err, ErrInvalidKey) {
			t.Errorf("Move(%q): %v", k, err)
		}
	}
	if err := s.RemoveTemp("tmp/../FORMAT"); err == nil {
		t.Error("RemoveTemp accepted traversal")
	}
	if err := s.RemoveTemp("FORMAT"); err == nil {
		t.Error("RemoveTemp accepted non-tmp path")
	}
}

func TestWalkAndQuarantine(t *testing.T) {
	s, d := newStore(t)
	k1 := putObject(t, s, "one")
	k2 := putObject(t, s, "two")
	if err := s.Move(Objects, Trash, k2); err != nil {
		t.Fatal(err)
	}
	// stray file with a non-key name
	if err := os.WriteFile(d.Abs("objects/stray.txt"), []byte("x"), 0o600); err != nil {
		t.Fatal(err)
	}

	var keys, strays []string
	err := s.Walk(Objects, func(f File) error {
		if f.Key != "" {
			keys = append(keys, f.Key)
		} else {
			strays = append(strays, f.Rel)
		}
		return nil
	})
	if err != nil {
		t.Fatal(err)
	}
	if len(keys) != 1 || keys[0] != k1 || len(strays) != 1 || strays[0] != "objects/stray.txt" {
		t.Fatalf("walk: keys=%v strays=%v", keys, strays)
	}

	q, err := s.Quarantine(Objects, k1, time.UnixMilli(1700000000000))
	if err != nil {
		t.Fatal(err)
	}
	if !strings.HasPrefix(q, "quarantine/1700000000000-") {
		t.Fatalf("quarantine path %q", q)
	}
	if _, err := s.Stat(Objects, k1); !errors.Is(err, fs.ErrNotExist) {
		t.Fatal("orphan still in objects")
	}
	q2, err := s.QuarantineRaw("objects/stray.txt", time.UnixMilli(1700000000000))
	if err != nil || !strings.HasPrefix(q2, "quarantine/") {
		t.Fatalf("raw quarantine: %q %v", q2, err)
	}
}

func TestThumbs(t *testing.T) {
	s, _ := newStore(t)
	id := ids.NewImageID()
	if err := s.WriteThumb(id, "jpg", []byte("thumb")); err != nil {
		t.Fatal(err)
	}
	f, err := s.OpenThumb(id, "jpg")
	if err != nil {
		t.Fatal(err)
	}
	b, _ := io.ReadAll(f)
	f.Close()
	if string(b) != "thumb" {
		t.Fatalf("thumb %q", b)
	}
	if err := s.WriteThumb(id, "gif", nil); !errors.Is(err, ErrInvalidKey) {
		t.Fatalf("bad ext: %v", err)
	}
	if err := s.WriteThumb("../x", "jpg", nil); !errors.Is(err, ErrInvalidKey) {
		t.Fatalf("bad id: %v", err)
	}
	if err := s.RemoveThumbs(id); err != nil {
		t.Fatal(err)
	}
	if _, err := s.OpenThumb(id, "jpg"); !errors.Is(err, fs.ErrNotExist) {
		t.Fatalf("thumb not removed: %v", err)
	}
	if tl, _ := s.ListTemp(); len(tl) != 0 {
		t.Fatalf("thumb temp leaked: %v", tl)
	}
}

func TestListTempAndRemove(t *testing.T) {
	s, _ := newStore(t)
	tmp, err := s.CreateTemp()
	if err != nil {
		t.Fatal(err)
	}
	tmp.Close()
	l, err := s.ListTemp()
	if err != nil || len(l) != 1 || l[0].Name != tmp.Name() {
		t.Fatalf("ListTemp = %v, %v", l, err)
	}
	if err := s.RemoveTemp(tmp.Name()); err != nil {
		t.Fatal(err)
	}
	if err := s.RemoveTemp(tmp.Name()); err != nil {
		t.Fatalf("second remove: %v", err)
	}
}
