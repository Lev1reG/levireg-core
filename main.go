package main

import (
	"context"
	"errors"
	"net"
	"net/http"
	"os/signal"
	"syscall"
	"time"

	"go.uber.org/zap"

	"github.com/Lev1reG/levireg-core/internal/config"
	"github.com/Lev1reG/levireg-core/internal/logger"
	"github.com/Lev1reG/levireg-core/internal/server"
)

func main() {
	cfg := config.Load()

	log, err := logger.New(cfg.IsProd)
	if err != nil {
		panic(err)
	}
	defer func() { _ = log.Sync() }()

	srv := server.New(log, net.JoinHostPort("", cfg.Port))

	ctx, stop := signal.NotifyContext(context.Background(), syscall.SIGINT, syscall.SIGTERM)
	defer stop()

	go func() {
		log.Info("server listening", zap.String("addr", srv.Addr))
		if err := srv.ListenAndServe(); err != nil && !errors.Is(err, http.ErrServerClosed) {
			log.Fatal("server failed", zap.Error(err))
		}
	}()

	<-ctx.Done()
	log.Info("shutting down")

	shutdownCtx, cancel := context.WithTimeout(context.Background(), 10*time.Second)
	defer cancel()
	if err := srv.Shutdown(shutdownCtx); err != nil {
		log.Error("graceful shutdown failed", zap.Error(err))
	}
}
