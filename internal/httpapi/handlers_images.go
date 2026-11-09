package httpapi

import (
	"errors"
	"io"
	"mime"
	"net/http"
	"strconv"
	"strings"
	"time"

	"picberth/internal/core"
)

const multipartSlack = 64 << 10 // headroom for multipart framing on top of maxUploadBytes

const maxFormParts = 8

// maxFolderPartBytes bounds the `folder` part: a 16-character id plus surrounding blanks.
const maxFolderPartBytes = 64

func (s *Server) uploadImage(w http.ResponseWriter, r *http.Request, c *call) {
	ctx := r.Context()
	maxBytes := s.core.Settings().MaxUploadBytes
	d := core.UploadTimeout(maxBytes)
	extendDeadlines(w, d, d+30*time.Second)

	up, err := s.core.BeginUpload(ctx, c.p)
	if err != nil {
		s.writeError(w, r, err)
		return
	}
	defer up.Close()

	if r.ContentLength > maxBytes+multipartSlack {
		s.writeError(w, r, &core.Error{Code: core.CodeTooLarge, Detail: "The request exceeds the upload size limit."})
		return
	}
	if mt, _, err := mime.ParseMediaType(r.Header.Get("Content-Type")); err != nil || mt != "multipart/form-data" {
		s.writeError(w, r, &core.Error{Code: core.CodeValidation, Detail: "Content-Type must be multipart/form-data."})
		return
	}
	r.Body = http.MaxBytesReader(w, r.Body, maxBytes+multipartSlack)
	mr, err := r.MultipartReader() // streaming; never ParseMultipartForm (it would spill to the OS temp dir)
	if err != nil {
		s.writeError(w, r, &core.Error{Code: core.CodeValidation, Detail: "Malformed multipart request."})
		return
	}
	parts := 0
	for {
		part, err := mr.NextPart()
		if err == io.EOF {
			break
		}
		if err != nil {
			s.writeError(w, r, multipartError(err))
			return
		}
		if part.FormName() != "tag" { // legacy `tag` parts are ignored entirely, including the part limit
			parts++
			if parts > maxFormParts {
				s.writeError(w, r, &core.Error{Code: core.CodeValidation, Detail: "Too many form parts."})
				return
			}
		}
		switch part.FormName() {
		case "file":
			if up.HasFile() {
				s.writeError(w, r, &core.Error{Code: core.CodeValidation, Detail: "Exactly one file part is allowed."})
				return
			}
			if err := up.WriteFile(ctx, part, part.FileName()); err != nil {
				s.writeError(w, r, uploadReadError(err))
				return
			}
		case "folder":
			v, err := io.ReadAll(io.LimitReader(part, maxFolderPartBytes+1))
			if err != nil {
				s.writeError(w, r, multipartError(err))
				return
			}
			if len(v) > maxFolderPartBytes {
				s.writeError(w, r, &core.Error{Code: core.CodeValidation, Detail: "The folder value is too long."})
				return
			}
			up.SetFolder(string(v))
		}
		// anything else is ignored; NextPart skips the unread remainder
	}
	res, err := up.Commit(ctx)
	if err != nil {
		s.writeError(w, r, err)
		return
	}
	if res.Duplicate {
		writeJSON(w, http.StatusOK, res.Image)
		return
	}
	w.Header().Set("Location", "/api/v1/images/"+res.Image.ID)
	writeJSON(w, http.StatusCreated, res.Image)
}

// multipartError classifies failures of the multipart reader itself: an over-long body is a 413,
// anything else (malformed framing, aborted client, read deadline) is the client's problem.
func multipartError(err error) error {
	var mbe *http.MaxBytesError
	if errors.As(err, &mbe) {
		return &core.Error{Code: core.CodeTooLarge, Detail: "The upload exceeds the size limit."}
	}
	return &core.Error{Code: core.CodeValidation, Detail: "Malformed or interrupted multipart request."}
}

// uploadReadError separates oversized or broken request bodies from domain and I/O errors.
func uploadReadError(err error) error {
	var ce *core.Error
	if errors.As(err, &ce) {
		return err
	}
	var mbe *http.MaxBytesError
	if errors.As(err, &mbe) {
		return &core.Error{Code: core.CodeTooLarge, Detail: "The upload exceeds the size limit."}
	}
	var sre *core.SourceReadError
	if errors.As(err, &sre) {
		return &core.Error{Code: core.CodeValidation, Detail: "The upload could not be read (interrupted or malformed)."}
	}
	if strings.HasPrefix(err.Error(), "multipart:") || errors.Is(err, io.ErrUnexpectedEOF) {
		return &core.Error{Code: core.CodeValidation, Detail: "Malformed multipart request."}
	}
	return err
}

func (s *Server) listImages(w http.ResponseWriter, r *http.Request, c *call) {
	q := r.URL.Query()
	lp := core.ListParams{
		State: q.Get("state"), Q: q.Get("q"), Formats: q["format"], Sources: q["source"],
		Folder: q.Get("folder"), Sort: q.Get("sort"), Cursor: q.Get("cursor"),
	}
	if v := q.Get("recursive"); v != "" {
		switch v {
		case "true":
			lp.Recursive = true
		case "false":
		default:
			s.writeError(w, r, &core.Error{Code: core.CodeValidation, Detail: "recursive must be true or false."})
			return
		}
	}
	for name, dst := range map[string]**time.Time{"since": &lp.Since, "until": &lp.Until} {
		if v := q.Get(name); v != "" {
			t, err := time.Parse(time.RFC3339, v)
			if err != nil {
				s.writeError(w, r, &core.Error{Code: core.CodeValidation, Detail: name + " must be an RFC 3339 timestamp."})
				return
			}
			*dst = &t
		}
	}
	if v := q.Get("tokenId"); v != "" {
		n, err := strconv.ParseInt(v, 10, 64)
		if err != nil || n < 1 {
			s.writeError(w, r, &core.Error{Code: core.CodeValidation, Detail: "tokenId must be an integer."})
			return
		}
		lp.TokenID = &n
	}
	if v := q.Get("limit"); v != "" {
		n, err := strconv.Atoi(v)
		if err != nil {
			s.writeError(w, r, &core.Error{Code: core.CodeValidation, Detail: "limit must be an integer."})
			return
		}
		lp.Limit = n
		if n == 0 {
			lp.Limit = -1 // 0 is not a valid explicit limit
		}
	}
	items, next, err := s.core.ListImages(r.Context(), c.p, lp)
	if err != nil {
		s.writeError(w, r, err)
		return
	}
	var nextCursor any
	if next != "" {
		nextCursor = next
	}
	if items == nil {
		items = []core.ImageView{}
	}
	writeJSON(w, http.StatusOK, map[string]any{"items": items, "nextCursor": nextCursor})
}

