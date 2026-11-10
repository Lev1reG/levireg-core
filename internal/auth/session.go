package auth

import (
	"context"
	"crypto/rand"
	"crypto/sha256"
	"encoding/base64"
	"errors"
	"time"

	"github.com/jackc/pgx/v5"
	"github.com/jackc/pgx/v5/pgtype"
	"go.uber.org/zap"

	"github.com/Lev1reG/levireg-core/internal/db"
)

const (
	sessionTTL = 7 * 24 * time.Hour
	// Starting a Session past this many ends the User's oldest ones.
	maxSessionsPerUser = 5
)

// ErrNoSession means the token is empty, unknown or expired.
var ErrNoSession = errors.New("no session")

// Sessions owns the Session lifecycle: token creation and hashing, lifetime,
// the per-User cap and expiry. Only the token hash is stored.
type Sessions struct{ Q *db.Queries }

// Start opens a Session for username and returns its token and expiry.
func (s Sessions) Start(ctx context.Context, username string) (token string, expires time.Time, err error) {
	expires = time.Now().Add(sessionTTL)
	token, err = s.start(ctx, username, expires)
	return token, expires, err
}

func (s Sessions) start(ctx context.Context, username string, expires time.Time) (string, error) {
	raw := make([]byte, 32)
	if _, err := rand.Read(raw); err != nil {
		return "", err
	}
	token := base64.RawURLEncoding.EncodeToString(raw)
	err := s.Q.CreateSession(ctx, db.CreateSessionParams{
		TokenHash: hashToken(token), Username: username, MaxSessions: maxSessionsPerUser,
		ExpiresAt: pgtype.Timestamptz{Time: expires, Valid: true},
	})
	return token, err
}

// Resolve returns the User holding the live Session for token, or ErrNoSession.
func (s Sessions) Resolve(ctx context.Context, token string) (db.AuthUser, error) {
	if token == "" {
		return db.AuthUser{}, ErrNoSession
	}
	u, err := s.Q.GetUserBySessionToken(ctx, hashToken(token))
	if errors.Is(err, pgx.ErrNoRows) {
		return db.AuthUser{}, ErrNoSession
	}
	return u, err
}

// End deletes the Session for token; an unknown token is not an error.
func (s Sessions) End(ctx context.Context, token string) error {
	return s.Q.DeleteSession(ctx, hashToken(token))
}

// PurgeExpired deletes expired Sessions now and then every interval until ctx
// ends. Failures are logged and retried on the next tick; expired Sessions are
// already rejected by Resolve, so this only reclaims storage.
func (s Sessions) PurgeExpired(ctx context.Context, log *zap.Logger, interval time.Duration) {
	t := time.NewTicker(interval)
	defer t.Stop()
	for {
		switch n, err := s.Q.DeleteExpiredSessions(ctx); {
		case ctx.Err() != nil:
			return
		case err != nil:
			log.Error("purge expired sessions failed", zap.Error(err))
		case n > 0:
			log.Info("purged expired sessions", zap.Int64("count", n))
		}
		select {
		case <-ctx.Done():
			return
		case <-t.C:
		}
	}
}

func hashToken(token string) []byte {
	h := sha256.Sum256([]byte(token))
	return h[:]
}
