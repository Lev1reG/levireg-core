package main

import (
	"bufio"
	"context"
	"errors"
	"flag"
	"fmt"
	"io"
	"os"
	"strings"

	"golang.org/x/term"

	"github.com/Lev1reG/levireg-core/internal/auth"
	"github.com/Lev1reG/levireg-core/internal/db"
)

// createSuperadmin implements `levireg-core create-superadmin`. The password
// comes from stdin so it never lands in shell history or env.
func createSuperadmin(ctx context.Context, q *db.Queries, args []string, stdin io.Reader) error {
	fs := flag.NewFlagSet("create-superadmin", flag.ContinueOnError)
	u := auth.NewUser{Role: auth.RoleSuperadmin}
	fs.StringVar(&u.Username, "username", "", "username (3-32 chars: a-z, 0-9, _)")
	fs.StringVar(&u.Email, "email", "", "email address")
	fs.StringVar(&u.Name, "name", "", "display name")
	if err := fs.Parse(args); err != nil {
		return err
	}

	if f, ok := stdin.(*os.File); ok && term.IsTerminal(int(f.Fd())) {
		fmt.Fprint(os.Stderr, "Password: ")
		pw, err := term.ReadPassword(int(f.Fd()))
		fmt.Fprintln(os.Stderr)
		if err != nil {
			return err
		}
		u.Password = string(pw)
	} else {
		line, err := bufio.NewReader(stdin).ReadString('\n')
		if err != nil && line == "" {
			return fmt.Errorf("read password from stdin: %w", err)
		}
		u.Password = strings.TrimRight(line, "\r\n")
	}

	if errs := u.Validate(); len(errs) > 0 {
		var b strings.Builder
		for f, m := range errs {
			fmt.Fprintf(&b, "\n  %s: %s", f, m)
		}
		return fmt.Errorf("invalid input:%s", b.String())
	}
	_, err := auth.CreateUser(ctx, q, u)
	switch {
	case errors.Is(err, auth.ErrUsernameTaken):
		return fmt.Errorf("username %q already exists", u.Username)
	case errors.Is(err, auth.ErrEmailTaken):
		return fmt.Errorf("email %q already exists", u.Email)
	}
	return err
}
