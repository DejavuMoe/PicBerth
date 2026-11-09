package core

import (
	"fmt"
	"slices"
	"strings"

	"picberth/internal/meta"
)

// Kind says how a request was authenticated.
type Kind int

const (
	KindAnonymous Kind = iota
	KindSession
	KindToken
	KindCLI // local operator commands
)

func (k Kind) String() string {
	switch k {
	case KindSession:
		return "session"
	case KindToken:
		return "token"
	case KindCLI:
		return "cli"
	}
	return "anonymous"
}

// Token scopes.
const (
	ScopeUpload = "upload"
	ScopeManage = "manage"
)

var allScopes = []string{ScopeUpload, ScopeManage}

// Principal is who is acting. Every Service method authorizes against it.
type Principal struct {
	Kind      Kind
	UserID    int64
	Username  string
	TokenID   int64
	TokenName string
	Scopes    []string
	IP        string // client address, for the audit log
}

// CLIPrincipal is the local operator.
func CLIPrincipal() Principal { return Principal{Kind: KindCLI, Username: "cli"} }

// Actor is the audit-log name of the principal.
func (p Principal) Actor() string {
	switch p.Kind {
	case KindSession:
		return "user:" + p.Username
	case KindToken:
		return fmt.Sprintf("token:%d:%s", p.TokenID, p.TokenName)
	case KindCLI:
		if p.Username != "" {
			return p.Username
		}
		return "cli"
	}
	return "anonymous"
}

// IsAdmin is true for the interactive administrator (browser session or local CLI). Only they can manage
// tokens, settings, checks, snapshots and the audit log.
func (p Principal) IsAdmin() bool { return p.Kind == KindSession || p.Kind == KindCLI }

func (p Principal) hasScope(s string) bool {
	return p.Kind == KindToken && slices.Contains(p.Scopes, s)
}

// CanUpload: sessions, CLI and tokens with upload or manage.
func (p Principal) CanUpload() bool {
	return p.IsAdmin() || p.hasScope(ScopeUpload) || p.hasScope(ScopeManage)
}

// CanReadAll: may list and read every image of the owner.
func (p Principal) CanReadAll() bool {
	return p.IsAdmin() || p.hasScope(ScopeManage)
}

// CanManage: may change images, folders, trash, restore and purge.
func (p Principal) CanManage() bool { return p.IsAdmin() || p.hasScope(ScopeManage) }

// canReadOwnUploads: an upload-only token sees just what it uploaded itself.
func (p Principal) canReadOwnUploads() bool { return p.hasScope(ScopeUpload) }

// dedupeScope is the record set an upload may deduplicate against: one folder of the owner-scoped
// library the principal can see, i.e. its own uploads for an upload-only token.
func (p Principal) dedupeScope(ownerID int64, folderID *string) meta.DupScope {
	_, token := p.listScope()
	return meta.DupScope{OwnerID: ownerID, TokenID: token, FolderID: folderID}
}

// folderOwner is the owner whose folders the principal works in (nil = every owner, CLI only).
func (p Principal) folderOwner() *int64 {
	if p.Kind == KindCLI && p.UserID == 0 {
		return nil
	}
	uid := p.UserID
	return &uid
}

// canSee applies ownership and scope rules to a record. Callers answer "not found" when it is false.
func (p Principal) canSee(im *meta.Image) bool {
	if im.State == meta.StatePurged {
		return false
	}
	if im.State == meta.StateTrashed && !p.CanManage() {
		return false
	}
	switch p.Kind {
	case KindCLI:
		return true
	case KindSession:
		return im.OwnerID == p.UserID
	case KindToken:
		if im.OwnerID != p.UserID {
			return false
		}
		if p.CanReadAll() {
			return true
		}
		return p.canReadOwnUploads() && im.TokenID != nil && *im.TokenID == p.TokenID
	}
	return false
}

func parseScopes(csv string) []string {
	if csv == "" {
		return nil
	}
	return strings.Split(csv, ",")
}

// normalizeScopes validates, deduplicates and orders scopes canonically.
func normalizeScopes(in []string) ([]string, error) {
	if len(in) == 0 {
		return nil, validation("At least one scope is required.")
	}
	var out []string
	for _, want := range allScopes {
		if slices.Contains(in, want) {
			out = append(out, want)
		}
	}
	for _, s := range in {
		if !slices.Contains(allScopes, s) {
			return nil, newErr(CodeInvalidScope, "Unknown scope %q (valid: upload, manage).", s)
		}
	}
	return out, nil
}
