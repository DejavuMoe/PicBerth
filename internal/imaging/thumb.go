package imaging

import (
	"bufio"
	"bytes"
	"fmt"
	"image"
	"image/gif"
	"image/jpeg"
	"image/png"
	"io"

	"golang.org/x/image/draw"
	"golang.org/x/image/webp"
)

const (
	// defaultThumbPixels is used when maxPixels <= 0 (design baseline §13.3: 25 megapixels).
	defaultThumbPixels int64 = 25_000_000
	jpegQuality              = 82
	// preReduceRatio: sources at least this many times larger than the target (per axis) are first
	// shrunk with the cheap ApproxBiLinear scaler to twice the target size, then finished with CatmullRom.
	preReduceRatio = 4
)

func thumbnail(r io.ReaderAt, size int64, info Info, maxDim int, maxPixels int64) (out []byte, ext string, err error) {
	defer func() {
		if rec := recover(); rec != nil {
			out, ext, err = nil, "", fmt.Errorf("%w: internal decode failure", ErrInvalid)
		}
	}()
	if maxDim <= 0 {
		return nil, "", fmt.Errorf("imaging: maxDim must be positive")
	}
	if maxPixels <= 0 {
		maxPixels = defaultThumbPixels
	}
	if info.Width <= 0 || info.Height <= 0 || r == nil || size <= 0 {
		return nil, "", fmt.Errorf("%w: bad dimensions", ErrInvalid)
	}
	// Checked from the header values before anything is decoded or allocated.
	if uint64(info.Width) > uint64(maxPixels)/uint64(info.Height) {
		return nil, "", fmt.Errorf("%w: thumbnail source", ErrTooManyPixels)
	}
	switch info.Format {
	case JPEG, PNG, GIF:
	case WebP:
		if info.Frames > 1 {
			return nil, "", fmt.Errorf("%w: animated webp", ErrUnsupported)
		}
	default: // AVIF has no decoder
		return nil, "", fmt.Errorf("%w: no decoder for %s", ErrUnsupported, info.Format)
	}

	src, err := decodeStill(r, size, info, maxPixels)
	if err != nil {
		return nil, "", err
	}
	opaque := isOpaque(src)
	sb := src.Bounds()
	tw, th := fitWithin(sb.Dx(), sb.Dy(), maxDim)

	var img *image.RGBA
	switch {
	case tw == sb.Dx() && th == sb.Dy():
		img = image.NewRGBA(image.Rect(0, 0, tw, th))
		draw.Draw(img, img.Bounds(), src, sb.Min, draw.Src)
	default:
		if sb.Dx() >= preReduceRatio*tw && sb.Dy() >= preReduceRatio*th {
			mid := image.NewRGBA(image.Rect(0, 0, 2*tw, 2*th))
			draw.ApproxBiLinear.Scale(mid, mid.Bounds(), src, sb, draw.Src, nil)
			src, sb = mid, mid.Bounds()
		}
		img = image.NewRGBA(image.Rect(0, 0, tw, th))
		draw.CatmullRom.Scale(img, img.Bounds(), src, sb, draw.Src, nil)
	}
	if opaque {
		// Resampling arithmetic can leave alpha at 254; the source had none.
		for i := 3; i < len(img.Pix); i += 4 {
			img.Pix[i] = 0xFF
		}
	}
	if info.Format == JPEG || info.Format == WebP {
		img = orient(img, info.Orientation)
	}

	var buf bytes.Buffer
	if opaque {
		if err := jpeg.Encode(&buf, img, &jpeg.Options{Quality: jpegQuality}); err != nil {
			return nil, "", fmt.Errorf("imaging: encode jpeg: %w", err)
		}
		return buf.Bytes(), "jpg", nil
	}
	enc := png.Encoder{CompressionLevel: png.BestSpeed}
	if err := enc.Encode(&buf, img); err != nil {
		return nil, "", fmt.Errorf("imaging: encode png: %w", err)
	}
	return buf.Bytes(), "png", nil
}

// fitWithin scales (w, h) down so the longer side is at most maxDim, never up, and never below 1 pixel.
func fitWithin(w, h, maxDim int) (int, int) {
	long := w
	if h > long {
		long = h
	}
	if long <= maxDim {
		return w, h
	}
	s := func(v int) int {
		n := int((int64(v)*int64(maxDim) + int64(long)/2) / int64(long))
		if n < 1 {
			n = 1
		}
		return n
	}
	return s(w), s(h)
}

type opaquer interface{ Opaque() bool }

func isOpaque(img image.Image) bool {
	if o, ok := img.(opaquer); ok {
		return o.Opaque()
	}
	return false
}

