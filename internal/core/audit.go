package core

import (
	"context"
	"strconv"
	"strings"
	"time"
	"unicode"
	"unicode/utf8"

	"picberth/internal/meta"
)

// audit records an action. A failing audit write is logged but never fails the operation it describes.
func (s *Service) audit(ctx context.Context, p Principal, action, target, detail string) {
	s.auditActor(ctx, p.Actor(), p.IP, action, target, detail)
}

func (s *Service) auditActor(ctx context.Context, actor, ip, action, target, detail string) {
	// Detached from request cancellation: an audit row must survive a client disconnect.
	if ctx.Err() != nil {
		ctx = context.WithoutCancel(ctx)
	}
	err := s.db.AddAudit(ctx, meta.AuditEntry{At: s.nowMs(), Actor: actor, Action: action, Target: target, Detail: detail, IP: ip})
	if err != nil {
		s.log.Error("audit write failed", "action", action, "err", err)
	}
}

// printable keeps attacker-controlled text (login names, file names) safe for logs and audit rows.
func printable(s string, maxRunes int) string {
	s = strings.ToValidUTF8(s, "")
	var b strings.Builder
	n := 0
	for _, r := range s {
		if unicode.IsControl(r) || unicode.Is(unicode.Cf, r) || r == ' ' || r == ' ' {
			continue
		}
		if n++; n > maxRunes {
			break
		}
		b.WriteRune(r)
	}
	return b.String()
}

// AuditItem is one audit log row as served by the API.
type AuditItem struct {
	ID     int64  `json:"id"`
	At     string `json:"at"`
	Actor  string `json:"actor"`
	Action string `json:"action"`
	Target string `json:"target"`
	Detail string `json:"detail"`
	IP     string `json:"ip"`
}

// AuditQuery filters the audit log: Action keeps exactly one action string, Kind keeps one category of
// AuditKinds. Both may be combined; empty means no filter.
type AuditQuery struct {
	Action string
	Kind   string
}

// AuditKinds groups action strings into the console's filter categories (docs/api.md §7).
var AuditKinds = map[string]meta.AuditFilter{
	"image":    {Prefixes: []string{"image."}},
	"folder":   {Prefixes: []string{"folder."}},
	"token":    {Prefixes: []string{"token."}},
	"settings": {Prefixes: []string{"settings."}},
	"system":   {Prefixes: []string{"check.", "admin."}, Exact: []string{"snapshot"}},
	"session":  {Prefixes: []string{"login."}, Exact: []string{"logout", "setup", "setup.failure"}},
}

// ListAudit pages the audit log, newest first, filtered by q. The cursor is the id of the last item of
// the previous page.
func (s *Service) ListAudit(ctx context.Context, p Principal, q AuditQuery, limit int, cursor string) ([]AuditItem, string, error) {
	if !p.IsAdmin() {
		return nil, "", s.denyAdmin(p)
	}
	filter := meta.AuditFilter{Action: q.Action}
	if q.Kind != "" {
		k, ok := AuditKinds[q.Kind]
		if !ok {
			return nil, "", validation("Unknown kind %q (valid: image, folder, token, settings, system, session).", q.Kind)
		}
		filter.Exact, filter.Prefixes = k.Exact, k.Prefixes
	}
	if limit == 0 {
		limit = 10
	}
	if limit < 1 || limit > 100 {
		return nil, "", validation("limit must be between 1 and 100.")
	}
	var before int64
	if cursor != "" {
		n, err := strconv.ParseInt(cursor, 10, 64)
		if err != nil || n < 1 {
			return nil, "", validation("Invalid cursor.")
		}
		before = n
	}
	rows, err := s.db.ListAudit(ctx, before, filter, limit+1)
	if err != nil {
		return nil, "", err
	}
	next := ""
	if len(rows) > limit {
		rows = rows[:limit]
		next = strconv.FormatInt(rows[limit-1].ID, 10)
	}
	items := make([]AuditItem, len(rows))
	for i, r := range rows {
		items[i] = AuditItem{ID: r.ID, At: fmtTime(r.At), Actor: r.Actor, Action: r.Action, Target: r.Target, Detail: r.Detail, IP: r.IP}
	}
	return items, next, nil
}

// ListAuditActions returns the distinct action strings in the log, for the console's filter.
func (s *Service) ListAuditActions(ctx context.Context, p Principal) ([]string, error) {
	if !p.IsAdmin() {
		return nil, s.denyAdmin(p)
	}
	return s.db.AuditActions(ctx)
}

// PruneExpiredAudit deletes audit rows older than the retention setting in batches. It writes no audit
// row of its own: the job only logs the count.
func (s *Service) PruneExpiredAudit(ctx context.Context) (int64, error) {
	days := s.Settings().AuditRetentionDays
	cutoff := s.now().Add(-time.Duration(days) * 24 * time.Hour).UnixMilli()
	var total int64
	for {
		if err := ctx.Err(); err != nil {
			return total, err
		}
		n, err := s.db.DeleteAuditBefore(ctx, cutoff, 1000)
		if err != nil {
			return total, err
		}
		total += n
		if n < 1000 {
			return total, nil
		}
	}
}

// utf8Len counts characters, which is what the API limits mean.
func utf8Len(s string) int { return utf8.RuneCountInString(s) }
