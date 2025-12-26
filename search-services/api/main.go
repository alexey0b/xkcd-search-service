package main

import (
	"context"
	"errors"
	"flag"
	"fmt"
	"log/slog"
	"net/http"
	"os"
	"os/signal"
	"search-service/api/adapters/rest"
	"search-service/api/adapters/rest/middleware"
	"search-service/api/adapters/search"
	"search-service/api/adapters/update"
	"search-service/api/adapters/words"
	"search-service/api/config"
	"search-service/api/core"
	"syscall"
	"time"
)

func main() {
	var configPath string
	flag.StringVar(&configPath, "config", "config.yaml", "server configuration file")
	flag.Parse()

	var cfg config.Config
	config.MustLoad(configPath, &cfg)

	// Logger
	log := mustMakeLogger(cfg.LogLevel)

	if err := run(cfg, log); err != nil {
		log.Error("server failed", "error", err)
		os.Exit(1)
	}
}

func run(cfg config.Config, log *slog.Logger) error {
	log.Info("starting api server")
	log.Debug("debug messages are enabled")

	// Update adapter
	update, err := update.NewClient(cfg.UpdateAddress, log)
	if err != nil {
		return fmt.Errorf("cannot init Update adapter: %w", err)
	}
	defer update.Close()

	// Words adapter
	words, err := words.NewClient(cfg.WordsAddress, log)
	if err != nil {
		return fmt.Errorf("cannot init Words adapter: %w", err)
	}
	defer words.Close()

	// Search adapter
	search, err := search.NewClient(cfg.SearchAddress, log)
	if err != nil {
		return fmt.Errorf("cannot init Search adapter: %w", err)
	}
	defer search.Close()

	// Search limiters
	searchConcLimiter := middleware.NewConcurrencyLimiter(cfg.Limits.SearchConcurrency)
	searchRateLimiter := middleware.NewRateLimiter(cfg.Limits.SearchRate)

	// JWT authenticator
	jwtAth, err := middleware.NewJwtAuthenticator(cfg.Auth.AdminUser, cfg.Auth.AdminPassword, cfg.Auth.JwtSecret, cfg.Auth.TokenTtl)
	if err != nil {
		return fmt.Errorf("cannot init jwt authenticator: %w", err)
	}

	mux := http.NewServeMux()

	// API endpoints
	mux.Handle("POST /api/login", rest.NewLoginHandler(log, jwtAth))
	mux.Handle("GET /api/search", searchConcLimiter.Limit(rest.NewSearchHandler(log, search)))
	mux.Handle("GET /api/isearch", searchRateLimiter.Limit(rest.NewISearchHandler(log, search)))

	// API admin endpoints (requires JWT)
	mux.Handle("POST /api/db/update", jwtAth.CheckToken(rest.NewUpdateHandler(log, update)))
	mux.Handle("DELETE /api/db", jwtAth.CheckToken(rest.NewDropHandler(log, update)))

	// API statistics endpoints
	mux.Handle("GET /api/db/stats", rest.NewUpdateStatsHandler(log, update))
	mux.Handle("GET /api/db/status", rest.NewUpdateStatusHandler(log, update))
	mux.Handle("GET /api/ping", rest.NewPingHandler(
		log,
		map[string]core.Pinger{
			"update": update,
			"words":  words,
			"search": search,
		}),
	)

	handler := middleware.Logging(mux, log)
	handler = middleware.PanicRecovery(handler, log)

	server := http.Server{
		Addr:        cfg.ApiConfig.Address,
		ReadTimeout: cfg.ApiConfig.Timeout,
		Handler:     handler,
	}

	ctx, cancel := signal.NotifyContext(context.Background(), syscall.SIGINT, syscall.SIGTERM)
	defer cancel()

	go func() {
		<-ctx.Done()
		log.Debug("shutting down Api server...")

		ctxTimeout, cancel := context.WithTimeout(context.Background(), 30*time.Second)
		defer cancel()
		if err := server.Shutdown(ctxTimeout); err != nil {
			log.Error("erroneous shutdown", "error", err)
			return
		}
		log.Debug("Api server stopped gracefully")
	}()

	log.Info("Running Api server", "address", cfg.ApiConfig.Address)
	if err := server.ListenAndServe(); err != nil {
		if !errors.Is(err, http.ErrServerClosed) {
			return fmt.Errorf("server closed unexpectedly: %w", err)
		}
	}
	return nil
}

func mustMakeLogger(logLevel string) *slog.Logger {
	var level slog.Level
	switch logLevel {
	case "DEBUG":
		level = slog.LevelDebug
	case "INFO":
		level = slog.LevelInfo
	case "ERROR":
		level = slog.LevelError
	default:
		panic("unknown log level: " + logLevel)
	}
	handler := slog.NewTextHandler(os.Stderr, &slog.HandlerOptions{AddSource: true, Level: level})
	return slog.New(handler)
}
