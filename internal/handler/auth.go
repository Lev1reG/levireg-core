package handler

import (
	"context"
	"encoding/json"
	"errors"
	"net/http"
	"strings"
	"time"

	"github.com/jackc/pgx/v5"
	"github.com/jackc/pgx/v5/pgtype"

	"github.com/Lev1reG/levireg-core/internal/auth"
	"github.com/Lev1reG/levireg-core/internal/db"
)

const sessionTTL = 7 * 24 * time.Hour

// Fixed hash verified for unknown usernames so login timing does not reveal
// which usernames exist.
var dummyHash = func() string {
	h, err := auth.HashPassword("dummy-password")
	if err != nil {
		panic(err)
	}
	return h
}()

type userCtxKey struct{}

// UserJSON is the public shape of a User; password_hash is never included.
type UserJSON struct {
	Username  string    `json:"username"`
	Email     string    `json:"email"`
	Name      string    `json:"name"`
	Role      string    `json:"role"`
	CreatedAt time.Time `json:"created_at"`
}

func toUserJSON(u db.AuthUser) UserJSON {
	return UserJSON{u.Username, u.Email, u.Name, u.Role, u.CreatedAt.Time}
}

// Auth serves the /auth endpoints and the middleware guarding protected routes.
type Auth struct{ Q *db.Queries }

func bearer(r *http.Request) string {
	tok, ok := strings.CutPrefix(r.Header.Get("Authorization"), "Bearer ")
	if !ok {
		return ""
	}
	return strings.TrimSpace(tok)
}

func unauthorized(w http.ResponseWriter) {
	WriteError(w, http.StatusUnauthorized, CodeUnauthorized, "missing or invalid token", nil)
}

func serverError(w http.ResponseWriter) {
	WriteError(w, http.StatusInternalServerError, CodeInternalError, "something went wrong", nil)
}

// Require puts the session's User in the request context or answers 401.
func (a Auth) Require(next http.Handler) http.Handler {
	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		tok := bearer(r)
		if tok == "" {
			unauthorized(w)
			return
		}
		u, err := a.Q.GetUserBySessionToken(r.Context(), auth.HashToken(tok))
		if errors.Is(err, pgx.ErrNoRows) {
			unauthorized(w)
			return
		}
		if err != nil {
			serverError(w)
			return
		}
		next.ServeHTTP(w, r.WithContext(context.WithValue(r.Context(), userCtxKey{}, u)))
	})
}

// RequireSuperadmin must run after Require; Members get 403.
func RequireSuperadmin(next http.Handler) http.Handler {
	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if currentUser(r).Role != auth.RoleSuperadmin {
			WriteError(w, http.StatusForbidden, CodeForbidden, "superadmin only", nil)
			return
		}
		next.ServeHTTP(w, r)
	})
}

func currentUser(r *http.Request) db.AuthUser {
	u, _ := r.Context().Value(userCtxKey{}).(db.AuthUser)
	return u
}

func decode(w http.ResponseWriter, r *http.Request, v any) bool {
	if err := json.NewDecoder(http.MaxBytesReader(w, r.Body, 1<<16)).Decode(v); err != nil {
		WriteError(w, http.StatusBadRequest, CodeBadRequest, "invalid JSON body", nil)
		return false
	}
	return true
}

func (a Auth) Login(w http.ResponseWriter, r *http.Request) {
	var in struct{ Username, Password string }
	if !decode(w, r, &in) {
		return
	}
	if in.Username == "" || in.Password == "" {
		WriteError(w, http.StatusUnprocessableEntity, CodeValidationFailed, "username and password are required", map[string]string{"username": "is required", "password": "is required"})
		return
	}
	u, err := a.Q.GetUserByUsername(r.Context(), strings.ToLower(in.Username))
	hash := dummyHash
	if err == nil {
		hash = u.PasswordHash
	} else if !errors.Is(err, pgx.ErrNoRows) {
		serverError(w)
		return
	}
	if !auth.VerifyPassword(hash, in.Password) || err != nil {
		WriteError(w, http.StatusUnauthorized, CodeInvalidCredentials, "wrong username or password", nil)
		return
	}

	token, tokenHash, err := auth.NewSessionToken()
	if err != nil {
		serverError(w)
		return
	}
	exp := time.Now().Add(sessionTTL)
	err = a.Q.CreateSession(r.Context(), db.CreateSessionParams{
		TokenHash: tokenHash, Username: u.Username,
		ExpiresAt: pgtype.Timestamptz{Time: exp, Valid: true},
	})
	if err != nil {
		serverError(w)
		return
	}
	Write(w, http.StatusOK, struct {
		Token     string    `json:"token"`
		ExpiresAt time.Time `json:"expires_at"`
		User      UserJSON  `json:"user"`
	}{token, exp, toUserJSON(u)})
}

func (a Auth) Logout(w http.ResponseWriter, r *http.Request) {
	if err := a.Q.DeleteSession(r.Context(), auth.HashToken(bearer(r))); err != nil {
		serverError(w)
		return
	}
	WriteCode(w, http.StatusOK, CodeOK, "logged out")
}

func (a Auth) Me(w http.ResponseWriter, r *http.Request) {
	Write(w, http.StatusOK, toUserJSON(currentUser(r)))
}

func (a Auth) Register(w http.ResponseWriter, r *http.Request) {
	var in auth.NewUser
	if !decode(w, r, &in) {
		return
	}
	if errs := in.Validate(); len(errs) > 0 {
		WriteError(w, http.StatusUnprocessableEntity, CodeValidationFailed, "invalid fields", errs)
		return
	}
	u, err := auth.CreateUser(r.Context(), a.Q, in)
	switch {
	case errors.Is(err, auth.ErrUsernameTaken):
		WriteError(w, http.StatusConflict, CodeUsernameTaken, "username already exists", nil)
	case errors.Is(err, auth.ErrEmailTaken):
		WriteError(w, http.StatusConflict, CodeEmailTaken, "email already exists", nil)
	case err != nil:
		serverError(w)
	default:
		Write(w, http.StatusCreated, toUserJSON(u))
	}
}
