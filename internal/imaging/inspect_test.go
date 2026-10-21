package imaging

import (
	"bytes"
	"errors"
	"runtime"
	"strings"
	"testing"
	"time"
)

func inspectBytes(b []byte, lim Limits) (Info, error) {
	return Inspect(bytes.NewReader(b), int64(len(b)), lim)
}

var testLimits = Limits{MaxPixels: 50_000_000}

func TestSniff(t *testing.T) {
	cases := []struct {
		name string
		in   []byte
		want Format
		ok   bool
	}{
		{"jpeg", []byte{0xFF, 0xD8, 0xFF, 0xE0}, JPEG, true},
		{"jpeg short", []byte{0xFF, 0xD8}, "", false},
		{"png", append(append([]byte(nil), pngSignature...), 0, 0), PNG, true},
		{"png short", pngSignature[:7], "", false},
		{"gif87", []byte("GIF87a.."), GIF, true},
		{"gif89", []byte("GIF89a.."), GIF, true},
		{"gif bad", []byte("GIF88a.."), "", false},
		{"webp", []byte("RIFF\x10\x00\x00\x00WEBPVP8 "), WebP, true},
		{"riff wave", []byte("RIFF\x10\x00\x00\x00WAVEfmt "), "", false},
		{"avif major", ftypBox("avif"), AVIF, true},
		{"avis major", ftypBox("avis"), AVIF, true},
		{"avif compat", ftypBox("mif1", "miaf", "avif"), AVIF, true},
		{"heic", ftypBox("heic", "mif1", "heic"), "", false},
		{"mp4", ftypBox("isom", "iso2", "mp41"), "", false},
		{"avif brand beyond box", append(ftypBox("heic", "mif1"), []byte("avif")...), "", false},
		{"svg", []byte(`<svg xmlns="http://www.w3.org/2000/svg">`), "", false},
		{"html", []byte("<!DOCTYPE html><html>"), "", false},
		{"bmp", []byte("BM\x36\x00\x00\x00\x00\x00"), "", false},
		{"tiff le", []byte("II*\x00\x08\x00\x00\x00"), "", false},
		{"tiff be", []byte("MM\x00*\x00\x00\x00\x08"), "", false},
		{"ico", []byte{0, 0, 1, 0, 1, 0, 16, 16}, "", false},
		{"empty", nil, "", false},
		{"one byte", []byte{0xFF}, "", false},
	}
	for _, c := range cases {
		t.Run(c.name, func(t *testing.T) {
			got, ok := Sniff(c.in)
			if got != c.want || ok != c.ok {
				t.Fatalf("Sniff = %q, %v; want %q, %v", got, ok, c.want, c.ok)
			}
		})
	}
}

