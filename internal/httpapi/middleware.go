package httpapi

import (
	"context"
	"net/http"
	"runtime/debug"
	"strings"
	"time"

	"picberth/internal/buildinfo"
	"picberth/internal/core"
	"picberth/internal/ids"
)

type ctxKey int

const reqInfoKey ctxKey = 1

// reqInfo is request-scoped data shared between middleware and handlers (a pointer, so auth can fill it in
// for the access log).
type reqInfo struct {
	id        string
	ip        string
	principal core.Kind
}

func infoFrom(r *http.Request) *reqInfo {
	if ri, ok := r.Context().Value(reqInfoKey).(*reqInfo); ok {
		return ri
	}
	return &reqInfo{}
}

func requestID(r *http.Request) string { return infoFrom(r).id }

// statusWriter records the response status and size for the access log. Unwrap keeps
// http.ResponseController (deadlines, flushing) working.
type statusWriter struct {
	http.ResponseWriter
	status int
	bytes  int64
}

func (w *statusWriter) WriteHeader(code int) {
	if w.status == 0 {
		w.status = code
	}
	w.ResponseWriter.WriteHeader(code)
}

func (w *statusWriter) Write(b []byte) (int, error) {
	if w.status == 0 {
		w.status = http.StatusOK
	}
	n, err := w.ResponseWriter.Write(b)
	w.bytes += int64(n)
	return n, err
}

func (w *statusWriter) Unwrap() http.ResponseWriter { return w.ResponseWriter }

// recoverer turns panics into 500s.
func (s *Server) recoverer(next http.Handler) http.Handler {
	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		defer func() {
			if rec := recover(); rec != nil {
				if rec == http.ErrAbortHandler {
					panic(rec)
				}
				s.log.Error("panic in handler", "id", requestID(r), "path", r.URL.Path, "panic", rec, "stack", string(debug.Stack()))
				writeProblem(w, core.CodeInternal, "An unexpected error occurred.", 0)
			}
		}()
		next.ServeHTTP(w, r)
	})
}

// accessLog logs one line per request: never headers, cookies, tokens or query strings.
func (s *Server) accessLog(next http.Handler) http.Handler {
	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		start := time.Now()
		ri := &reqInfo{id: ids.RandomHex(6)}
		r = r.WithContext(context.WithValue(r.Context(), reqInfoKey, ri))
		w.Header().Set("X-Request-Id", ri.id)
		sw := &statusWriter{ResponseWriter: w}
		// Normal requests get 30 s to arrive and be answered; uploads, images and long operations extend this.
		rc := http.NewResponseController(w)
		_ = rc.SetReadDeadline(start.Add(30 * time.Second))
		_ = rc.SetWriteDeadline(start.Add(30 * time.Second))
		next.ServeHTTP(sw, r)
		status := sw.status
		if status == 0 {
			status = http.StatusOK
		}
		level := s.log.Info
		switch {
		case status >= 500:
			level = s.log.Warn
		case r.URL.Path == "/healthz" || r.URL.Path == "/readyz":
			level = s.log.Debug
		}
		level("request", "id", ri.id, "method", r.Method, "path", r.URL.Path, "status", status, "bytes", sw.bytes,
			"ms", time.Since(start).Milliseconds(), "principal", ri.principal.String(), "ip", ri.ip)
	})
}

// baseHeaders applies the headers every response carries.
func (s *Server) baseHeaders(next http.Handler) http.Handler {
	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		h := w.Header()
		h.Set("X-Content-Type-Options", "nosniff")
		h.Set("X-PicBerth-Version", buildinfo.Version)
		next.ServeHTTP(w, r)
	})
}

// rejectOddPaths answers 404 for paths no route could legitimately use (dot segments, empty segments, backslashes, NUL),
// before the mux can redirect them to a cleaned path.
func (s *Server) rejectOddPaths(next http.Handler) http.Handler {
	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		p := r.URL.Path
		if strings.ContainsAny(p, "\\\x00") || strings.Contains(p, "//") || strings.Contains(r.URL.RawPath, "\x00") {
			s.notFoundFor(w, r)
			return
		}
		for _, seg := range strings.Split(p, "/") {
			if seg == ".." || seg == "." {
				s.notFoundFor(w, r)
				return
			}
		}
		next.ServeHTTP(w, r)
	})
}

func (s *Server) notFoundFor(w http.ResponseWriter, r *http.Request) {
	if strings.HasPrefix(r.URL.Path, "/api/") {
		apiHeaders(w)
		writeProblem(w, core.CodeNotFound, "Not found.", 0)
		return
	}
	plainNotFound(w)
}

func apiHeaders(w http.ResponseWriter) {
	h := w.Header()
	h.Set("Cache-Control", "no-store")
	h.Set("Content-Security-Policy", "default-src 'none'; frame-ancestors 'none'")
}

// extendDeadlines lets a handler run longer than the default 30 s. A zero duration removes the deadline.
func extendDeadlines(w http.ResponseWriter, read, write time.Duration) {
	rc := http.NewResponseController(w)
	now := time.Now()
	if read > 0 {
		_ = rc.SetReadDeadline(now.Add(read))
	}
	switch {
	case write > 0:
		_ = rc.SetWriteDeadline(now.Add(write))
	case write < 0:
		_ = rc.SetWriteDeadline(time.Time{})
	}
}
