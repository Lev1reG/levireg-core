package main

import (
	"context"
	"os"
	"strings"
	"testing"
	"time"

	"github.com/jackc/pgx/v5/pgxpool"

	"github.com/Lev1reG/levireg-core/internal/db"
)

func TestCreateSuperadmin(t *testing.T) {
	url := os.Getenv("DATABASE_URL")
	if url == "" {
		url = "postgres://levireg:levireg@localhost:5432/levireg?sslmode=disable"
	}
	ctx, cancel := context.WithTimeout(context.Background(), 3*time.Second)
	defer cancel()
	pool, err := pgxpool.New(ctx, url)
	if err == nil {
		err = pool.Ping(ctx)
	}
	if err != nil {
		t.Skipf("postgres unavailable: %v", err)
	}
	defer pool.Close()
	// Own username/email so this does not collide with server tests sharing the DB.
	_, _ = pool.Exec(ctx, "DELETE FROM auth.users WHERE username = 'clitest'")
	t.Cleanup(func() { _, _ = pool.Exec(context.Background(), "DELETE FROM auth.users WHERE username = 'clitest'") })

	q := db.New(pool)
	args := []string{"-username", "clitest", "-email", "cli@example.com", "-name", "Cli"}

	if err := createSuperadmin(ctx, q, args, strings.NewReader("short\n")); err == nil {
		t.Fatal("short password accepted")
	}
	if err := createSuperadmin(ctx, q, args, strings.NewReader("password123\n")); err != nil {
		t.Fatal(err)
	}
	u, err := q.GetUserByUsername(ctx, "clitest")
	if err != nil || u.Role != "superadmin" {
		t.Fatalf("user = %+v, err = %v", u, err)
	}
	err = createSuperadmin(ctx, q, args, strings.NewReader("password123\n"))
	if err == nil || !strings.Contains(err.Error(), "already exists") {
		t.Fatalf("duplicate err = %v", err)
	}
}
