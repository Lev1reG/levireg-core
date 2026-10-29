package logger

import (
	"net/http"
	"net/http/httptest"
	"testing"

	"github.com/go-chi/chi/v5/middleware"
	"go.uber.org/zap"
	"go.uber.org/zap/zapcore"
	"go.uber.org/zap/zaptest/observer"
)

func TestRecovererLogsPanicWithRequestID(t *testing.T) {
	core, logs := observer.New(zap.DebugLevel)
	h := middleware.RequestID(RequestLogger(zap.New(core))(Recoverer(
		http.HandlerFunc(func(http.ResponseWriter, *http.Request) { panic("boom") }),
	)))

	rec := httptest.NewRecorder()
	h.ServeHTTP(rec, httptest.NewRequest(http.MethodGet, "/x", nil))

	if rec.Code != http.StatusInternalServerError {
		t.Fatalf("status = %d, want 500", rec.Code)
	}
	entries := logs.All()
	if len(entries) != 2 || entries[0].Message != "panic" || entries[1].Message != "request" {
		t.Fatalf("entries = %+v, want panic then request", entries)
	}
	for _, e := range entries {
		if e.Level != zapcore.ErrorLevel || e.ContextMap()["request_id"] == nil {
			t.Errorf("%q: level %v, fields %v; want error with request_id", e.Message, e.Level, e.ContextMap())
		}
	}
}

func TestLevel(t *testing.T) {
	for _, tc := range []struct {
		path   string
		status int
		want   zapcore.Level
	}{
		{"/api/v1/auth/me", 200, zap.InfoLevel},
		{"/healthz", 200, zap.DebugLevel},
		{"/readyz", 503, zap.ErrorLevel},
		{"/x", 404, zap.WarnLevel},
	} {
		if got := level(tc.path, tc.status); got != tc.want {
			t.Errorf("level(%q, %d) = %v, want %v", tc.path, tc.status, got, tc.want)
		}
	}
}
