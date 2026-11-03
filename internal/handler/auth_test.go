package handler

import (
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"
)

// These paths answer before touching the DB, so a zero Auth is enough.
func TestRequireRejectsMissingSessionCookie(t *testing.T) {
	h := Auth{}.Require(http.HandlerFunc(func(http.ResponseWriter, *http.Request) {
		t.Fatal("next called")
	}))
	// A Bearer header is not a way in any more.
	for _, hdr := range []string{"", "Authorization: Bearer abc", "Cookie: __Host-session=", "Cookie: session=abc"} {
		req := httptest.NewRequest("GET", "/", nil)
		if k, v, ok := strings.Cut(hdr, ": "); ok {
			req.Header.Set(k, v)
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
