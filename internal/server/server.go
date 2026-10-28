package server

import (
	"net/http"
	"time"

	"github.com/go-chi/chi/v5"
	"github.com/go-chi/chi/v5/middleware"
	"go.uber.org/zap"

	"github.com/Lev1reG/levireg-core/internal/db"
	"github.com/Lev1reG/levireg-core/internal/handler"
	"github.com/Lev1reG/levireg-core/internal/logger"
)

// DB is what the server needs from the database; *pgxpool.Pool satisfies it.
type DB interface {
	handler.Pinger
	db.DBTX
}

func New(log *zap.Logger, addr string, pool DB) *http.Server {
	r := chi.NewRouter()
	r.Use(middleware.RequestID)
	// nginx on the same host sets X-Real-IP; trust it only from loopback.
	r.Use(middleware.ClientIPFromHeader("X-Real-IP", "127.0.0.1/32", "::1/128"))
	r.Use(middleware.Recoverer)
	r.Use(logger.RequestLogger(log))

	// Outside /api/v1 on purpose: probes must not inherit API middleware
	// such as auth or rate limiting.
	r.Get("/healthz", handler.Healthz)
	r.Get("/readyz", handler.Readyz(pool))

	r.Route("/api/v1", func(r chi.Router) {
		a := handler.Auth{Q: db.New(pool)}
		r.Route("/auth", func(r chi.Router) {
			r.With(handler.RateLimit(5)).Post("/login", a.Login)
			r.Group(func(r chi.Router) {
				r.Use(a.Require)
				r.Post("/logout", a.Logout)
				r.Get("/me", a.Me)
				r.With(handler.RequireSuperadmin).Post("/register", a.Register)
			})
		})
	})

	return &http.Server{
		Addr:              addr,
		Handler:           r,
		ReadHeaderTimeout: 5 * time.Second,
		ReadTimeout:       10 * time.Second,
		WriteTimeout:      15 * time.Second,
		IdleTimeout:       60 * time.Second,
	}
}
