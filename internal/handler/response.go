package handler

import (
	"encoding/json"
	"net/http"
)

// Response codes clients branch on.
const (
	CodeOK                 = "ok"
	CodeUnauthorized       = "unauthorized"
	CodeForbidden          = "forbidden"
	CodeInternalError      = "internal_error"
	CodeValidationFailed   = "validation_failed"
	CodeInvalidCredentials = "invalid_credentials"
	CodeBadRequest         = "bad_request"
	CodeUsernameTaken      = "username_taken"
	CodeEmailTaken         = "email_taken"
	CodeRateLimited        = "rate_limited"
)

// Envelope is the shape of every /api/v1 response. Code is the stable contract
// clients branch on; the rest is omitted when empty.
type Envelope[T any] struct {
	Code    string            `json:"code"`
	Message string            `json:"message,omitempty"`
	Data    *T                `json:"data,omitempty"`
	Errors  map[string]string `json:"errors,omitempty"`
	Meta    *Meta             `json:"meta,omitempty"`
}

// Meta is the Pagination meta carried by list responses.
type Meta struct {
	Page    int `json:"page"`
	PerPage int `json:"per_page"`
	Total   int `json:"total"`
}

// Write sends a successful envelope with the given status and payload.
func Write[T any](w http.ResponseWriter, status int, data T) {
	write(w, status, Envelope[T]{Code: CodeOK, Data: &data})
}

// WriteList sends a successful list envelope. A nil items slice serializes as
// [], never null.
func WriteList[T any](w http.ResponseWriter, items []T, meta Meta) {
	if items == nil {
		items = []T{}
	}
	write(w, http.StatusOK, Envelope[[]T]{Code: CodeOK, Data: &items, Meta: &meta})
}

// WriteCode sends a successful envelope with no payload and a custom code, for
// outcomes such as "accepted" that carry nothing back.
func WriteCode(w http.ResponseWriter, status int, code, message string) {
	write(w, status, Envelope[struct{}]{Code: code, Message: message})
}

// WriteError sends a failed envelope. fieldErrors may be nil.
func WriteError(w http.ResponseWriter, status int, code, message string, fieldErrors map[string]string) {
	write(w, status, Envelope[struct{}]{Code: code, Message: message, Errors: fieldErrors})
}

func write[T any](w http.ResponseWriter, status int, e Envelope[T]) {
	w.Header().Set("Content-Type", "application/json")
	w.WriteHeader(status)
	// Client disconnect is the only realistic failure; the logger already
	// records the status, so there is nothing useful to report here.
	_ = json.NewEncoder(w).Encode(e)
}
