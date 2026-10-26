package server

import (
	"context"
	"net/http"
	"net/http/httptest"
	"testing"

	"go.uber.org/zap"
)

type okPinger struct{}

func (okPinger) Ping(context.Context) error { return nil }

func TestHealthz(t *testing.T) {
	srv := New(zap.NewNop(), ":0", okPinger{})

	rec := httptest.NewRecorder()
	srv.Handler.ServeHTTP(rec, httptest.NewRequest(http.MethodGet, "/healthz", nil))

	if rec.Code != http.StatusOK {
		t.Fatalf("status = %d, want %d", rec.Code, http.StatusOK)
	}
	if got, want := rec.Body.String(), "{\"status\":\"ok\"}\n"; got != want {
		t.Fatalf("body = %q, want %q", got, want)
	}
}
