package imaging

import (
	"bytes"
	"testing"
	"time"
)

// Worst-case shapes for the walkers: huge numbers of tiny elements. Each must finish promptly.
func TestInspectPathologicalInputs(t *testing.T) {
	const n = 16 << 20
	hdr := func(b []byte, rest ...byte) []byte { return append(append([]byte(nil), b...), rest...) }
	repeat := func(unit []byte, total int) []byte { return bytes.Repeat(unit, total/len(unit)) }

	jpegHead := jpegDims(8, 8)[:2+13+10] // SOI, SOF0, SOS header
	cases := map[string][]byte{
		"jpeg all FF":         append(hdr(jpegHead), repeat([]byte{0xFF}, n)...),
		"jpeg FF00 stuffing":  append(hdr(jpegHead), repeat([]byte{0xFF, 0x00}, n)...),
		"jpeg RST markers":    append(hdr(jpegHead), repeat([]byte{0xFF, 0xD0}, n)...),
		"jpeg empty segments": append(hdr(jpegHead), repeat([]byte{0xFF, 0xFE, 0x00, 0x02}, n)...),
		"jpeg no markers":     append(hdr(jpegHead), make([]byte, n)...),
		"gif empty extension": append(hdr(rawGIF(4, 4, 0, false)), repeat([]byte{0x21, 0xFE, 0x00}, n)...),
		"gif one-byte blocks": append(hdr(rawGIF(4, 4, 1, false), 0x21, 0xFE), repeat([]byte{0x01, 0x00}, n)...),
		"png tiny chunks": func() []byte {
			cs := splitPNG(t, encodePNG(t, gradient(2, 2)))
			out := []pchunk{cs[0]}
			for i := 0; i < n/12; i++ {
				out = append(out, pchunk{"tEXt", nil})
			}
			return joinPNG(append(out, cs[1:]...))
		}(),
		"webp tiny chunks": riff(vp8x(0, 4, 4), repeat(riffChunk("JUNK", nil), n), riffChunk("VP8L", vp8lSized(4, 4))),
		"avif tiny boxes":  append(mkAVIF(avifOpts{w: 4, h: 4}), repeat(isobox("free"), n)...),
	}
	for name, data := range cases {
		t.Run(name, func(t *testing.T) {
			start := time.Now()
			_, err := inspectBytes(data, Limits{MaxPixels: 1 << 20})
			t.Logf("%d MiB in %v: %v", len(data)>>20, time.Since(start), err)
			if d := time.Since(start); d > 5*time.Second {
				t.Fatalf("took %v", d)
			}
		})
	}
}
