package imaging

// inspectWebP walks the RIFF chunk list. Lossy (VP8), lossless (VP8L) and extended (VP8X, including
// animation) layouts are supported; only the chunk headers and the few bitstream header bytes that carry
// dimensions are read.
func inspectWebP(w *window, lim Limits) (Info, error) {
	var info Info
	hd, err := w.get(0, 12)
	if err != nil {
		return info, err
	}
	riffSize := int64(le32(hd[4:]))
	end := riffSize + 8
	if riffSize < 12 {
		return info, invalid(WebP, "bad RIFF size")
	}
	if end > w.size {
		return info, invalid(WebP, "truncated")
	}
	if w.size-end > 1 { // one pad byte is tolerated
		return info, invalid(WebP, "RIFF size mismatch")
	}

	var (
		haveVP8X, animated, haveImage, haveEXIF bool
		canvasW, canvasH                        uint64
		frames                                  int
	)
	off := int64(12)
	for first := true; off < end; first = false {
		if end-off < 8 {
			return info, invalid(WebP, "truncated chunk header")
		}
		ch, err := w.get(off, 8)
		if err != nil {
			return info, err
		}
		tag, size := string(ch[:4]), int64(le32(ch[4:]))
		body := off + 8
		if size > end-body {
			return info, invalid(WebP, "chunk exceeds RIFF")
		}
		if first && tag != "VP8X" && tag != "VP8 " && tag != "VP8L" {
			return info, invalid(WebP, "bad first chunk")
		}

		switch tag {
		case "VP8X":
			if !first || size < 10 {
				return info, invalid(WebP, "bad VP8X")
			}
			d, err := w.get(body, 10)
			if err != nil {
				return info, err
			}
			canvasW, canvasH = uint64(le24(d[4:]))+1, uint64(le24(d[7:]))+1
			if err := checkDims(WebP, canvasW, canvasH, lim); err != nil {
				return info, err
			}
			haveVP8X = true
			animated = d[0]&0x02 != 0
			info.HasAlpha = d[0]&0x10 != 0
			info.Width, info.Height = int(canvasW), int(canvasH)
		case "VP8 ", "VP8L":
			if animated || haveImage {
				break
			}
			var iw, ih uint64
			if tag == "VP8 " {
				if size < 10 {
					return info, invalid(WebP, "short VP8")
				}
				d, err := w.get(body, 10)
				if err != nil {
					return info, err
				}
				if d[0]&1 != 0 || d[3] != 0x9D || d[4] != 0x01 || d[5] != 0x2A {
					return info, invalid(WebP, "bad VP8 frame header")
				}
				iw, ih = uint64(le16(d[6:])&0x3fff), uint64(le16(d[8:])&0x3fff)
			} else {
				if size < 5 {
					return info, invalid(WebP, "short VP8L")
				}
				d, err := w.get(body, 5)
				if err != nil {
					return info, err
				}
				bits := le32(d[1:])
				if d[0] != 0x2F || bits>>29 != 0 {
					return info, invalid(WebP, "bad VP8L header")
				}
				iw, ih = uint64(bits&0x3fff)+1, uint64(bits>>14&0x3fff)+1
				if bits>>28&1 != 0 {
					info.HasAlpha = true
				}
			}
			if err := checkDims(WebP, iw, ih, lim); err != nil {
				return info, err
			}
			if haveVP8X {
				if iw != canvasW || ih != canvasH {
					return info, invalid(WebP, "canvas size mismatch")
				}
			} else {
				info.Width, info.Height = int(iw), int(ih)
			}
			haveImage = true
		case "ALPH":
			info.HasAlpha = true
		case "ANMF":
			if !animated {
				break
			}
			if size < 16 {
				return info, invalid(WebP, "short ANMF")
			}
			d, err := w.get(body, 16)
			if err != nil {
				return info, err
			}
			fx, fy := uint64(le24(d))*2, uint64(le24(d[3:]))*2
			fw, fh := uint64(le24(d[6:]))+1, uint64(le24(d[9:]))+1
			if err := checkDims(WebP, fw, fh, lim); err != nil {
				return info, err
			}
			if fx+fw > canvasW || fy+fh > canvasH {
				return info, invalid(WebP, "frame outside canvas")
			}
			frames++
			if err := checkFrames(WebP, frames, lim); err != nil {
				return info, err
			}
		case "EXIF":
			if !haveEXIF {
				if b, err := w.copyN(body, size, maxExifBytes); err == nil {
					haveEXIF = true
					ex := parseEXIF(b)
					info.Orientation, info.HasGPS = ex.orientation, ex.hasGPS
				}
			}
		}
		off = body + size + size&1
	}

	switch {
	case animated && frames == 0:
		return info, invalid(WebP, "animation without frames")
	case !animated && !haveImage:
		return info, invalid(WebP, "no image data")
	}
	if animated {
		info.Frames = frames
	}
	if err := checkTrailing(w, WebP, end); err != nil {
		return Info{}, err
	}
	return info, nil
}
