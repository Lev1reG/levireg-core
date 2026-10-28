package main

import (
	"context"
	"errors"
	"fmt"
	"net"
	"net/http"
	"os"
	"os/signal"
	"syscall"
	"time"

	"github.com/jackc/pgx/v5/pgxpool"
	"go.uber.org/zap"

	"github.com/Lev1reG/levireg-core/internal/config"
	"github.com/Lev1reG/levireg-core/internal/db"
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

	ctx, stop := signal.NotifyContext(context.Background(), syscall.SIGINT, syscall.SIGTERM)
	defer stop()

	pool, err := pgxpool.New(ctx, cfg.DatabaseURL)
	if err != nil {
		log.Fatal("database config invalid", zap.Error(err))
	}
	defer pool.Close()

	pingCtx, cancelPing := context.WithTimeout(ctx, 5*time.Second)
	err = pool.Ping(pingCtx)
	cancelPing()
	if err != nil {
		log.Fatal("database unreachable", zap.Error(err))
	}

	if len(os.Args) > 1 {
		if os.Args[1] != "create-superadmin" {
			log.Fatal("unknown subcommand", zap.String("name", os.Args[1]))
		}
		if err := createSuperadmin(ctx, db.New(pool), os.Args[2:], os.Stdin); err != nil {
			fmt.Fprintln(os.Stderr, "error:", err)
			os.Exit(1)
		}
		fmt.Println("superadmin created")
		return
	}

	srv := server.New(log, net.JoinHostPort("", cfg.Port), pool)

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
