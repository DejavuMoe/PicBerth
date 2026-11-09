package core

import (
	"context"
	"crypto/subtle"
	"errors"
	"regexp"
	"strings"
	"time"

	"picberth/internal/ids"
	"picberth/internal/meta"
)

var usernameRE = regexp.MustCompile(`^[A-Za-z0-9._-]{1,64}$`)

// Fixed session lifetimes (approved 2026-11-09): 7 days idle, 30 days absolute.
const (
	sessionIdleTTL     = 7 * 24 * time.Hour
	sessionAbsoluteTTL = 30 * 24 * time.Hour
)

func validateCredentials(username, pw string) error {
	if !usernameRE.MatchString(username) {
		return validation("Username must be 1-64 characters from A-Z a-z 0-9 . _ -")
	}
	if n := utf8Len(pw); n < 10 || n > 256 {
		return validation("Password must be 10-256 characters.")
	}
	return nil
}

// NewSession is a freshly issued browser session.
type NewSession struct {
	ID        string // cookie value; only its hash is stored
	ExpiresAt time.Time
	MaxAge    time.Duration
}

// SetupRequired reports whether no administrator exists yet.
func (s *Service) SetupRequired(ctx context.Context) (bool, error) {
	n, err := s.db.CountUsers(ctx)
	return n == 0, err
}

// EnsureSetupCode returns the one-time setup code, generating it on first use. It returns "" when an
// administrator already exists.
func (s *Service) EnsureSetupCode(ctx context.Context) (string, error) {
	required, err := s.SetupRequired(ctx)
	if err != nil || !required {
		return "", err
	}
	s.setupMu.Lock()
	defer s.setupMu.Unlock()
	if s.setupCode == "" {
		s.setupCode = ids.NewSetupCode()
	}
	return s.setupCode, nil
}

// Setup creates the first administrator, authorized by the one-time setup code, and logs them in.
func (s *Service) Setup(ctx context.Context, ip, code, username, pw string) (NewSession, error) {
	if ok, wait := s.setupLimit.Allow("global"); !ok {
		return NewSession{}, rateLimited(wait)
	}
	if ok, wait := s.loginIPLimit.Allow(ip); !ok {
		return NewSession{}, rateLimited(wait)
	}
	required, err := s.SetupRequired(ctx)
	if err != nil {
		return NewSession{}, err
	}
	if !required {
		return NewSession{}, &Error{Code: CodeSetupDone, Detail: "Setup has already been completed."}
	}
	if err := validateCredentials(username, pw); err != nil {
		return NewSession{}, err
	}
	want, err := s.EnsureSetupCode(ctx)
	if err != nil {
		return NewSession{}, err
	}
	got := ids.NormalizeSetupCode(code)
	if subtle.ConstantTimeCompare([]byte(got), []byte(ids.NormalizeSetupCode(want))) != 1 {
		s.auditActor(ctx, "anonymous", ip, "setup.failure", "", "invalid setup code")
		return NewSession{}, forbidden("The setup code is not valid.")
	}
	hash, err := s.hasher.Hash(ctx, pw)
	if err != nil {
		return NewSession{}, err
	}
	uid, created, err := s.db.CreateFirstUser(ctx, username, hash, s.nowMs())
	if err != nil {
		return NewSession{}, err
	}
	if !created {
		return NewSession{}, &Error{Code: CodeSetupDone, Detail: "Setup has already been completed."}
	}
	s.setupMu.Lock()
	s.setupCode = ""
	s.setupMu.Unlock()
	s.auditActor(ctx, "user:"+username, ip, "setup", "", "administrator created")
	return s.issueSession(ctx, uid)
}

