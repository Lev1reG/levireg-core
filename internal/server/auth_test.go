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

func (e *testEnv) seed(username, role string, approved bool) {
	e.t.Helper()
	u := auth.NewUser{Username: username, Email: username + "@example.com", Name: strings.ToUpper(username), Password: "password123", Role: role, Approved: approved}
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
	Raw     []byte
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
	r.Raw = rec.Body.Bytes()
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
	e.seed("alice", auth.RoleMember, true)

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
	e.seed("alice", auth.RoleMember, true)
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
	e.seed("alice", auth.RoleMember, true)
	a, b := e.token("alice"), e.token("alice")

	want(t, e.do("POST", "/api/v1/auth/logout", a, nil), 200, "ok")
	want(t, e.do("GET", "/api/v1/auth/me", a, nil), 401, "unauthorized")
	want(t, e.do("GET", "/api/v1/auth/me", b, nil), 200, "ok")
	want(t, e.do("POST", "/api/v1/auth/logout", "", nil), 401, "unauthorized")
}

func (e *testEnv) register(username string, extra map[string]string) resp {
	body := map[string]string{"username": username, "email": username + "@example.com", "name": username, "password": "password123"}
	for k, v := range extra {
		body[k] = v
	}
	return e.do("POST", "/api/v1/auth/register", "", body)
}

func TestRegisterCreatesPendingMember(t *testing.T) {
	e := newEnv(t)
	r := e.register("Bob", map[string]string{"role": "superadmin"})
	want(t, r, 201, "ok")
	var u map[string]any
	_ = json.Unmarshal(r.Data, &u)
	if u["username"] != "bob" || u["role"] != "member" || u["approved_at"] != nil {
		t.Fatalf("data = %s", r.Data)
	}
	if _, leaked := u["password_hash"]; leaked {
		t.Fatal("password_hash leaked")
	}

	want(t, e.register("bob", nil), 409, "username_taken")
	want(t, e.register("bob2", map[string]string{"email": "BOB@example.com"}), 409, "email_taken")
}

func TestRegisterValidation(t *testing.T) {
	e := newEnv(t)
	r := e.do("POST", "/api/v1/auth/register", "", map[string]string{"username": "x", "email": "bad", "password": "short"})
	want(t, r, 422, "validation_failed")
	for _, f := range []string{"username", "email", "name", "password"} {
		if r.Errors[f] == "" {
			t.Errorf("no field error for %s: %v", f, r.Errors)
		}
	}
}

func TestRegisterRateLimit(t *testing.T) {
	e := newEnv(t)
	for _, name := range []string{"user_a", "user_b", "user_c"} {
		want(t, e.register(name, nil), 201, "ok")
	}
	want(t, e.register("user_d", nil), 429, "rate_limited")
}

func TestPendingUserCannotLogin(t *testing.T) {
	e := newEnv(t)
	e.seed("root", auth.RoleSuperadmin, true)
	want(t, e.register("bob", nil), 201, "ok")

	want(t, e.login("bob", "password123"), 403, "user_pending")
	want(t, e.login("bob", "wrong"), 401, "invalid_credentials")
	var n int
	if err := e.pool.QueryRow(context.Background(), "SELECT count(*) FROM auth.sessions WHERE username = 'bob'").Scan(&n); err != nil || n != 0 {
		t.Fatalf("sessions = %d, err = %v", n, err)
	}

	root := e.token("root")
	r := e.do("POST", "/api/v1/users/bob/approve", root, nil)
	want(t, r, 200, "ok")
	if strings.Contains(string(r.Data), `"approved_at":null`) {
		t.Fatalf("data = %s", r.Data)
	}
	want(t, e.login("bob", "password123"), 200, "ok")
	want(t, e.do("POST", "/api/v1/users/bob/approve", root, nil), 409, "user_not_pending")
}

func TestRejectFreesUsernameAndEmail(t *testing.T) {
	e := newEnv(t)
	e.seed("root", auth.RoleSuperadmin, true)
	e.seed("alice", auth.RoleMember, true)
	root := e.token("root")
	want(t, e.register("bob", nil), 201, "ok")

	r := e.do("POST", "/api/v1/users/bob/reject", root, nil)
	want(t, r, 200, "ok")
	if len(r.Data) != 0 {
		t.Fatalf("data = %s, want none", r.Data)
	}
	want(t, e.register("bob", nil), 201, "ok")

	want(t, e.do("POST", "/api/v1/users/ghost/reject", root, nil), 404, "user_not_found")
	want(t, e.do("POST", "/api/v1/users/ghost/approve", root, nil), 404, "user_not_found")
	want(t, e.do("POST", "/api/v1/users/alice/reject", root, nil), 409, "user_not_pending")
	want(t, e.login("alice", "password123"), 200, "ok")
}

