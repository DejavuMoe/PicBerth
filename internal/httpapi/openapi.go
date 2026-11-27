package httpapi

import (
	"crypto/sha256"
	_ "embed"
	"encoding/hex"
	"net/http"
)

// openAPIDoc is the static OpenAPI 3.1 description of the public API. It is served without
// authentication and is the API contract.
//
//go:embed openapi.json
var openAPIDoc []byte

// openAPIETag identifies the exact document served, so a client holding an older contract revalidates.
var openAPIETag = func() string {
	sum := sha256.Sum256(openAPIDoc)
	return `"` + hex.EncodeToString(sum[:]) + `"`
}()

// handleOpenAPI serves the embedded contract. The document changes only with a release, so it is
// cacheable for a day but always revalidated by ETag.
func (s *Server) handleOpenAPI(w http.ResponseWriter, r *http.Request) {
	etag := openAPIETag
	h := w.Header()
	h.Set("Content-Type", "application/json")
	h.Set("ETag", etag)
	h.Set("Cache-Control", "public, max-age=86400")
	if etagMatches(r.Header.Get("If-None-Match"), etag) {
		w.WriteHeader(http.StatusNotModified)
		return
	}
	_, _ = w.Write(openAPIDoc)
}
