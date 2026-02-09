package main

import (
	"context"
	"flag"
	"fmt"
	"log/slog"
	"net"
	"net/http"
	"os"
	"os/signal"
	updatepb "search-service/proto/update"
	"search-service/update/adapters/db"
	updategrpc "search-service/update/adapters/grpc"
	"search-service/update/adapters/metrics"
	"search-service/update/adapters/publisher"
	"search-service/update/adapters/words"
	"search-service/update/adapters/xkcd"
	"search-service/update/config"
	"search-service/update/core"
	"sync"
	"syscall"
	"time"

	"github.com/prometheus/client_golang/prometheus/promhttp"
	"google.golang.org/grpc/health"
	healthgrpc "google.golang.org/grpc/health/grpc_health_v1"

	"google.golang.org/grpc"
	"google.golang.org/grpc/reflection"
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

	// Database adapter
	storage, err := db.New(log, cfg.DBAddress)
	if err != nil {
		return fmt.Errorf("failed to connect to db: %v", err)
	}
	defer storage.Close()

	if err := storage.Migrate(); err != nil {
		return fmt.Errorf("failed to migrate db: %v", err)
	}

	// xkcd adapter
	xkcd, err := xkcd.NewClient(cfg.XKCD.URL, cfg.XKCD.Timeout, log)
	if err != nil {
		return fmt.Errorf("failed create XKCD client: %v", err)
	}

	// Words adapter
	words, err := words.NewClient(cfg.WordsAddress, log)
	if err != nil {
		return fmt.Errorf("failed create Words client: %v", err)
	}
	defer words.Close()

	// Publisher adapter
	publisher, err := publisher.NewNatsPublisher(cfg.Broker.Address, cfg.Broker.Subject, log)
	if err != nil {
		return fmt.Errorf("failed create Nats publisher: %w", err)
	}
	defer publisher.Close()

	// Metrics adapter
	promCollector := metrics.NewPrometheusCollector()

	// Service
	updater, err := core.NewService(log, promCollector, storage, xkcd, words, publisher, cfg.XKCD.Concurrency)
	if err != nil {
		return fmt.Errorf("failed create Update service: %v", err)
	}

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

	// gRPC server
	listener, err := net.Listen("tcp", cfg.UpdateAddress)
	if err != nil {
		return fmt.Errorf("failed to listen: %v", err)
	}

	s := grpc.NewServer()
	healthcheck := health.NewServer()

	// register gRPC services
	healthgrpc.RegisterHealthServer(s, healthcheck)
	updatepb.RegisterUpdateServer(s, updategrpc.NewServer(updater))
	reflection.Register(s)

	ctx, cancel := signal.NotifyContext(context.Background(), syscall.SIGINT, syscall.SIGTERM)
	defer cancel()

	// Graceful shutdown
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

		// Stop gRPC server
		wg.Go(func() {
			log.Debug("Stopping gRPC server...")
			done := make(chan struct{})
			go func() {
				s.GracefulStop()
				close(done)
			}()
			select {
			case <-done:
				log.Debug("gRPC server stopped")
			case <-time.After(30 * time.Second):
				log.Debug("gRPC server forcing shutdown")
				s.Stop()
			}
		})

		wg.Wait()
		log.Debug("All servers stopped")
	}()

	log.Info("Update server started", "address", cfg.UpdateAddress, "log_level", cfg.LogLevel)
	if err := s.Serve(listener); err != nil {
		return fmt.Errorf("failed to serve: %v", err)
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