func TestCreateUserIsApproved(t *testing.T) {
	e := newEnv(t)
	e.seed("root", auth.RoleSuperadmin, true)
	e.seed("alice", auth.RoleMember, true)
	body := map[string]string{"username": "Bob", "email": "bob@example.com", "name": "Bob", "password": "password123", "role": "superadmin"}

	want(t, e.do("POST", "/api/v1/users", "", body), 401, "unauthorized")
	want(t, e.do("POST", "/api/v1/users", e.token("alice"), body), 403, "forbidden")

	r := e.do("POST", "/api/v1/users", e.token("root"), body)
	want(t, r, 201, "ok")
	if !strings.Contains(string(r.Data), `"role":"superadmin"`) || strings.Contains(string(r.Data), `"approved_at":null`) {
		t.Fatalf("data = %s", r.Data)
	}
	want(t, e.login("bob", "password123"), 200, "ok")
	want(t, e.do("POST", "/api/v1/users", e.token("root"), body), 409, "username_taken")
}

func TestUsersRefuseNonSuperadmin(t *testing.T) {
	e := newEnv(t)
	e.seed("alice", auth.RoleMember, true)
	tok := e.token("alice")
	for _, ep := range [][2]string{{"GET", "/api/v1/users"}, {"POST", "/api/v1/users"}, {"POST", "/api/v1/users/alice/approve"}, {"POST", "/api/v1/users/alice/reject"}} {
		want(t, e.do(ep[0], ep[1], "", nil), 401, "unauthorized")
		want(t, e.do(ep[0], ep[1], tok, nil), 403, "forbidden")
	}
}

type listResp struct {
	Data []map[string]any `json:"data"`
	Meta map[string]int   `json:"meta"`
}

func (e *testEnv) list(token, query string) (resp, listResp) {
	e.t.Helper()
	r := e.do("GET", "/api/v1/users"+query, token, nil)
	var l listResp
	_ = json.Unmarshal(r.Raw, &l)
	return r, l
}

func TestListUsers(t *testing.T) {
	e := newEnv(t)
	e.seed("root", auth.RoleSuperadmin, true)
	for _, n := range []string{"pen1", "pen2", "pen3"} {
		e.seed(n, auth.RoleMember, false)
	}
	e.seed("apr1", auth.RoleMember, true)
	// Equal created_at forces the username tie-breaker.
	if _, err := e.pool.Exec(context.Background(), "UPDATE auth.users SET created_at = '2026-01-01'"); err != nil {
		t.Fatal(err)
	}

	root := e.token("root")
	r, l := e.list(root, "")
	want(t, r, 200, "ok")
	if l.Meta["total"] != 5 || l.Meta["page"] != 1 || l.Meta["per_page"] != 20 || len(l.Data) != 5 {
		t.Fatalf("all: %+v", l)
	}
	if l.Data[0]["username"] != "apr1" || l.Data[4]["username"] != "root" {
		t.Fatalf("order: %+v", l.Data)
	}

	_, l = e.list(root, "?status=pending")
	if l.Meta["total"] != 3 || len(l.Data) != 3 || l.Data[0]["approved_at"] != nil {
		t.Fatalf("pending: %+v", l)
	}
	_, l = e.list(root, "?status=approved")
	if l.Meta["total"] != 2 || len(l.Data) != 2 || l.Data[0]["approved_at"] == nil {
		t.Fatalf("approved: %+v", l)
	}

	_, l = e.list(root, "?per_page=2&page=2")
	if l.Meta["total"] != 5 || len(l.Data) != 2 || l.Data[0]["username"] != "pen2" {
		t.Fatalf("page 2: %+v", l)
	}
	r, l = e.list(root, "?page=99999999999")
	want(t, r, 200, "ok")
	if l.Data == nil || len(l.Data) != 0 || l.Meta["total"] != 5 {
		t.Fatalf("past end: %s", r.Raw)
	}

	for _, q := range []string{"?page=0", "?per_page=101", "?page=x", "?status=bogus"} {
		r, _ := e.list(root, q)
		want(t, r, 422, "validation_failed")
		if len(r.Errors) == 0 {
			t.Errorf("%s: no field errors", q)
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
