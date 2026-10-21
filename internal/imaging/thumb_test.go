package imaging

import (
	"bytes"
	"errors"
	"image"
	"image/color"
	"image/gif"
	"testing"
)

func thumbOf(t *testing.T, data []byte, maxDim int, maxPixels int64) ([]byte, string, error) {
	t.Helper()
	info, err := inspectBytes(data, testLimits)
	if err != nil {
		t.Fatalf("Inspect: %v", err)
	}
	return Thumbnail(bytes.NewReader(data), int64(len(data)), info, maxDim, maxPixels)
}

func decodeOut(t *testing.T, out []byte, ext string) image.Image {
	t.Helper()
	img, format, err := image.Decode(bytes.NewReader(out))
	if err != nil {
		t.Fatalf("output does not decode: %v", err)
	}
	want := map[string]string{"jpg": "jpeg", "png": "png"}[ext]
	if format != want {
		t.Fatalf("ext %q but encoded as %s", ext, format)
	}
	return img
}

func TestThumbnailFormats(t *testing.T) {
	cases := []struct {
		name         string
		data         []byte
		maxDim       int
		wantExt      string
		wantW, wantH int
	}{
		{"jpeg downscale", mkJPEG(t, 200, 100), 50, "jpg", 50, 25},
		{"jpeg portrait", mkJPEG(t, 100, 300), 60, "jpg", 20, 60},
		{"jpeg no upscale", mkJPEG(t, 20, 10), 100, "jpg", 20, 10},
		{"jpeg exact size", mkJPEG(t, 64, 64), 64, "jpg", 64, 64},
		{"jpeg heavy reduction", mkJPEG(t, 1200, 800), 100, "jpg", 100, 67},
		{"jpeg to one pixel", mkJPEG(t, 300, 10), 1, "jpg", 1, 1},
		{"png opaque", encodePNG(t, gradient(120, 80)), 60, "jpg", 60, 40},
		{"png alpha", encodePNG(t, translucent(120, 80)), 60, "png", 60, 40},
		{"png no upscale alpha", encodePNG(t, translucent(10, 8)), 60, "png", 10, 8},
		{"apng first frame", mkAPNG(t, 40, 40, 3), 20, "jpg", 20, 20},
		{"gif opaque", mkGIF(t, 40, 20, 3, false), 20, "jpg", 20, 10},
		{"gif transparent", mkGIF(t, 40, 20, 3, true), 20, "png", 20, 10},
		{"webp lossless opaque", webpLossless(400, 200, false), 100, "jpg", 100, 50},
		{"webp lossless alpha", webpLossless(400, 200, true), 100, "png", 100, 50},
		{"webp with exif orientation 6", webpWithEXIF(80, 40, exifOpts{orient: 6}, true), 100, "jpg", 40, 80},
		{"jpeg orientation 6", jpegWithEXIF(t, 200, 100, exifOpts{orient: 6}), 50, "jpg", 25, 50},
		{"jpeg orientation 5 no scaling", jpegWithEXIF(t, 30, 10, exifOpts{orient: 5}), 100, "jpg", 10, 30},
		{"jpeg orientation 3", jpegWithEXIF(t, 30, 10, exifOpts{orient: 3}), 100, "jpg", 30, 10},
	}
	for _, c := range cases {
		t.Run(c.name, func(t *testing.T) {
			out, ext, err := thumbOf(t, c.data, c.maxDim, 1<<24)
			if err != nil {
				t.Fatal(err)
			}
			if ext != c.wantExt {
				t.Fatalf("ext = %q; want %q", ext, c.wantExt)
			}
			b := decodeOut(t, out, ext).Bounds()
			if b.Dx() != c.wantW || b.Dy() != c.wantH {
				t.Fatalf("size = %dx%d; want %dx%d", b.Dx(), b.Dy(), c.wantW, c.wantH)
			}
		})
	}
}

func TestThumbnailAlphaPreserved(t *testing.T) {
	out, ext, err := thumbOf(t, encodePNG(t, translucent(100, 20)), 50, 1<<20)
	if err != nil || ext != "png" {
		t.Fatalf("ext=%q err=%v", ext, err)
	}
	img := decodeOut(t, out, ext)
	_, _, _, aLeft := img.At(1, 5).RGBA()
	_, _, _, aRight := img.At(48, 5).RGBA()
	if aLeft > 0x3000 || aRight < 0xC000 {
		t.Fatalf("alpha ramp lost: left=%x right=%x", aLeft, aRight)
	}
}

