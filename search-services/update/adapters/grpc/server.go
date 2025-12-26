package grpc

import (
	"context"
	"errors"
	updatepb "search-service/proto/update"
	"search-service/update/core"

	"google.golang.org/grpc/codes"
	"google.golang.org/grpc/status"
	"google.golang.org/protobuf/types/known/emptypb"
)

type Server struct {
	updatepb.UnimplementedUpdateServer
	service core.Updater
}

func NewServer(service core.Updater) *Server {
	return &Server{service: service}
}

func (s *Server) Ping(_ context.Context, _ *emptypb.Empty) (*emptypb.Empty, error) {
	return nil, nil
}

func (s *Server) Status(ctx context.Context, _ *emptypb.Empty) (*updatepb.StatusReply, error) {
	var status updatepb.Status
	switch s.service.Status(ctx) {
	case core.StatusRunning:
		status = updatepb.Status_STATUS_RUNNING
	case core.StatusIdle:
		status = updatepb.Status_STATUS_IDLE
	default:
		status = updatepb.Status_STATUS_UNSPECIFIED
	}
	return &updatepb.StatusReply{Status: status}, nil
}

func (s *Server) Update(ctx context.Context, _ *emptypb.Empty) (*emptypb.Empty, error) {
	err := s.service.Update(ctx)
	if err != nil {
		if errors.Is(err, core.ErrAlreadyExists) {
			return nil, status.Error(codes.AlreadyExists, err.Error())
		}
		return nil, status.Error(codes.Internal, err.Error())
	}
	return nil, err
}

func (s *Server) Stats(ctx context.Context, _ *emptypb.Empty) (*updatepb.StatsReply, error) {
	stats, err := s.service.Stats(ctx)
	if err != nil {
		return nil, status.Error(codes.Internal, err.Error())
	}
	return &updatepb.StatsReply{
		WordsTotal:    stats.WordsTotal,
		WordsUnique:   stats.WordsUnique,
		ComicsFetched: stats.ComicsFetched,
		ComicsTotal:   stats.ComicsTotal,
	}, nil
}

func (s *Server) Drop(ctx context.Context, _ *emptypb.Empty) (*emptypb.Empty, error) {
	if err := s.service.Drop(ctx); err != nil {
		return nil, status.Error(codes.Internal, err.Error())
	}
	return nil, nil
}
