package handler

import (
	"net/http"
	"net/url"
	"slices"
)

// Origins answers CORS for the allowed origins (with credentials, so the
// Session cookie travels) and blocks cross-site request forgery: an unsafe
// request whose Origin is neither allowed nor the API's own host gets 403.
// SameSite alone does not stop sibling subdomains. Requests without Origin
// (curl, server-to-server) pass, since browsers always send it on unsafe methods.
func Origins(allowed []string) func(http.Handler) http.Handler {
	return func(next http.Handler) http.Handler {
		return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
			origin := r.Header.Get("Origin")
			if origin == "" {
				next.ServeHTTP(w, r)
				return
			}
			w.Header().Add("Vary", "Origin")
			if slices.Contains(allowed, origin) {
				h := w.Header()
				h.Set("Access-Control-Allow-Origin", origin)
				h.Set("Access-Control-Allow-Credentials", "true")
				if r.Method == http.MethodOptions && r.Header.Get("Access-Control-Request-Method") != "" {
					h.Set("Access-Control-Allow-Methods", "GET, POST, PUT, PATCH, DELETE")
					h.Set("Access-Control-Allow-Headers", "Content-Type")
					h.Set("Access-Control-Max-Age", "600")
					w.WriteHeader(http.StatusNoContent)
					return
				}
			} else if !safeMethod(r.Method) && !sameHost(origin, r.Host) {
				WriteError(w, http.StatusForbidden, CodeForbidden, "origin not allowed", nil)
				return
			}
			next.ServeHTTP(w, r)
		})
	}
}

func safeMethod(m string) bool {
	return m == http.MethodGet || m == http.MethodHead || m == http.MethodOptions
}

// sameHost compares hosts only: behind nginx the API sees plain HTTP.
func sameHost(origin, host string) bool {
	u, err := url.Parse(origin)
	return err == nil && u.Host == host
}
