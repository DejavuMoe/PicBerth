package imaging

import (
	"errors"
	"fmt"
	"io"
)

func inspect(r io.ReaderAt, size int64, lim Limits) (info Info, err error) {
	defer func() {
		// The walkers are bounds-checked, but this code parses hostile input: never let a bug escape as a panic.
		if rec := recover(); rec != nil {
			info, err = Info{}, fmt.Errorf("%w: internal parse failure", ErrInvalid)
		}
	}()
	if r == nil || size <= 0 {
		return Info{}, fmt.Errorf("%w: empty input", ErrUnsupported)
	}
	if lim.MaxPixels <= 0 {
		lim.MaxPixels = defaultPixels
	}
	if lim.MaxFrames <= 0 {
		lim.MaxFrames = defaultFrames
	}
	w := newWindow(r, size)
	headLen := size
	if headLen > 4096 {
		headLen = 4096
	}
	head, err := w.get(0, int(headLen))
	if err != nil {
		if errors.Is(err, errShort) {
			return Info{}, fmt.Errorf("%w: unreadable header", ErrInvalid)
		}
		return Info{}, err
	}
	f, ok := sniff(head)
	if !ok {
		return Info{}, fmt.Errorf("%w: unknown signature", ErrUnsupported)
	}
	switch f {
	case JPEG:
		info, err = inspectJPEG(w, lim)
	case PNG:
		info, err = inspectPNG(w, lim)
	case GIF:
		info, err = inspectGIF(w, lim)
	case WebP:
		info, err = inspectWebP(w, lim)
	case AVIF:
		info, err = inspectAVIF(w, lim)
	}
	if err != nil {
		if errors.Is(err, errShort) {
			err = invalid(f, "truncated")
		}
		return Info{}, err
	}
	info.Format = f
	if info.Frames < 1 {
		info.Frames = 1
	}
	if info.Orientation < 1 || info.Orientation > 8 {
		info.Orientation = 1
	}
	return info, nil
}
