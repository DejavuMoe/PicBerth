package imaging

import (
	"bytes"
	"encoding/binary"
	"hash/crc32"
	"image"
	"image/color"
	"image/gif"
	"image/jpeg"
	"image/png"
	"testing"
)

// ---- raster helpers ----

func gradient(w, h int) *image.RGBA {
	img := image.NewRGBA(image.Rect(0, 0, w, h))
	for y := 0; y < h; y++ {
		for x := 0; x < w; x++ {
			img.SetRGBA(x, y, color.RGBA{uint8(x * 255 / w), uint8(y * 255 / h), uint8((x + y) & 0xFF), 0xFF})
		}
	}
	return img
}

// halves is left half red, right half blue (opaque).
func halves(w, h int) *image.RGBA {
	img := image.NewRGBA(image.Rect(0, 0, w, h))
	for y := 0; y < h; y++ {
		for x := 0; x < w; x++ {
			if x < w/2 {
				img.SetRGBA(x, y, color.RGBA{255, 0, 0, 255})
			} else {
				img.SetRGBA(x, y, color.RGBA{0, 0, 255, 255})
			}
		}
	}
	return img
}

func translucent(w, h int) *image.NRGBA {
	img := image.NewNRGBA(image.Rect(0, 0, w, h))
	for y := 0; y < h; y++ {
		for x := 0; x < w; x++ {
			img.SetNRGBA(x, y, color.NRGBA{200, 30, 30, uint8(255 * x / w)})
		}
	}
	return img
}

func encodeJPEG(tb testing.TB, img image.Image) []byte {
	tb.Helper()
	var b bytes.Buffer
	if err := jpeg.Encode(&b, img, &jpeg.Options{Quality: 80}); err != nil {
		tb.Fatal(err)
	}
	return b.Bytes()
}

func encodePNG(tb testing.TB, img image.Image) []byte {
	tb.Helper()
	var b bytes.Buffer
	if err := png.Encode(&b, img); err != nil {
		tb.Fatal(err)
	}
	return b.Bytes()
}

// ---- EXIF ----

type exifOpts struct {
	orient     int // 0 = no orientation tag
	gps        bool
	gpsRefOnly bool // GPS IFD with only the N/E reference tags
	bigEndian  bool
}

func buildEXIF(o exifOpts) []byte {
	var bo binary.ByteOrder = binary.LittleEndian
	hdr := "II"
	if o.bigEndian {
		bo, hdr = binary.BigEndian, "MM"
	}
	hasGPS := o.gps || o.gpsRefOnly
	n := 0
	if o.orient > 0 {
		n++
	}
	if hasGPS {
		n++
	}
	b := make([]byte, 8)
	copy(b, hdr)
	bo.PutUint16(b[2:], 42)
	bo.PutUint32(b[4:], 8)
	ent := func(tag, typ uint16, count, val uint32) []byte {
		e := make([]byte, 12)
		bo.PutUint16(e, tag)
		bo.PutUint16(e[2:], typ)
		bo.PutUint32(e[4:], count)
		if typ == 3 && count == 1 {
			bo.PutUint16(e[8:], uint16(val))
		} else {
			bo.PutUint32(e[8:], val)
		}
		return e
	}
	gpsOff := uint32(8 + 2 + 12*n + 4)
	ifd := make([]byte, 2)
	bo.PutUint16(ifd, uint16(n))
	if o.orient > 0 {
		ifd = append(ifd, ent(0x0112, 3, 1, uint32(o.orient))...)
	}
	if hasGPS {
		ifd = append(ifd, ent(0x8825, 4, 1, gpsOff)...)
	}
	b = append(b, ifd...)
	b = append(b, 0, 0, 0, 0)
	if hasGPS {
		ref := func(tag uint16, c byte) []byte {
			e := ent(tag, 2, 2, 0)
			e[8] = c
			return e
		}
		g := make([]byte, 2)
		entries := 2
		if o.gps {
			entries = 4
		}
		bo.PutUint16(g, uint16(entries))
		dataOff := gpsOff + 2 + uint32(entries)*12 + 4
		g = append(g, ref(1, 'N')...)
		if o.gps {
			g = append(g, ent(2, 5, 3, dataOff)...)
		}
		g = append(g, ref(3, 'E')...)
		if o.gps {
			g = append(g, ent(4, 5, 3, dataOff+24)...)
		}
		g = append(g, 0, 0, 0, 0)
		if o.gps {
			g = append(g, make([]byte, 48)...)
		}
		b = append(b, g...)
	}
	return b
}