func TestInspectAccepts(t *testing.T) {
	cases := []struct {
		name string
		data []byte
		want Info
	}{
		{"jpeg", mkJPEG(t, 40, 30), Info{Format: JPEG, Width: 40, Height: 30, Frames: 1, Orientation: 1}},
		{"jpeg orientation+gps", jpegWithEXIF(t, 40, 30, exifOpts{orient: 6, gps: true}),
			Info{Format: JPEG, Width: 40, Height: 30, Frames: 1, HasGPS: true, Orientation: 6}},
		{"jpeg exif big-endian orientation 3", jpegWithEXIF(t, 40, 30, exifOpts{orient: 3, bigEndian: true}),
			Info{Format: JPEG, Width: 40, Height: 30, Frames: 1, Orientation: 3}},
		{"jpeg exif without gps", jpegWithEXIF(t, 40, 30, exifOpts{orient: 8}),
			Info{Format: JPEG, Width: 40, Height: 30, Frames: 1, Orientation: 8}},
		{"jpeg gps reference tags only", jpegWithEXIF(t, 40, 30, exifOpts{gpsRefOnly: true}),
			Info{Format: JPEG, Width: 40, Height: 30, Frames: 1, Orientation: 1}},
		{"jpeg orientation out of range", jpegWithEXIF(t, 40, 30, exifOpts{orient: 9}),
			Info{Format: JPEG, Width: 40, Height: 30, Frames: 1, Orientation: 1}},
		{"jpeg stuffing/RST/fill/inner segment", jpegDims(123, 77), Info{Format: JPEG, Width: 123, Height: 77, Frames: 1, Orientation: 1}},
		{"png opaque", encodePNG(t, gradient(20, 10)), Info{Format: PNG, Width: 20, Height: 10, Frames: 1, Orientation: 1}},
		{"png alpha", encodePNG(t, translucent(20, 10)), Info{Format: PNG, Width: 20, Height: 10, Frames: 1, HasAlpha: true, Orientation: 1}},
		{"png tRNS", pngInsert(t, encodePNG(t, gradient(8, 8)), pchunk{"tRNS", []byte{0, 0, 0, 0, 0, 0}}),
			Info{Format: PNG, Width: 8, Height: 8, Frames: 1, HasAlpha: true, Orientation: 1}},
		{"png eXIf gps+orientation", pngInsert(t, encodePNG(t, gradient(8, 8)), pchunk{"eXIf", buildEXIF(exifOpts{orient: 5, gps: true})}),
			Info{Format: PNG, Width: 8, Height: 8, Frames: 1, HasGPS: true, Orientation: 5}},
		{"apng", mkAPNG(t, 16, 16, 3), Info{Format: PNG, Width: 16, Height: 16, Frames: 3, Orientation: 1}},
		{"gif static", mkGIF(t, 12, 9, 1, false), Info{Format: GIF, Width: 12, Height: 9, Frames: 1, Orientation: 1}},
		{"gif animated", mkGIF(t, 12, 9, 4, false), Info{Format: GIF, Width: 12, Height: 9, Frames: 4, Orientation: 1}},
		{"gif transparent", mkGIF(t, 12, 9, 2, true), Info{Format: GIF, Width: 12, Height: 9, Frames: 2, HasAlpha: true, Orientation: 1}},
		{"webp lossless", webpLossless(33, 21, false), Info{Format: WebP, Width: 33, Height: 21, Frames: 1, Orientation: 1}},
		{"webp lossless alpha", webpLossless(33, 21, true), Info{Format: WebP, Width: 33, Height: 21, Frames: 1, HasAlpha: true, Orientation: 1}},
		{"webp lossy", webpLossyStub(640, 480), Info{Format: WebP, Width: 640, Height: 480, Frames: 1, Orientation: 1}},
		{"webp exif with prefix", webpWithEXIF(10, 10, exifOpts{orient: 6, gps: true}, true),
			Info{Format: WebP, Width: 10, Height: 10, Frames: 1, HasGPS: true, Orientation: 6}},
		{"webp exif bare", webpWithEXIF(10, 10, exifOpts{orient: 2}, false),
			Info{Format: WebP, Width: 10, Height: 10, Frames: 1, Orientation: 2}},
		{"webp animated", webpAnimated(30, 20, 3), Info{Format: WebP, Width: 30, Height: 20, Frames: 3, Orientation: 1}},
		{"webp odd chunk padding", riff(vp8x(0, 5, 5), riffChunk("ICCP", []byte{1, 2, 3}), riffChunk("VP8L", vp8lSized(5, 5))),
			Info{Format: WebP, Width: 5, Height: 5, Frames: 1, Orientation: 1}},
		{"webp trailing pad byte", append(webpLossless(4, 4, false), 0), Info{Format: WebP, Width: 4, Height: 4, Frames: 1, Orientation: 1}},
		{"avif", mkAVIF(avifOpts{w: 800, h: 600}), Info{Format: AVIF, Width: 800, Height: 600, Frames: 1, Orientation: 1}},
		{"avif alpha", mkAVIF(avifOpts{w: 800, h: 600, alpha: true}), Info{Format: AVIF, Width: 800, Height: 600, Frames: 1, HasAlpha: true, Orientation: 1}},
		{"avif sequence brand", mkAVIF(avifOpts{w: 8, h: 8, major: "avis"}), Info{Format: AVIF, Width: 8, Height: 8, Frames: 2, Orientation: 1}},
		{"avif moov", mkAVIF(avifOpts{w: 8, h: 8, moov: true}), Info{Format: AVIF, Width: 8, Height: 8, Frames: 2, Orientation: 1}},
		{"avif compat brand", mkAVIF(avifOpts{w: 8, h: 8, major: "mif1", compat: []string{"miaf", "avif"}}),
			Info{Format: AVIF, Width: 8, Height: 8, Frames: 1, Orientation: 1}},
	}
	for _, c := range cases {
		t.Run(c.name, func(t *testing.T) {
			got, err := inspectBytes(c.data, testLimits)
			if err != nil {
				t.Fatalf("Inspect: %v", err)
			}
			if got != c.want {
				t.Fatalf("got  %+v\nwant %+v", got, c.want)
			}
		})
	}
}

