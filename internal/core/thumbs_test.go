package core

import (
	"context"
	"errors"
	"fmt"
	"io/fs"
	"os"
	"slices"
	"testing"

	"picberth/internal/meta"
)

func TestThumbnailBackfillPassesUnsettledPage(t *testing.T) {
	for _, broken := range []bool{false, true} {
		t.Run(fmt.Sprint("broken=", broken), func(t *testing.T) {
			e := newEnv(t)
			ctx := context.Background()
			for i := 0; i < 200; i++ {
				id := fmt.Sprintf("00000000000000000%03d", i)
				_, err := e.db.InsertImage(ctx, meta.Image{ID: id, OwnerID: e.admin.UserID,
					Source: "web", ObjectKey: "00/00/" + id + ".png", OriginalName: "missing.png",
					Mime: "image/png", Ext: "png", Size: 1, Width: 1, Height: 1, Frames: 1, SHA256: id}, nil)
				if err != nil {
					t.Fatal(err)
				}
				if broken {
					at := int64(1)
					if _, err := e.db.SetBroken(ctx, id, &at); err != nil {
						t.Fatal(err)
					}
				}
			}
			im := e.mustUpload(e.admin, "healthy.png", makePNG(t, 3, 3, 123))
			e.svc.ProcessThumbnails(ctx)
			got, err := e.db.GetImage(ctx, im.ID)
			if err != nil || got.Thumb == "" {
				t.Fatalf("unreadable head page blocked a healthy image: %+v, %v", got, err)
			}
		})
	}
}

func TestFullCheckRepairsMissingThumbnail(t *testing.T) {
	e := newEnv(t)
	ctx := context.Background()
	missing := e.mustUpload(e.admin, "missing-thumb.png", makePNG(t, 3, 3, 124))
	cached := e.mustUpload(e.admin, "cached.png", makePNG(t, 3, 3, 125))
	e.svc.ProcessThumbnails(ctx)
	if err := e.svc.blobs.RemoveThumbs(missing.ID); err != nil {
		t.Fatal(err)
	}

	if rep := e.mustCheck(CheckFull); !rep.OK || len(rep.Issues) != 0 {
		t.Fatalf("full check: %+v", rep)
	}
	for _, tc := range []struct {
		id   string
		want string
	}{{missing.ID, ""}, {cached.ID, "png"}} {
		im, err := e.db.GetImage(ctx, tc.id)
		if err != nil || im.Thumb != tc.want {
			t.Errorf("image %s: %+v, %v, want thumb state %q", tc.id, im, err, tc.want)
		}
	}
	e.svc.ProcessThumbnails(ctx)
	thumb, err := e.svc.OpenThumb(ctx, e.admin, missing.ID)
	if err != nil {
		t.Fatal(err)
	}
	thumb.File.Close()
}

func TestFullCheckContinuesAfterUnreadableThumbnail(t *testing.T) {
	e := newEnv(t)
	ctx := context.Background()
	first := e.mustUpload(e.admin, "first.png", makePNG(t, 5, 5, 126))
	next := e.mustUpload(e.admin, "next.png", makePNG(t, 5, 5, 127))
	if next.ID < first.ID { // Check visits records in ID order.
		first, next = next, first
	}
	e.svc.ProcessThumbnails(ctx)
	thumbPath := "cache/thumbs/" + first.ID[:2] + "/" + first.ID + ".png"
	if err := os.Chmod(e.dir.Abs(thumbPath), 0); err != nil {
		t.Fatal(err)
	}
	f, err := e.svc.blobs.OpenThumb(first.ID, "png")
	if err == nil {
		f.Close()
		t.Skip("this account can read a mode-000 file")
	}
	if !errors.Is(err, fs.ErrPermission) {
		t.Fatal(err)
	}

	for _, id := range []string{first.ID, next.ID} {
		im, err := e.db.GetImage(ctx, id)
		if err != nil {
			t.Fatal(err)
		}
		name := "objects/" + im.ObjectKey
		data, err := e.dir.Root().ReadFile(name)
		if err != nil {
			t.Fatal(err)
		}
		data[len(data)-1] ^= 1
		if err := e.dir.Root().WriteFile(name, data, 0o600); err != nil {
			t.Fatal(err)
		}
	}

	rep := e.mustCheck(CheckFull)
	if rep.OK || rep.Checked != 2 || len(rep.Issues) != 3 {
		t.Fatalf("full check: %+v", rep)
	}
	if !slices.ContainsFunc(rep.Issues, func(i Issue) bool {
		return i.Kind == IssueMissingFile && !i.Repaired && i.ImageID != nil && *i.ImageID == first.ID &&
			i.Path != nil && *i.Path == thumbPath
	}) {
		t.Fatalf("missing unreadable thumbnail in report: %+v", rep.Issues)
	}
	for _, id := range []string{first.ID, next.ID} {
		if !slices.ContainsFunc(rep.Issues, func(i Issue) bool {
			return i.Kind == IssueHashMismatch && !i.Repaired && i.ImageID != nil && *i.ImageID == id
		}) {
			t.Errorf("missing hash mismatch for %s", id)
		}
		_, err := e.svc.LookupPublic(ctx, id, "png")
		wantCode(t, err, CodeNotFound)
	}
}