// CreateAdmin creates the first administrator from the local CLI. It fails when a user already exists.
func (s *Service) CreateAdmin(ctx context.Context, username, pw string) error {
	if err := validateCredentials(username, pw); err != nil {
		return err
	}
	hash, err := s.hasher.Hash(ctx, pw)
	if err != nil {
		return err
	}
	_, created, err := s.db.CreateFirstUser(ctx, username, hash, s.nowMs())
	if err != nil {
		return err
	}
	if !created {
		return conflict("An administrator already exists; use reset-password.")
	}
	s.auditActor(ctx, "cli", "", "admin.create", username, "")
	return nil
}

// ResetPassword sets a new password for an existing user from the local CLI and ends their sessions.
func (s *Service) ResetPassword(ctx context.Context, username, pw string) error {
	if err := validateCredentials(username, pw); err != nil {
		return err
	}
	u, err := s.db.GetUserByName(ctx, username)
	if err != nil {
		if errors.Is(err, meta.ErrNotFound) {
			return notFound()
		}
		return err
	}
	hash, err := s.hasher.Hash(ctx, pw)
	if err != nil {
		return err
	}
	if err := s.db.SetPassword(ctx, u.ID, hash); err != nil {
		return err
	}
	s.auditActor(ctx, "cli", "", "admin.reset-password", u.Username, "")
	return nil
}

// Login verifies credentials and issues a session. Failures never reveal whether the user exists.
func (s *Service) Login(ctx context.Context, ip, username, pw string) (NewSession, error) {
	userKey := strings.ToLower(printable(username, 64))
	if ok, wait := s.loginIPLimit.Allow(ip); !ok {
		return NewSession{}, rateLimited(wait)
	}
	if ok, wait := s.loginUserLimit.Allow(userKey); !ok {
		s.loginIPLimit.Refund(ip)
		return NewSession{}, rateLimited(wait)
	}
	if required, err := s.SetupRequired(ctx); err != nil {
		return NewSession{}, err
	} else if required {
		return NewSession{}, &Error{Code: CodeSetupRequired, Detail: "No administrator exists yet; complete setup first."}
	}

	var u *meta.User
	if usernameRE.MatchString(username) && utf8Len(pw) <= 256 {
		var err error
		u, err = s.db.GetUserByName(ctx, username)
		if err != nil && !errors.Is(err, meta.ErrNotFound) {
			return NewSession{}, err
		}
	}
	ok := false
	if u == nil {
		if err := s.hasher.Dummy(ctx, pw); err != nil {
			return NewSession{}, err
		}
	} else {
		var err error
		ok, err = s.hasher.Verify(ctx, pw, u.PasswordHash)
		if err != nil {
			if ctx.Err() != nil {
				return NewSession{}, err
			}
			s.log.Error("stored password hash is unusable", "user", u.Username, "err", err)
		}
	}
	if !ok {
		s.auditActor(ctx, "anonymous", ip, "login.failure", "", "user: "+printable(username, 64))
		return NewSession{}, unauthorized()
	}
	s.loginIPLimit.Refund(ip)
	s.loginUserLimit.Refund(userKey)
	sess, err := s.issueSession(ctx, u.ID)
	if err != nil {
		return NewSession{}, err
	}
	s.auditActor(ctx, "user:"+u.Username, ip, "login.success", "", "")
	return sess, nil
}

func (s *Service) issueSession(ctx context.Context, userID int64) (NewSession, error) {
	now := s.now()
	id := ids.NewSessionID()
	err := s.db.CreateSession(ctx, meta.Session{
		IDHash: ids.Hash(id), UserID: userID, CreatedAt: now.UnixMilli(), LastSeenAt: now.UnixMilli(),
	})
	if err != nil {
		return NewSession{}, err
	}
	return NewSession{ID: id, ExpiresAt: now.Add(sessionAbsoluteTTL), MaxAge: sessionAbsoluteTTL}, nil
}

// Logout ends a session. Unknown sessions are ignored.
func (s *Service) Logout(ctx context.Context, p Principal, sessionID string) error {
	if sessionID == "" {
		return nil
	}
	if err := s.db.DeleteSession(ctx, ids.Hash(sessionID)); err != nil {
		return err
	}
	if p.Kind == KindSession {
		s.audit(ctx, p, "logout", "", "")
	}
	return nil
}

