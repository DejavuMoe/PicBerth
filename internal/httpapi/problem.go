package httpapi

import (
	"encoding/json"
	"errors"
	"net/http"
	"strconv"

	"picberth/internal/core"
)

type problem struct {
	Type   string `json:"type"`
	Title  string `json:"title"`
	Status int    `json:"status"`
	Detail string `json:"detail,omitempty"`
	Code   string `json:"code"`
}

var problemMeta = map[string]struct {
	status int
	title  string
}{
	core.CodeValidation:          {http.StatusBadRequest, "Invalid request"},
	core.CodeUnauthorized:        {http.StatusUnauthorized, "Authentication required"},
	core.CodeForbidden:           {http.StatusForbidden, "Forbidden"},
	core.CodeNotFound:            {http.StatusNotFound, "Not found"},
	core.CodeConflict:            {http.StatusConflict, "Conflict"},
	core.CodeTooLarge:            {http.StatusRequestEntityTooLarge, "File too large"},
	core.CodeUnsupportedFormat:   {http.StatusUnsupportedMediaType, "Unsupported format"},
	core.CodeInvalidImage:        {http.StatusUnprocessableEntity, "Invalid image"},
	core.CodeTooManyPixels:       {http.StatusUnprocessableEntity, "Image too large in pixels"},
	core.CodeSetupRequired:       {http.StatusConflict, "Setup required"},
	core.CodeSetupDone:           {http.StatusConflict, "Setup already completed"},
	core.CodeRateLimited:         {http.StatusTooManyRequests, "Too many requests"},
	core.CodeInsufficientStorage: {http.StatusInsufficientStorage, "Insufficient storage"},
	core.CodeFolderExists:        {http.StatusConflict, "Folder already exists"},
	core.CodeFolderNotEmpty:      {http.StatusConflict, "Folder not empty"},
	core.CodeFolderDepth:         {http.StatusUnprocessableEntity, "Folder depth exceeded"},
	core.CodeInvalidScope:        {http.StatusUnprocessableEntity, "Invalid token scope"},
	core.CodeInternal:            {http.StatusInternalServerError, "Internal error"},
}

// writeProblem sends an RFC 9457 problem document for a stable code.
func writeProblem(w http.ResponseWriter, code, detail string, retryAfterSecs int) {
	m, ok := problemMeta[code]
	if !ok {
		code = core.CodeInternal
		m = problemMeta[code]
	}
	if retryAfterSecs > 0 {
		w.Header().Set("Retry-After", strconv.Itoa(retryAfterSecs))
	}
	w.Header().Set("Content-Type", "application/problem+json")
	w.WriteHeader(m.status)
	_ = json.NewEncoder(w).Encode(problem{Type: "urn:picberth:problem:" + code, Title: m.title, Status: m.status, Detail: detail, Code: code})
}

// writeError maps any error to a problem response. Unexpected errors are logged with the request id and
// never expose internals.
func (s *Server) writeError(w http.ResponseWriter, r *http.Request, err error) {
	var ce *core.Error
	if errors.As(err, &ce) {
		secs := 0
		if ce.RetryAfter > 0 {
			secs = int((ce.RetryAfter + 999_999_999) / 1_000_000_000)
		}
		if ce.Code == core.CodeUnauthorized {
			w.Header().Add("WWW-Authenticate", `Bearer realm="picberth"`)
		}
		writeProblem(w, ce.Code, ce.Detail, secs)
		return
	}
	var mbe *http.MaxBytesError
	if errors.As(err, &mbe) {
		writeProblem(w, core.CodeTooLarge, "The request body is too large.", 0)
		return
	}
	s.log.Error("request failed", "id", requestID(r), "method", r.Method, "path", r.URL.Path, "err", err)
	writeProblem(w, core.CodeInternal, "An unexpected error occurred.", 0)
}

func writeJSON(w http.ResponseWriter, status int, v any) {
	w.Header().Set("Content-Type", "application/json")
	w.WriteHeader(status)
	_ = json.NewEncoder(w).Encode(v)
}
