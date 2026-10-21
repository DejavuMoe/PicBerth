package imaging

import "bytes"

var (
	pngSignature = []byte{0x89, 'P', 'N', 'G', '\r', '\n', 0x1a, '\n'}
	avifBrand    = []byte("avif")
	avisBrand    = []byte("avis")
)

// sniff identifies a format from the leading bytes only. It never reads past len(head).
func sniff(head []byte) (Format, bool) {
	switch {
	case len(head) >= 3 && head[0] == 0xFF && head[1] == 0xD8 && head[2] == 0xFF:
		return JPEG, true
	case len(head) >= 8 && bytes.Equal(head[:8], pngSignature):
		return PNG, true
	case len(head) >= 6 && (string(head[:6]) == "GIF87a" || string(head[:6]) == "GIF89a"):
		return GIF, true
	case len(head) >= 12 && string(head[:4]) == "RIFF" && string(head[8:12]) == "WEBP":
		return WebP, true
	case isAVIFHead(head):
		return AVIF, true
	}
	return "", false
}

// isAVIFHead reports whether head starts with an ISO-BMFF ftyp box listing avif or avis as the major or a
// compatible brand. Only brands that lie inside both the box and head are considered.
func isAVIFHead(head []byte) bool {
	avif, avis := ftypBrands(head)
	return avif || avis
}

// ftypBrands scans an ftyp box at the start of b for the avif and avis brands (major or compatible).
func ftypBrands(b []byte) (avif, avis bool) {
	if len(b) < 12 || string(b[4:8]) != "ftyp" {
		return false, false
	}
	boxSize := int64(be32(b))
	end := int64(len(b))
	if boxSize >= 16 && boxSize < end {
		end = boxSize
	}
	check := func(p []byte) {
		avif = avif || bytes.Equal(p, avifBrand)
		avis = avis || bytes.Equal(p, avisBrand)
	}
	check(b[8:12])
	// bytes 12..16 are the minor version, compatible brands start at 16
	for off := int64(16); off+4 <= end; off += 4 {
		check(b[off : off+4])
	}
	return avif, avis
}
