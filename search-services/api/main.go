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
	"sync"
	"syscall"
	"time"

	"github.com/prometheus/client_golang/prometheus/promhttp"
)

func main() {
	var configPath string
	flag.StringVar(&configPath, "config", "config.yaml", "server configuration file")
	flag.Parse()

	var cfg config.Config
	config.MustLoad(configPath, &cfg)

	log := mustMakeLogger(cfg.LogLevel)

	if err := run(cfg, log); err != nil {
		log.Error("server failed", "error", err)
		os.Exit(1)
	}
}

func run(cfg config.Config, log *slog.Logger) error {
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
	mux.Handle("GET /api/health", rest.NewHealthHandler(
		log,
		map[string]core.HealthChecker{
			"update": update,
			"words":  words,
			"search": search,
		}),
	)

	handler := middleware.Metrics(mux)
	handler = middleware.Logging(handler, log)
	handler = middleware.PanicRecovery(handler, log)

	// Metrics server
	metricsServer := http.Server{
		Addr:        cfg.PromServer.Address,
		ReadTimeout: cfg.PromServer.Timeout,
		Handler:     promhttp.Handler(),
	}

	go func() {
		log.Info("Metrics server started", "address", cfg.PromServer.Address)
		if err := metricsServer.ListenAndServe(); err != nil {
			log.Error("metrics server error", "error", err)
		}
	}()

	// API server
	apiServer := http.Server{
		Addr:        cfg.ApiServer.Address,
		ReadTimeout: cfg.ApiServer.Timeout,
		Handler:     handler,
	}

	// Graceful shutdown
	ctx, cancel := signal.NotifyContext(context.Background(), syscall.SIGINT, syscall.SIGTERM)
	defer cancel()

	go func() {
		<-ctx.Done()
		var wg sync.WaitGroup

		// Stop Metrics server
		wg.Go(func() {
			ctxTimeout, cancel := context.WithTimeout(context.Background(), 10*time.Second)
			defer cancel()
			log.Debug("Stopping Metrics server...")
			if err := metricsServer.Shutdown(ctxTimeout); err != nil {
				log.Error("metrics shutdown error", "error", err)
			} else {
				log.Debug("Metrics server stopped")
			}
		})

		// Stop API server
		wg.Go(func() {
			ctxTimeout, cancel := context.WithTimeout(context.Background(), 30*time.Second)
			defer cancel()
			log.Debug("Stopping API server...")
			if err := apiServer.Shutdown(ctxTimeout); err != nil {
				log.Error("api shutdown error", "error", err)
			} else {
				log.Debug("API server stopped")
			}
		})

		wg.Wait()
		log.Debug("All servers stopped")
	}()

	log.Info("API server started", "address", cfg.ApiServer.Address)
	if err := apiServer.ListenAndServe(); err != nil {
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
