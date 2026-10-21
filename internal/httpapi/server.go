// Package httpapi is PicBerth's HTTP surface: public image delivery, the JSON API, the SPA host and
// health probes. It adapts transport details to internal/core and contains no business rules.
package httpapi

import (
	"io/fs"
	"log/slog"
	"net/http"
	"net/url"
	"strings"
	"sync/atomic"

	"picberth/internal/core"
	"picberth/internal/netx"
)

// Options configure a Server.
type Options struct {
	Core      *core.Service
	PublicURL string // normalized base URL, no trailing slash
	ImageURL  string
	Trusted   *netx.Trusted
	Web       fs.FS // SPA build output (index.html, assets/...)
	Logger    *slog.Logger
}

// Server is the HTTP handler tree.
type Server struct {
	core      *core.Service
	log       *slog.Logger
	trusted   *netx.Trusted
	web       fs.FS
	publicURL string
	imageURL  string
	secure    bool // public URL is https: Secure / __Host- cookies
	cookie    string
	csp       string
	ready     atomic.Bool
	handler   http.Handler
}

// New builds the server.
func New(o Options) *Server {
	if o.Logger == nil {
		o.Logger = slog.Default()
	}
	s := &Server{core: o.Core, log: o.Logger, trusted: o.Trusted, web: o.Web, publicURL: o.PublicURL, imageURL: o.ImageURL}
	if u, err := url.Parse(o.PublicURL); err == nil && u.Scheme == "https" {
		s.secure = true
	}
	s.cookie = "pb_session"
	if s.secure {
		s.cookie = "__Host-pb_session"
	}
	s.csp = spaCSP(o.PublicURL, o.ImageURL)
	s.ready.Store(true)
	s.handler = s.routes()
	return s
}

// Handler is the root http.Handler.
func (s *Server) Handler() http.Handler { return s.handler }

// SetReady flips the readiness probe (false during shutdown).
func (s *Server) SetReady(v bool) { s.ready.Store(v) }

// spaCSP builds the SPA Content-Security-Policy; the image origin is added when it differs from the UI origin.
func spaCSP(publicURL, imageURL string) string {
	img := "'self' blob: data:"
	if imageURL != "" && imageURL != publicURL {
		img += " " + imageURL
	}
	return strings.Join([]string{
		"default-src 'self'",
		"img-src " + img,
		"style-src 'self' 'unsafe-inline'",
		"script-src 'self'",
		"connect-src 'self'",
		"object-src 'none'",
		"base-uri 'none'",
		"form-action 'self'",
		"frame-ancestors 'none'",
	}, "; ")
}

func (s *Server) routes() http.Handler {
	mux := http.NewServeMux()

	mux.HandleFunc("GET /healthz", s.handleHealthz)
	mux.HandleFunc("GET /readyz", s.handleReadyz)
	mux.HandleFunc("GET /i/{file}", s.handlePublicImage)

	api := s.apiMux()
	mux.Handle("/api/", api) // "/api" itself is answered by handleRootFile (a method-less "/api" pattern would conflict)

	mux.HandleFunc("GET /assets/{path...}", s.handleAsset)
	for _, p := range []string{"/{$}", "/login", "/setup", "/library", "/trash", "/tokens", "/settings", "/status"} {
		mux.HandleFunc("GET "+p, s.handleIndex)
	}
	mux.HandleFunc("GET /{name}", s.handleRootFile)
	mux.HandleFunc("/", func(w http.ResponseWriter, r *http.Request) { plainNotFound(w) })

	var h http.Handler = mux
	h = s.rejectOddPaths(h)
	h = s.baseHeaders(h)
	h = s.accessLog(h)
	h = s.recoverer(h)
	return h
}

func plainNotFound(w http.ResponseWriter) {
	w.Header().Set("Content-Type", "text/plain; charset=utf-8")
	w.Header().Set("Cache-Control", "no-store")
	w.WriteHeader(http.StatusNotFound)
	_, _ = w.Write([]byte("404 not found\n"))
}