func TestInspectUnsupported(t *testing.T) {
	cases := map[string][]byte{
		"svg":   []byte(`<?xml version="1.0"?><svg xmlns="http://www.w3.org/2000/svg" width="1" height="1"/>`),
		"html":  []byte("<!DOCTYPE html><html><body>hi</body></html>"),
		"php":   []byte("<?php echo 1; ?>"),
		"bmp":   append([]byte("BM"), make([]byte, 60)...),
		"tiff":  append([]byte("II*\x00\x08\x00\x00\x00"), make([]byte, 60)...),
		"heic":  append(ftypBox("heic", "mif1", "heic"), make([]byte, 60)...),
		"ico":   append([]byte{0, 0, 1, 0, 1, 0}, make([]byte, 60)...),
		"empty": nil,
		"text":  []byte("hello world, definitely not an image"),
	}
	for name, data := range cases {
		t.Run(name, func(t *testing.T) {
			_, err := inspectBytes(data, testLimits)
			if !errors.Is(err, ErrUnsupported) {
				t.Fatalf("err = %v; want ErrUnsupported", err)
			}
		})
	}
	if _, err := Inspect(nil, 10, testLimits); !errors.Is(err, ErrUnsupported) {
		t.Fatalf("nil reader: %v", err)
	}
}

func TestInspectTruncated(t *testing.T) {
	cases := []struct {
		name string
		data []byte
		min  int // shortest prefix that still sniffs
	}{
		{"jpeg", jpegWithEXIF(t, 24, 16, exifOpts{orient: 6, gps: true}), 3},
		{"jpeg hand-built", jpegDims(10, 10), 3},
		{"png", encodePNG(t, translucent(12, 12)), 8},
		{"apng", mkAPNG(t, 8, 8, 2), 8},
		{"gif", mkGIF(t, 10, 10, 3, true), 6},
		{"webp", webpLossless(16, 16, true), 12},
		{"webp animated", webpAnimated(8, 8, 2), 12},
	}
	for _, c := range cases {
		t.Run(c.name, func(t *testing.T) {
			if _, err := inspectBytes(c.data, testLimits); err != nil {
				t.Fatalf("full file: %v", err)
			}
			for n := 1; n < len(c.data); n++ {
				_, err := inspectBytes(c.data[:n], testLimits)
				switch {
				case err == nil:
					t.Fatalf("prefix of %d/%d bytes accepted", n, len(c.data))
				case n < c.min:
					if !errors.Is(err, ErrUnsupported) {
						t.Fatalf("prefix %d: %v; want ErrUnsupported", n, err)
					}
				case !errors.Is(err, ErrInvalid):
					t.Fatalf("prefix %d: %v; want ErrInvalid", n, err)
				}
			}
		})
	}
	t.Run("avif", func(t *testing.T) {
		data := mkAVIF(avifOpts{w: 10, h: 10})
		metaEnd := len(data) - (64 + 8)
		for n := 1; n < len(data); n++ {
			if n > metaEnd && n < metaEnd+8 {
				continue // fewer than a box header left: tolerated as trailing bytes
			}
			if _, err := inspectBytes(data[:n], testLimits); err == nil && n != metaEnd {
				t.Fatalf("prefix of %d/%d bytes accepted", n, len(data))
			}
		}
	})
}

