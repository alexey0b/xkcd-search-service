package update

import (
	"context"
	"log/slog"
	"search-service/api/core"
	updatepb "search-service/proto/update"
	"time"

	"google.golang.org/grpc"
	"google.golang.org/grpc/backoff"
	"google.golang.org/grpc/codes"
	"google.golang.org/grpc/credentials/insecure"
	healthpb "google.golang.org/grpc/health/grpc_health_v1"
	"google.golang.org/grpc/status"

	"google.golang.org/protobuf/types/known/emptypb"
)

// Client is a gRPC client for the Update service.
type Client struct {
	log          *slog.Logger
	conn         *grpc.ClientConn
	client       updatepb.UpdateClient
	healthClient healthpb.HealthClient
}

// NewClient creates a new Update service gRPC client with exponential backoff retry.
func NewClient(address string, log *slog.Logger) (*Client, error) {
	conn, err := grpc.NewClient(
		address,
		grpc.WithTransportCredentials(insecure.NewCredentials()),
		grpc.WithConnectParams(grpc.ConnectParams{
			Backoff: backoff.Config{
				BaseDelay:  1 * time.Second,
				Multiplier: 1.6,
				MaxDelay:   10 * time.Second,
			},
			MinConnectTimeout: 10 * time.Second,
		}),
	)
	if err != nil {
		return nil, err
	}
	return &Client{
		log:          log,
		conn:         conn,
		client:       updatepb.NewUpdateClient(conn),
		healthClient: healthpb.NewHealthClient(conn),
	}, nil
}

// Close closes the gRPC connection.
func (c *Client) Close() {
	if err := c.conn.Close(); err != nil {
		c.log.Warn("failed to close gRPC connection", "error", err)
	}
}

// HealthCheck checks if the Update service is healthy and serving requests.
func (c *Client) HealthCheck(ctx context.Context) error {
	resp, err := c.healthClient.Check(ctx, &healthpb.HealthCheckRequest{})
	if err != nil {
		if status.Code(err) == codes.Unavailable {
			return core.ErrServiceUnavailable
		}
		return err
	}
	if resp.Status != healthpb.HealthCheckResponse_SERVING {
		return core.ErrServiceUnavailable
	}
	return nil
}

// Status returns the current status of the database update process.
func (c *Client) Status(ctx context.Context) (core.UpdateStatus, error) {
	reply, err := c.client.Status(ctx, &emptypb.Empty{})
	if err != nil {
		if status.Code(err) == codes.Unavailable {
			return core.UpdateUnknown, core.ErrServiceUnavailable
		}
		return core.UpdateUnknown, err
	}
	switch reply.GetStatus() {
	case updatepb.Status_STATUS_IDLE:
		return core.UpdateIdle, nil
	case updatepb.Status_STATUS_RUNNING:
		return core.UpdateRunning, nil
	default:
		return core.UpdateUnknown, nil
	}
}

// Stats returns current database statistics.
// Includes total/unique words count and fetched/total comics count.
func (c *Client) Stats(ctx context.Context) (core.UpdateStats, error) {
	reply, err := c.client.Stats(ctx, &emptypb.Empty{})
	if err != nil {
		if status.Code(err) == codes.Unavailable {
			return core.UpdateStats{}, core.ErrServiceUnavailable
		}
		return core.UpdateStats{}, err
	}
	return core.UpdateStats{
		WordsTotal:    reply.GetWordsTotal(),
		WordsUnique:   reply.GetWordsUnique(),
		ComicsFetched: reply.GetComicsFetched(),
		ComicsTotal:   reply.GetComicsTotal(),
	}, nil
}

// Update triggers asynchronous database update to fetch new comics from xkcd API.
func (c *Client) Update(ctx context.Context) error {
	_, err := c.client.Update(ctx, &emptypb.Empty{})
	if err != nil {
		switch status.Code(err) {
		case codes.Unavailable:
			return core.ErrServiceUnavailable
		case codes.AlreadyExists:
			return core.ErrAlreadyExists
		default:
			return err
		}
	}
	return nil
}

// Drop removes all comics and indexed words from the database.
func (c *Client) Drop(ctx context.Context) error {
	if _, err := c.client.Drop(ctx, &emptypb.Empty{}); err != nil {
		if status.Code(err) == codes.Unavailable {
			return core.ErrServiceUnavailable
		}
		return err
	}
	return nil
}
