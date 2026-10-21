package httpapi

import (
	"bytes"
	"io"
	"io/fs"
	"net/http"
	"path"
	"strings"

	"picberth/internal/core"
)

// Content types for the SPA build output, fixed here so they do not depend on the OS mime registry.
var assetTypes = map[string]string{
	".js":          "text/javascript; charset=utf-8",
	".mjs":         "text/javascript; charset=utf-8",
	".css":         "text/css; charset=utf-8",
	".html":        "text/html; charset=utf-8",
	".json":        "application/json",
	".map":         "application/json",
	".svg":         "image/svg+xml",
	".png":         "image/png",
	".jpg":         "image/jpeg",
	".jpeg":        "image/jpeg",
	".gif":         "image/gif",
	".webp":        "image/webp",
	".avif":        "image/avif",
	".ico":         "image/x-icon",
	".woff":        "font/woff",
	".woff2":       "font/woff2",
	".txt":         "text/plain; charset=utf-8",
	".webmanifest": "application/manifest+json",
}

// handleIndex serves index.html for the exact SPA routes.
func (s *Server) handleIndex(w http.ResponseWriter, r *http.Request) {
	data, err := fs.ReadFile(s.web, "index.html")
	if err != nil {
		s.log.Error("SPA index.html unavailable", "err", err)
		http.Error(w, "web UI not available", http.StatusInternalServerError)
		return
	}
	h := w.Header()
	h.Set("Content-Type", "text/html; charset=utf-8")
	h.Set("Content-Security-Policy", s.csp)
	h.Set("Referrer-Policy", "same-origin")
	h.Set("X-Frame-Options", "DENY")
	h.Set("Cross-Origin-Opener-Policy", "same-origin")
	h.Set("Cache-Control", "no-cache")
	_, _ = w.Write(data)
}

// handleAsset serves hashed build assets with an immutable cache policy.
func (s *Server) handleAsset(w http.ResponseWriter, r *http.Request) {
	name := "assets/" + r.PathValue("path")
	s.serveWebFile(w, r, name, "public, max-age=31536000, immutable")
}

// handleRootFile serves other files at the dist root (favicon.svg, ...). index.html is not one of them.
func (s *Server) handleRootFile(w http.ResponseWriter, r *http.Request) {
	name := r.PathValue("name")
	if name == "api" {
		apiHeaders(w)
		writeProblem(w, core.CodeNotFound, "Not found.", 0)
		return
	}
	if name == "index.html" || strings.HasPrefix(name, ".") {
		plainNotFound(w)
		return
	}
	s.serveWebFile(w, r, name, "public, max-age=86400")
}

func (s *Server) serveWebFile(w http.ResponseWriter, r *http.Request, name, cacheControl string) {
	if !fs.ValidPath(name) || path.Clean(name) != name {
		plainNotFound(w)
		return
	}
	f, err := s.web.Open(name)
	if err != nil {
		plainNotFound(w)
		return
	}
	defer f.Close()
	st, err := f.Stat()
	if err != nil || !st.Mode().IsRegular() {
		plainNotFound(w)
		return
	}
	var rs io.ReadSeeker
	if seeker, ok := f.(io.ReadSeeker); ok {
		rs = seeker
	} else {
		data, err := io.ReadAll(f)
		if err != nil {
			plainNotFound(w)
			return
		}
		rs = bytes.NewReader(data)
	}
	ct := assetTypes[strings.ToLower(path.Ext(name))]
	if ct == "" {
		ct = "application/octet-stream"
	}
	h := w.Header()
	h.Set("Content-Type", ct)
	h.Set("Cache-Control", cacheControl)
	http.ServeContent(w, r, "", st.ModTime(), rs)
}