// AuthenticateSession resolves a session cookie value to a Principal.
func (s *Service) AuthenticateSession(ctx context.Context, sessionID, ip string) (Principal, error) {
	if sessionID == "" || len(sessionID) > 128 {
		return Principal{}, unauthorized()
	}
	h := ids.Hash(sessionID)
	sess, err := s.db.GetSession(ctx, h)
	if err != nil {
		if errors.Is(err, meta.ErrNotFound) {
			return Principal{}, unauthorized()
		}
		return Principal{}, err
	}
	now := s.now()
	if now.After(time.UnixMilli(sess.LastSeenAt).Add(sessionIdleTTL)) || now.After(time.UnixMilli(sess.CreatedAt).Add(sessionAbsoluteTTL)) {
		_ = s.db.DeleteSession(ctx, h)
		return Principal{}, unauthorized()
	}
	if now.UnixMilli()-sess.LastSeenAt >= touchInterval.Milliseconds() {
		if err := s.db.TouchSession(ctx, h, now.UnixMilli()); err != nil {
			s.log.Warn("could not update session activity", "err", err)
		}
	}
	return Principal{Kind: KindSession, UserID: sess.UserID, Username: sess.Username, IP: ip}, nil
}

// AuthenticateToken resolves a bearer secret to a Principal.
func (s *Service) AuthenticateToken(ctx context.Context, secret, ip string) (Principal, error) {
	if !ids.ValidTokenSecret(secret) {
		return Principal{}, unauthorized()
	}
	t, err := s.db.GetTokenBySecretHash(ctx, ids.Hash(secret))
	if err != nil {
		if errors.Is(err, meta.ErrNotFound) {
			return Principal{}, unauthorized()
		}
		return Principal{}, err
	}
	now := s.now().UnixMilli()
	if t.RevokedAt != nil || (t.ExpiresAt != nil && now >= *t.ExpiresAt) {
		return Principal{}, unauthorized()
	}
	if t.LastUsedAt == nil || now-*t.LastUsedAt >= touchInterval.Milliseconds() {
		if err := s.db.TouchToken(ctx, t.ID, now); err != nil {
			s.log.Warn("could not update token activity", "err", err)
		}
	}
	u, err := s.db.GetUser(ctx, t.OwnerID)
	if err != nil {
		return Principal{}, unauthorized()
	}
	return Principal{Kind: KindToken, UserID: t.OwnerID, Username: u.Username, TokenID: t.ID, TokenName: t.Name, Scopes: parseScopes(t.Scopes), IP: ip}, nil
}

// SessionState is what GET /api/v1/session reports.
type SessionState struct {
	Authenticated bool
	SetupRequired bool
	Username      string
	Limits        *SessionLimits
}

// SessionLimits are the client-relevant upload limits.
type SessionLimits struct {
	MaxUploadBytes int64    `json:"maxUploadBytes"`
	AllowedFormats []string `json:"allowedFormats"`
}

// SessionInfo builds the session state for a (possibly anonymous) principal.
func (s *Service) SessionInfo(ctx context.Context, p Principal) (SessionState, error) {
	required, err := s.SetupRequired(ctx)
	if err != nil {
		return SessionState{}, err
	}
	st := SessionState{SetupRequired: required}
	if p.Kind == KindSession {
		cfg := s.Settings()
		st.Authenticated = true
		st.Username = p.Username
		st.Limits = &SessionLimits{MaxUploadBytes: cfg.MaxUploadBytes, AllowedFormats: cfg.AllowedFormats}
	}
	return st, nil
}

// PurgeExpiredSessions deletes sessions past their idle or absolute lifetime.
func (s *Service) PurgeExpiredSessions(ctx context.Context) (int64, error) {
	return s.db.DeleteExpiredSessions(ctx, s.nowMs(), sessionIdleTTL.Milliseconds(), sessionAbsoluteTTL.Milliseconds())
}
