package server

import (
	"bytes"
	"context"
	"crypto/sha256"
	"encoding/json"
	"io"
	"net/http"
	"net/http/httptest"
	"os"
	"strings"
	"testing"
	"time"

	"github.com/jackc/pgx/v5/pgtype"
	"github.com/jackc/pgx/v5/pgxpool"
	"go.uber.org/zap"

	"github.com/Lev1reG/levireg-core/internal/auth"
	"github.com/Lev1reG/levireg-core/internal/db"
)

// Needs the compose Postgres with migrations applied (make up migrate-up);
// skips when it is unreachable.
func testPool(t *testing.T) *pgxpool.Pool {
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
	return pool
}

type testEnv struct {
	t    *testing.T
	h    http.Handler
	pool *pgxpool.Pool
}

func newEnv(t *testing.T) *testEnv {
	pool := testPool(t)
	return &testEnv{t: t, h: New(zap.NewNop(), ":0", pool).Handler, pool: pool}
}

func (e *testEnv) seed(username, role string) {
	e.t.Helper()
	u := auth.NewUser{Username: username, Email: username + "@example.com", Name: strings.ToUpper(username), Password: "password123", Role: role}
	if _, err := auth.CreateUser(context.Background(), db.New(e.pool), u); err != nil {
		e.t.Fatal(err)
	}
}

type resp struct {
	Code    string          `json:"code"`
	Data    json.RawMessage `json:"data"`
	Errors  map[string]string
	Status  int
	Message string
}

func (e *testEnv) do(method, path, token string, body any) resp {
	e.t.Helper()
	var rd io.Reader = http.NoBody
	if body != nil {
		b, _ := json.Marshal(body)
		rd = bytes.NewReader(b)
	}
	req := httptest.NewRequest(method, path, rd)
	req.RemoteAddr = "127.0.0.1:1"
	req.Header.Set("X-Real-IP", "10.0.0.1")
	if token != "" {
		req.Header.Set("Authorization", "Bearer "+token)
	}
	rec := httptest.NewRecorder()
	e.h.ServeHTTP(rec, req)
	var r resp
	if err := json.Unmarshal(rec.Body.Bytes(), &r); err != nil {
		e.t.Fatalf("bad body %q: %v", rec.Body.String(), err)
	}
	r.Status = rec.Code
	return r
}

func (e *testEnv) login(username, password string) resp {
	return e.do("POST", "/api/v1/auth/login", "", map[string]string{"username": username, "password": password})
}

func (e *testEnv) token(username string) string {
	e.t.Helper()
	r := e.login(username, "password123")
	var d struct{ Token string }
	_ = json.Unmarshal(r.Data, &d)
	if d.Token == "" {
		e.t.Fatalf("no token: %+v", r)
	}
	return d.Token
}

func want(t *testing.T, r resp, status int, code string) {
	t.Helper()
	if r.Status != status || r.Code != code {
		t.Fatalf("got %d %q, want %d %q (%s)", r.Status, r.Code, status, code, r.Message)
	}
}

func TestLogin(t *testing.T) {
	e := newEnv(t)
	e.seed("alice", auth.RoleMember)

	r := e.login("Alice", "password123") // username is case-insensitive
	want(t, r, 200, "ok")
	var d struct {
		Token     string
		ExpiresAt time.Time `json:"expires_at"`
		User      map[string]any
	}
	if err := json.Unmarshal(r.Data, &d); err != nil {
		t.Fatal(err)
	}
	if d.Token == "" || d.User["username"] != "alice" || d.User["role"] != "member" {
		t.Fatalf("bad data: %s", r.Data)
	}
	if _, leaked := d.User["password_hash"]; leaked {
		t.Fatal("password_hash leaked")
	}
	if got := time.Until(d.ExpiresAt); got < 167*time.Hour || got > 169*time.Hour {
		t.Fatalf("expires in %v, want ~7 days", got)
	}

	// Only the hash is stored.
	h := sha256.Sum256([]byte(d.Token))
	if _, err := db.New(e.pool).GetUserBySessionToken(context.Background(), h[:]); err != nil {
		t.Fatalf("session not stored by sha256: %v", err)
	}

	wrong := e.login("alice", "nope")
	want(t, wrong, 401, "invalid_credentials")
	unknown := e.login("ghost", "password123")
	want(t, unknown, 401, "invalid_credentials")
	if wrong.Message != unknown.Message {
		t.Fatal("wrong password and unknown user responses differ")
	}
}

