package httpapi

import (
	"net/http"

	"picberth/internal/core"
)

func (s *Server) listFolders(w http.ResponseWriter, r *http.Request, c *call) {
	items, err := s.core.ListFolders(r.Context(), c.p)
	if err != nil {
		s.writeError(w, r, err)
		return
	}
	if items == nil {
		items = []core.FolderView{}
	}
	writeJSON(w, http.StatusOK, map[string]any{"items": items})
}

func (s *Server) createFolder(w http.ResponseWriter, r *http.Request, c *call) {
	if !s.requireManage(w, r, c) {
		return
	}
	var req struct {
		Name     string         `json:"name"`
		ParentID nullableString `json:"parentId"`
	}
	if err := decodeJSON(w, r, &req); err != nil {
		s.writeError(w, r, err)
		return
	}
	v, err := s.core.CreateFolder(r.Context(), c.p, req.Name, req.ParentID.Value)
	if err != nil {
		s.writeError(w, r, err)
		return
	}
	writeJSON(w, http.StatusCreated, v)
}

func (s *Server) patchFolder(w http.ResponseWriter, r *http.Request, c *call) {
	if !s.requireManage(w, r, c) {
		return
	}
	var req struct {
		Name     *string        `json:"name"`
		ParentID nullableString `json:"parentId"`
	}
	if err := decodeJSON(w, r, &req); err != nil {
		s.writeError(w, r, err)
		return
	}
	v, err := s.core.UpdateFolder(r.Context(), c.p, r.PathValue("id"), core.FolderPatch{
		Name: req.Name, ParentID: req.ParentID.Value, Move: req.ParentID.Set,
	})
	if err != nil {
		s.writeError(w, r, err)
		return
	}
	writeJSON(w, http.StatusOK, v)
}

func (s *Server) deleteFolder(w http.ResponseWriter, r *http.Request, c *call) {
	if !s.requireManage(w, r, c) {
		return
	}
	if err := s.core.DeleteFolder(r.Context(), c.p, r.PathValue("id")); err != nil {
		s.writeError(w, r, err)
		return
	}
	w.WriteHeader(http.StatusNoContent)
}
