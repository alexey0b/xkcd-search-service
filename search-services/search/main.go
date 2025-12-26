package main

import (
	"context"
	"flag"
	"fmt"
	"log/slog"
	"net"
	"os"
	"os/signal"
	searchpb "search-service/proto/search"
	"search-service/search/adapters/db"
	searchgrpc "search-service/search/adapters/grpc"
	"search-service/search/adapters/scheduler"
	"search-service/search/adapters/subscriber"
	"search-service/search/adapters/words"
	"search-service/search/config"
	"search-service/search/core"
	"syscall"
	"time"

	"google.golang.org/grpc"
	"google.golang.org/grpc/reflection"
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
	log.Info("starting Search service...")
	log.Debug("debug messages are enabled")

	// Database adapter
	storage, err := db.New(log, cfg.DBAddress)
	if err != nil {
		return fmt.Errorf("failed to connect to db: %w", err)
	}
	defer storage.Close()

	// Words adapter
	words, err := words.NewClient(cfg.WordsAddress, log)
	if err != nil {
		return fmt.Errorf("failed create Words client: %w", err)
	}
	defer words.Close()

	// Service
	searcher, err := core.NewService(log, storage, words)
	if err != nil {
		return fmt.Errorf("failed create Search service: %w", err)
	}

	// Subscriber adapter
	subscriber, err := subscriber.NewNatsSubscriber(cfg.Broker.Address, cfg.Broker.Subject, searcher, log)
	if err != nil {
		return fmt.Errorf("failed create Nats subscriber: %w", err)
	}
	defer subscriber.Unsubscribe()

	// Searcher scheduler
	searchSched := scheduler.NewSearcherScheduler(log, searcher, cfg.IndexTTL)

	// gRPC server
	listener, err := net.Listen("tcp", cfg.Address)
	if err != nil {
		return fmt.Errorf("failed to listen: %w", err)
	}

	s := grpc.NewServer()
	searchpb.RegisterSearchServer(s, searchgrpc.NewServer(searcher))
	reflection.Register(s)

	ctx, cancel := signal.NotifyContext(context.Background(), syscall.SIGINT, syscall.SIGTERM)
	defer cancel()

	if err := searchSched.Start(ctx); err != nil {
		return fmt.Errorf("failed to start searcher scheduler: %w", err)
	}

	go func() {
		<-ctx.Done()
		log.Debug("shutting down Search service...")

		done := make(chan struct{})
		go func() {
			s.GracefulStop()
			close(done)
		}()

		select {
		case <-done:
			log.Debug("Search service stopped gracefully")
		case <-time.After(30 * time.Second):
			log.Debug("Search service forcing shutdown")
			s.Stop()
		}
	}()

	log.Info("Search service started", "address", cfg.Address, "log_level", cfg.LogLevel)
	if err := s.Serve(listener); err != nil {
		return fmt.Errorf("failed to serve: %w", err)
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
