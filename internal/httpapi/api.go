package httpapi

import (
	"encoding/json"
	"errors"
	"io"
	"mime"
	"net/http"
	"net/url"
	"strings"

	"picberth/internal/core"
)

// call is what an authenticated API handler receives.
type call struct {
	p         core.Principal
	ip        string
	sessionID string // cookie value when the request carried one (valid or not)
}

type apiFunc func(w http.ResponseWriter, r *http.Request, c *call)

func (s *Server) apiMux() http.Handler {
	mux := http.NewServeMux()
	route := func(pattern string, h apiFunc) { mux.HandleFunc(pattern, s.authed(h, true)) }
	open := func(pattern string, h apiFunc) { mux.HandleFunc(pattern, s.authed(h, false)) }

	// Browser session endpoints: cookie only, never bearer tokens.
	open("GET /api/v1/session", s.getSession)
	open("POST /api/v1/session", s.login)
	open("DELETE /api/v1/session", s.logout)
	open("POST /api/v1/setup", s.setup)

	route("POST /api/v1/images", s.uploadImage)
	route("GET /api/v1/images", s.listImages)
	route("POST /api/v1/images/batch", s.batchImages)
	route("GET /api/v1/images/facets", s.imageFacets) // before {id}: "facets" is not an image id
	route("GET /api/v1/images/{id}", s.getImage)
	route("PATCH /api/v1/images/{id}", s.patchImage)
	route("DELETE /api/v1/images/{id}", s.deleteImage)
	route("POST /api/v1/images/{id}/restore", s.restoreImage)
	route("POST /api/v1/images/{id}/purge", s.purgeImage)
	route("GET /api/v1/images/{id}/raw", s.rawImage)
	route("GET /api/v1/images/{id}/thumb", s.thumbImage)

	route("GET /api/v1/tokens", s.listTokens)
	route("POST /api/v1/tokens", s.createToken)
	route("DELETE /api/v1/tokens/{id}", s.revokeToken)

	route("GET /api/v1/folders", s.listFolders)
	route("POST /api/v1/folders", s.createFolder)
	route("PATCH /api/v1/folders/{id}", s.patchFolder)
	route("DELETE /api/v1/folders/{id}", s.deleteFolder)

	route("GET /api/v1/settings", s.getSettings)
	route("PATCH /api/v1/settings", s.patchSettings)
	route("GET /api/v1/status", s.getStatus)
	route("POST /api/v1/checks", s.runCheck)
	route("POST /api/v1/snapshots", s.runSnapshot)
	route("GET /api/v1/audit/actions", s.getAuditActions)
	route("GET /api/v1/audit", s.getAudit)

	// Public, machine-readable contract; no credentials required.
	mux.HandleFunc("GET /api/v1/openapi.json", s.handleOpenAPI)

	mux.HandleFunc("/api/", func(w http.ResponseWriter, r *http.Request) {
		writeProblem(w, core.CodeNotFound, "Not found.", 0)
	})
	mux.HandleFunc("/api", func(w http.ResponseWriter, r *http.Request) {
		writeProblem(w, core.CodeNotFound, "Not found.", 0)
	})

	cop := http.NewCrossOriginProtection()
	if u, err := url.Parse(s.publicURL); err == nil && u.Scheme != "" {
		_ = cop.AddTrustedOrigin(u.Scheme + "://" + u.Host)
	}
	cop.SetDenyHandler(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		apiHeaders(w)
		writeProblem(w, core.CodeForbidden, "Cross-origin request refused.", 0)
	}))
	inner := cop.Handler(mux)
	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		apiHeaders(w)
		inner.ServeHTTP(w, r)
	})
}

func isSafeMethod(m string) bool {
	return m == http.MethodGet || m == http.MethodHead || m == http.MethodOptions
}

// clientIP resolves the real client address and raises the untrusted-forwarded warning when needed.
func (s *Server) clientIP(r *http.Request) string {
	res := s.trusted.ClientIP(r.RemoteAddr, r.Header.Values("X-Forwarded-For"))
	if res.UntrustedForwarded {
		s.core.NoteUntrustedForwarded()
	}
	if res.Malformed {
		s.log.Warn("unusable X-Forwarded-For chain from a trusted proxy; using the peer address", "id", requestID(r))
	}
	if !res.IP.IsValid() {
		return "unknown"
	}
	return res.IP.String()
}

