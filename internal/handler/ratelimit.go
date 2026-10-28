package handler

import (
	"net"
	"net/http"
	"sync"
	"time"

	"github.com/go-chi/chi/v5/middleware"
	"golang.org/x/time/rate"
)

type limiterEntry struct {
	l    *rate.Limiter
	seen time.Time
}

// RateLimit allows n requests per minute per client IP and counts every
// request. shortcut: in-memory, per-process; move to Postgres/Redis if running >1 instance
func RateLimit(n int) func(http.Handler) http.Handler {
	var mu sync.Mutex
	entries := map[string]*limiterEntry{}
	var lastSweep time.Time

	allow := func(ip string) bool {
		mu.Lock()
		defer mu.Unlock()
		now := time.Now()
		if now.Sub(lastSweep) > time.Minute { // evict idle entries
			for k, e := range entries {
				if now.Sub(e.seen) > 2*time.Minute {
					delete(entries, k)
				}
			}
			lastSweep = now
		}
		e := entries[ip]
		if e == nil {
			e = &limiterEntry{l: rate.NewLimiter(rate.Every(time.Minute/time.Duration(n)), n)}
			entries[ip] = e
		}
		e.seen = now
		return e.l.AllowN(now, 1)
	}

	return func(next http.Handler) http.Handler {
		return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
			ip := middleware.GetClientIP(r.Context())
			if ip == "" {
				ip, _, _ = net.SplitHostPort(r.RemoteAddr)
			}
			if !allow(ip) {
				WriteError(w, http.StatusTooManyRequests, "rate_limited", "too many attempts, try again later", nil)
				return
			}
			next.ServeHTTP(w, r)
		})
	}
}
