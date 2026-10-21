package imaging

import "hash/crc32"

// crcRange computes the CRC-32 (IEEE) of n bytes at off, streaming through the window.
func (w *window) crcRange(off, n int64) (uint32, error) {
	if n < 0 || off < 0 || off > w.size || n > w.size-off {
		return 0, errShort
	}
	var sum uint32
	for n > 0 {
		b, err := w.rest(off)
		if err != nil {
			return 0, err
		}
		if int64(len(b)) > n {
			b = b[:n]
		}
		sum = crc32.Update(sum, crc32.IEEETable, b)
		off += int64(len(b))
		n -= int64(len(b))
	}
	return sum, nil
}

// inspectPNG walks every chunk, verifying lengths and CRCs, and requires IHDR first and IDAT + IEND.
func inspectPNG(w *window, lim Limits) (Info, error) {
	var info Info
	var haveIHDR, haveIDAT, havePLTE bool
	var colorType byte
	var canvasW, canvasH uint32
	declaredFrames := 0
	haveEXIF := false
	off := int64(8)

	for {
		hd, err := w.get(off, 8)
		if err != nil {
			return info, invalid(PNG, "missing IEND")
		}
		length := int64(be32(hd))
		typ := [4]byte{hd[4], hd[5], hd[6], hd[7]}
		for _, c := range typ {
			if !(c >= 'A' && c <= 'Z' || c >= 'a' && c <= 'z') {
				return info, invalid(PNG, "bad chunk type")
			}
		}
		if length > 0x7fffffff || off+12+length > w.size {
			return info, invalid(PNG, "chunk exceeds file")
		}
		body := off + 8
		name := string(typ[:])

		if !haveIHDR && name != "IHDR" {
			return info, invalid(PNG, "IHDR not first")
		}
		sum, err := w.crcRange(off+4, length+4)
		if err != nil {
			return info, err
		}
		cb, err := w.get(body+length, 4)
		if err != nil {
			return info, err
		}
		if sum != be32(cb) {
			return info, invalid(PNG, "bad chunk CRC")
		}

		switch name {
		case "IHDR":
			if haveIHDR || length != 13 {
				return info, invalid(PNG, "bad IHDR")
			}
			d, err := w.get(body, 13)
			if err != nil {
				return info, err
			}
			wd, h := be32(d), be32(d[4:])
			depth, ct := d[8], d[9]
			if wd > 0x7fffffff || h > 0x7fffffff {
				return info, invalid(PNG, "bad dimensions")
			}
			if !pngDepthOK(ct, depth) || d[10] != 0 || d[11] != 0 || d[12] > 1 {
				return info, invalid(PNG, "bad IHDR fields")
			}
			if err := checkDims(PNG, uint64(wd), uint64(h), lim); err != nil {
				return info, err
			}
			haveIHDR, colorType = true, ct
			canvasW, canvasH = wd, h
			info.Width, info.Height = int(wd), int(h)
			info.HasAlpha = ct == 4 || ct == 6
		case "PLTE":
			if length == 0 || length%3 != 0 || length > 768 {
				return info, invalid(PNG, "bad PLTE")
			}
			havePLTE = true
		case "tRNS":
			info.HasAlpha = true
		case "acTL":
			if haveIDAT || declaredFrames != 0 || length != 8 {
				return info, invalid(PNG, "bad acTL")
			}
			d, err := w.get(body, 8)
			if err != nil {
				return info, err
			}
			n := be32(d)
			if n == 0 || n > 0x7fffffff {
				return info, invalid(PNG, "bad frame count")
			}
			declaredFrames = int(n)
			if err := checkFrames(PNG, declaredFrames, lim); err != nil {
				return info, err
			}
		case "fcTL":
			if length != 26 {
				return info, invalid(PNG, "bad fcTL")
			}
			d, err := w.get(body, 26)
			if err != nil {
				return info, err
			}
			fw, fh, fx, fy := be32(d[4:]), be32(d[8:]), be32(d[12:]), be32(d[16:])
			if fw == 0 || fh == 0 || uint64(fx)+uint64(fw) > uint64(canvasW) || uint64(fy)+uint64(fh) > uint64(canvasH) {
				return info, invalid(PNG, "frame outside canvas")
			}
		case "eXIf":
			if !haveEXIF {
				if b, err := w.copyN(body, length, maxExifBytes); err == nil {
					haveEXIF = true
					ex := parseEXIF(b)
					info.Orientation, info.HasGPS = ex.orientation, ex.hasGPS
				}
			}
		case "IDAT":
			if colorType == 3 && !havePLTE {
				return info, invalid(PNG, "missing PLTE")
			}
			haveIDAT = true
		case "IEND":
			if length != 0 {
				return info, invalid(PNG, "bad IEND")
			}
			if !haveIDAT {
				return info, invalid(PNG, "missing IDAT")
			}
			end := off + 12
			if err := checkTrailing(w, PNG, end); err != nil {
				return Info{}, err
			}
			info.Frames = 1
			if declaredFrames > 0 {
				info.Frames = declaredFrames
			}
			return info, nil
		}
		off += 12 + length
	}
}

func pngDepthOK(colorType, depth byte) bool {
	switch colorType {
	case 0:
		return depth == 1 || depth == 2 || depth == 4 || depth == 8 || depth == 16
	case 2, 4, 6:
		return depth == 8 || depth == 16
	case 3:
		return depth == 1 || depth == 2 || depth == 4 || depth == 8
	}
	return false
}
