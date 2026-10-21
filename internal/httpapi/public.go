package httpapi

import (
	"errors"
	"net/http"
	"strings"

	"picberth/internal/core"
)

func (s *Server) handleHealthz(w http.ResponseWriter, r *http.Request) {
	w.Header().Set("Content-Type", "text/plain; charset=utf-8")
	w.Header().Set("Cache-Control", "no-store")
	_, _ = w.Write([]byte("ok\n"))
}

func (s *Server) handleReadyz(w http.ResponseWriter, r *http.Request) {
	w.Header().Set("Content-Type", "text/plain; charset=utf-8")
	w.Header().Set("Cache-Control", "no-store")
	if !s.ready.Load() {
		w.WriteHeader(http.StatusServiceUnavailable)
		_, _ = w.Write([]byte("shutting down\n"))
		return
	}
	if err := s.core.Ready(r.Context()); err != nil {
		s.log.Warn("readiness check failed", "err", err)
		w.WriteHeader(http.StatusServiceUnavailable)
		_, _ = w.Write([]byte("not ready\n"))
		return
	}
	_, _ = w.Write([]byte("ok\n"))
}

// handlePublicImage serves /i/{id}.{ext}: public, active, healthy images only. Everything else is a 404
// that reveals nothing about private, trashed or unknown images.
func (s *Server) handlePublicImage(w http.ResponseWriter, r *http.Request) {
	file := r.PathValue("file")
	dot := strings.LastIndexByte(file, '.')
	if dot <= 0 || dot == len(file)-1 {
		plainNotFound(w)
		return
	}
	id, ext := file[:dot], file[dot+1:]
	info, err := s.core.LookupPublic(r.Context(), id, ext)
	if err != nil {
		if errors.Is(err, core.ErrNotFound) {
			plainNotFound(w)
			return
		}
		s.log.Error("public lookup failed", "id", requestID(r), "err", err)
		http.Error(w, "internal error", http.StatusInternalServerError)
		return
	}
	etag := `"` + info.SHA256 + `"`
	h := w.Header()
	h.Set("Content-Type", info.ContentType) // from the database; never sniffed
	h.Set("Cache-Control", "public, max-age=31536000, immutable")
	h.Set("ETag", etag)
	h.Set("Last-Modified", info.ModTime.UTC().Format(http.TimeFormat))
	h.Set("Content-Security-Policy", "default-src 'none'; sandbox")
	h.Set("Cross-Origin-Resource-Policy", "cross-origin")
	h.Set("Content-Disposition", `inline; filename="`+info.ID+"."+info.Ext+`"`)

	if etagMatches(r.Header.Get("If-None-Match"), etag) {
		// The record alone decides: the file is never opened.
		h.Del("Content-Type")
		h.Del("Content-Disposition")
		w.WriteHeader(http.StatusNotModified)
		return
	}
	f, err := s.core.OpenPublic(info)
	if err != nil {
		if errors.Is(err, core.ErrNotFound) {
			plainNotFound(w)
			return
		}
		s.log.Error("opening public image failed", "id", requestID(r), "err", err)
		http.Error(w, "internal error", http.StatusInternalServerError)
		return
	}
	defer f.Close()
	extendDeadlines(w, 0, writeTimeoutFor(info.Size))
	http.ServeContent(w, r, "", info.ModTime, f)
}

// etagMatches implements If-None-Match with weak comparison (RFC 9110 §13.1.2).
func etagMatches(header, etag string) bool {
	if header == "" {
		return false
	}
	for _, cand := range strings.Split(header, ",") {
		cand = strings.TrimSpace(cand)
		if cand == "*" {
			return true
		}
		if strings.TrimPrefix(cand, "W/") == etag {
			return true
		}
	}
	return false
}
