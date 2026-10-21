package core

import (
	"context"
	"errors"
	"strings"
	"time"

	"picberth/internal/ids"
	"picberth/internal/meta"
)

// TokenView is the API representation of a token (never the secret).
type TokenView struct {
	ID         int64    `json:"id"`
	Name       string   `json:"name"`
	Prefix     string   `json:"prefix"`
	Scopes     []string `json:"scopes"`
	CreatedAt  string   `json:"createdAt"`
	ExpiresAt  *string  `json:"expiresAt"`
	LastUsedAt *string  `json:"lastUsedAt"`
	RevokedAt  *string  `json:"revokedAt"`
}

func tokenView(t meta.Token) TokenView {
	return TokenView{
		ID: t.ID, Name: t.Name, Prefix: t.Prefix, Scopes: parseScopes(t.Scopes), CreatedAt: fmtTime(t.CreatedAt),
		ExpiresAt: fmtTimePtr(t.ExpiresAt), LastUsedAt: fmtTimePtr(t.LastUsedAt), RevokedAt: fmtTimePtr(t.RevokedAt),
	}
}

// CreateToken issues a token; the secret is returned once and never stored. Only an administrator session can do this.
func (s *Service) CreateToken(ctx context.Context, p Principal, name string, scopes []string, expiresInDays *int) (TokenView, string, error) {
	if !p.IsAdmin() {
		return TokenView{}, "", s.denyAdmin(p)
	}
	name = strings.TrimSpace(name)
	if n := utf8Len(name); n < 1 || n > 80 {
		return TokenView{}, "", validation("Token name must be 1-80 characters.")
	}
	if printable(name, 1000) != name {
		return TokenView{}, "", validation("Token name must not contain control characters.")
	}
	norm, err := normalizeScopes(scopes)
	if err != nil {
		return TokenView{}, "", err
	}
	var expires *int64
	if expiresInDays != nil {
		if *expiresInDays < 1 || *expiresInDays > 3650 {
			return TokenView{}, "", validation("expiresInDays must be between 1 and 3650 or null.")
		}
		e := s.now().Add(time.Duration(*expiresInDays) * 24 * time.Hour).UnixMilli()
		expires = &e
	}
	secret := ids.NewTokenSecret()
	t, err := s.db.CreateToken(ctx, meta.Token{
		OwnerID: p.UserID, Name: name, Prefix: ids.TokenDisplay(secret), SecretHash: ids.Hash(secret),
		Scopes: strings.Join(norm, ","), CreatedAt: s.nowMs(), ExpiresAt: expires,
	})
	if err != nil {
		return TokenView{}, "", err
	}
	s.audit(ctx, p, "token.create", itoa(t.ID), "name: "+name+"; scopes: "+t.Scopes)
	return tokenView(*t), secret, nil
}

// ListTokens returns the owner's tokens, newest first.
func (s *Service) ListTokens(ctx context.Context, p Principal) ([]TokenView, error) {
	if !p.IsAdmin() {
		return nil, s.denyAdmin(p)
	}
	ts, err := s.db.ListTokens(ctx, p.UserID)
	if err != nil {
		return nil, err
	}
	out := make([]TokenView, len(ts))
	for i, t := range ts {
		out[i] = tokenView(t)
	}
	return out, nil
}

// RevokeToken revokes a token (idempotent).
func (s *Service) RevokeToken(ctx context.Context, p Principal, id int64) error {
	if !p.IsAdmin() {
		return s.denyAdmin(p)
	}
	t, err := s.db.GetToken(ctx, id)
	if err != nil {
		if errors.Is(err, meta.ErrNotFound) {
			return notFound()
		}
		return err
	}
	if p.Kind == KindSession && t.OwnerID != p.UserID {
		return notFound()
	}
	if t.RevokedAt != nil {
		return nil
	}
	if err := s.db.RevokeToken(ctx, id, s.nowMs()); err != nil {
		return err
	}
	s.audit(ctx, p, "token.revoke", itoa(id), "name: "+t.Name)
	return nil
}
