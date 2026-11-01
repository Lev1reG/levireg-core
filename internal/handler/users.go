package handler

import (
	"errors"
	"net/http"
	"strings"

	"github.com/go-chi/chi/v5"
	"github.com/jackc/pgx/v5"

	"github.com/Lev1reG/levireg-core/internal/db"
)

// ListUsers serves GET /users with ?status, ?page and ?per_page.
func (a Auth) ListUsers(w http.ResponseWriter, r *http.Request) {
	p, errs := ParsePagination(r)
	status := r.URL.Query().Get("status")
	if status != "" && status != "pending" && status != "approved" {
		if errs == nil {
			errs = map[string]string{}
		}
		errs["status"] = "must be pending or approved"
	}
	if len(errs) > 0 {
		WriteError(w, http.StatusUnprocessableEntity, CodeValidationFailed, "invalid query", errs)
		return
	}

	total, err := a.Q.CountUsers(r.Context(), status)
	if err != nil {
		serverError(w, r, err)
		return
	}
	rows, err := a.Q.ListUsers(r.Context(), db.ListUsersParams{
		Status: status, PageLimit: int32(p.PerPage), PageOffset: int64(p.Offset()),
	})
	if err != nil {
		serverError(w, r, err)
		return
	}
	items := make([]UserJSON, len(rows))
	for i, u := range rows {
		items[i] = toUserJSON(u)
	}
	WriteList(w, items, Meta{Page: p.Page, PerPage: p.PerPage, Total: int(total)})
}

// Approve turns a Pending User into an Approved User.
func (a Auth) Approve(w http.ResponseWriter, r *http.Request) {
	username := strings.ToLower(chi.URLParam(r, "username"))
	u, err := a.Q.ApproveUser(r.Context(), username)
	if errors.Is(err, pgx.ErrNoRows) {
		a.notPendingOrMissing(w, r, username)
		return
	}
	if err != nil {
		serverError(w, r, err)
		return
	}
	Write(w, http.StatusOK, toUserJSON(u))
}

// Reject deletes a Pending User; an Approved User is never touched.
func (a Auth) Reject(w http.ResponseWriter, r *http.Request) {
	username := strings.ToLower(chi.URLParam(r, "username"))
	n, err := a.Q.RejectUser(r.Context(), username)
	if err != nil {
		serverError(w, r, err)
		return
	}
	if n == 0 {
		a.notPendingOrMissing(w, r, username)
		return
	}
	WriteCode(w, http.StatusOK, CodeOK, "rejected")
}

// notPendingOrMissing explains why a conditional approve/reject changed nothing.
func (a Auth) notPendingOrMissing(w http.ResponseWriter, r *http.Request, username string) {
	_, err := a.Q.GetUserByUsername(r.Context(), username)
	switch {
	case errors.Is(err, pgx.ErrNoRows):
		WriteError(w, http.StatusNotFound, CodeUserNotFound, "user not found", nil)
	case err != nil:
		serverError(w, r, err)
	default:
		WriteError(w, http.StatusConflict, CodeUserNotPending, "user is not pending", nil)
	}
}
