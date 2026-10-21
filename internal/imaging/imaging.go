// Package imaging identifies, validates and thumbnails uploaded images.
//
// Validation never trusts file names or client-declared content types: the format is sniffed from the
// leading bytes, then the container structure is walked without decoding pixels (design baseline §6.3).
// Full decoding only happens in Thumbnail, behind a pixel limit checked from the header first.
package imaging

import (
	"errors"
	"io"
)

// Format is an allowed image format.
type Format string

const (
	JPEG Format = "jpeg"
	PNG  Format = "png"
	GIF  Format = "gif"
	WebP Format = "webp"
	AVIF Format = "avif"
)

// AllFormats lists every format PicBerth can accept, in display order.
var AllFormats = []Format{JPEG, PNG, GIF, WebP, AVIF}

// MIME returns the canonical media type.
func (f Format) MIME() string {
	switch f {
	case JPEG:
		return "image/jpeg"
	case PNG:
		return "image/png"
	case GIF:
		return "image/gif"
	case WebP:
		return "image/webp"
	case AVIF:
		return "image/avif"
	}
	return "application/octet-stream"
}

// Ext returns the canonical file extension without a dot.
func (f Format) Ext() string {
	switch f {
	case JPEG:
		return "jpg"
	case PNG, GIF, WebP, AVIF:
		return string(f)
	}
	return "bin"
}

// ParseFormat parses a format name as used in settings ("jpeg", "png", …).
func ParseFormat(s string) (Format, bool) {
	for _, f := range AllFormats {
		if string(f) == s {
			return f, true
		}
	}
	return "", false
}

// Info is what Inspect learns from the container structure.
type Info struct {
	Format      Format
	Width       int
	Height      int
	Frames      int  // 1 for still images
	HasGPS      bool // EXIF GPS latitude/longitude present (JPEG APP1, PNG eXIf, WebP EXIF)
	HasAlpha    bool // best effort
	Orientation int  // EXIF orientation 1–8; 1 when absent
}

// Limits bound what Inspect accepts.
type Limits struct {
	MaxPixels int64 // per frame: width*height
	MaxFrames int   // 0 = default (10000)
}

var (
	// ErrUnsupported: the content is not one of the supported formats (or cannot be decoded for thumbnails).
	ErrUnsupported = errors.New("imaging: unsupported format")
	// ErrInvalid: the content claims a supported format but its structure is broken or suspicious.
	ErrInvalid = errors.New("imaging: invalid image")
	// ErrTooManyPixels: a frame exceeds Limits.MaxPixels, or there are too many frames.
	ErrTooManyPixels = errors.New("imaging: too many pixels")
)

// Sniff identifies the format from leading bytes (at least 32 bytes should be supplied).
func Sniff(head []byte) (Format, bool) {
	return sniff(head)
}

// Inspect validates the container structure of r (size bytes long) and returns its metadata.
// Errors wrap ErrUnsupported, ErrInvalid or ErrTooManyPixels.
func Inspect(r io.ReaderAt, size int64, lim Limits) (Info, error) {
	return inspect(r, size, lim)
}

// Thumbnail decodes the image (first frame for animations), applies EXIF orientation and scales it so the
// longest side is at most maxDim. It returns encoded bytes and the extension ("jpg" for opaque images,
// "png" when the image has transparency). It refuses images above maxPixels and returns ErrUnsupported
// for formats that cannot be decoded (AVIF, animated WebP).
func Thumbnail(r io.ReaderAt, size int64, info Info, maxDim int, maxPixels int64) ([]byte, string, error) {
	return thumbnail(r, size, info, maxDim, maxPixels)
}
