package search

import (
	"context"
	"log/slog"
	"net"
	"search-service/api/core"
	"testing"

	searchpb "search-service/proto/search"

	"github.com/stretchr/testify/require"
	"google.golang.org/grpc"
	"google.golang.org/grpc/codes"
	"google.golang.org/grpc/credentials/insecure"
	"google.golang.org/grpc/status"
	"google.golang.org/grpc/test/bufconn"
)

type mockSearchServer struct {
	searchpb.UnimplementedSearchServer
	expectedSearchReplies []*searchpb.SearchReply
	err                   error
}

func (m *mockSearchServer) Search(_ *searchpb.SearchRequest, stream searchpb.Search_SearchServer) error {
	if m.err != nil {
		return m.err
	}
	for _, reply := range m.expectedSearchReplies {
		if err := stream.Send(reply); err != nil {
			return status.Error(codes.Internal, err.Error())
		}
	}
	return nil
}

func (m *mockSearchServer) ISearch(_ *searchpb.SearchRequest, stream searchpb.Search_SearchServer) error {
	if m.err != nil {
		return m.err
	}
	for _, reply := range m.expectedSearchReplies {
		if err := stream.Send(reply); err != nil {
			return status.Error(codes.Internal, err.Error())
		}
	}
	return nil
}

var testCases = []struct {
	name                  string
	mock                  mockSearchServer
	expectedSearchReplies []*searchpb.SearchReply
	expectedErr           error
}{
	{
		name: "success",
		mock: mockSearchServer{
			UnimplementedSearchServer: searchpb.UnimplementedSearchServer{},
			expectedSearchReplies: []*searchpb.SearchReply{
				{Id: 1, Url: "test1"},
				{Id: 2, Url: "test2"},
			},
		},
		expectedSearchReplies: []*searchpb.SearchReply{
			{Id: 1, Url: "test1"},
			{Id: 2, Url: "test2"},
		},
	},
	{
		name: "error_unavailable",
		mock: mockSearchServer{
			UnimplementedSearchServer: searchpb.UnimplementedSearchServer{},
			err:                       status.Error(codes.Unavailable, "unavailable"),
		},
		expectedErr: core.ErrServiceUnavailable,
	},
	{
		name: "error_invalid_argument",
		mock: mockSearchServer{
			UnimplementedSearchServer: searchpb.UnimplementedSearchServer{},
			err:                       status.Error(codes.InvalidArgument, "bad request"),
		},
		expectedErr: core.ErrBadArguments,
	},
}

func TestSearch(t *testing.T) {
	for _, tc := range testCases {
		t.Run(tc.name, func(t *testing.T) {
			client := setupTestClient(t, &tc.mock)
			actualSearchReplies, err := client.Search(context.Background(), "", 0)
			if tc.expectedErr != nil {
				require.ErrorIs(t, tc.expectedErr, err)
				return
			}

			require.NoError(t, err)
			require.Equal(t, len(tc.expectedSearchReplies), len(actualSearchReplies))
			for i := range actualSearchReplies {
				require.Equal(t, tc.expectedSearchReplies[i].Id, actualSearchReplies[i].ID)
				require.Equal(t, tc.expectedSearchReplies[i].Url, actualSearchReplies[i].URL)
			}
		})
	}
}

func TestISearch(t *testing.T) {
	for _, tc := range testCases {
		t.Run(tc.name, func(t *testing.T) {
			client := setupTestClient(t, &tc.mock)
			actualSearchReplies, err := client.ISearch(context.Background(), "", 0)
			if tc.expectedErr != nil {
				require.ErrorIs(t, tc.expectedErr, err)
				return
			}

			require.NoError(t, err)
			require.Equal(t, len(tc.expectedSearchReplies), len(actualSearchReplies))
			for i := range actualSearchReplies {
				require.Equal(t, tc.expectedSearchReplies[i].Id, actualSearchReplies[i].ID)
				require.Equal(t, tc.expectedSearchReplies[i].Url, actualSearchReplies[i].URL)
			}
		})
	}
}

// setupTestClient creates a test gRPC client with in-memory bufconn connection.
func setupTestClient(t *testing.T, mock searchpb.SearchServer) *Client {
	// сreate in-memory listener
	lis := bufconn.Listen(1024 * 1024)
	s := grpc.NewServer()
	searchpb.RegisterSearchServer(s, mock)

	go func() {
		if err := s.Serve(lis); err != nil {
			t.Logf("Server exited with error: %v", err)
		}
	}()
	t.Cleanup(s.Stop)

	// сreate client connection using bufconn dialer
	conn, err := grpc.NewClient("passthrough://bufnet",
		grpc.WithContextDialer(func(context.Context, string) (net.Conn, error) {
			return lis.Dial()
		}),
		grpc.WithTransportCredentials(insecure.NewCredentials()),
	)
	require.NoError(t, err)
	t.Cleanup(func() { _ = conn.Close() })

	return &Client{
		log:    slog.Default(),
		conn:   conn,
		client: searchpb.NewSearchClient(conn),
	}
}