// authed resolves the principal. With tokens=true a Bearer token is honoured (and wins over a cookie);
// session-authenticated unsafe requests must carry X-PicBerth-Client: web.
func (s *Server) authed(h apiFunc, tokens bool) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		c := &call{ip: s.clientIP(r)}
		ri := infoFrom(r)
		ri.ip = c.ip
		ctx := r.Context()

		if ck, err := r.Cookie(s.cookie); err == nil {
			c.sessionID = ck.Value
		}
		if auth := r.Header.Get("Authorization"); auth != "" && tokens {
			scheme, secret, ok := strings.Cut(auth, " ")
			if !ok || !strings.EqualFold(scheme, "Bearer") {
				s.writeError(w, r, &core.Error{Code: core.CodeUnauthorized, Detail: "Use Authorization: Bearer <token>."})
				return
			}
			p, err := s.core.AuthenticateToken(ctx, strings.TrimSpace(secret), c.ip)
			if err != nil {
				s.writeError(w, r, err)
				return
			}
			c.p = p
		} else if c.sessionID != "" {
			p, err := s.core.AuthenticateSession(ctx, c.sessionID, c.ip)
			if err != nil && core.ErrorCode(err) != core.CodeUnauthorized {
				s.writeError(w, r, err)
				return
			}
			if err == nil {
				c.p = p
			}
		}
		ri.principal = c.p.Kind
		if c.p.Kind == core.KindSession && !isSafeMethod(r.Method) && r.Header.Get("X-PicBerth-Client") != "web" {
			writeProblem(w, core.CodeForbidden, "Missing X-PicBerth-Client header.", 0)
			return
		}
		if c.p.Kind == core.KindAnonymous {
			c.p.IP = c.ip
		}
		h(w, r, c)
	}
}

// decodeJSON reads a small JSON body into dst, rejecting unknown fields and trailing data.
func decodeJSON(w http.ResponseWriter, r *http.Request, dst any) error {
	if mt, _, err := mime.ParseMediaType(r.Header.Get("Content-Type")); err != nil || mt != "application/json" {
		return &core.Error{Code: core.CodeValidation, Detail: "Content-Type must be application/json."}
	}
	body := http.MaxBytesReader(w, r.Body, 256<<10)
	dec := json.NewDecoder(body)
	dec.DisallowUnknownFields()
	if err := dec.Decode(dst); err != nil {
		var mbe *http.MaxBytesError
		if errors.As(err, &mbe) {
			return err
		}
		return &core.Error{Code: core.CodeValidation, Detail: "The request body is not valid JSON for this endpoint: " + jsonErrDetail(err)}
	}
	if _, err := dec.Token(); err != io.EOF {
		return &core.Error{Code: core.CodeValidation, Detail: "Unexpected data after the JSON body."}
	}
	return nil
}

func jsonErrDetail(err error) string {
	var syn *json.SyntaxError
	var typ *json.UnmarshalTypeError
	switch {
	case errors.Is(err, io.EOF):
		return "empty body."
	case errors.As(err, &syn):
		return "syntax error."
	case errors.As(err, &typ):
		if typ.Field != "" {
			return "wrong type for field " + typ.Field + "."
		}
		return "wrong type."
	}
	msg := err.Error()
	if strings.HasPrefix(msg, "json: unknown field") {
		return strings.TrimPrefix(msg, "json: ")
	}
	return "invalid body."
}

// nullableString distinguishes an absent JSON field from an explicit null. UnmarshalJSON runs only when
// the field is present, so Set records exactly that.
type nullableString struct {
	Set   bool
	Value *string
}

func (n *nullableString) UnmarshalJSON(b []byte) error {
	n.Set = true
	if string(b) == "null" {
		n.Value = nil
		return nil
	}
	var s string
	if err := json.Unmarshal(b, &s); err != nil {
		return err
	}
	n.Value = &s
	return nil
}

// requireAdmin answers 401/403 before any request body is parsed. core re-checks every operation.
func (s *Server) requireAdmin(w http.ResponseWriter, r *http.Request, c *call) bool {
	if c.p.IsAdmin() {
		return true
	}
	s.writeError(w, r, denyFor(c.p, "This operation requires an interactive administrator session."))
	return false
}

func (s *Server) requireManage(w http.ResponseWriter, r *http.Request, c *call) bool {
	if c.p.CanManage() {
		return true
	}
	s.writeError(w, r, denyFor(c.p, "This operation requires the manage scope."))
	return false
}

func denyFor(p core.Principal, detail string) error {
	if p.Kind == core.KindAnonymous {
		return &core.Error{Code: core.CodeUnauthorized, Detail: "Authentication required or credentials invalid."}
	}
	return &core.Error{Code: core.CodeForbidden, Detail: detail}
}
