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
	"google.golang.org/grpc/status"
	"google.golang.org/protobuf/types/known/emptypb"
)

type Client struct {
	log    *slog.Logger
	conn   *grpc.ClientConn
	client updatepb.UpdateClient
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
		client: updatepb.NewUpdateClient(conn),
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

func (c *Client) Status(ctx context.Context) (core.UpdateStatus, error) {
	reply, err := c.client.Status(ctx, &emptypb.Empty{})
	if err != nil {
		if status.Code(err) == codes.Unavailable {
			return core.StatusUpdateUnknown, core.ErrServiceUnavailable
		}
		return core.StatusUpdateUnknown, err
	}
	switch reply.GetStatus() {
	case updatepb.Status_STATUS_IDLE:
		return core.StatusUpdateIdle, nil
	case updatepb.Status_STATUS_RUNNING:
		return core.StatusUpdateRunning, nil
	default:
		return core.StatusUpdateUnknown, nil
	}
}

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

func (c *Client) Drop(ctx context.Context) error {
	if _, err := c.client.Drop(ctx, &emptypb.Empty{}); err != nil {
		if status.Code(err) == codes.Unavailable {
			return core.ErrServiceUnavailable
		}
		return err
	}
	return nil
}
