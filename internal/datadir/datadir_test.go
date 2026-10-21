package datadir

import (
	"errors"
	"os"
	"path/filepath"
	"testing"
)

func TestOpenCreatesLayout(t *testing.T) {
	p := filepath.Join(t.TempDir(), "data")
	d, err := Open(p, Options{Create: true})
	if err != nil {
		t.Fatal(err)
	}
	defer d.Close()
	for _, sub := range []string{"objects", "trash", "tmp", "cache/thumbs", "quarantine"} {
		if st, err := d.Root().Stat(sub); err != nil || !st.IsDir() {
			t.Errorf("%s missing: %v", sub, err)
		}
	}
	b, err := d.Root().ReadFile("FORMAT")
	if err != nil || string(b) != "picberth-data 1\n" {
		t.Fatalf("FORMAT = %q, %v", b, err)
	}
	if free, total, err := d.FreeSpace(); err != nil || total == 0 || free > total {
		t.Fatalf("FreeSpace = %d/%d, %v", free, total, err)
	}
}

func TestSecondInstanceFails(t *testing.T) {
	p := t.TempDir()
	d1, err := Open(p, Options{Create: true})
	if err != nil {
		t.Fatal(err)
	}
	d2, err := Open(p, Options{})
	if err == nil {
		d2.Close()
		t.Fatal("second Open succeeded while the first holds the lock")
	}
	if !errors.Is(err, ErrLocked) {
		t.Fatalf("want ErrLocked, got %v", err)
	}
	d1.Close()
	d3, err := Open(p, Options{})
	if err != nil {
		t.Fatalf("Open after release failed: %v", err)
	}
	d3.Close()
}

func TestMissingDirWithoutCreate(t *testing.T) {
	if _, err := Open(filepath.Join(t.TempDir(), "nope"), Options{}); err == nil {
		t.Fatal("expected error")
	}
}

func TestRefusesForeignDirectory(t *testing.T) {
	p := t.TempDir()
	if err := os.WriteFile(filepath.Join(p, "something.txt"), []byte("x"), 0o600); err != nil {
		t.Fatal(err)
	}
	if _, err := Open(p, Options{Create: true}); err == nil {
		t.Fatal("non-empty directory without FORMAT must be refused")
	}
}

func TestUnknownFormatMajor(t *testing.T) {
	p := t.TempDir()
	if err := os.WriteFile(filepath.Join(p, "FORMAT"), []byte("picberth-data 2\n"), 0o600); err != nil {
		t.Fatal(err)
	}
	if _, err := Open(p, Options{}); err == nil {
		t.Fatal("unknown major version must be refused")
	}
	p2 := t.TempDir()
	os.WriteFile(filepath.Join(p2, "FORMAT"), []byte("garbage"), 0o600)
	if _, err := Open(p2, Options{}); err == nil {
		t.Fatal("garbage FORMAT must be refused")
	}
}

func TestRequireEmpty(t *testing.T) {
	p := t.TempDir()
	d, err := Open(p, Options{Create: true, RequireEmpty: true})
	if err != nil {
		t.Fatal(err)
	}
	d.Close()
	if _, err := Open(p, Options{RequireEmpty: true}); err == nil {
		t.Fatal("populated directory accepted as empty")
	}
}
