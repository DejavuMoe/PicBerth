package imaging

// gifBlock describes one image descriptor or extension block seen by walkGIF.
type gifBlock struct {
	frame  bool
	fw, fh uint32 // frame size (frames only)
	label  byte   // extension label (extensions only)
	first  []byte // first sub-block of an extension (copy, at most 255 bytes)
	end    int64  // offset just past the block
}

// gifSubBlocks skips a chain of data sub-blocks starting at off and returns the offset after the
// terminator plus a copy of the first sub-block.
func gifSubBlocks(w *window, off int64) (int64, []byte, error) {
	var first []byte
	for i := 0; ; i++ {
		b, err := w.get(off, 1)
		if err != nil {
			return 0, nil, err
		}
		n := int64(b[0])
		if n == 0 {
			return off + 1, first, nil
		}
		if i == 0 {
			d, err := w.get(off+1, int(n))
			if err != nil {
				return 0, nil, err
			}
			first = append([]byte(nil), d...)
		} else if n > w.size-off-1 {
			return 0, nil, errShort
		}
		off += 1 + n
	}
}

func gifTableSize(packed byte) int64 {
	if packed&0x80 == 0 {
		return 0
	}
	return 3 << ((packed & 7) + 1)
}

// walkGIF parses the logical screen and then every block, calling fn for frames and extensions. Frame
// sizes are checked against the pixel limit. fn may stop the walk early; otherwise the walk ends at the
// trailer, whose end offset is returned. A missing trailer is ErrInvalid.
func walkGIF(w *window, lim Limits, fn func(gifBlock) (stop bool, err error)) (width, height int, end int64, err error) {
	hd, err := w.get(0, 13)
	if err != nil {
		return 0, 0, 0, err
	}
	sw, sh := uint64(le16(hd[6:])), uint64(le16(hd[8:]))
	if err := checkDims(GIF, sw, sh, lim); err != nil {
		return 0, 0, 0, err
	}
	width, height = int(sw), int(sh)
	off := 13 + gifTableSize(hd[10])
	for {
		b, err := w.get(off, 1)
		if err != nil {
			return 0, 0, 0, invalid(GIF, "missing trailer")
		}
		switch b[0] {
		case 0x3B:
			return width, height, off + 1, nil
		case 0x2C:
			d, err := w.get(off+1, 9)
			if err != nil {
				return 0, 0, 0, err
			}
			fw, fh := uint32(le16(d[4:])), uint32(le16(d[6:]))
			if int64(fw)*int64(fh) > lim.MaxPixels {
				return 0, 0, 0, checkDims(GIF, uint64(fw), uint64(fh), lim)
			}
			off += 10 + gifTableSize(d[8])
			lzw, err := w.get(off, 1)
			if err != nil {
				return 0, 0, 0, err
			}
			if lzw[0] < 1 || lzw[0] > 8 {
				return 0, 0, 0, invalid(GIF, "bad LZW code size")
			}
			next, _, err := gifSubBlocks(w, off+1)
			if err != nil {
				return 0, 0, 0, err
			}
			off = next
			stop, err := fn(gifBlock{frame: true, fw: fw, fh: fh, end: off})
			if err != nil {
				return 0, 0, 0, err
			}
			if stop {
				return width, height, off, nil
			}
		case 0x21:
			l, err := w.get(off+1, 1)
			if err != nil {
				return 0, 0, 0, err
			}
			next, first, err := gifSubBlocks(w, off+2)
			if err != nil {
				return 0, 0, 0, err
			}
			off = next
			stop, err := fn(gifBlock{label: l[0], first: first, end: off})
			if err != nil {
				return 0, 0, 0, err
			}
			if stop {
				return width, height, off, nil
			}
		default:
			return 0, 0, 0, invalid(GIF, "bad block")
		}
	}
}

func inspectGIF(w *window, lim Limits) (Info, error) {
	var info Info
	var frames int
	width, height, end, err := walkGIF(w, lim, func(b gifBlock) (bool, error) {
		if b.frame {
			frames++
			return false, checkFrames(GIF, frames, lim)
		}
		if b.label == 0xF9 && len(b.first) >= 4 && b.first[0]&1 != 0 {
			info.HasAlpha = true
		}
		return false, nil
	})
	if err != nil {
		return info, err
	}
	if frames == 0 {
		return info, invalid(GIF, "no frames")
	}
	if err := checkTrailing(w, GIF, end); err != nil {
		return Info{}, err
	}
	info.Width, info.Height, info.Frames = width, height, frames
	return info, nil
}
