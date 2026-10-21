package httpapi

import (
	"net/http"
	"strconv"

	"picberth/internal/core"
)

func (s *Server) listTokens(w http.ResponseWriter, r *http.Request, c *call) {
	items, err := s.core.ListTokens(r.Context(), c.p)
	if err != nil {
		s.writeError(w, r, err)
		return
	}
	if items == nil {
		items = []core.TokenView{}
	}
	writeJSON(w, http.StatusOK, map[string]any{"items": items})
}

func (s *Server) createToken(w http.ResponseWriter, r *http.Request, c *call) {
	if !s.requireAdmin(w, r, c) {
		return
	}
	var req struct {
		Name          string   `json:"name"`
		Scopes        []string `json:"scopes"`
		ExpiresInDays *int     `json:"expiresInDays"`
	}
	if err := decodeJSON(w, r, &req); err != nil {
		s.writeError(w, r, err)
		return
	}
	tv, secret, err := s.core.CreateToken(r.Context(), c.p, req.Name, req.Scopes, req.ExpiresInDays)
	if err != nil {
		s.writeError(w, r, err)
		return
	}
	writeJSON(w, http.StatusCreated, map[string]any{"token": tv, "secret": secret})
}

func (s *Server) revokeToken(w http.ResponseWriter, r *http.Request, c *call) {
	id, err := strconv.ParseInt(r.PathValue("id"), 10, 64)
	if err != nil || id < 1 {
		// authorization comes first so non-admins cannot probe, then the id shape
		if !s.requireAdmin(w, r, c) {
			return
		}
		s.writeError(w, r, &core.Error{Code: core.CodeNotFound, Detail: "Not found."})
		return
	}
	if err := s.core.RevokeToken(r.Context(), c.p, id); err != nil {
		s.writeError(w, r, err)
		return
	}
	w.WriteHeader(http.StatusNoContent)
}

func (s *Server) getSettings(w http.ResponseWriter, r *http.Request, c *call) {
	v, err := s.core.GetSettings(c.p)
	if err != nil {
		s.writeError(w, r, err)
		return
	}
	writeJSON(w, http.StatusOK, v)
}

func (s *Server) patchSettings(w http.ResponseWriter, r *http.Request, c *call) {
	if !s.requireAdmin(w, r, c) {
		return
	}
	var patch core.SettingsPatch
	if err := decodeJSON(w, r, &patch); err != nil {
		s.writeError(w, r, err)
		return
	}
	v, err := s.core.UpdateSettings(r.Context(), c.p, patch)
	if err != nil {
		s.writeError(w, r, err)
		return
	}
	writeJSON(w, http.StatusOK, v)
}

func (s *Server) getStatus(w http.ResponseWriter, r *http.Request, c *call) {
	st, err := s.core.Status(r.Context(), c.p)
	if err != nil {
		s.writeError(w, r, err)
		return
	}
	writeJSON(w, http.StatusOK, st)
}

func (s *Server) runCheck(w http.ResponseWriter, r *http.Request, c *call) {
	var req struct {
		Mode string `json:"mode"`
	}
	if !s.requireAdmin(w, r, c) {
		return
	}
	if err := decodeJSON(w, r, &req); err != nil {
		s.writeError(w, r, err)
		return
	}
	extendDeadlines(w, 0, -1) // a full check hashes every file
	rep, err := s.core.Check(r.Context(), c.p, req.Mode)
	if err != nil {
		s.writeError(w, r, err)
		return
	}
	writeJSON(w, http.StatusOK, rep)
}

func (s *Server) runSnapshot(w http.ResponseWriter, r *http.Request, c *call) {
	extendDeadlines(w, 0, -1)
	info, err := s.core.Snapshot(r.Context(), c.p, "")
	if err != nil {
		s.writeError(w, r, err)
		return
	}
	writeJSON(w, http.StatusCreated, info)
}

func (s *Server) getAudit(w http.ResponseWriter, r *http.Request, c *call) {
	limit := 0
	if v := r.URL.Query().Get("limit"); v != "" {
		n, err := strconv.Atoi(v)
		if err != nil || n == 0 {
			s.writeError(w, r, &core.Error{Code: core.CodeValidation, Detail: "limit must be between 1 and 200."})
			return
		}
		limit = n
	}
	items, next, err := s.core.ListAudit(r.Context(), c.p, limit, r.URL.Query().Get("cursor"))
	if err != nil {
		s.writeError(w, r, err)
		return
	}
	if items == nil {
		items = []core.AuditItem{}
	}
	var nextCursor any
	if next != "" {
		nextCursor = next
	}
	writeJSON(w, http.StatusOK, map[string]any{"items": items, "nextCursor": nextCursor})
}
