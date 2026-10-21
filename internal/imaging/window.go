package imaging

import (
	"bytes"
	"errors"
	"fmt"
	"io"
)

// Internal limits that keep every walk and read bounded regardless of what the file claims.
const (
	windowSize    = 64 << 10 // read-ahead block for the structure walkers
	maxExifBytes  = 1 << 20  // EXIF/TIFF payloads larger than this are ignored, not read
	trailingScan  = 4 << 10  // bytes of trailing data inspected for embedded markup
	defaultFrames = 10000
	// defaultPixels is used when Limits.MaxPixels is not set (design baseline §13.3: 50 megapixels).
	defaultPixels int64 = 50_000_000
)

// errShort is returned by window reads that run past the end of the file; inspect converts it to ErrInvalid.
var errShort = errors.New("short read")

func invalid(format Format, what string) error {
	return fmt.Errorf("%w: %s: %s", ErrInvalid, format, what)
}

func be32(b []byte) uint32 {
	return uint32(b[0])<<24 | uint32(b[1])<<16 | uint32(b[2])<<8 | uint32(b[3])
}
func le32(b []byte) uint32 {
	return uint32(b[3])<<24 | uint32(b[2])<<16 | uint32(b[1])<<8 | uint32(b[0])
}
func le24(b []byte) uint32 { return uint32(b[2])<<16 | uint32(b[1])<<8 | uint32(b[0]) }
func le16(b []byte) uint32 { return uint32(b[1])<<8 | uint32(b[0]) }

// window is a read-ahead cache over an io.ReaderAt of known size. Every access is bounds-checked against
// size, so no length field read from the file can cause an out-of-range read or a large allocation.
type window struct {
	r    io.ReaderAt
	size int64
	buf  []byte
	off  int64 // file offset of buf[0]
	n    int   // valid bytes in buf
}

func newWindow(r io.ReaderAt, size int64) *window {
	return &window{r: r, size: size, buf: make([]byte, windowSize)}
}

func (w *window) fill(off int64) error {
	want := int64(len(w.buf))
	if rem := w.size - off; rem < want {
		want = rem
	}
	if want <= 0 {
		w.n = 0
		return errShort
	}
	n, err := w.r.ReadAt(w.buf[:want], off)
	if int64(n) < want {
		w.n = 0
		if err == nil || errors.Is(err, io.EOF) || errors.Is(err, io.ErrUnexpectedEOF) {
			return errShort
		}
		return fmt.Errorf("imaging: read: %w", err)
	}
	w.off, w.n = off, n
	return nil
}

func (w *window) cached(off int64, n int) bool {
	return w.n > 0 && off >= w.off && off+int64(n) <= w.off+int64(w.n)
}

// get returns n bytes (n <= windowSize) at off. The slice is only valid until the next window call.
func (w *window) get(off int64, n int) ([]byte, error) {
	if off < 0 || n < 0 || n > len(w.buf) || off > w.size || int64(n) > w.size-off {
		return nil, errShort
	}
	if !w.cached(off, n) {
		if err := w.fill(off); err != nil {
			return nil, err
		}
	}
	i := int(off - w.off)
	return w.buf[i : i+n], nil
}

// rest returns the non-empty cached bytes starting at off (up to the end of the cache block).
func (w *window) rest(off int64) ([]byte, error) {
	if off < 0 || off >= w.size {
		return nil, errShort
	}
	if !w.cached(off, 1) {
		if err := w.fill(off); err != nil {
			return nil, err
		}
	}
	return w.buf[int(off-w.off):w.n], nil
}

// copyN reads n bytes at off into a fresh slice. n is capped by max before anything is allocated.
func (w *window) copyN(off int64, n int64, max int64) ([]byte, error) {
	if n < 0 || n > max || off < 0 || off > w.size || n > w.size-off {
		return nil, errShort
	}
	b := make([]byte, n)
	if n == 0 {
		return b, nil
	}
	m, err := w.r.ReadAt(b, off)
	if int64(m) < n {
		if err == nil || errors.Is(err, io.EOF) || errors.Is(err, io.ErrUnexpectedEOF) {
			return nil, errShort
		}
		return nil, fmt.Errorf("imaging: read: %w", err)
	}
	return b, nil
}

// checkDims validates declared dimensions against the limits without ever multiplying unchecked values.
func checkDims(f Format, w, h uint64, lim Limits) error {
	if w == 0 || h == 0 {
		return invalid(f, "zero dimension")
	}
	if w > uint64(lim.MaxPixels)/h || w*h > uint64(lim.MaxPixels) {
		return fmt.Errorf("%w: %s: %dx%d", ErrTooManyPixels, f, w, h)
	}
	return nil
}

func checkFrames(f Format, frames int, lim Limits) error {
	if frames > lim.MaxFrames {
		return fmt.Errorf("%w: %s: too many frames", ErrTooManyPixels, f)
	}
	return nil
}

// suspicious are markers that should never follow the logical end of an image.
var suspicious = [][]byte{[]byte("<script"), []byte("<html"), []byte("<!doctype"), []byte("<?php"), []byte("<svg"), []byte("<iframe")}

// checkTrailing looks at (at most) the first 4 KiB after the logical end of the image for embedded markup.
// Other trailing data (for example a motion-photo video) is allowed.
func checkTrailing(w *window, f Format, end int64) error {
	if end >= w.size {
		return nil
	}
	n := w.size - end
	if n > trailingScan {
		n = trailingScan
	}
	b, err := w.copyN(end, n, trailingScan)
	if err != nil {
		return err
	}
	for i, c := range b { // ASCII lower-case in place
		if c >= 'A' && c <= 'Z' {
			b[i] = c + 32
		}
	}
	for _, s := range suspicious {
		if bytes.Contains(b, s) {
			return invalid(f, "markup after end of image")
		}
	}
	return nil
}
