package auth

import (
	"context"
	"time"

	"go.uber.org/zap"
)

// SessionPurger deletes expired Sessions and returns how many it removed.
type SessionPurger interface {
	DeleteExpiredSessions(ctx context.Context) (int64, error)
}

// PurgeExpiredSessions deletes expired Sessions now and then every interval
// until ctx ends. Failures are logged and retried on the next tick; expired
// Sessions are already rejected on use, so this only reclaims storage.
func PurgeExpiredSessions(ctx context.Context, q SessionPurger, log *zap.Logger, interval time.Duration) {
	t := time.NewTicker(interval)
	defer t.Stop()
	for {
		switch n, err := q.DeleteExpiredSessions(ctx); {
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