// exifLoop builds a TIFF whose next-IFD pointer and GPS pointer both point back at IFD0.
func exifLoop() []byte {
	b := make([]byte, 8)
	copy(b, "II")
	binary.LittleEndian.PutUint16(b[2:], 42)
	binary.LittleEndian.PutUint32(b[4:], 8)
	ifd := make([]byte, 2+12+4)
	binary.LittleEndian.PutUint16(ifd, 1)
	binary.LittleEndian.PutUint16(ifd[2:], 0x8825)
	binary.LittleEndian.PutUint16(ifd[4:], 4)
	binary.LittleEndian.PutUint32(ifd[6:], 1)
	binary.LittleEndian.PutUint32(ifd[10:], 8) // GPS pointer -> IFD0
	binary.LittleEndian.PutUint32(ifd[14:], 8) // next IFD -> IFD0
	return append(b, ifd...)
}

// ---- JPEG ----

func mkJPEG(tb testing.TB, w, h int) []byte { return encodeJPEG(tb, gradient(w, h)) }

func app1(payload []byte) []byte {
	seg := []byte{0xFF, 0xE1, 0, 0}
	binary.BigEndian.PutUint16(seg[2:], uint16(len(payload)+2))
	return append(seg, payload...)
}

func jpegWithSegment(j, seg []byte) []byte {
	out := append([]byte(nil), j[:2]...)
	out = append(out, seg...)
	return append(out, j[2:]...)
}

func jpegWithEXIF(tb testing.TB, w, h int, o exifOpts) []byte {
	return jpegWithSegment(mkJPEG(tb, w, h), app1(append([]byte("Exif\x00\x00"), buildEXIF(o)...)))
}

func jpegDims(w, h int) []byte {
	// hand-built minimal JPEG: SOI, SOF0, SOS, entropy data with stuffing/RST/fill bytes, EOI
	b := []byte{0xFF, 0xD8}
	sof := []byte{0xFF, 0xC0, 0, 11, 8, byte(h >> 8), byte(h), byte(w >> 8), byte(w), 1, 1, 0x11, 0}
	b = append(b, sof...)
	b = append(b, 0xFF, 0xDA, 0, 8, 1, 1, 0, 0, 63, 0)
	b = append(b, 0x12, 0xFF, 0x00, 0x34, 0xFF, 0xD0, 0x56, 0xFF, 0xFF, 0xD1, 0x78)
	// a DHT-like segment inside the stream whose payload contains FF D9 must not end the image
	b = append(b, 0xFF, 0xC4, 0, 6, 0xFF, 0xD9, 0x01, 0x02)
	b = append(b, 0x9A)
	return append(b, 0xFF, 0xD9)
}

// ---- PNG ----

type pchunk struct {
	typ  string
	data []byte
}

func splitPNG(tb testing.TB, b []byte) []pchunk {
	tb.Helper()
	var cs []pchunk
	for off := 8; off < len(b); {
		n := int(binary.BigEndian.Uint32(b[off:]))
		cs = append(cs, pchunk{string(b[off+4 : off+8]), append([]byte(nil), b[off+8:off+8+n]...)})
		off += 12 + n
	}
	return cs
}

func joinPNG(cs []pchunk) []byte {
	out := append([]byte(nil), pngSignature...)
	for _, c := range cs {
		hd := make([]byte, 8)
		binary.BigEndian.PutUint32(hd, uint32(len(c.data)))
		copy(hd[4:], c.typ)
		out = append(out, hd...)
		out = append(out, c.data...)
		sum := crc32.Update(crc32.ChecksumIEEE([]byte(c.typ)), crc32.IEEETable, c.data)
		out = binary.BigEndian.AppendUint32(out, sum)
	}
	return out
}

// pngInsert returns the PNG with extra chunks inserted directly after IHDR.
func pngInsert(tb testing.TB, b []byte, extra ...pchunk) []byte {
	cs := splitPNG(tb, b)
	out := append([]pchunk{cs[0]}, extra...)
	return joinPNG(append(out, cs[1:]...))
}

func pngSetIHDR(tb testing.TB, b []byte, w, h uint32) []byte {
	cs := splitPNG(tb, b)
	binary.BigEndian.PutUint32(cs[0].data, w)
	binary.BigEndian.PutUint32(cs[0].data[4:], h)
	return joinPNG(cs)
}

func be32b(v uint32) []byte { return binary.BigEndian.AppendUint32(nil, v) }

