package main

import (
	"context"
	"log/slog"
	"net/http"
	"os"
	"os/signal"
	"strings"
	"syscall"
	"time"

	"strangerbot/internal/bot"
	"strangerbot/internal/config"
	"strangerbot/internal/store"
)

func main() {
	cfg := config.Load()
	if err := cfg.Validate(); err != nil {
		slog.Error("invalid configuration", "err", err)
		os.Exit(1)
	}

	log := slog.New(slog.NewTextHandler(os.Stderr, &slog.HandlerOptions{Level: parseLevel(cfg.LogLevel)}))
	slog.SetDefault(log)

	st, err := store.Open(cfg.DBPath, cfg)
	if err != nil {
		log.Error("open store", "err", err)
		os.Exit(1)
	}
	if driver := strings.ToLower(strings.TrimSpace(cfg.DBDriver)); driver == "mongo" || driver == "mongodb" {
		log.Info("store opened", "driver", "mongodb", "database", cfg.MongoDatabase)
	} else {
		log.Info("store opened", "driver", "json", "path", cfg.DBPath)
	}

	b, err := bot.New(cfg, st, log)
	if err != nil {
		log.Error("start bot", "err", err)
		os.Exit(1)
	}

	if cfg.Port != "" {
		go serveHealth(cfg.Port, log)
	}

	ctx, stop := signal.NotifyContext(context.Background(), os.Interrupt, syscall.SIGTERM)
	defer stop()

	log.Info("strangerbot is running", "name", cfg.BotName)
	if err := b.Run(ctx); err != nil {
		log.Error("bot exited", "err", err)
		os.Exit(1)
	}
	log.Info("bye")
}

// parseLevel maps a log level string to a slog level.
func parseLevel(s string) slog.Level {
	switch strings.ToLower(strings.TrimSpace(s)) {
	case "debug":
		return slog.LevelDebug
	case "warn", "warning":
		return slog.LevelWarn
	case "error":
		return slog.LevelError
	default:
		return slog.LevelInfo
	}
}

// serveHealth starts a minimal HTTP endpoint used by deployment probes.
func serveHealth(port string, log *slog.Logger) {
	if !strings.HasPrefix(port, ":") {
		port = ":" + port
	}
	mux := http.NewServeMux()
	mux.HandleFunc("/healthz", func(w http.ResponseWriter, _ *http.Request) {
		w.Header().Set("Content-Type", "text/plain; charset=utf-8")
		_, _ = w.Write([]byte("ok"))
	})
	srv := &http.Server{
		Addr:              port,
		Handler:           mux,
		ReadHeaderTimeout: 5 * time.Second,
	}
	log.Info("health server listening", "addr", port)
	if err := srv.ListenAndServe(); err != nil && err != http.ErrServerClosed {
		log.Warn("health server stopped", "err", err)
	}
}
