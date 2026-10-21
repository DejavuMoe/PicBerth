package imaging

import (
	"bytes"
	"image"
	"image/color"
	"sync"
	"testing"
)

var (
	bigJPEGOnce sync.Once
	bigJPEG     []byte
)

// photoLike returns a 6000x4000 (24 MP) image with smooth structure plus noise, so that the JPEG has a
// realistic entropy-coding cost.
func photoLike(w, h int) *image.RGBA {
	img := image.NewRGBA(image.Rect(0, 0, w, h))
	seed := uint32(12345)
	for y := 0; y < h; y++ {
		row := img.Pix[y*img.Stride:]
		for x := 0; x < w; x++ {
			seed = seed*1664525 + 1013904223
			n := int(seed>>28) - 8
			v := func(base int) uint8 {
				c := base + n
				if c < 0 {
					return 0
				}
				if c > 255 {
					return 255
				}
				return uint8(c)
			}
			p := row[x*4:]
			p[0], p[1], p[2], p[3] = v((x/16+y/24)&0xFF), v((x*y>>12)&0xFF), v(128+(x-y)/40), 255
		}
	}
	_ = color.RGBA{}
	return img
}

func benchJPEG(b *testing.B) ([]byte, Info) {
	bigJPEGOnce.Do(func() { bigJPEG = encodeJPEG(b, photoLike(6000, 4000)) })
	info, err := Inspect(bytes.NewReader(bigJPEG), int64(len(bigJPEG)), Limits{MaxPixels: 50_000_000})
	if err != nil {
		b.Fatal(err)
	}
	return bigJPEG, info
}

// BenchmarkThumbnail24MP decodes and thumbnails a 24-megapixel JPEG (design target: < 1.5 s).
func BenchmarkThumbnail24MP(b *testing.B) {
	data, info := benchJPEG(b)
	b.SetBytes(int64(len(data)))
	b.ReportAllocs()
	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		if _, _, err := Thumbnail(bytes.NewReader(data), int64(len(data)), info, 640, 25_000_000); err != nil {
			b.Fatal(err)
		}
	}
}

func BenchmarkInspect24MP(b *testing.B) {
	data, _ := benchJPEG(b)
	b.SetBytes(int64(len(data)))
	for i := 0; i < b.N; i++ {
		if _, err := Inspect(bytes.NewReader(data), int64(len(data)), Limits{MaxPixels: 50_000_000}); err != nil {
			b.Fatal(err)
		}
	}
}

func BenchmarkInspectPNG(b *testing.B) {
	data := encodePNG(b, gradient(2000, 1500))
	b.SetBytes(int64(len(data)))
	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		if _, err := Inspect(bytes.NewReader(data), int64(len(data)), Limits{}); err != nil {
			b.Fatal(err)
		}
	}
}