func fcTL(seq, w, h, x, y uint32) pchunk {
	d := append(be32b(seq), be32b(w)...)
	d = append(d, be32b(h)...)
	d = append(d, be32b(x)...)
	d = append(d, be32b(y)...)
	return pchunk{"fcTL", append(d, 0, 1, 0, 10, 0, 0)}
}

// mkAPNG builds an APNG with n frames over a w x h opaque PNG (frame 0 uses IDAT).
func mkAPNG(tb testing.TB, w, h, n int) []byte {
	cs := splitPNG(tb, encodePNG(tb, gradient(w, h)))
	out := []pchunk{cs[0], {"acTL", append(be32b(uint32(n)), be32b(0)...)}, fcTL(0, uint32(w), uint32(h), 0, 0)}
	out = append(out, cs[1:len(cs)-1]...)
	seq := uint32(1)
	for i := 1; i < n; i++ {
		out = append(out, fcTL(seq, uint32(w), uint32(h), 0, 0))
		out = append(out, pchunk{"fdAT", append(be32b(seq+1), cs[1].data[:4]...)})
		seq += 2
	}
	return joinPNG(append(out, cs[len(cs)-1]))
}

// ---- GIF ----

func mkGIF(tb testing.TB, w, h, frames int, transparent bool) []byte {
	pal := color.Palette{color.RGBA{255, 0, 0, 255}, color.RGBA{0, 0, 255, 255}}
	if transparent {
		pal = color.Palette{color.RGBA{0, 0, 0, 0}, color.RGBA{0, 0, 255, 255}}
	}
	g := &gif.GIF{Config: image.Config{Width: w, Height: h, ColorModel: pal}}
	for i := 0; i < frames; i++ {
		f := image.NewPaletted(image.Rect(0, 0, w, h), pal)
		for j := range f.Pix {
			f.Pix[j] = uint8((j + i) & 1)
		}
		g.Image = append(g.Image, f)
		g.Delay = append(g.Delay, 5)
	}
	var b bytes.Buffer
	if err := gif.EncodeAll(&b, g); err != nil {
		tb.Fatal(err)
	}
	return b.Bytes()
}

// rawGIF is a minimal hand-written GIF with n 1x1 frames inside a w x h screen.
func rawGIF(w, h, n int, trailer bool) []byte {
	b := []byte("GIF89a")
	b = binary.LittleEndian.AppendUint16(b, uint16(w))
	b = binary.LittleEndian.AppendUint16(b, uint16(h))
	b = append(b, 0x80, 0, 0, 0, 0, 0, 255, 255, 255)
	for i := 0; i < n; i++ {
		b = append(b, 0x2C, 0, 0, 0, 0, 1, 0, 1, 0, 0, 0x02, 0x02, 0x44, 0x01, 0x00)
	}
	if trailer {
		b = append(b, 0x3B)
	}
	return b
}

// ---- WebP ----

type bitWriter struct {
	buf  []byte
	bits uint
}

func (w *bitWriter) put(v uint32, n uint) {
	for i := uint(0); i < n; i++ {
		if w.bits%8 == 0 {
			w.buf = append(w.buf, 0)
		}
		if v>>i&1 != 0 {
			w.buf[len(w.buf)-1] |= 1 << (w.bits % 8)
		}
		w.bits++
	}
}

// vp8l builds a valid lossless bitstream of size w x h in which every pixel has the same ARGB value
// (each of the four colour channels uses a single-symbol prefix code, so no pixel data is needed).
func vp8l(w, h int, r, g, b, a byte) []byte {
	bw := &bitWriter{}
	bw.put(0x2F, 8)
	bw.put(uint32(w-1), 14)
	bw.put(uint32(h-1), 14)
	if a != 255 {
		bw.put(1, 1)
	} else {
		bw.put(0, 1)
	}
	bw.put(0, 3)
	bw.put(0, 1) // no transform
	bw.put(0, 1) // no colour cache
	bw.put(0, 1) // no meta prefix codes
	for _, sym := range []byte{g, r, b, a, 0} {
		bw.put(1, 1) // simple code
		bw.put(0, 1) // one symbol
		bw.put(1, 1) // 8-bit symbol
		bw.put(uint32(sym), 8)
	}
	return bw.buf
}

func vp8lSized(w, h int) []byte {
	b := vp8l(1, 1, 0, 0, 0, 255)
	// patch the 14-bit dimensions without touching the rest of the stream
	v := binary.LittleEndian.Uint32(b[1:5])
	v = v&^0x0FFFFFFF | uint32(w-1)&0x3FFF | uint32(h-1)&0x3FFF<<14
	binary.LittleEndian.PutUint32(b[1:5], v)
	return b
}