// decodeStill decodes the first frame of the image, after DecodeConfig has confirmed the dimensions match
// what Inspect reported.
func decodeStill(r io.ReaderAt, size int64, info Info, maxPixels int64) (image.Image, error) {
	open := func() io.Reader { return bufio.NewReaderSize(io.NewSectionReader(r, 0, size), windowSize) }

	var cfg image.Config
	var err error
	switch info.Format {
	case JPEG:
		cfg, err = jpeg.DecodeConfig(open())
	case PNG:
		cfg, err = png.DecodeConfig(open())
	case GIF:
		cfg, err = gif.DecodeConfig(open())
	case WebP:
		cfg, err = webp.DecodeConfig(open())
	}
	if err != nil {
		return nil, invalid(info.Format, "unreadable header")
	}
	if cfg.Width != info.Width || cfg.Height != info.Height {
		return nil, invalid(info.Format, "dimensions differ from inspection")
	}

	var img image.Image
	switch info.Format {
	case JPEG:
		img, err = jpeg.Decode(open())
	case PNG:
		img, err = png.Decode(open())
	case WebP:
		img, err = webp.Decode(open())
	case GIF:
		return decodeGIFFirstFrame(r, size, info, maxPixels)
	}
	if err != nil {
		return nil, invalid(info.Format, "decode failed")
	}
	if b := img.Bounds(); b.Dx() != info.Width || b.Dy() != info.Height {
		return nil, invalid(info.Format, "decoded size differs from header")
	}
	return img, nil
}

// decodeGIFFirstFrame decodes only the first frame (gif.Decode stops after it, but we also cut the stream
// there so nothing later is read) and composites it onto a transparent logical-screen-sized canvas.
func decodeGIFFirstFrame(r io.ReaderAt, size int64, info Info, maxPixels int64) (image.Image, error) {
	w := newWindow(r, size)
	found := false
	_, _, end, err := walkGIF(w, Limits{MaxPixels: maxPixels, MaxFrames: 1}, func(b gifBlock) (bool, error) {
		found = b.frame
		return b.frame, nil
	})
	if err != nil {
		return nil, gifErr(err)
	}
	if !found {
		return nil, invalid(GIF, "no frames")
	}
	stream := io.MultiReader(io.NewSectionReader(r, 0, end), bytes.NewReader([]byte{0x3B}))
	g, err := gif.DecodeAll(stream)
	if err != nil || len(g.Image) == 0 {
		return nil, invalid(GIF, "decode failed")
	}
	frame := g.Image[0]
	canvas := image.NewRGBA(image.Rect(0, 0, info.Width, info.Height))
	draw.Draw(canvas, frame.Bounds().Intersect(canvas.Bounds()), frame, frame.Bounds().Min, draw.Over)
	return canvas, nil
}

func gifErr(err error) error {
	if err == errShort {
		return invalid(GIF, "truncated")
	}
	return err
}

// orient applies an EXIF orientation (1-8) to img and returns the displayed image.
func orient(img *image.RGBA, o int) *image.RGBA {
	if o < 2 || o > 8 {
		return img
	}
	W, H := img.Rect.Dx(), img.Rect.Dy()
	ow, oh := W, H
	if o >= 5 {
		ow, oh = H, W
	}
	// source coordinate for output (x, y): sx = a1*x + b1*y + c1, sy = a2*x + b2*y + c2
	var a1, b1, c1, a2, b2, c2 int
	switch o {
	case 2: // mirror horizontally
		a1, c1, b2 = -1, W-1, 1
	case 3: // rotate 180
		a1, c1, b2, c2 = -1, W-1, -1, H-1
	case 4: // mirror vertically
		a1, b2, c2 = 1, -1, H-1
	case 5: // transpose
		b1, a2 = 1, 1
	case 6: // rotate 90 clockwise
		b1, a2, c2 = 1, -1, H-1
	case 7: // transverse
		b1, c1, a2, c2 = -1, W-1, -1, H-1
	case 8: // rotate 90 counter-clockwise
		b1, c1, a2 = -1, W-1, 1
	}
	out := image.NewRGBA(image.Rect(0, 0, ow, oh))
	for y := 0; y < oh; y++ {
		for x := 0; x < ow; x++ {
			sx, sy := a1*x+b1*y+c1, a2*x+b2*y+c2
			copy(out.Pix[y*out.Stride+x*4:y*out.Stride+x*4+4], img.Pix[sy*img.Stride+sx*4:sy*img.Stride+sx*4+4])
		}
	}
	return out
}
