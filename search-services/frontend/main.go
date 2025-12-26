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
	"syscall"
	"time"
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
	log.Info("starting Web server")
	log.Debug("debug messages are enabled")

	// API adapter
	api := api.NewClient(cfg.Api.ApiAddress, cfg.Api.Timeout, log)

	jwtAth, err := middleware.NewJwtAuthenticator(cfg.Auth.AdminUser, cfg.Auth.AdminPassword, cfg.Auth.JwtSecret, cfg.Auth.TokenTtl)
	if err != nil {
		return fmt.Errorf("cannot init jwt authenticator: %w", err)
	}

	mux := http.NewServeMux()

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

	handler := middleware.Logging(mux, log)
	handler = middleware.PanicRecovery(handler, log)

	server := http.Server{
		Addr:        cfg.Web.Address,
		ReadTimeout: cfg.Web.Timeout,
		Handler:     handler,
	}

	ctx, cancel := signal.NotifyContext(context.Background(), syscall.SIGINT, syscall.SIGTERM)
	defer cancel()

	go func() {
		<-ctx.Done()
		log.Debug("shutting down Web server...")

		ctxTimeout, cancel := context.WithTimeout(context.Background(), 30*time.Second)
		defer cancel()
		if err := server.Shutdown(ctxTimeout); err != nil {
			log.Error("erroneous shutdown", "error", err)
			return
		}
		log.Debug("Web server stopped gracefully")
	}()

	log.Info("Running Web server", "address", cfg.Web.Address)
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
