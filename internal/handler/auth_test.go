package handler

import (
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"
)

// These paths answer before touching the DB, so a zero Auth is enough.
func TestRequireRejectsMissingOrMalformedAuthorization(t *testing.T) {
	h := Auth{}.Require(http.HandlerFunc(func(http.ResponseWriter, *http.Request) {
		t.Fatal("next called")
	}))
	for _, hdr := range []string{"", "Bearer", "Bearer ", "Basic abc", "abc", "bearer abc"} {
		req := httptest.NewRequest("GET", "/", nil)
		if hdr != "" {
			req.Header.Set("Authorization", hdr)
		}
		rec := httptest.NewRecorder()
		h.ServeHTTP(rec, req)
		if rec.Code != 401 || !strings.Contains(rec.Body.String(), `"`+CodeUnauthorized+`"`) {
			t.Errorf("header %q: got %d %s", hdr, rec.Code, rec.Body)
		}
	}
}

func TestLoginEmptyFields(t *testing.T) {
	for _, body := range []string{`{}`, `{"username":"a"}`, `{"password":"x"}`, `{"username":"","password":""}`} {
		rec := httptest.NewRecorder()
		Auth{}.Login(rec, httptest.NewRequest("POST", "/", strings.NewReader(body)))
		if rec.Code != 422 || !strings.Contains(rec.Body.String(), `"`+CodeValidationFailed+`"`) {
			t.Errorf("body %s: got %d %s", body, rec.Code, rec.Body)
		}
	}
}
