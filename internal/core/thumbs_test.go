package core

import (
	"context"
	"fmt"
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
