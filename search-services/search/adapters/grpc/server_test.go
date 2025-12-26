package grpc_test

import (
	"context"
	"errors"
	searchpb "search-service/proto/search"
	"search-service/search/adapters/grpc"
	"search-service/search/core"
	"testing"

	"github.com/stretchr/testify/require"
	"go.uber.org/mock/gomock"
	"google.golang.org/grpc/codes"
	"google.golang.org/grpc/status"
	"google.golang.org/protobuf/types/known/emptypb"
)

type mockSearchStream struct {
	searchpb.Search_SearchServer
	sent []*searchpb.SearchReply
}

func (m *mockSearchStream) Send(reply *searchpb.SearchReply) error {
	m.sent = append(m.sent, reply)
	return nil
}

func (m *mockSearchStream) Context() context.Context {
	return context.Background()
}

func TestPing(t *testing.T) {
	ctrl := gomock.NewController(t)
	defer ctrl.Finish()

	mockSearcher := core.NewMockSearcher(ctrl)
	server := grpc.NewServer(mockSearcher)

	_, err := server.Ping(context.Background(), &emptypb.Empty{})
	require.NoError(t, err)
}

func TestSearch(t *testing.T) {
	testCases := []struct {
		desc          string
		phrase        string
		limit         int64
		serviceResult []core.Comic
		serviceError  error
		expectedSent  int
		expectedCode  codes.Code
		wantErr       bool
	}{
		{
			desc:   "success - returns comics",
			phrase: "test",
			limit:  10,
			serviceResult: []core.Comic{
				{ID: 1, URL: "http://example.com/1"},
				{ID: 2, URL: "http://example.com/2"},
			},
			expectedSent: 2,
			wantErr:      false,
		},
		{
			desc:          "success - empty result",
			phrase:        "notfound",
			limit:         10,
			serviceResult: []core.Comic{},
			expectedSent:  0,
			wantErr:       false,
		},
		{
			desc:         "error - bad arguments",
			phrase:       "",
			limit:        -1,
			serviceError: core.ErrBadArguments,
			expectedCode: codes.InvalidArgument,
			wantErr:      true,
		},
		{
			desc:         "error - internal error",
			phrase:       "test",
			limit:        10,
			serviceError: errors.New("database error"),
			expectedCode: codes.Internal,
			wantErr:      true,
		},
	}

	for _, tc := range testCases {
		t.Run(tc.desc, func(t *testing.T) {
			ctrl := gomock.NewController(t)
			defer ctrl.Finish()

			mockSearcher := core.NewMockSearcher(ctrl)
			mockSearcher.EXPECT().Search(gomock.Any(), tc.phrase, tc.limit).Return(tc.serviceResult, tc.serviceError)

			server := grpc.NewServer(mockSearcher)
			stream := &mockSearchStream{}

			err := server.Search(&searchpb.SearchRequest{Phrase: tc.phrase, Limit: tc.limit}, stream)

			if tc.wantErr {
				require.Error(t, err)
				require.Equal(t, tc.expectedCode, status.Code(err))
			} else {
				require.NoError(t, err)
				require.Len(t, stream.sent, tc.expectedSent)
				for i, comic := range tc.serviceResult {
					require.Equal(t, comic.ID, stream.sent[i].Id)
					require.Equal(t, comic.URL, stream.sent[i].Url)
				}
			}
		})
	}
}

func TestISearch(t *testing.T) {
	testCases := []struct {
		desc          string
		phrase        string
		limit         int64
		serviceResult []core.Comic
		serviceError  error
		expectedSent  int
		expectedCode  codes.Code
		wantErr       bool
	}{
		{
			desc:   "success - returns comics",
			phrase: "test",
			limit:  10,
			serviceResult: []core.Comic{
				{ID: 1, URL: "http://example.com/1"},
				{ID: 2, URL: "http://example.com/2"},
			},
			expectedSent: 2,
			wantErr:      false,
		},
		{
			desc:          "success - empty result",
			phrase:        "notfound",
			limit:         10,
			serviceResult: []core.Comic{},
			expectedSent:  0,
			wantErr:       false,
		},
		{
			desc:         "error - bad arguments",
			phrase:       "",
			limit:        -1,
			serviceError: core.ErrBadArguments,
			expectedCode: codes.InvalidArgument,
			wantErr:      true,
		},
		{
			desc:         "error - internal error",
			phrase:       "test",
			limit:        10,
			serviceError: errors.New("database error"),
			expectedCode: codes.Internal,
			wantErr:      true,
		},
	}

	for _, tc := range testCases {
		t.Run(tc.desc, func(t *testing.T) {
			ctrl := gomock.NewController(t)
			defer ctrl.Finish()

			mockSearcher := core.NewMockSearcher(ctrl)
			mockSearcher.EXPECT().ISearch(gomock.Any(), tc.phrase, tc.limit).Return(tc.serviceResult, tc.serviceError)

			server := grpc.NewServer(mockSearcher)
			stream := &mockSearchStream{}

			err := server.ISearch(&searchpb.SearchRequest{Phrase: tc.phrase, Limit: tc.limit}, stream)

			if tc.wantErr {
				require.Error(t, err)
				require.Equal(t, tc.expectedCode, status.Code(err))
			} else {
				require.NoError(t, err)
				require.Len(t, stream.sent, tc.expectedSent)
				for i, comic := range tc.serviceResult {
					require.Equal(t, comic.ID, stream.sent[i].Id)
					require.Equal(t, comic.URL, stream.sent[i].Url)
				}
			}
		})
	}
}
