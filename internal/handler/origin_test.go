package handler

import (
	"net/http"
	"net/http/httptest"
	"testing"
)

func TestOrigins(t *testing.T) {
	h := Origins([]string{"https://console.levireg.dev"})(http.HandlerFunc(func(w http.ResponseWriter, _ *http.Request) {
		w.WriteHeader(299)
	}))
	for _, c := range []struct {
		method, origin string
		status         int
		cors           bool
	}{
		{"POST", "", 299, false},                           // curl, server-to-server
		{"POST", "https://console.levireg.dev", 299, true}, // allowed
		{"POST", "https://api.levireg.dev", 299, false},    // own host, e.g. /docs try-it
		{"POST", "https://levireg.dev", 403, false},        // sibling subdomain
		{"POST", "https://evil.example", 403, false},
		{"GET", "https://evil.example", 299, false}, // no cookie use without CORS headers
		{"OPTIONS", "https://console.levireg.dev", 204, true},
	} {
		req := httptest.NewRequest(c.method, "https://api.levireg.dev/api/v1/x", nil)
		if c.origin != "" {
			req.Header.Set("Origin", c.origin)
		}
		if c.method == "OPTIONS" {
			req.Header.Set("Access-Control-Request-Method", "POST")
		}
		rec := httptest.NewRecorder()
		h.ServeHTTP(rec, req)
		gotCORS := rec.Header().Get("Access-Control-Allow-Credentials") == "true"
		if rec.Code != c.status || gotCORS != c.cors {
			t.Errorf("%s from %q: got %d cors=%v, want %d cors=%v", c.method, c.origin, rec.Code, gotCORS, c.status, c.cors)
		}
	}
}