func riffChunk(tag string, data []byte) []byte {
	b := append([]byte(tag), 0, 0, 0, 0)
	binary.LittleEndian.PutUint32(b[4:], uint32(len(data)))
	b = append(b, data...)
	if len(data)%2 == 1 {
		b = append(b, 0)
	}
	return b
}

func riff(chunks ...[]byte) []byte {
	body := []byte("WEBP")
	for _, c := range chunks {
		body = append(body, c...)
	}
	b := append([]byte("RIFF"), 0, 0, 0, 0)
	binary.LittleEndian.PutUint32(b[4:], uint32(len(body)))
	return append(b, body...)
}

func le24b(v uint32) []byte { return []byte{byte(v), byte(v >> 8), byte(v >> 16)} }

func vp8x(flags byte, w, h uint32) []byte {
	d := append([]byte{flags, 0, 0, 0}, le24b(w-1)...)
	return riffChunk("VP8X", append(d, le24b(h-1)...))
}

func webpLossless(w, h int, alpha bool) []byte {
	a := byte(255)
	if alpha {
		a = 128
	}
	return riff(riffChunk("VP8L", vp8l(w, h, 200, 100, 50, a)))
}

// webpLossyStub has a valid VP8 frame header but no usable bitstream (enough for Inspect only).
func webpLossyStub(w, h int) []byte {
	d := []byte{0x50, 0, 0, 0x9D, 0x01, 0x2A, byte(w), byte(w >> 8), byte(h), byte(h >> 8), 0, 0}
	return riff(riffChunk("VP8 ", d))
}

func anmf(x, y, w, h uint32) []byte {
	d := append(le24b(x/2), le24b(y/2)...)
	d = append(d, le24b(w-1)...)
	d = append(d, le24b(h-1)...)
	d = append(d, le24b(100)...)
	d = append(d, 0)
	d = append(d, riffChunk("VP8L", vp8lSized(int(w), int(h)))...)
	return riffChunk("ANMF", d)
}

func webpAnimated(w, h uint32, frames int) []byte {
	cs := [][]byte{vp8x(0x02, w, h), riffChunk("ANIM", make([]byte, 6))}
	for i := 0; i < frames; i++ {
		cs = append(cs, anmf(0, 0, w, h))
	}
	return riff(cs...)
}

func webpWithEXIF(w, h int, o exifOpts, prefix bool) []byte {
	ex := buildEXIF(o)
	if prefix {
		ex = append([]byte("Exif\x00\x00"), ex...)
	}
	return riff(vp8x(0x08, uint32(w), uint32(h)), riffChunk("VP8L", vp8lSized(w, h)), riffChunk("EXIF", ex))
}

// ---- AVIF / ISO-BMFF ----

func isobox(typ string, parts ...[]byte) []byte {
	var body []byte
	for _, p := range parts {
		body = append(body, p...)
	}
	return append(be32b(uint32(len(body)+8)), append([]byte(typ), body...)...)
}

func fullbox(typ string, parts ...[]byte) []byte {
	return isobox(typ, append([][]byte{{0, 0, 0, 0}}, parts...)...)
}

func ftypBox(major string, compat ...string) []byte {
	parts := [][]byte{[]byte(major), {0, 0, 0, 0}}
	for _, c := range compat {
		parts = append(parts, []byte(c))
	}
	return isobox("ftyp", parts...)
}

type avifOpts struct {
	w, h   uint32
	noIspe bool
	alpha  bool
	moov   bool
	major  string
	compat []string
}

func mkAVIF(o avifOpts) []byte {
	if o.major == "" {
		o.major = "avif"
	}
	var props [][]byte
	if !o.noIspe {
		props = append(props, fullbox("ispe", be32b(o.w), be32b(o.h)))
	}
	props = append(props, isobox("pixi", []byte{0, 0, 0, 0, 3, 8, 8, 8}))
	if o.alpha {
		props = append(props, fullbox("auxC", []byte("urn:mpeg:mpegB:cicp:systems:auxiliary:alpha\x00")))
	}
	meta := fullbox("meta",
		fullbox("hdlr", []byte{0, 0, 0, 0}, []byte("pict"), make([]byte, 13)),
		isobox("iprp", isobox("ipco", props...)))
	out := append(ftypBox(o.major, o.compat...), meta...)
	if o.moov {
		out = append(out, isobox("moov", make([]byte, 16))...)
	}
	return append(out, isobox("mdat", make([]byte, 64))...)
}
