package core

import (
	"context"
	"testing"

	"picberth/internal/blob"
)

func TestQuickCheckRemovesEveryPurgedFile(t *testing.T) {
	e := newEnv(t)
	ctx := context.Background()
	v := e.mustUpload(e.admin, "a.png", makePNG(t, 3, 3, 42))
	e.svc.ProcessThumbnails(ctx)
	im, err := e.db.GetImage(ctx, v.ID)
	if err != nil {
		t.Fatal(err)
	}
	if _, err := e.db.MarkTrashed(ctx, im.ID, 1); err != nil {
		t.Fatal(err)
	}
	if _, err := e.db.MarkPurged(ctx, im.ID, 2); err != nil {
		t.Fatal(err)
	}
	// The database is committed but the original object and thumbnail are still present.
	rep := e.mustCheck(CheckQuick)
	if !rep.OK || len(rep.Issues) != 2 || !hasIssue(rep, IssueOrphanFile) {
		t.Fatalf("cleanup report: %+v", rep)
	}
	if e.fileExists("objects/"+im.ObjectKey) || e.fileExists("cache/thumbs/"+im.ID[:2]+"/"+im.ID+".png") {
		t.Fatal("purged bytes survived the quick check")
	}
	// A cache-only leftover must be retried even when both object areas are empty.
	if err := e.svc.blobs.WriteThumb(im.ID, "png", []byte("leftover")); err != nil {
		t.Fatal(err)
	}
	if rep := e.mustCheck(CheckQuick); !rep.OK || len(rep.Issues) != 1 {
		t.Fatalf("cache-only cleanup: %+v", rep)
	}
	if _, err := e.svc.blobs.Open(blob.Objects, im.ObjectKey); err == nil {
		t.Fatal("purged object returned")
	}
}
