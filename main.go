package main

import (
	"context"
	"errors"
	"log/slog"
	"net"
	"net/http"
	"os"
	"os/signal"
	"strconv"
	"syscall"
	"time"

	"chatgpt2api/api"
	"chatgpt2api/internal/config"
	"chatgpt2api/internal/storage"
	"chatgpt2api/internal/token"
)

func main() {
	logger := slog.New(slog.NewTextHandler(os.Stdout, &slog.HandlerOptions{Level: slog.LevelInfo}))
	slog.SetDefault(logger)

	cfg := config.New("")
	if err := cfg.Load(); err != nil {
		logger.Error("load config", slog.Any("error", err))
		os.Exit(1)
	}

	// Initialize token manager with local storage
	tokenMgr := token.GetInstance()
	tokenMgr.SetStorage(storage.NewLocalStorage())
	if err := tokenMgr.Load(); err != nil {
		logger.Warn("load token pool", slog.Any("error", err))
	}

	// Seed initial token from env if pool is empty and ACCESS_TOKEN is set
	if tokenMgr.GetStats().Total == 0 {
		if envToken := os.Getenv("ACCESS_TOKEN"); envToken != "" {
			tokenMgr.AddToken(envToken, "env:ACCESS_TOKEN")
			tokenMgr.Save()
			logger.Info("seeded token pool from ACCESS_TOKEN env var")
		}
	}

	if tokenMgr.GetStats().Total == 0 {
		logger.Warn("no tokens available — add tokens via /v1/admin/tokens or ACCESS_TOKEN env var")
	}

	host := envString("SERVER_HOST", "0.0.0.0")
	port := envInt("SERVER_PORT", 8080)
	addr := net.JoinHostPort(host, strconv.Itoa(port))

	server := &http.Server{
		Addr:              addr,
		Handler:           api.SetupRouter(cfg),
		ReadHeaderTimeout: 5 * time.Second,
	}

	logger.Info("chatgpt2api listening", slog.String("addr", addr), slog.Int("tokens", tokenMgr.GetStats().Total))

	errCh := make(chan error, 1)
	go func() {
		if err := server.ListenAndServe(); err != nil && !errors.Is(err, http.ErrServerClosed) {
			errCh <- err
		}
	}()

	ctx, stop := signal.NotifyContext(context.Background(), os.Interrupt, syscall.SIGTERM)
	defer stop()

	select {
	case err := <-errCh:
		logger.Error("server error", slog.Any("error", err))
		os.Exit(1)
	case <-ctx.Done():
		logger.Info("shutdown signal received")
	}

	shutCtx, cancel := context.WithTimeout(context.Background(), 10*time.Second)
	defer cancel()
	server.Shutdown(shutCtx)
	tokenMgr.Save()
	logger.Info("server stopped")
}

func envString(key, fallback string) string {
	if v := os.Getenv(key); v != "" {
		return v
	}
	return fallback
}

func envInt(key string, fallback int) int {
	if v := os.Getenv(key); v != "" {
		if n, err := strconv.Atoi(v); err == nil {
			return n
		}
	}
	return fallback
}
