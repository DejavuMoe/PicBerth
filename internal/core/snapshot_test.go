package core

import (
	"context"
	"path/filepath"
	"testing"
)

func TestSnapshotWhileImageMovesBetweenObjectsAndTrash(t *testing.T) {
	e := newEnv(t)
	ctx := context.Background()
	img := e.mustUpload(e.admin, "moving.png", makePNG(t, 6, 6, 128))
	started, stop, done := make(chan struct{}), make(chan struct{}), make(chan struct{})
	var moveErr error
	go func() {
		defer close(done)
		for i := 0; ; i++ {
			if moveErr = e.svc.Trash(ctx, e.admin, img.ID); moveErr != nil {
				return
			}
			if _, moveErr = e.svc.Restore(ctx, e.admin, img.ID); moveErr != nil {
				return
			}
			if i == 0 {
				close(started)
			}
			select {
			case <-stop:
				return
			default:
			}
		}
	}()
	defer func() {
		close(stop)
		<-done
		if moveErr != nil {
			t.Errorf("moving source: %v", moveErr)
		}
	}()
	select {
	case <-started:
	case <-done:
		return // the deferred check reports the move error
	}

	var info SnapshotInfo
	for range 40 {
		var err error
		info, err = e.svc.Snapshot(ctx, e.admin, t.TempDir())
		if err != nil {
			t.Fatal(err)
		}
	}
	rep, err := Restore(ctx, info.Path, filepath.Join(t.TempDir(), "restored"), testConfig(t.TempDir()), quietLogger())
	if err != nil || !rep.OK || rep.Checked != 1 {
		t.Fatalf("restoring snapshot of moving image: %+v, %v", rep, err)
	}
}
