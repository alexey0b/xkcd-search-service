package words

import (
	"context"
	"log/slog"
	"search-service/api/core"
	wordspb "search-service/proto/words"
	"time"

	"google.golang.org/grpc"
	"google.golang.org/grpc/backoff"
	"google.golang.org/grpc/codes"
	"google.golang.org/grpc/credentials/insecure"
	healthpb "google.golang.org/grpc/health/grpc_health_v1"
	"google.golang.org/grpc/status"
)

// Client is a gRPC client for the Words service.
type Client struct {
	log          *slog.Logger
	conn         *grpc.ClientConn
	client       wordspb.WordsClient
	healthClient healthpb.HealthClient
}

// NewClient creates a new Words service gRPC client with exponential backoff retry.
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
		client:       wordspb.NewWordsClient(conn),
		healthClient: healthpb.NewHealthClient(conn),
	}, nil
}

// Close closes the gRPC connection.
func (c *Client) Close() {
	if err := c.conn.Close(); err != nil {
		c.log.Warn("failed to close gRPC connection", "error", err)
	}
}

// HealthCheck checks if the Words service is healthy and serving requests.
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

// Norm normalizes words in the phrase.
func (c *Client) Norm(ctx context.Context, phrase string) ([]string, error) {
	reply, err := c.client.Norm(ctx, &wordspb.WordsRequest{Phrase: phrase})
	if err != nil {
		switch status.Code(err) {
		case codes.Unavailable:
			return nil, core.ErrServiceUnavailable
		case codes.ResourceExhausted:
			return nil, core.ErrBadArguments
		default:
			return nil, err
		}
	}
	return reply.GetWords(), nil
}
