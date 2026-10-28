package auth

import (
	"context"
	"errors"

	"github.com/jackc/pgx/v5/pgconn"

	"github.com/Lev1reG/levireg-core/internal/db"
)

var (
	ErrUsernameTaken = errors.New("username taken")
	ErrEmailTaken    = errors.New("email taken")
)

// CreateUser hashes the password and inserts u, which must already be
// validated. Unique violations map to ErrUsernameTaken / ErrEmailTaken.
func CreateUser(ctx context.Context, q *db.Queries, u NewUser) (db.AuthUser, error) {
	hash, err := HashPassword(u.Password)
	if err != nil {
		return db.AuthUser{}, err
	}
	row, err := q.CreateUser(ctx, db.CreateUserParams{
		Username: u.Username, Email: u.Email, Name: u.Name, PasswordHash: hash, Role: u.Role,
	})
	var pgErr *pgconn.PgError
	if errors.As(err, &pgErr) && pgErr.Code == "23505" {
		switch pgErr.ConstraintName {
		case "users_pkey":
			return db.AuthUser{}, ErrUsernameTaken
		case "users_email_lower_key":
			return db.AuthUser{}, ErrEmailTaken
		}
	}
	return row, err
}