func (s *Server) imageFacets(w http.ResponseWriter, r *http.Request, c *call) {
	f, err := s.core.ImageFacets(r.Context(), c.p, r.URL.Query().Get("state"))
	if err != nil {
		s.writeError(w, r, err)
		return
	}
	writeJSON(w, http.StatusOK, f)
}

func (s *Server) getImage(w http.ResponseWriter, r *http.Request, c *call) {
	v, err := s.core.GetImage(r.Context(), c.p, r.PathValue("id"))
	if err != nil {
		s.writeError(w, r, err)
		return
	}
	writeJSON(w, http.StatusOK, v)
}

func (s *Server) patchImage(w http.ResponseWriter, r *http.Request, c *call) {
	if !s.requireManage(w, r, c) {
		return
	}
	var req struct {
		Filename *string        `json:"filename"`
		FolderID nullableString `json:"folderId"`
	}
	if err := decodeJSON(w, r, &req); err != nil {
		s.writeError(w, r, err)
		return
	}
	v, err := s.core.PatchImage(r.Context(), c.p, r.PathValue("id"), core.ImagePatch{
		Filename: req.Filename, FolderID: req.FolderID.Value, MoveFolder: req.FolderID.Set,
	})
	if err != nil {
		s.writeError(w, r, err)
		return
	}
	writeJSON(w, http.StatusOK, v)
}

func (s *Server) deleteImage(w http.ResponseWriter, r *http.Request, c *call) {
	if err := s.core.Trash(r.Context(), c.p, r.PathValue("id")); err != nil {
		s.writeError(w, r, err)
		return
	}
	w.WriteHeader(http.StatusNoContent)
}

func (s *Server) restoreImage(w http.ResponseWriter, r *http.Request, c *call) {
	v, err := s.core.Restore(r.Context(), c.p, r.PathValue("id"))
	if err != nil {
		s.writeError(w, r, err)
		return
	}
	writeJSON(w, http.StatusOK, v)
}

func (s *Server) purgeImage(w http.ResponseWriter, r *http.Request, c *call) {
	if err := s.core.Purge(r.Context(), c.p, r.PathValue("id")); err != nil {
		s.writeError(w, r, err)
		return
	}
	w.WriteHeader(http.StatusNoContent)
}

func (s *Server) batchImages(w http.ResponseWriter, r *http.Request, c *call) {
	if !s.requireManage(w, r, c) {
		return
	}
	var req struct {
		Action   string         `json:"action"`
		IDs      []string       `json:"ids"`
		FolderID nullableString `json:"folderId"`
	}
	if err := decodeJSON(w, r, &req); err != nil {
		s.writeError(w, r, err)
		return
	}
	res, err := s.core.Batch(r.Context(), c.p, req.Action, req.IDs, req.FolderID.Value)
	if err != nil {
		s.writeError(w, r, err)
		return
	}
	writeJSON(w, http.StatusOK, map[string]any{"results": res})
}

func (s *Server) rawImage(w http.ResponseWriter, r *http.Request, c *call) {
	o, err := s.core.OpenRaw(r.Context(), c.p, r.PathValue("id"))
	if err != nil {
		s.writeError(w, r, err)
		return
	}
	s.serveOpened(w, r, o, o.Image.ID+"."+o.Image.Ext)
}

func (s *Server) thumbImage(w http.ResponseWriter, r *http.Request, c *call) {
	o, err := s.core.OpenThumb(r.Context(), c.p, r.PathValue("id"))
	if err != nil {
		s.writeError(w, r, err)
		return
	}
	ext := "jpg"
	if o.ContentType == "image/png" {
		ext = "png"
	}
	s.serveOpened(w, r, o, o.Image.ID+"-thumb."+ext)
}

// serveOpened sends an authenticated image body: private, uncached, same-origin only.
func (s *Server) serveOpened(w http.ResponseWriter, r *http.Request, o *core.Opened, name string) {
	defer o.File.Close()
	h := w.Header()
	h.Set("Content-Type", o.ContentType)
	h.Set("Cache-Control", "private, no-store")
	h.Set("Cross-Origin-Resource-Policy", "same-origin")
	h.Set("Content-Security-Policy", "default-src 'none'; sandbox")
	h.Set("Content-Disposition", `inline; filename="`+name+`"`)
	extendDeadlines(w, 0, writeTimeoutFor(o.Size))
	http.ServeContent(w, r, "", o.ModTime, o.File)
}

// writeTimeoutFor allows slow downloads: at least 60 s, otherwise size at 32 KiB/s.
func writeTimeoutFor(size int64) time.Duration {
	d := time.Duration(size/(32<<10)) * time.Second
	if d < time.Minute {
		d = time.Minute
	}
	return d
}