func TestInspectInvalid(t *testing.T) {
	badCRC := encodePNG(t, gradient(20, 20))
	badCRC[len(badCRC)-20] ^= 0xFF // inside IDAT data or its CRC
	idatFlip := encodePNG(t, gradient(64, 64))
	idatFlip[60] ^= 0x01
	noIDAT := joinPNG(func() []pchunk {
		cs := splitPNG(t, encodePNG(t, gradient(4, 4)))
		return []pchunk{cs[0], cs[len(cs)-1]}
	}())
	noIEND := encodePNG(t, gradient(4, 4))
	noIEND = noIEND[:len(noIEND)-12]
	ihdrNotFirst := func() []byte {
		cs := splitPNG(t, encodePNG(t, gradient(4, 4)))
		return joinPNG(append([]pchunk{{"tEXt", []byte("a\x00b")}}, cs...))
	}()
	badDepth := func() []byte {
		cs := splitPNG(t, encodePNG(t, gradient(4, 4)))
		cs[0].data[8] = 3
		return joinPNG(cs)
	}()
	apngOutside := func() []byte {
		cs := splitPNG(t, encodePNG(t, gradient(4, 4)))
		return joinPNG(append([]pchunk{cs[0], {"acTL", append(be32b(2), be32b(0)...)}, fcTL(0, 9, 9, 0, 0)}, cs[1:]...))
	}()
	zeroFrames := func() []byte {
		cs := splitPNG(t, encodePNG(t, gradient(4, 4)))
		return joinPNG(append([]pchunk{cs[0], {"acTL", append(be32b(0), be32b(0)...)}}, cs[1:]...))
	}()

	gifNoTrailer := rawGIF(4, 4, 2, false)
	gifZeroScreen := rawGIF(0, 4, 1, true)
	gifNoFrames := rawGIF(4, 4, 0, true)
	gifBadBlock := append(rawGIF(4, 4, 1, false), 0x55, 0x3B)
	gifBadLZW := rawGIF(4, 4, 1, true)
	gifBadLZW[13+6+10] = 0x20

	webpBadRiff := webpLossless(4, 4, false)
	webpBadRiff[4] += 16 // claims more than the file holds
	webpTrailing := append(webpLossless(4, 4, false), 1, 2, 3)
	webpFirstChunk := riff(riffChunk("EXIF", buildEXIF(exifOpts{})), riffChunk("VP8L", vp8lSized(2, 2)))
	webpNoImage := riff(vp8x(0, 4, 4), riffChunk("ICCP", []byte{1, 2}))
	webpBadVP8 := riff(riffChunk("VP8 ", []byte{0x50, 0, 0, 1, 2, 3, 4, 0, 4, 0}))
	webpZeroVP8 := webpLossyStub(0, 5)
	webpMismatch := riff(vp8x(0, 9, 9), riffChunk("VP8L", vp8lSized(4, 4)))
	webpBadVP8L := riff(riffChunk("VP8L", []byte{0x2E, 0, 0, 0, 0}))
	webpAnimNoFrames := riff(vp8x(0x02, 4, 4), riffChunk("ANIM", make([]byte, 6)))
	webpFrameOutside := riff(vp8x(0x02, 4, 4), riffChunk("ANIM", make([]byte, 6)), anmf(2, 0, 4, 4))
	webpChunkOverflow := riff(vp8x(0, 4, 4), append([]byte("VP8L"), 0xFF, 0xFF, 0xFF, 0x7F))

	avifNoIspe := mkAVIF(avifOpts{w: 4, h: 4, noIspe: true})
	avifZero := mkAVIF(avifOpts{w: 0, h: 4})
	avifNoMeta := append(ftypBox("avif"), isobox("mdat", make([]byte, 32))...)
	avifBoxOverflow := append(ftypBox("avif"), append(be32b(0x7FFFFFF0), []byte("meta")...)...)
	avifLargesize := append(ftypBox("avif"), append(be32b(1), append([]byte("free"), 0xFF, 0xFF, 0xFF, 0xFF, 0xFF, 0xFF, 0xFF, 0xFF)...)...)
	avifTinyBox := append(ftypBox("avif"), append(be32b(4), append([]byte("free"), make([]byte, 20)...)...)...)
	avifShortFtyp := append(be32b(12), append([]byte("ftypavif"), make([]byte, 40)...)...)

	jpegZero := jpegDims(0, 10)
	jpegNoSOF := []byte{0xFF, 0xD8, 0xFF, 0xDA, 0, 8, 1, 1, 0, 0, 63, 0, 0x12, 0xFF, 0xD9}
	jpegEOIFirst := []byte{0xFF, 0xD8, 0xFF, 0xD9}
	jpegGarbage := append(mkJPEG(t, 8, 8)[:2:2], 0xFF, 0xE0, 0, 2, 0x11, 0x22)
	jpegBadSeg := []byte{0xFF, 0xD8, 0xFF, 0xE0, 0, 1, 0, 0}
	jpegSegOverflow := []byte{0xFF, 0xD8, 0xFF, 0xE0, 0xFF, 0xFF, 0, 0}
	jpegDupSOF := func() []byte {
		b := jpegDims(8, 8)
		sof := b[2:15]
		return append(append(append([]byte(nil), b[:2]...), sof...), b[2:]...)
	}()

	cases := []struct {
		name string
		data []byte
	}{
		{"png bad crc", badCRC}, {"png flipped IDAT", idatFlip}, {"png no IDAT", noIDAT}, {"png no IEND", noIEND},
		{"png IHDR not first", ihdrNotFirst}, {"png bad depth/type", badDepth},
		{"png zero width", pngSetIHDR(t, encodePNG(t, gradient(4, 4)), 0, 4)},
		{"png zero height", pngSetIHDR(t, encodePNG(t, gradient(4, 4)), 4, 0)},
		{"png apng frame outside canvas", apngOutside}, {"png zero frames", zeroFrames},
		{"gif no trailer", gifNoTrailer}, {"gif zero screen", gifZeroScreen}, {"gif no frames", gifNoFrames},
		{"gif unknown block", gifBadBlock}, {"gif bad lzw size", gifBadLZW},
		{"webp riff overstates size", webpBadRiff}, {"webp trailing bytes", webpTrailing},
		{"webp bad first chunk", webpFirstChunk}, {"webp no image", webpNoImage}, {"webp bad VP8", webpBadVP8},
		{"webp zero VP8 width", webpZeroVP8}, {"webp canvas mismatch", webpMismatch}, {"webp bad VP8L", webpBadVP8L},
		{"webp animation without frames", webpAnimNoFrames}, {"webp frame outside canvas", webpFrameOutside},
		{"webp chunk overflows riff", webpChunkOverflow},
		{"avif no ispe", avifNoIspe}, {"avif zero width", avifZero}, {"avif no meta", avifNoMeta},
		{"avif box overflow", avifBoxOverflow}, {"avif largesize overflow", avifLargesize},
		{"avif box smaller than header", avifTinyBox}, {"avif short ftyp", avifShortFtyp},
		{"jpeg zero width", jpegZero}, {"jpeg SOS before SOF", jpegNoSOF}, {"jpeg EOI first", jpegEOIFirst},
		{"jpeg garbage between segments", jpegGarbage}, {"jpeg zero segment length", jpegBadSeg},
		{"jpeg segment overflows", jpegSegOverflow}, {"jpeg duplicate SOF", jpegDupSOF},
	}
	for _, c := range cases {
		t.Run(c.name, func(t *testing.T) {
			_, err := inspectBytes(c.data, testLimits)
			if !errors.Is(err, ErrInvalid) {
				t.Fatalf("err = %v; want ErrInvalid", err)
			}
		})
	}
}

