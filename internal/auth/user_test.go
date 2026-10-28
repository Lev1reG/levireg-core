package auth

import (
	"strings"
	"testing"
)

func TestNewUserValidate(t *testing.T) {
	u := NewUser{Username: "Alice_1", Email: " a@b.co ", Name: " Alice ", Password: "longenough"}
	if errs := u.Validate(); len(errs) != 0 {
		t.Fatalf("unexpected errors: %v", errs)
	}
	if u.Username != "alice_1" || u.Email != "a@b.co" || u.Name != "Alice" || u.Role != RoleMember {
		t.Fatalf("not normalized: %+v", u)
	}

	bad := NewUser{Username: "a!", Email: "nope", Name: " ", Password: "short", Role: "root"}
	errs := bad.Validate()
	for _, f := range []string{"username", "email", "name", "password", "role"} {
		if errs[f] == "" {
			t.Errorf("no field error for %s", f)
		}
	}

	long := NewUser{Username: "alice", Email: "a@b.co", Name: "A", Password: strings.Repeat("x", 257)}
	if long.Validate()["password"] == "" {
		t.Error("257-char password accepted")
	}
}
