package grpc

import (
	"context"
	"errors"
	searchpb "search-service/proto/search"
	"search-service/search/core"

	"google.golang.org/grpc/codes"
	"google.golang.org/grpc/status"
	"google.golang.org/protobuf/types/known/emptypb"
)

func NewServer(service core.Searcher) *Server {
	return &Server{service: service}
}

type Server struct {
	service core.Searcher
	searchpb.UnimplementedSearchServer
}

func (s *Server) Ping(_ context.Context, _ *emptypb.Empty) (*emptypb.Empty, error) {
	return nil, nil
}

func (s *Server) Search(in *searchpb.SearchRequest, stream searchpb.Search_SearchServer) error {
	reply, err := s.service.Search(stream.Context(), in.GetPhrase(), in.GetLimit())
	if err != nil {
		if errors.Is(err, core.ErrBadArguments) {
			return status.Error(codes.InvalidArgument, err.Error())
		}
		return status.Error(codes.Internal, err.Error())
	}
	for _, comic := range reply {
		if err := stream.Send(&searchpb.SearchReply{Id: comic.ID, Url: comic.URL}); err != nil {
			return status.Error(codes.Internal, err.Error())
		}
	}
	return nil
}

func (s *Server) ISearch(in *searchpb.SearchRequest, stream searchpb.Search_SearchServer) error {
	reply, err := s.service.ISearch(stream.Context(), in.GetPhrase(), in.GetLimit())
	if err != nil {
		if errors.Is(err, core.ErrBadArguments) {
			return status.Error(codes.InvalidArgument, err.Error())
		}
		return status.Error(codes.Internal, err.Error())
	}
	for _, comic := range reply {
		if err := stream.Send(&searchpb.SearchReply{Id: comic.ID, Url: comic.URL}); err != nil {
			return status.Error(codes.Internal, err.Error())
		}
	}
	return nil
}
