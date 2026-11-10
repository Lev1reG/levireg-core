package auth

import (
	"context"
	"errors"
	"os"
	"testing"
	"time"

	"github.com/jackc/pgx/v5/pgxpool"

	"github.com/Lev1reG/levireg-core/internal/db"
)

// Needs the compose Postgres with migrations applied (make up migrate-up);
// skips when it is unreachable.
func testSessions(t *testing.T, users ...string) Sessions {
	t.Helper()
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
	t.Cleanup(pool.Close)
	if _, err := pool.Exec(ctx, "TRUNCATE auth.users CASCADE"); err != nil {
		t.Fatal(err)
	}
	q := db.New(pool)
	for _, u := range users {
		nu := NewUser{Username: u, Email: u + "@example.com", Name: u, Password: "password123", Role: RoleMember, Approved: true}
		if _, err := CreateUser(ctx, q, nu); err != nil {
			t.Fatal(err)
		}
	}
	return Sessions{Q: q}
}

func mustStart(t *testing.T, s Sessions, username string) string {
	t.Helper()
	tok, _, err := s.Start(context.Background(), username)
	if err != nil {
		t.Fatal(err)
	}
	return tok
}

func wantLive(t *testing.T, s Sessions, tok string, live bool) {
	t.Helper()
	_, err := s.Resolve(context.Background(), tok)
	if live && err != nil || !live && !errors.Is(err, ErrNoSession) {
		t.Fatalf("Resolve live=%v, err=%v", live, err)
	}
}

func TestSessionCapEndsOldestPerUser(t *testing.T) {
	s := testSessions(t, "alice", "bob")
	var toks []string
	for i := 0; i < maxSessionsPerUser+1; i++ {
		toks = append(toks, mustStart(t, s, "alice"))
	}
	bob := mustStart(t, s, "bob")

	wantLive(t, s, toks[0], false)
	for _, tok := range append(toks[1:], bob) {
		wantLive(t, s, tok, true)
	}
}

func TestExpiredSessionRejectedAndPurged(t *testing.T) {
	s := testSessions(t, "alice")
	ctx := context.Background()
	live := mustStart(t, s, "alice")
	expired, err := s.start(ctx, "alice", time.Now().Add(-time.Minute))
	if err != nil {
		t.Fatal(err)
	}

	wantLive(t, s, expired, false)
	if n, err := s.Q.DeleteExpiredSessions(ctx); err != nil || n != 1 {
		t.Fatalf("purged %d, err %v; want 1", n, err)
	}
	wantLive(t, s, live, true)
}

func TestEndOnlyEndsThatSession(t *testing.T) {
	s := testSessions(t, "alice")
	a, b := mustStart(t, s, "alice"), mustStart(t, s, "alice")
	if err := s.End(context.Background(), a); err != nil {
		t.Fatal(err)
	}
	wantLive(t, s, a, false)
	wantLive(t, s, b, true)
	wantLive(t, s, "", false)
}
