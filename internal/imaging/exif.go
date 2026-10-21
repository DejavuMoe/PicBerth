package imaging

import "bytes"

const (
	maxIFDEntries = 1000
	maxIFDChain   = 4

	tagOrientation = 0x0112
	tagGPSPointer  = 0x8825
	tagGPSLat      = 0x0002
	tagGPSLon      = 0x0004

	tiffShort = 3
	tiffLong  = 4
)

type exifInfo struct {
	orientation int // 1..8, 1 when absent or invalid
	hasGPS      bool
}

var exifPrefix = []byte("Exif\x00\x00")

// parseEXIF extracts the orientation and GPS presence from an EXIF payload. The payload may be a bare TIFF
// stream or carry the "Exif\0\0" prefix. Malformed data never fails: whatever was recovered before the
// problem is returned (GPS detection errs on the side of reporting).
func parseEXIF(b []byte) exifInfo {
	res := exifInfo{orientation: 1}
	b = bytes.TrimPrefix(b, exifPrefix)
	if len(b) < 8 {
		return res
	}
	var u16 func([]byte) uint32
	var u32 func([]byte) uint32
	switch string(b[:2]) {
	case "II":
		u16, u32 = le16, le32
	case "MM":
		u16 = func(p []byte) uint32 { return uint32(p[0])<<8 | uint32(p[1]) }
		u32 = be32
	default:
		return res
	}
	if u16(b[2:]) != 42 {
		return res
	}
	visited := make(map[uint32]bool, 8)
	off := u32(b[4:])
	for chain := 0; chain < maxIFDChain && off != 0; chain++ {
		if visited[off] || uint64(off)+2 > uint64(len(b)) {
			return res
		}
		visited[off] = true
		count := u16(b[off:])
		if count > maxIFDEntries {
			return res
		}
		entries := uint64(off) + 2
		if entries+uint64(count)*12+4 > uint64(len(b)) {
			// Entry table runs past the data: still look at the entries that fit, then stop.
			count = uint32((uint64(len(b)) - entries) / 12)
		}
		var gpsOff uint32
		for i := uint32(0); chain == 0 && i < count; i++ { // only IFD0 carries the tags we need
			e := b[entries+uint64(i)*12:]
			typ, cnt := u16(e[2:]), u32(e[4:])
			switch u16(e) {
			case tagOrientation:
				if typ == tiffShort && cnt >= 1 {
					if v := u16(e[8:]); v >= 1 && v <= 8 {
						res.orientation = int(v)
					}
				}
			case tagGPSPointer:
				if typ == tiffLong && cnt == 1 {
					gpsOff = u32(e[8:])
				}
			}
		}
		if chain == 0 && gpsOff != 0 && !visited[gpsOff] {
			visited[gpsOff] = true
			res.hasGPS = gpsIFDHasPosition(b, gpsOff, u16, u32)
		}
		next := entries + uint64(count)*12
		if next+4 > uint64(len(b)) {
			return res
		}
		off = u32(b[next:])
	}
	return res
}

// gpsIFDHasPosition reports whether the GPS IFD at off holds a latitude or longitude with a non-zero count.
func gpsIFDHasPosition(b []byte, off uint32, u16, u32 func([]byte) uint32) bool {
	if uint64(off)+2 > uint64(len(b)) {
		return false
	}
	count := u16(b[off:])
	if count > maxIFDEntries {
		return false
	}
	entries := uint64(off) + 2
	for i := uint64(0); i < uint64(count); i++ {
		p := entries + i*12
		if p+12 > uint64(len(b)) {
			return false
		}
		e := b[p:]
		if tag := u16(e); (tag == tagGPSLat || tag == tagGPSLon) && u32(e[4:]) > 0 {
			return true
		}
	}
	return false
}
