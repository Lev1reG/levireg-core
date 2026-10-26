package handler

import (
	"context"
	"errors"
	"net/http"
	"net/http/httptest"
	"testing"
)

type fakePinger struct{ err error }

func (f fakePinger) Ping(context.Context) error { return f.err }

func TestReadyz(t *testing.T) {
	tests := []struct {
		name   string
		err    error
		status int
		want   string
	}{
		{"up", nil, http.StatusOK, "{\"status\":\"ok\"}\n"},
		{"down", errors.New("boom"), http.StatusServiceUnavailable, "{\"status\":\"unavailable\"}\n"},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			rec := httptest.NewRecorder()
			Readyz(fakePinger{tt.err})(rec, httptest.NewRequest(http.MethodGet, "/readyz", nil))

			if rec.Code != tt.status {
				t.Fatalf("status = %d, want %d", rec.Code, tt.status)
			}
			if got := rec.Body.String(); got != tt.want {
				t.Fatalf("body = %q, want %q", got, tt.want)
			}
		})
	}
}