func TestInspectTooManyPixels(t *testing.T) {
	lim := Limits{MaxPixels: 1_000_000}
	cases := []struct {
		name string
		data []byte
	}{
		{"jpeg", jpegDims(65535, 65535)},
		{"jpeg just over", jpegDims(1001, 1000)},
		{"png", pngSetIHDR(t, encodePNG(t, gradient(4, 4)), 100000, 100000)},
		{"png max dims", pngSetIHDR(t, encodePNG(t, gradient(4, 4)), 0x7fffffff, 0x7fffffff)},
		{"gif screen", rawGIF(65535, 65535, 1, true)},
		{"gif frame", func() []byte { b := rawGIF(10, 10, 1, true); copy(b[24:28], []byte{0xFF, 0xFF, 0xFF, 0xFF}); return b }()},
		{"webp lossless", webpLossless(16384, 16384, false)},
		{"webp VP8X canvas", riff(vp8x(0, 1<<24, 1<<24))},
		{"webp animated frame", webpAnimated(1500, 1500, 1)},
		{"avif", mkAVIF(avifOpts{w: 0xFFFFFFFF, h: 0xFFFFFFFF})},
		{"avif one dimension huge", mkAVIF(avifOpts{w: 0xFFFFFFFF, h: 1000})},
	}
	for _, c := range cases {
		t.Run(c.name, func(t *testing.T) {
			var before, after runtime.MemStats
			runtime.ReadMemStats(&before)
			_, err := inspectBytes(c.data, lim)
			runtime.ReadMemStats(&after)
			if !errors.Is(err, ErrTooManyPixels) {
				t.Fatalf("err = %v; want ErrTooManyPixels", err)
			}
			if d := after.TotalAlloc - before.TotalAlloc; d > 512<<10 {
				t.Fatalf("allocated %d bytes for a rejected image", d)
			}
		})
	}
	// exactly at the limit is fine
	if _, err := inspectBytes(jpegDims(1000, 1000), lim); err != nil {
		t.Fatalf("at limit: %v", err)
	}
	// default limit applies when MaxPixels is zero
	if _, err := inspectBytes(jpegDims(65535, 65535), Limits{}); !errors.Is(err, ErrTooManyPixels) {
		t.Fatalf("default limit: %v", err)
	}
}