func TestFullCheckReportsInvalidThumbnailKey(t *testing.T) {
	for _, tc := range []struct {
		state, kind string
		checked     int
	}{
		{meta.StateActive, IssueMissingFile, 2},
		{meta.StateTrashed, IssueMissingFile, 2},
		{meta.StatePurged, IssueOrphanFile, 1},
	} {
		t.Run(tc.state, func(t *testing.T) {
			e := newEnv(t)
			ctx := context.Background()
			bad := e.mustUpload(e.admin, "bad-id.png", makePNG(t, 5, 5, 128))
			later := e.mustUpload(e.admin, "later.png", makePNG(t, 5, 5, 129))
			e.svc.ProcessThumbnails(ctx)
			if tc.state != meta.StateActive {
				if err := e.svc.Trash(ctx, e.admin, bad.ID); err != nil {
					t.Fatal(err)
				}
			}
			if tc.state == meta.StatePurged {
				if err := e.svc.Purge(ctx, e.admin, bad.ID); err != nil {
					t.Fatal(err)
				}
			}
			// The schema accepts a short ID. Its object key still points to the original.
			if _, err := e.db.Writer().ExecContext(ctx, "UPDATE images SET id = '!' WHERE id = ?", bad.ID); err != nil {
				t.Fatal(err)
			}
			if _, err := e.db.SetBroken(ctx, later.ID, ptr(int64(1))); err != nil {
				t.Fatal(err)
			}

			rep := e.mustCheck(CheckFull)
			if rep.OK || rep.Checked != tc.checked || len(rep.Issues) != 1 {
				t.Fatalf("full check: %+v", rep)
			}
			i := rep.Issues[0]
			if i.Kind != tc.kind || i.ImageID == nil || *i.ImageID != "!" || i.Path != nil || i.Repaired {
				t.Fatalf("invalid cache key: %+v", i)
			}
			im, err := e.db.GetImage(ctx, later.ID)
			if err != nil || im.BrokenAt != nil {
				t.Fatalf("later original did not pass its hash check: %+v, %v", im, err)
			}
		})
	}
}

func TestThumbnailRetry(t *testing.T) {
	for _, failure := range []string{"cache", "outcome"} {
		t.Run(failure, func(t *testing.T) {
			e := newEnv(t)
			ctx := context.Background()
			im := e.mustUpload(e.admin, "retry.png", makePNG(t, 3, 3, 120))
			exec := func(query string) {
				t.Helper()
				if _, err := e.db.Writer().ExecContext(ctx, query); err != nil {
					t.Fatal(err)
				}
			}
			shard := e.dir.Abs("cache/thumbs/" + im.ID[:2])
			if failure == "cache" {
				if err := os.WriteFile(shard, nil, 0600); err != nil {
					t.Fatal(err)
				}
			} else {
				exec("CREATE TRIGGER fail_thumb BEFORE UPDATE OF thumb ON images BEGIN SELECT RAISE(ABORT, 'temporary failure'); END")
			}
			e.svc.thumbOne(ctx, <-e.svc.thumbQ)
			if len(e.svc.thumbQ) != 0 || !e.svc.thumbDirty.Swap(false) {
				t.Fatal("expected retry on next tick with no immediate requeue")
			}
			if failure == "cache" {
				if err := os.Remove(shard); err != nil {
					t.Fatal(err)
				}
			} else {
				exec("DROP TRIGGER fail_thumb")
			}
			exec("ALTER TABLE images RENAME TO delayed_images")
			e.svc.ProcessThumbnails(ctx)
			if !e.svc.thumbDirty.Swap(false) {
				t.Fatal("failed pending-image query lost the scheduled retry")
			}
			exec("ALTER TABLE delayed_images RENAME TO images")
			e.svc.ProcessThumbnails(ctx)
			thumb, err := e.svc.OpenThumb(ctx, e.admin, im.ID)
			if err != nil {
				t.Fatal(err)
			}
			thumb.File.Close()
			if e.svc.thumbDirty.Load() {
				t.Fatal("successful retry scheduled another backfill")
			}
		})
	}
}
