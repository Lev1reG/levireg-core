package server

import (
	"net/http"
	"time"

	"github.com/go-chi/chi/v5"
	"github.com/go-chi/chi/v5/middleware"
	"go.uber.org/zap"

	"github.com/Lev1reG/levireg-core/internal/handler"
	"github.com/Lev1reG/levireg-core/internal/logger"
)

func New(log *zap.Logger, addr string) *http.Server {
	r := chi.NewRouter()
	r.Use(middleware.RequestID)
	r.Use(middleware.RealIP)
	r.Use(middleware.Recoverer)
	r.Use(logger.RequestLogger(log))

	// Outside /api/v1 on purpose: probes must not inherit API middleware
	// such as auth or rate limiting.
	r.Get("/healthz", handler.Healthz)

	r.Route("/api/v1", func(r chi.Router) {
		// No endpoints yet.
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