func TestInspectTooManyFrames(t *testing.T) {
	cases := []struct {
		name string
		data []byte
		lim  Limits
		ok   bool
	}{
		{"gif over", mkGIF(t, 4, 4, 3, false), Limits{MaxPixels: 1000, MaxFrames: 2}, false},
		{"gif at", mkGIF(t, 4, 4, 2, false), Limits{MaxPixels: 1000, MaxFrames: 2}, true},
		{"gif default limit exceeded", rawGIF(1, 1, 10001, true), Limits{}, false},
		{"gif default limit at", rawGIF(1, 1, 10000, true), Limits{}, true},
		{"apng declared", mkAPNG(t, 4, 4, 5), Limits{MaxFrames: 4}, false},
		{"apng huge declared", func() []byte {
			cs := splitPNG(t, encodePNG(t, gradient(4, 4)))
			return joinPNG(append([]pchunk{cs[0], {"acTL", append(be32b(0x7FFFFFFF), be32b(0)...)}}, cs[1:]...))
		}(), Limits{}, false},
		{"webp over", webpAnimated(4, 4, 3), Limits{MaxFrames: 2}, false},
		{"webp at", webpAnimated(4, 4, 2), Limits{MaxFrames: 2}, true},
	}
	for _, c := range cases {
		t.Run(c.name, func(t *testing.T) {
			_, err := inspectBytes(c.data, c.lim)
			if c.ok && err != nil {
				t.Fatal(err)
			}
			if !c.ok && !errors.Is(err, ErrTooManyPixels) {
				t.Fatalf("err = %v; want ErrTooManyPixels", err)
			}
		})
	}
}

func TestInspectTrailingData(t *testing.T) {
	mp4 := append(ftypBox("isom", "mp42"), isobox("mdat", bytes.Repeat([]byte{0xA5}, 2000))...)
	html := []byte("\r\n<HTML><body>hi</body></html>")
	bases := map[string][]byte{
		"jpeg": mkJPEG(t, 16, 16),
		"png":  encodePNG(t, gradient(8, 8)),
		"gif":  mkGIF(t, 8, 8, 2, false),
		"avif": mkAVIF(avifOpts{w: 8, h: 8}),
	}
	bad := [][]byte{
		html, []byte("<script>alert(1)</script>"), []byte("<?php system($_GET['c']); ?>"),
		[]byte("<!DocType html>"), []byte("<SVG xmlns='x'/>"), []byte("<iframe src=x>"),
		append(bytes.Repeat([]byte{0}, 4000), []byte("<script")...),
	}
	for name, base := range bases {
		t.Run(name+"/mp4 accepted", func(t *testing.T) {
			if _, err := inspectBytes(append(append([]byte(nil), base...), mp4...), testLimits); err != nil {
				t.Fatal(err)
			}
		})
		t.Run(name+"/binary junk accepted", func(t *testing.T) {
			if name == "avif" {
				t.Skip("non-box junk after an ISO-BMFF file is indistinguishable from a truncated box")
			}
			if _, err := inspectBytes(append(append([]byte(nil), base...), bytes.Repeat([]byte{0xAB}, 5000)...), testLimits); err != nil {
				t.Fatal(err)
			}
		})
		for i, tail := range bad {
			t.Run(name+"/markup "+string(rune('a'+i)), func(t *testing.T) {
				_, err := inspectBytes(append(append([]byte(nil), base...), tail...), testLimits)
				// AVIF: a '<' byte pair is not a valid box header so it may be rejected as a broken box instead.
				if !errors.Is(err, ErrInvalid) {
					t.Fatalf("err = %v; want ErrInvalid", err)
				}
			})
		}
		t.Run(name+"/markup past 4KiB not scanned", func(t *testing.T) {
			tail := append(bytes.Repeat([]byte{0}, 5000), []byte("<script>")...)
			if name == "avif" {
				t.Skip("padding would be parsed as a box")
			}
			if _, err := inspectBytes(append(append([]byte(nil), base...), tail...), testLimits); err != nil {
				t.Fatal(err)
			}
		})
	}
	t.Run("webp trailing data rejected", func(t *testing.T) {
		_, err := inspectBytes(append(webpLossless(4, 4, false), []byte("<html>")...), testLimits)
		if !errors.Is(err, ErrInvalid) {
			t.Fatal(err)
		}
	})
	t.Run("polyglot in jpeg comment is not trailing data", func(t *testing.T) {
		j := jpegWithSegment(mkJPEG(t, 8, 8), []byte{0xFF, 0xFE, 0, 10, '<', 's', 'c', 'r', 'i', 'p', 't', '>'})
		if _, err := inspectBytes(j, testLimits); err != nil {
			t.Fatal(err)
		}
	})
}