func TestMe(t *testing.T) {
	e := newEnv(t)
	e.seed("alice", auth.RoleMember)
	tok := e.token("alice")

	r := e.do("GET", "/api/v1/auth/me", tok, nil)
	want(t, r, 200, "ok")
	if !strings.Contains(string(r.Data), `"username":"alice"`) {
		t.Fatalf("data = %s", r.Data)
	}

	want(t, e.do("GET", "/api/v1/auth/me", "", nil), 401, "unauthorized")
	want(t, e.do("GET", "/api/v1/auth/me", "garbage", nil), 401, "unauthorized")

	// Expired session.
	h := sha256.Sum256([]byte("expiredtoken"))
	err := db.New(e.pool).CreateSession(context.Background(), db.CreateSessionParams{
		TokenHash: h[:], Username: "alice",
		ExpiresAt: pgtype.Timestamptz{Time: time.Now().Add(-time.Minute), Valid: true},
	})
	if err != nil {
		t.Fatal(err)
	}
	want(t, e.do("GET", "/api/v1/auth/me", "expiredtoken", nil), 401, "unauthorized")
}

func TestLogoutEndsOnlyCurrentSession(t *testing.T) {
	e := newEnv(t)
	e.seed("alice", auth.RoleMember)
	a, b := e.token("alice"), e.token("alice")

	want(t, e.do("POST", "/api/v1/auth/logout", a, nil), 200, "ok")
	want(t, e.do("GET", "/api/v1/auth/me", a, nil), 401, "unauthorized")
	want(t, e.do("GET", "/api/v1/auth/me", b, nil), 200, "ok")
	want(t, e.do("POST", "/api/v1/auth/logout", "", nil), 401, "unauthorized")
}

func TestRegister(t *testing.T) {
	e := newEnv(t)
	e.seed("root", auth.RoleSuperadmin)
	e.seed("alice", auth.RoleMember)
	body := map[string]string{"username": "Bob", "email": "bob@example.com", "name": "Bob", "password": "password123"}

	want(t, e.do("POST", "/api/v1/auth/register", "", body), 401, "unauthorized")
	want(t, e.do("POST", "/api/v1/auth/register", e.token("alice"), body), 403, "forbidden")

	root := e.token("root")
	r := e.do("POST", "/api/v1/auth/register", root, body)
	want(t, r, 201, "ok")
	if !strings.Contains(string(r.Data), `"username":"bob"`) || !strings.Contains(string(r.Data), `"role":"member"`) || strings.Contains(string(r.Data), "password") {
		t.Fatalf("data = %s", r.Data)
	}
	want(t, e.login("bob", "password123"), 200, "ok")

	want(t, e.do("POST", "/api/v1/auth/register", root, body), 409, "username_taken")
	body["username"] = "bob2"
	body["email"] = "BOB@example.com"
	want(t, e.do("POST", "/api/v1/auth/register", root, body), 409, "email_taken")

	r = e.do("POST", "/api/v1/auth/register", root, map[string]string{"username": "x", "email": "bad", "password": "short"})
	want(t, r, 422, "validation_failed")
	for _, f := range []string{"username", "email", "name", "password"} {
		if r.Errors[f] == "" {
			t.Errorf("no field error for %s: %v", f, r.Errors)
		}
	}
}

func TestLoginRateLimit(t *testing.T) {
	e := newEnv(t)
	for i := 0; i < 5; i++ {
		want(t, e.login("ghost", "x"), 401, "invalid_credentials")
	}
	want(t, e.login("ghost", "x"), 429, "rate_limited")
}
