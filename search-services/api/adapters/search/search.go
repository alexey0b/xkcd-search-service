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
	"google.golang.org/grpc/status"
	"google.golang.org/protobuf/types/known/emptypb"
)

type Client struct {
	log    *slog.Logger
	conn   *grpc.ClientConn
	client searchpb.SearchClient
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
		client: searchpb.NewSearchClient(conn),
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
	comics, err := collectCommics(stream)
	return comics, err
}

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
	comics, err := collectCommics(stream)
	return comics, err
}

func collectCommics(stream grpc.ServerStreamingClient[searchpb.SearchReply]) ([]core.Comic, error) {
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