func TestEXIF(t *testing.T) {
	t.Run("loop terminates", func(t *testing.T) {
		done := make(chan exifInfo, 1)
		go func() { done <- parseEXIF(exifLoop()) }()
		select {
		case got := <-done:
			if got.hasGPS || got.orientation != 1 {
				t.Fatalf("got %+v", got)
			}
		case <-time.After(2 * time.Second):
			t.Fatal("parseEXIF hangs on an IFD loop")
		}
		j := jpegWithSegment(mkJPEG(t, 8, 8), app1(append([]byte("Exif\x00\x00"), exifLoop()...)))
		if _, err := inspectBytes(j, testLimits); err != nil {
			t.Fatal(err)
		}
	})
	t.Run("malformed never panics or rejects", func(t *testing.T) {
		good := buildEXIF(exifOpts{orient: 6, gps: true})
		for n := 0; n <= len(good); n++ {
			parseEXIF(good[:n])
		}
		for i := range good {
			m := append([]byte(nil), good...)
			m[i] ^= 0xFF
			parseEXIF(m)
			m[i] = 0xFF
			parseEXIF(m)
		}
		garbage := []byte("Exif\x00\x00II*\x00\xFF\xFF\xFF\xFF zzzzzzzzzzzz")
		j := jpegWithSegment(mkJPEG(t, 8, 8), app1(garbage))
		info, err := inspectBytes(j, testLimits)
		if err != nil || info.HasGPS || info.Orientation != 1 {
			t.Fatalf("info=%+v err=%v", info, err)
		}
	})
	t.Run("too many entries ignored", func(t *testing.T) {
		b := buildEXIF(exifOpts{orient: 6})
		b[8], b[9] = 0xFF, 0xFF // entry count 65535
		if got := parseEXIF(b); got.orientation != 1 {
			t.Fatalf("got %+v", got)
		}
	})
	t.Run("oversized png eXIf skipped", func(t *testing.T) {
		big := append(buildEXIF(exifOpts{orient: 6, gps: true}), make([]byte, 2<<20)...)
		info, err := inspectBytes(pngInsert(t, encodePNG(t, gradient(4, 4)), pchunk{"eXIf", big}), testLimits)
		if err != nil || info.HasGPS || info.Orientation != 1 {
			t.Fatalf("info=%+v err=%v", info, err)
		}
	})
	t.Run("gps tag with zero count", func(t *testing.T) {
		b := buildEXIF(exifOpts{gps: true})
		// GPS IFD starts at 8+2+12+4 = 26; entries: 1 ref, 2 lat (count at +4), ...
		b[26+2+12+4] = 0 // latitude count := 0
		b[26+2+36+4] = 0 // longitude count := 0
		if got := parseEXIF(b); got.hasGPS {
			t.Fatal("GPS reported with zero-count coordinates")
		}
	})
}

func TestInspectReadErrors(t *testing.T) {
	data := mkJPEG(t, 16, 16)
	_, err := Inspect(failingReaderAt{}, int64(len(data)), testLimits)
	if err == nil || errors.Is(err, ErrUnsupported) || errors.Is(err, ErrInvalid) {
		t.Fatalf("I/O error should surface unchanged, got %v", err)
	}
	// reader shorter than the declared size
	_, err = Inspect(bytes.NewReader(data[:len(data)/2]), int64(len(data)), testLimits)
	if !errors.Is(err, ErrInvalid) {
		t.Fatalf("short reader: %v", err)
	}
	if !strings.Contains(ErrInvalid.Error(), "invalid") {
		t.Fatal("sanity")
	}
}

type failingReaderAt struct{}

func (failingReaderAt) ReadAt(p []byte, off int64) (int, error) { return 0, errors.New("disk on fire") }
