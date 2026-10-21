package imaging

import (
	"bytes"
	"testing"
)

func fuzzSeeds(f *testing.F) {
	seeds := [][]byte{
		mkJPEG(f, 24, 16),
		jpegWithEXIF(f, 24, 16, exifOpts{orient: 6, gps: true}),
		jpegWithEXIF(f, 24, 16, exifOpts{orient: 3, bigEndian: true, gps: true}),
		jpegWithSegment(mkJPEG(f, 8, 8), app1(append([]byte("Exif\x00\x00"), exifLoop()...))),
		jpegDims(40, 30),
		encodePNG(f, gradient(16, 16)),
		encodePNG(f, translucent(16, 16)),
		pngInsert(f, encodePNG(f, gradient(8, 8)), pchunk{"eXIf", buildEXIF(exifOpts{orient: 5, gps: true})}),
		pngInsert(f, encodePNG(f, gradient(8, 8)), pchunk{"tRNS", []byte{0, 0, 0, 0, 0, 0}}),
		mkAPNG(f, 8, 8, 3),
		mkGIF(f, 12, 9, 1, false),
		mkGIF(f, 12, 9, 3, true),
		rawGIF(10, 10, 2, true),
		webpLossless(33, 21, false),
		webpLossless(33, 21, true),
		webpLossyStub(64, 48),
		webpWithEXIF(10, 10, exifOpts{orient: 6, gps: true}, true),
		webpAnimated(30, 20, 3),
		mkAVIF(avifOpts{w: 800, h: 600, alpha: true}),
		mkAVIF(avifOpts{w: 8, h: 8, major: "avis", moov: true}),
		append(mkJPEG(f, 8, 8), []byte("<html>")...),
		[]byte("<svg xmlns='http://www.w3.org/2000/svg'/>"),
	}
	for _, s := range seeds {
		f.Add(s)
	}
}

func FuzzSniff(f *testing.F) {
	fuzzSeeds(f)
	f.Fuzz(func(t *testing.T, data []byte) {
		format, ok := Sniff(data)
		if ok {
			if _, valid := ParseFormat(string(format)); !valid {
				t.Fatalf("Sniff returned unknown format %q", format)
			}
		}
	})
}

func FuzzInspect(f *testing.F) {
	fuzzSeeds(f)
	f.Fuzz(func(t *testing.T, data []byte) {
		info, err := Inspect(bytes.NewReader(data), int64(len(data)), Limits{MaxPixels: 1 << 24, MaxFrames: 100})
		if err != nil {
			return
		}
		if info.Width <= 0 || info.Height <= 0 || int64(info.Width)*int64(info.Height) > 1<<24 ||
			info.Frames < 1 || info.Frames > 100 || info.Orientation < 1 || info.Orientation > 8 {
			t.Fatalf("accepted with implausible info %+v", info)
		}
	})
}

func FuzzThumbnail(f *testing.F) {
	fuzzSeeds(f)
	f.Fuzz(func(t *testing.T, data []byte) {
		info, err := Inspect(bytes.NewReader(data), int64(len(data)), Limits{MaxPixels: 1 << 20, MaxFrames: 100})
		if err != nil {
			return
		}
		out, ext, err := Thumbnail(bytes.NewReader(data), int64(len(data)), info, 64, 1<<18)
		if err != nil {
			return
		}
		if len(out) == 0 || (ext != "jpg" && ext != "png") {
			t.Fatalf("bad thumbnail output: %d bytes, ext %q", len(out), ext)
		}
	})
}
