package httpapi

import (
	_ "embed"
	"net/http"
)

// openAPIDoc is the static OpenAPI 3.1 description of the public API. It is served without
// authentication and kept in sync with docs/api.md.
//
//go:embed openapi.json
var openAPIDoc []byte

// handleOpenAPI serves the embedded contract. The document changes only with a release, so it is
// cacheable for a day but always revalidated by ETag.
func (s *Server) handleOpenAPI(w http.ResponseWriter, r *http.Request) {
	etag := `"openapi-` + openAPIVersion + `"`
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

// openAPIVersion is bumped whenever openapi.json changes, so clients revalidate.
const openAPIVersion = "1"
