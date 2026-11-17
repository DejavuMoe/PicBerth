package meta

import (
	"context"
	"path/filepath"
	"testing"
)

func BenchmarkPendingThumbsLargeLibrary(b *testing.B) {
	ctx := context.Background()
	db, err := Open(ctx, filepath.Join(b.TempDir(), "picberth.db"))
	if err != nil {
		b.Fatal(err)
	}
	b.Cleanup(func() { db.Close() })
	uid, _, err := db.CreateFirstUser(ctx, "bench", "fixture", 1)
	if err != nil {
		b.Fatal(err)
	}
	_, err = db.w.ExecContext(ctx, `WITH RECURSIVE n(i) AS (
		VALUES(1) UNION ALL SELECT i + 1 FROM n WHERE i < 20000
	) INSERT INTO images(id, owner_id, source, object_key, original_name, mime, ext, size,
		width, height, frames, sha256, has_gps, state, thumb, created_at, broken_at)
	SELECT printf('%020d', i), ?, 'web', printf('object-%d', i), 'fixture.png', 'image/png', 'png',
		1, 1, 1, 1, 'fixture', 0, 'active', CASE WHEN i % 2 = 0 THEN 'png' ELSE '' END, i,
		CASE WHEN i % 2 = 1 THEN 1 END FROM n`, uid)
	if err != nil {
		b.Fatal(err)
	}
	const pendingID = "zzzzzzzzzzzzzzzzzzzz"
	_, err = db.InsertImage(ctx, Image{ID: pendingID, OwnerID: uid, Source: "web", ObjectKey: "pending",
		OriginalName: "pending.png", Mime: "image/png", Ext: "png", Size: 1, Width: 1, Height: 1,
		Frames: 1, SHA256: "pending", CreatedAt: 20001}, nil)
	if err != nil {
		b.Fatal(err)
	}
	var node, parent, unused int
	var plan string
	if err := db.r.QueryRowContext(ctx, `EXPLAIN QUERY PLAN SELECT id FROM images
		WHERE state = 'active' AND broken_at IS NULL AND thumb = '' AND id > ? ORDER BY id LIMIT ?`, "", 200).
		Scan(&node, &parent, &unused, &plan); err != nil {
		b.Fatal(err)
	}
	b.Log(plan)
	b.ReportAllocs()
	b.ResetTimer()
	for b.Loop() {
		ids, err := db.PendingThumbs(ctx, "", 200)
		if err != nil || len(ids) != 1 || ids[0] != pendingID {
			b.Fatalf("pending images: %v, %v", ids, err)
		}
	}
}
