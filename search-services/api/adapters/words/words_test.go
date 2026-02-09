package words

import (
	"context"
	"log/slog"
	"net"
	"search-service/api/core"
	wordspb "search-service/proto/words"
	"testing"

	"github.com/stretchr/testify/require"
	"google.golang.org/grpc"
	"google.golang.org/grpc/codes"
	"google.golang.org/grpc/credentials/insecure"
	"google.golang.org/grpc/status"
	"google.golang.org/grpc/test/bufconn"
)

type mockWordsServer struct {
	wordspb.UnimplementedWordsServer
	wordsReply *wordspb.WordsReply
	err        error
}

func (m *mockWordsServer) Norm(_ context.Context, _ *wordspb.WordsRequest) (*wordspb.WordsReply, error) {
	if m.err != nil {
		return nil, m.err
	}
	return m.wordsReply, nil
}

func TestNorm(t *testing.T) {
	testCases := []struct {
		name          string
		mock          mockWordsServer
		expectedWords []string
		expectedErr   error
	}{
		{
			name: "success",
			mock: mockWordsServer{
				wordsReply: &wordspb.WordsReply{
					Words: []string{"test", "word"},
				},
			},
			expectedWords: []string{"test", "word"},
		},
		{
			name: "success_empty",
			mock: mockWordsServer{
				wordsReply: &wordspb.WordsReply{
					Words: []string{},
				},
			},
		},
		{
			name: "error_unavailable",
			mock: mockWordsServer{
				err: status.Error(codes.Unavailable, "unavailable"),
			},
			expectedErr: core.ErrServiceUnavailable,
		},
		{
			name: "error_resource_exhausted",
			mock: mockWordsServer{
				err: status.Error(codes.ResourceExhausted, "phrase too large"),
			},
			expectedErr: core.ErrBadArguments,
		},
	}

	for _, tc := range testCases {
		t.Run(tc.name, func(t *testing.T) {
			client := setupTestClient(t, &tc.mock)

			actualWords, err := client.Norm(context.Background(), "test phrase")
			if tc.expectedErr != nil {
				require.ErrorIs(t, tc.expectedErr, err)
				return
			}

			require.NoError(t, err)
			require.Equal(t, tc.expectedWords, actualWords)
		})
	}
}

// setupTestClient creates a test gRPC client with in-memory bufconn connection.
func setupTestClient(t *testing.T, mock wordspb.WordsServer) *Client {
	// сreate in-memory listener
	lis := bufconn.Listen(1024 * 1024)
	s := grpc.NewServer()
	wordspb.RegisterWordsServer(s, mock)

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
		client: wordspb.NewWordsClient(conn),
	}
}