func TestThumbnailOrientationPixels(t *testing.T) {
	// 64x32: left half red, right half blue. Display-correct results per EXIF orientation:
	//  6 (rotate 90 CW):  red on top, blue below      8 (rotate 90 CCW): blue on top, red below
	//  3 (rotate 180):    blue on the left, red right  2 (mirror):        blue left, red right
	//  1:                 red left, blue right
	type probe struct{ x, y int }
	cases := []struct {
		orient    int
		w, h      int
		red, blue probe
	}{
		{1, 64, 32, probe{8, 16}, probe{56, 16}},
		{2, 64, 32, probe{56, 16}, probe{8, 16}},
		{3, 64, 32, probe{56, 16}, probe{8, 16}},
		{6, 32, 64, probe{16, 8}, probe{16, 56}},
		{8, 32, 64, probe{16, 56}, probe{16, 8}},
		{5, 32, 64, probe{16, 8}, probe{16, 56}},
		{7, 32, 64, probe{16, 56}, probe{16, 8}},
	}
	for _, c := range cases {
		j := jpegWithSegment(encodeJPEG(t, halves(64, 32)), app1(append([]byte("Exif\x00\x00"), buildEXIF(exifOpts{orient: c.orient})...)))
		out, ext, err := thumbOf(t, j, 64, 1<<20)
		if err != nil {
			t.Fatal(err)
		}
		img := decodeOut(t, out, ext)
		if b := img.Bounds(); b.Dx() != c.w || b.Dy() != c.h {
			t.Fatalf("orientation %d: size %v", c.orient, b)
		}
		r, _, bl, _ := img.At(c.red.x, c.red.y).RGBA()
		if r < 0xC000 || bl > 0x4000 {
			t.Errorf("orientation %d: expected red at %v, got r=%x b=%x", c.orient, c.red, r, bl)
		}
		r, _, bl, _ = img.At(c.blue.x, c.blue.y).RGBA()
		if bl < 0xC000 || r > 0x4000 {
			t.Errorf("orientation %d: expected blue at %v, got r=%x b=%x", c.orient, c.blue, r, bl)
		}
	}
}

func TestOrientTable(t *testing.T) {
	// source (3 wide, 2 tall):  A B C / D E F, encoded as red channel values 1..6
	src := image.NewRGBA(image.Rect(0, 0, 3, 2))
	for i := 0; i < 6; i++ {
		src.SetRGBA(i%3, i/3, color.RGBA{uint8(i + 1), 0, 0, 255})
	}
	want := map[int][]string{
		1: {"ABC", "DEF"},
		2: {"CBA", "FED"},
		3: {"FED", "CBA"},
		4: {"DEF", "ABC"},
		5: {"AD", "BE", "CF"},
		6: {"DA", "EB", "FC"},
		7: {"FC", "EB", "DA"},
		8: {"CF", "BE", "AD"},
	}
	for o, rows := range want {
		got := orient(src, o)
		if got.Rect.Dy() != len(rows) || got.Rect.Dx() != len(rows[0]) {
			t.Fatalf("orientation %d: size %v", o, got.Rect)
		}
		for y, row := range rows {
			for x, ch := range row {
				if g := got.RGBAAt(x, y).R; g != uint8(ch-'A'+1) {
					t.Errorf("orientation %d at (%d,%d): got %d want %c", o, x, y, g, ch)
				}
			}
		}
	}
	if orient(src, 0) != src || orient(src, 9) != src {
		t.Error("out-of-range orientation must be a no-op")
	}
}

func TestThumbnailUnsupported(t *testing.T) {
	for name, data := range map[string][]byte{
		"avif":          mkAVIF(avifOpts{w: 10, h: 10}),
		"animated webp": webpAnimated(10, 10, 2),
	} {
		t.Run(name, func(t *testing.T) {
			_, _, err := thumbOf(t, data, 10, 1<<20)
			if !errors.Is(err, ErrUnsupported) {
				t.Fatalf("err = %v; want ErrUnsupported", err)
			}
		})
	}
}

type countingReaderAt struct {
	r     *bytes.Reader
	reads int
}

func (c *countingReaderAt) ReadAt(p []byte, off int64) (int, error) {
	c.reads++
	return c.r.ReadAt(p, off)
}

