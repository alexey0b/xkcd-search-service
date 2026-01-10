package main

import (
	"context"
	"embed"
	"errors"
	"flag"
	"fmt"
	"io/fs"
	"log/slog"
	"net/http"
	"os"
	"os/signal"
	"search-service/frontend/adapters/api"
	"search-service/frontend/adapters/web"
	"search-service/frontend/adapters/web/middleware"
	"search-service/frontend/config"
	"sync"
	"syscall"
	"time"

	"github.com/prometheus/client_golang/prometheus/promhttp"
)

//go:embed adapters/web/templates
var templatesFiles embed.FS

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
	log.Debug("debug messages are enabled")

	// API adapter
	api := api.NewClient(cfg.Api.Address, cfg.Api.Timeout, log)

	jwtAth, err := middleware.NewJwtAuthenticator(cfg.Auth.AdminUser, cfg.Auth.AdminPassword, cfg.Auth.JwtSecret, cfg.Auth.TokenTtl)
	if err != nil {
		return fmt.Errorf("cannot init jwt authenticator: %w", err)
	}

	mux := http.NewServeMux()

	// Web health
	mux.Handle("GET /health", web.NewHealthHandler())

	// HTML pages
	htmlFiles, err := fs.Sub(templatesFiles, "adapters/web/templates")
	if err != nil {
		return fmt.Errorf("cannot create html files subtree: %w", err)
	}
	mux.Handle("GET /", web.NewPageHandler(htmlFiles, "search.html"))
	mux.Handle("GET /login", web.NewPageHandler(htmlFiles, "admin-login.html"))
	mux.Handle("GET /admin", jwtAth.CheckToken(web.NewPageHandler(htmlFiles, "admin.html")))

	// Static files
	staticFiles, err := fs.Sub(templatesFiles, "adapters/web/templates/static")
	if err != nil {
		return fmt.Errorf("cannot create html files subtree: %w", err)
	}
	mux.Handle("GET /static/", http.StripPrefix("/static/", http.FileServerFS(staticFiles)))

	// API endpoints
	mux.Handle("GET /api/search", web.NewSearchHandler(log, api))
	mux.Handle("POST /api/login", web.NewLoginHandler(log, jwtAth, cfg.Auth.TokenTtl))
	mux.Handle("GET /api/ping", web.NewPingHandler(log, api))

	// API admin endpoints (requires JWT)
	mux.Handle("GET /api/admin/statistics", jwtAth.CheckToken(web.NewStatisticsHandler(log, api)))
	mux.Handle("POST /api/admin/update", jwtAth.CheckToken(web.NewUpdateHandler(log, api)))
	mux.Handle("DELETE /api/admin/db", jwtAth.CheckToken(web.NewDropHandler(log, api)))

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

	// Web server
	webServer := http.Server{
		Addr:        cfg.WebServer.Address,
		ReadTimeout: cfg.WebServer.Timeout,
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

		// Stop Web server
		wg.Go(func() {
			ctxTimeout, cancel := context.WithTimeout(context.Background(), 30*time.Second)
			defer cancel()
			log.Debug("Stopping Web server...")
			if err := webServer.Shutdown(ctxTimeout); err != nil {
				log.Error("web shutdown error", "error", err)
			} else {
				log.Debug("Web server stopped")
			}
		})

		wg.Wait()
		log.Debug("All servers stopped")
	}()

	log.Info("Web server started", "address", cfg.WebServer.Address)
	if err := webServer.ListenAndServe(); err != nil {
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
