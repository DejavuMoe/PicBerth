package imaging

import (
	"bytes"
	"errors"
	"math"
)

const maxBoxDepth = 8

// Box header problems. At the top level a bad header after the metadata is treated as trailing data;
// everywhere else both are ErrInvalid.
var (
	errBoxHeader = errors.New("bad box header") // fewer than 8 bytes left, or size smaller than the header
	errBoxBounds = errors.New("box exceeds container")
)

type isoBox struct {
	typ        string
	start, end int64
	body       int64 // first payload byte
}

// readBox parses the box header at off inside a container ending at limit.
func readBox(w *window, off, limit int64) (isoBox, error) {
	var b isoBox
	if limit-off < 8 {
		return b, errBoxHeader
	}
	hd, err := w.get(off, 8)
	if err != nil {
		return b, err
	}
	for _, c := range hd[4:8] { // box types are printable four-character codes; anything else is not a box
		if c < 0x20 || c > 0x7E {
			if c != 0xA9 {
				return b, errBoxHeader
			}
		}
	}
	size, hdr := int64(be32(hd)), int64(8)
	b.typ, b.start = string(hd[4:8]), off
	switch size {
	case 0: // extends to the end of the container
		size = limit - off
	case 1:
		lg, err := w.get(off+8, 8)
		if err != nil {
			return b, errBoxBounds
		}
		v := uint64(be32(lg))<<32 | uint64(be32(lg[4:]))
		if v > math.MaxInt64 {
			return b, errBoxBounds
		}
		size, hdr = int64(v), 16
	}
	if size < hdr {
		return b, errBoxHeader
	}
	if size > limit-off {
		return b, errBoxBounds
	}
	b.end, b.body = off+size, off+hdr
	return b, nil
}

// forEachBox calls fn for every child box in [start, end). fn returns true to stop early.
func forEachBox(w *window, start, end int64, fn func(isoBox) (bool, error)) error {
	for off := start; off < end; {
		b, err := readBox(w, off, end)
		if err != nil {
			if errors.Is(err, errBoxHeader) || errors.Is(err, errBoxBounds) {
				return invalid(AVIF, "bad box")
			}
			return err
		}
		stop, err := fn(b)
		if err != nil || stop {
			return err
		}
		off = b.end
	}
	return nil
}

// findPath returns the first box reached by following path (box types) down from [start, end).
// "meta" is a full box, so its children begin four bytes into the payload.
func findPath(w *window, start, end int64, path []string, depth int) (isoBox, bool, error) {
	if depth > maxBoxDepth {
		return isoBox{}, false, invalid(AVIF, "boxes nested too deeply")
	}
	var found isoBox
	var ok bool
	err := forEachBox(w, start, end, func(b isoBox) (bool, error) {
		if b.typ != path[0] {
			return false, nil
		}
		if len(path) == 1 {
			found, ok = b, true
			return true, nil
		}
		cs := b.body
		if b.typ == "meta" {
			cs += 4
		}
		if cs > b.end {
			return false, invalid(AVIF, "short box")
		}
		var err error
		found, ok, err = findPath(w, cs, b.end, path[1:], depth+1)
		return ok || err != nil, err
	})
	return found, ok, err
}

// inspectAVIF validates the ISO-BMFF box structure and reads the primary dimensions from the first
// ispe property in meta/iprp/ipco. Animation (avis brand or a moov box) is reported as Frames = 2: it is a
// marker for "animated", not an exact count, since counting samples would mean parsing moov's sample
// tables. HasGPS is never set for AVIF.
func inspectAVIF(w *window, lim Limits) (Info, error) {
	var info Info
	first, err := readBox(w, 0, w.size)
	if err != nil {
		if errors.Is(err, errBoxHeader) || errors.Is(err, errBoxBounds) {
			return info, invalid(AVIF, "bad ftyp box")
		}
		return info, err
	}
	if first.typ != "ftyp" || first.end-first.start < 16 {
		return info, invalid(AVIF, "bad ftyp box")
	}
	n := first.end
	if n > 4096 {
		n = 4096
	}
	fb, err := w.get(0, int(n))
	if err != nil {
		return info, err
	}
	avif, avis := ftypBrands(fb)
	if !avif && !avis {
		return info, invalid(AVIF, "no avif brand")
	}
	animated := avis

	var haveMeta, haveSize bool
	off := first.end
	for off < w.size {
		b, err := readBox(w, off, w.size)
		if err != nil {
			if errors.Is(err, errBoxHeader) && haveMeta {
				break // not a box any more: trailing data, checked below
			}
			if errors.Is(err, errBoxHeader) || errors.Is(err, errBoxBounds) {
				return info, invalid(AVIF, "bad box")
			}
			return info, err
		}
		switch b.typ {
		case "meta":
			if haveMeta {
				return info, invalid(AVIF, "duplicate meta")
			}
			haveMeta = true
			if b.body+4 > b.end {
				return info, invalid(AVIF, "short meta")
			}
			ipco, ok, err := findPath(w, b.body+4, b.end, []string{"iprp", "ipco"}, 1)
			if err != nil {
				return info, err
			}
			if !ok {
				return info, invalid(AVIF, "missing ipco")
			}
			err = forEachBox(w, ipco.body, ipco.end, func(p isoBox) (bool, error) {
				switch p.typ {
				case "ispe":
					if haveSize {
						return false, nil
					}
					if p.end-p.body < 12 {
						return false, invalid(AVIF, "short ispe")
					}
					d, err := w.get(p.body+4, 8)
					if err != nil {
						return false, err
					}
					wd, ht := uint64(be32(d)), uint64(be32(d[4:]))
					if err := checkDims(AVIF, wd, ht, lim); err != nil {
						return false, err
					}
					info.Width, info.Height = int(wd), int(ht)
					haveSize = true
				case "auxC":
					n := p.end - p.body - 4
					if n > 128 {
						n = 128
					}
					if n > 0 {
						if d, err := w.get(p.body+4, int(n)); err == nil && bytes.Contains(d, []byte("alpha")) {
							info.HasAlpha = true
						}
					}
				}
				return false, nil
			})
			if err != nil {
				return info, err
			}
			if !haveSize {
				return info, invalid(AVIF, "missing ispe")
			}
		case "moov":
			animated = true
		}
		off = b.end
	}
	if !haveMeta {
		return info, invalid(AVIF, "missing meta")
	}
	if animated {
		info.Frames = 2
	}
	if err := checkTrailing(w, AVIF, off); err != nil {
		return Info{}, err
	}
	return info, nil
}
