package auth

import (
	"net/mail"
	"regexp"
	"strings"
	"unicode/utf8"
)

const (
	RoleSuperadmin = "superadmin"
	RoleMember     = "member"
)

var usernameRE = regexp.MustCompile(`^[a-z0-9_]{3,32}$`)

// NewUser is the input to register and create-superadmin.
type NewUser struct {
	Username string `json:"username"`
	Email    string `json:"email"`
	Name     string `json:"name"`
	Password string `json:"password"`
	Role     string `json:"role"`
}

// Validate normalizes u in place (lowercase username, trimmed fields, default
// role) and returns Field errors keyed by JSON field name; empty means valid.
func (u *NewUser) Validate() map[string]string {
	errs := map[string]string{}
	u.Username = strings.ToLower(u.Username)
	u.Email = strings.TrimSpace(u.Email)
	u.Name = strings.TrimSpace(u.Name)
	if u.Role == "" {
		u.Role = RoleMember
	}

	if !usernameRE.MatchString(u.Username) {
		errs["username"] = "must be 3-32 characters: a-z, 0-9, underscore"
	}
	if a, err := mail.ParseAddress(u.Email); err != nil || a.Address != u.Email {
		errs["email"] = "must be a valid email address"
	}
	if u.Name == "" {
		errs["name"] = "is required"
	}
	if n := utf8.RuneCountInString(u.Password); n < 8 || n > 256 {
		errs["password"] = "must be 8 to 256 characters"
	}
	if u.Role != RoleSuperadmin && u.Role != RoleMember {
		errs["role"] = "must be superadmin or member"
	}
	return errs
}
