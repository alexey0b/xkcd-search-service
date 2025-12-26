package main

import (
	"context"
	"flag"
	"fmt"
	"log/slog"
	"net"
	"os"
	"os/signal"
	updatepb "search-service/proto/update"
	"search-service/update/adapters/db"
	updategrpc "search-service/update/adapters/grpc"
	"search-service/update/adapters/publisher"
	"search-service/update/adapters/words"
	"search-service/update/adapters/xkcd"
	"search-service/update/config"
	"search-service/update/core"
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
	log.Info("starting Update service...")
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

	// Service
	updater, err := core.NewService(log, storage, xkcd, words, publisher, cfg.XKCD.Concurrency)
	if err != nil {
		return fmt.Errorf("failed create Update service: %v", err)
	}

	// gRPC server
	listener, err := net.Listen("tcp", cfg.Address)
	if err != nil {
		return fmt.Errorf("failed to listen: %v", err)
	}

	s := grpc.NewServer()
	updatepb.RegisterUpdateServer(s, updategrpc.NewServer(updater))
	reflection.Register(s)

	ctx, cancel := signal.NotifyContext(context.Background(), syscall.SIGINT, syscall.SIGTERM)
	defer cancel()

	go func() {
		<-ctx.Done()
		log.Debug("shutting down Update service...")

		done := make(chan struct{})
		go func() {
			s.GracefulStop()
			close(done)
		}()

		select {
		case <-done:
			log.Debug("Update service stopped gracefully")
		case <-time.After(30 * time.Second):
			log.Debug("Update service forcing shutdown")
			s.Stop()
		}
	}()

	log.Info("Update service started", "address", cfg.Address, "log_level", cfg.LogLevel)
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
