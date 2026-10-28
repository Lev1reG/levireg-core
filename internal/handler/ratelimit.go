package handler

import (
	"net"
	"net/http"
	"sync"
	"time"

	"github.com/go-chi/chi/v5/middleware"
)

// RateLimit allows n requests per rolling minute per client IP (sliding log,
// so no 60s span can exceed n). shortcut: in-memory, per-process; move to
// Postgres/Redis if running >1 instance
func RateLimit(n int) func(http.Handler) http.Handler {
	var mu sync.Mutex
	hits := map[string][]time.Time{}
	var lastSweep time.Time

	allow := func(ip string) bool {
		mu.Lock()
		defer mu.Unlock()
		now := time.Now()
		cut := now.Add(-time.Minute)
		if now.Sub(lastSweep) > time.Minute { // evict idle IPs
			for k, ts := range hits {
				if ts[len(ts)-1].Before(cut) {
					delete(hits, k)
				}
			}
			lastSweep = now
		}
		ts := hits[ip]
		for len(ts) > 0 && ts[0].Before(cut) {
			ts = ts[1:]
		}
		if len(ts) >= n {
			hits[ip] = ts
			return false
		}
		hits[ip] = append(ts, now)
		return true
	}

	return func(next http.Handler) http.Handler {
		return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
			ip := middleware.GetClientIP(r.Context())
			if ip == "" {
				ip, _, _ = net.SplitHostPort(r.RemoteAddr)
			}
			if !allow(ip) {
				WriteError(w, http.StatusTooManyRequests, CodeRateLimited, "too many attempts, try again later", nil)
				return
			}
			next.ServeHTTP(w, r)
		})
	}
}
