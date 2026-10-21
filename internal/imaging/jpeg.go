package imaging

import "bytes"

// inspectJPEG walks the marker segments up to SOS, then scans the entropy-coded data for EOI.
func inspectJPEG(w *window, lim Limits) (Info, error) {
	var info Info
	var haveSOF, haveEXIF bool
	off := int64(2) // after SOI
	inScan := false

	for {
		// Locate the next marker. Between header segments the next byte must be 0xFF; inside entropy-coded
		// data anything else is image data.
		buf, err := w.rest(off)
		if err != nil {
			return info, invalid(JPEG, "missing EOI")
		}
		if !inScan {
			if buf[0] != 0xFF {
				return info, invalid(JPEG, "expected marker")
			}
		} else {
			j := bytes.IndexByte(buf, 0xFF)
			if j < 0 {
				off += int64(len(buf))
				continue
			}
			off += int64(j)
		}
		p, err := w.get(off, 2)
		if err != nil {
			return info, invalid(JPEG, "missing EOI")
		}
		m := p[1]
		switch {
		case m == 0xFF: // fill byte
			off++
			continue
		case m == 0x00 || (m >= 0xD0 && m <= 0xD7) || m == 0x01:
			// byte stuffing (only meaningful in a scan), RSTn and TEM are standalone
			if !inScan && m != 0x01 {
				return info, invalid(JPEG, "unexpected marker")
			}
			off += 2
			continue
		case m == 0xD9:
			if !haveSOF || !inScan {
				return info, invalid(JPEG, "EOI before image data")
			}
			off += 2
			if err := checkTrailing(w, JPEG, off); err != nil {
				return Info{}, err
			}
			info.Frames = 1
			return info, nil
		case m == 0xD8:
			if !inScan {
				return info, invalid(JPEG, "unexpected SOI")
			}
			off += 2
			continue
		}

		// Segment with a length field.
		lp, err := w.get(off+2, 2)
		if err != nil {
			return info, invalid(JPEG, "truncated segment")
		}
		segLen := int64(lp[0])<<8 | int64(lp[1])
		if segLen < 2 {
			return info, invalid(JPEG, "bad segment length")
		}
		body, bodyLen := off+4, segLen-2
		if body+bodyLen > w.size {
			return info, invalid(JPEG, "segment exceeds file")
		}

		switch {
		case m >= 0xC0 && m <= 0xCF && m != 0xC4 && m != 0xC8 && m != 0xCC: // SOFn
			if inScan {
				break // another frame header inside the stream: ignore
			}
			if haveSOF {
				return info, invalid(JPEG, "duplicate SOF")
			}
			if bodyLen < 6 {
				return info, invalid(JPEG, "short SOF")
			}
			sof, err := w.get(body, 6)
			if err != nil {
				return info, err
			}
			h, wd := uint64(sof[1])<<8|uint64(sof[2]), uint64(sof[3])<<8|uint64(sof[4])
			if err := checkDims(JPEG, wd, h, lim); err != nil {
				return info, err
			}
			info.Width, info.Height = int(wd), int(h)
			haveSOF = true
		case m == 0xDA: // SOS
			if !haveSOF {
				return info, invalid(JPEG, "SOS before SOF")
			}
			inScan = true
		case m == 0xE1 && !haveEXIF && !inScan && bodyLen >= int64(len(exifPrefix))+8:
			if hd, err := w.get(body, len(exifPrefix)); err == nil && bytes.Equal(hd, exifPrefix) {
				// Oversized or unreadable EXIF is skipped, never fatal.
				if b, err := w.copyN(body, bodyLen, maxExifBytes); err == nil {
					haveEXIF = true
					ex := parseEXIF(b)
					info.Orientation, info.HasGPS = ex.orientation, ex.hasGPS
				}
			}
		}
		off = body + bodyLen
	}
}
