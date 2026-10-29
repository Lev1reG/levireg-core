package handler

import (
	"context"
	"net/http"
	"time"

	"go.uber.org/zap"

	"github.com/Lev1reG/levireg-core/internal/logger"
)

// Pinger is the dependency check Readyz needs; *pgxpool.Pool satisfies it.
type Pinger interface {
	Ping(ctx context.Context) error
}

// Readyz reports whether the service can reach its dependencies. Unlike
// Healthz it fails when the DB is down, so a probe can stop routing traffic.
func Readyz(db Pinger) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		ctx, cancel := context.WithTimeout(r.Context(), 2*time.Second)
		defer cancel()

		w.Header().Set("Content-Type", "application/json")
		if err := db.Ping(ctx); err != nil {
			logger.From(r.Context()).Warn("dependency unavailable", zap.Error(err))
			w.WriteHeader(http.StatusServiceUnavailable)
			_, _ = w.Write([]byte("{\"status\":\"unavailable\"}\n"))
			return
		}
		_, _ = w.Write([]byte("{\"status\":\"ok\"}\n"))
	}
}