func TestThumbnailRefusals(t *testing.T) {
	data := mkJPEG(t, 100, 100)
	info, err := inspectBytes(data, testLimits)
	if err != nil {
		t.Fatal(err)
	}
	t.Run("too many pixels before any read", func(t *testing.T) {
		cr := &countingReaderAt{r: bytes.NewReader(data)}
		_, _, err := Thumbnail(cr, int64(len(data)), info, 50, 9999)
		if !errors.Is(err, ErrTooManyPixels) || cr.reads != 0 {
			t.Fatalf("err=%v reads=%d", err, cr.reads)
		}
		_, _, err = Thumbnail(cr, int64(len(data)), info, 50, 10000)
		if err != nil {
			t.Fatalf("at the limit: %v", err)
		}
	})
	t.Run("absurd header values do not overflow", func(t *testing.T) {
		huge := Info{Format: JPEG, Width: 1 << 40, Height: 1 << 40, Frames: 1}
		_, _, err := Thumbnail(bytes.NewReader(data), int64(len(data)), huge, 50, 1<<40)
		if !errors.Is(err, ErrTooManyPixels) {
			t.Fatalf("err = %v", err)
		}
	})
	t.Run("dimension mismatch", func(t *testing.T) {
		bad := info
		bad.Width = 99
		_, _, err := Thumbnail(bytes.NewReader(data), int64(len(data)), bad, 50, 1<<20)
		if !errors.Is(err, ErrInvalid) {
			t.Fatalf("err = %v", err)
		}
	})
	t.Run("wrong format for content", func(t *testing.T) {
		bad := info
		bad.Format = PNG
		_, _, err := Thumbnail(bytes.NewReader(data), int64(len(data)), bad, 50, 1<<20)
		if !errors.Is(err, ErrInvalid) {
			t.Fatalf("err = %v", err)
		}
	})
	t.Run("corrupt pixel data", func(t *testing.T) {
		png := encodePNG(t, gradient(64, 64))
		pinfo, _ := inspectBytes(png, testLimits)
		broken := append([]byte(nil), png...)
		cs := splitPNG(t, broken)
		for i := range cs {
			if cs[i].typ == "IDAT" {
				for j := range cs[i].data { // valid CRCs, garbage deflate stream
					cs[i].data[j] = 0xFF
				}
			}
		}
		broken = joinPNG(cs)
		if _, err := inspectBytes(broken, testLimits); err != nil {
			t.Fatalf("structure is intact: %v", err)
		}
		_, _, err := Thumbnail(bytes.NewReader(broken), int64(len(broken)), pinfo, 32, 1<<20)
		if !errors.Is(err, ErrInvalid) {
			t.Fatalf("err = %v", err)
		}
	})
	t.Run("bad arguments", func(t *testing.T) {
		if _, _, err := Thumbnail(bytes.NewReader(data), int64(len(data)), info, 0, 1<<20); err == nil {
			t.Fatal("maxDim 0 accepted")
		}
		if _, _, err := Thumbnail(bytes.NewReader(data), int64(len(data)), Info{Format: JPEG}, 10, 1<<20); !errors.Is(err, ErrInvalid) {
			t.Fatalf("zero info: %v", err)
		}
		if _, _, err := Thumbnail(bytes.NewReader(data), int64(len(data)), Info{Format: "tiff", Width: 1, Height: 1}, 10, 1<<20); !errors.Is(err, ErrUnsupported) {
			t.Fatalf("unknown format: %v", err)
		}
	})
}

func TestThumbnailGIFComposite(t *testing.T) {
	pal := color.Palette{color.RGBA{0, 0, 0, 0}, color.RGBA{255, 0, 0, 255}, color.RGBA{0, 255, 0, 255}}
	first := image.NewPaletted(image.Rect(2, 2, 6, 6), pal)
	for i := range first.Pix {
		first.Pix[i] = 1
	}
	second := image.NewPaletted(image.Rect(0, 0, 10, 10), pal)
	for i := range second.Pix {
		second.Pix[i] = 2
	}
	g := &gif.GIF{Image: []*image.Paletted{first, second}, Delay: []int{1, 1}, Config: image.Config{Width: 10, Height: 10, ColorModel: pal}}
	var buf bytes.Buffer
	if err := gif.EncodeAll(&buf, g); err != nil {
		t.Fatal(err)
	}
	out, ext, err := thumbOf(t, buf.Bytes(), 100, 1<<20)
	if err != nil || ext != "png" {
		t.Fatalf("ext=%q err=%v", ext, err)
	}
	img := decodeOut(t, out, ext)
	if b := img.Bounds(); b.Dx() != 10 || b.Dy() != 10 {
		t.Fatalf("size %v; thumbnail must use the logical screen", b)
	}
	if _, _, _, a := img.At(0, 0).RGBA(); a != 0 {
		t.Errorf("uncovered area should be transparent, alpha=%x", a)
	}
	if r, gr, _, a := img.At(3, 3).RGBA(); r != 0xFFFF || gr != 0 || a != 0xFFFF {
		t.Errorf("first frame pixel wrong: r=%x g=%x a=%x", r, gr, a)
	}
	// the second (green) frame must not leak into the thumbnail
	if _, gr, _, _ := img.At(8, 8).RGBA(); gr != 0 {
		t.Errorf("second frame leaked: g=%x", gr)
	}
}

func TestThumbnailGIFFrameBeyondLimit(t *testing.T) {
	b := rawGIF(10, 10, 1, true)
	copy(b[24:28], []byte{0x88, 0x13, 0x88, 0x13}) // frame is 5000x5000 inside a 10x10 screen
	info := Info{Format: GIF, Width: 10, Height: 10, Frames: 1}
	_, _, err := Thumbnail(bytes.NewReader(b), int64(len(b)), info, 10, 1<<20)
	if !errors.Is(err, ErrTooManyPixels) {
		t.Fatalf("err = %v", err)
	}
}
