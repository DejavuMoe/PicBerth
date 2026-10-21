package httpapi

import (
	"net/http"

	"picberth/internal/buildinfo"
	"picberth/internal/core"
)

type sessionResponse struct {
	Authenticated bool                `json:"authenticated"`
	SetupRequired bool                `json:"setupRequired"`
	User          *sessionUser        `json:"user"`
	Limits        *core.SessionLimits `json:"limits"`
	Version       string              `json:"version"`
}

type sessionUser struct {
	Name string `json:"name"`
}

func (s *Server) getSession(w http.ResponseWriter, r *http.Request, c *call) {
	st, err := s.core.SessionInfo(r.Context(), c.p)
	if err != nil {
		s.writeError(w, r, err)
		return
	}
	resp := sessionResponse{Authenticated: st.Authenticated, SetupRequired: st.SetupRequired, Limits: st.Limits, Version: buildinfo.Version}
	if st.Authenticated {
		resp.User = &sessionUser{Name: st.Username}
	}
	writeJSON(w, http.StatusOK, resp)
}

func (s *Server) setCookie(w http.ResponseWriter, sess core.NewSession) {
	http.SetCookie(w, &http.Cookie{
		Name: s.cookie, Value: sess.ID, Path: "/", MaxAge: int(sess.MaxAge.Seconds()),
		HttpOnly: true, Secure: s.secure, SameSite: http.SameSiteStrictMode,
	})
}

func (s *Server) clearCookie(w http.ResponseWriter) {
	http.SetCookie(w, &http.Cookie{
		Name: s.cookie, Value: "", Path: "/", MaxAge: -1,
		HttpOnly: true, Secure: s.secure, SameSite: http.SameSiteStrictMode,
	})
}

func (s *Server) login(w http.ResponseWriter, r *http.Request, c *call) {
	var req struct {
		Username string `json:"username"`
		Password string `json:"password"`
	}
	if err := decodeJSON(w, r, &req); err != nil {
		s.writeError(w, r, err)
		return
	}
	sess, err := s.core.Login(r.Context(), c.ip, r.UserAgent(), req.Username, req.Password)
	if err != nil {
		s.writeError(w, r, err)
		return
	}
	s.setCookie(w, sess)
	w.WriteHeader(http.StatusNoContent)
}

func (s *Server) logout(w http.ResponseWriter, r *http.Request, c *call) {
	if err := s.core.Logout(r.Context(), c.p, c.sessionID); err != nil {
		s.writeError(w, r, err)
		return
	}
	s.clearCookie(w)
	w.WriteHeader(http.StatusNoContent)
}

func (s *Server) setup(w http.ResponseWriter, r *http.Request, c *call) {
	var req struct {
		SetupCode string `json:"setupCode"`
		Username  string `json:"username"`
		Password  string `json:"password"`
	}
	if err := decodeJSON(w, r, &req); err != nil {
		s.writeError(w, r, err)
		return
	}
	sess, err := s.core.Setup(r.Context(), c.ip, r.UserAgent(), req.SetupCode, req.Username, req.Password)
	if err != nil {
		s.writeError(w, r, err)
		return
	}
	s.setCookie(w, sess)
	w.WriteHeader(http.StatusNoContent)
}
