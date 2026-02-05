package search

import (
	"context"
	"io"
	"log/slog"
	"search-service/api/core"
	searchpb "search-service/proto/search"
	"time"

	"google.golang.org/grpc"
	"google.golang.org/grpc/backoff"
	"google.golang.org/grpc/codes"
	"google.golang.org/grpc/credentials/insecure"
	healthpb "google.golang.org/grpc/health/grpc_health_v1"
	"google.golang.org/grpc/status"
)

// Client is a gRPC client for the Search service.
type Client struct {
	log          *slog.Logger
	conn         *grpc.ClientConn
	client       searchpb.SearchClient
	healthClient healthpb.HealthClient
}

// NewClient creates a new Search service gRPC client with exponential backoff retry.
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
		client:       searchpb.NewSearchClient(conn),
		healthClient: healthpb.NewHealthClient(conn),
	}, nil
}

// Close closes the gRPC connection.
func (c *Client) Close() {
	if err := c.conn.Close(); err != nil {
		c.log.Warn("failed to close gRPC connection", "error", err)
	}
}

// HealthCheck checks if the Search service is healthy and serving requests.
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

// Search performs exact phrase search and returns matching comics.
func (c *Client) Search(ctx context.Context, phrase string, limite int64) ([]core.Comic, error) {
	stream, err := c.client.Search(ctx, &searchpb.SearchRequest{Phrase: phrase, Limit: limite})
	if err != nil {
		switch status.Code(err) {
		case codes.Unavailable:
			return nil, core.ErrServiceUnavailable
		case codes.InvalidArgument, codes.ResourceExhausted:
			return nil, core.ErrBadArguments
		default:
			return nil, err
		}
	}
	comics, err := collectComics(stream)
	return comics, err
}

// ISearch performs indexed search and returns matching comics.
func (c *Client) ISearch(ctx context.Context, phrase string, limite int64) ([]core.Comic, error) {
	stream, err := c.client.ISearch(ctx, &searchpb.SearchRequest{Phrase: phrase, Limit: limite})
	if err != nil {
		switch status.Code(err) {
		case codes.Unavailable:
			return nil, core.ErrServiceUnavailable
		case codes.InvalidArgument, codes.ResourceExhausted:
			return nil, core.ErrBadArguments
		default:
			return nil, err
		}
	}
	comics, err := collectComics(stream)
	return comics, err
}

// collectComics reads all comics from the gRPC stream.
func collectComics(stream grpc.ServerStreamingClient[searchpb.SearchReply]) ([]core.Comic, error) {
	var comics []core.Comic
	for {
		reply, err := stream.Recv()
		if err != nil {
			if err == io.EOF {
				break
			}
			return nil, err
		}
		comics = append(comics, core.Comic{ID: reply.GetId(), URL: reply.GetUrl()})
	}
	return comics, nil
}
