package handler

import (
	"net/http"
	"net/http/httptest"
	"testing"
)

func TestEnvelope(t *testing.T) {
	tests := []struct {
		name   string
		call   func(http.ResponseWriter)
		status int
		want   string
	}{
		{"data", func(w http.ResponseWriter) {
			Write(w, http.StatusOK, []string{"a"})
		}, http.StatusOK, `{"code":"ok","data":["a"]}` + "\n"},
		{"no data", func(w http.ResponseWriter) {
			WriteCode(w, http.StatusAccepted, "accepted", "")
		}, http.StatusAccepted, `{"code":"accepted"}` + "\n"},
		{"error with fields", func(w http.ResponseWriter) {
			WriteError(w, http.StatusBadRequest, "validation_failed", "invalid", map[string]string{"email": "required"})
		}, http.StatusBadRequest, `{"code":"validation_failed","message":"invalid","errors":{"email":"required"}}` + "\n"},
		{"list", func(w http.ResponseWriter) {
			WriteList(w, []string{"a"}, Meta{Page: 2, PerPage: 20, Total: 21})
		}, http.StatusOK, `{"code":"ok","data":["a"],"meta":{"page":2,"per_page":20,"total":21}}` + "\n"},
		{"empty list from nil", func(w http.ResponseWriter) {
			WriteList[string](w, nil, Meta{Page: 1, PerPage: 20})
		}, http.StatusOK, `{"code":"ok","data":[],"meta":{"page":1,"per_page":20,"total":0}}` + "\n"},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			rec := httptest.NewRecorder()
			tt.call(rec)

			if rec.Code != tt.status {
				t.Fatalf("status = %d, want %d", rec.Code, tt.status)
			}
			if got := rec.Header().Get("Content-Type"); got != "application/json" {
				t.Fatalf("Content-Type = %q, want application/json", got)
			}
			if got := rec.Body.String(); got != tt.want {
				t.Fatalf("body = %q, want %q", got, tt.want)
			}
		})
	}
}
