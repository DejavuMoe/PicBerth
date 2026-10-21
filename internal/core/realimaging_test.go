package core

import (
	"context"
	"testing"

	"picberth/internal/imaging"
)

// TestRealImagingSmoke runs the upload and thumbnail pipeline with the real imaging package. It skips
// while imaging is still a stub.
func TestRealImagingSmoke(t *testing.T) {
	png := makePNG(t, 40, 30, 7)
	if _, ok := imaging.Sniff(png); !ok {
		t.Skip("imaging is not implemented yet")
	}
	e := newEnv(t)
	e.svc.Sniff, e.svc.Inspect, e.svc.Thumbnail = imaging.Sniff, imaging.Inspect, imaging.Thumbnail
	v := e.mustUpload(e.admin, "real.png", png)
	if v.Width != 40 || v.Height != 30 || v.Ext != "png" {
		t.Fatalf("view: %+v", v)
	}
	jp := e.mustUpload(e.admin, "real.jpg", makeJPEG(t, 9))
	if jp.Ext != "jpg" || jp.Mime != "image/jpeg" {
		t.Fatalf("jpeg view: %+v", jp)
	}
	e.svc.ProcessThumbnails(context.Background())
	im, err := e.db.GetImage(context.Background(), v.ID)
	if err != nil || (im.Thumb != "png" && im.Thumb != "jpg") {
		t.Fatalf("thumb state %q (%v)", im.Thumb, err)
	}
	_, err = e.upload(e.admin, "x.png", []byte("<svg xmlns='http://www.w3.org/2000/svg'/>"))
	wantCode(t, err, CodeUnsupportedFormat)
	assertInvariants(t, e)
}
