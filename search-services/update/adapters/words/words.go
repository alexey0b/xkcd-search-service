package words

import (
	"context"
	"log/slog"
	wordspb "search-service/proto/words"
	"search-service/update/core"
	"time"

	"google.golang.org/grpc"
	"google.golang.org/grpc/backoff"
	"google.golang.org/grpc/codes"
	"google.golang.org/grpc/credentials/insecure"
	"google.golang.org/grpc/status"
	"google.golang.org/protobuf/types/known/emptypb"
)

type Client struct {
	log    *slog.Logger
	conn   *grpc.ClientConn
	client wordspb.WordsClient
}

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
		log:    log,
		conn:   conn,
		client: wordspb.NewWordsClient(conn),
	}, nil
}

func (c *Client) Close() {
	if err := c.conn.Close(); err != nil {
		c.log.Warn("failed to close gRPC connection", "error", err)
	}
}

func (c *Client) Ping(ctx context.Context) error {
	if _, err := c.client.Ping(ctx, &emptypb.Empty{}); err != nil {
		if status.Code(err) == codes.Unavailable {
			return core.ErrServiceUnavailable
		}
		return err
	}
	return nil
}

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
