package update

import (
	"context"
	"log/slog"
	"net"
	"search-service/api/core"
	updatepb "search-service/proto/update"
	"testing"

	"github.com/stretchr/testify/require"
	"google.golang.org/grpc"
	"google.golang.org/grpc/codes"
	"google.golang.org/grpc/credentials/insecure"
	"google.golang.org/grpc/status"
	"google.golang.org/grpc/test/bufconn"
	"google.golang.org/protobuf/types/known/emptypb"
)

type mockUpdateServer struct {
	updatepb.UnimplementedUpdateServer
	statusReply *updatepb.StatusReply
	statsReply  *updatepb.StatsReply
	err         error
}

func (m *mockUpdateServer) Status(_ context.Context, _ *emptypb.Empty) (*updatepb.StatusReply, error) {
	if m.err != nil {
		return nil, m.err
	}
	return m.statusReply, nil
}

func (m *mockUpdateServer) Stats(_ context.Context, _ *emptypb.Empty) (*updatepb.StatsReply, error) {
	if m.err != nil {
		return nil, m.err
	}
	return m.statsReply, nil
}

func (m *mockUpdateServer) Update(_ context.Context, _ *emptypb.Empty) (*emptypb.Empty, error) {
	if m.err != nil {
		return nil, m.err
	}
	return &emptypb.Empty{}, nil
}

func (m *mockUpdateServer) Drop(_ context.Context, _ *emptypb.Empty) (*emptypb.Empty, error) {
	if m.err != nil {
		return nil, m.err
	}
	return &emptypb.Empty{}, nil
}

func TestStatus(t *testing.T) {
	testCases := []struct {
		name           string
		mock           mockUpdateServer
		expectedStatus core.UpdateStatus
		expectedErr    error
	}{
		{
			name: "success_idle",
			mock: mockUpdateServer{
				statusReply: &updatepb.StatusReply{Status: updatepb.Status_STATUS_IDLE},
			},
			expectedStatus: core.UpdateIdle,
		},
		{
			name: "success_running",
			mock: mockUpdateServer{
				statusReply: &updatepb.StatusReply{Status: updatepb.Status_STATUS_RUNNING},
			},
			expectedStatus: core.UpdateRunning,
		},
		{
			name: "error_unavailable",
			mock: mockUpdateServer{
				err: status.Error(codes.Unavailable, "unavailable"),
			},
			expectedStatus: core.UpdateUnknown,
			expectedErr:    core.ErrServiceUnavailable,
		},
	}

	for _, tc := range testCases {
		t.Run(tc.name, func(t *testing.T) {
			client := setupTestClient(t, &tc.mock)

			actualStatus, err := client.Status(context.Background())
			if tc.expectedErr != nil {
				require.ErrorIs(t, tc.expectedErr, err)
				require.Equal(t, tc.expectedStatus, actualStatus)
				return
			}

			require.NoError(t, err)
			require.Equal(t, tc.expectedStatus, actualStatus)
		})
	}
}

func TestStats(t *testing.T) {
	testCases := []struct {
		name          string
		mock          mockUpdateServer
		expectedStats core.UpdateStats
		expectedErr   error
	}{
		{
			name: "success",
			mock: mockUpdateServer{
				statsReply: &updatepb.StatsReply{
					WordsTotal:    100,
					WordsUnique:   50,
					ComicsFetched: 10,
					ComicsTotal:   20,
				},
			},
			expectedStats: core.UpdateStats{
				WordsTotal:    100,
				WordsUnique:   50,
				ComicsFetched: 10,
				ComicsTotal:   20,
			},
		},
		{
			name: "error_unavailable",
			mock: mockUpdateServer{
				err: status.Error(codes.Unavailable, "unavailable"),
			},
			expectedErr: core.ErrServiceUnavailable,
		},
	}

	for _, tc := range testCases {
		t.Run(tc.name, func(t *testing.T) {
			client := setupTestClient(t, &tc.mock)

			actualStats, err := client.Stats(context.Background())
			if tc.expectedErr != nil {
				require.ErrorIs(t, tc.expectedErr, err)
				return
			}

			require.NoError(t, err)
			require.Equal(t, tc.expectedStats, actualStats)
		})
	}
}

func TestUpdate(t *testing.T) {
	testCases := []struct {
		name        string
		mock        mockUpdateServer
		expectedErr error
	}{
		{
			name: "success",
			mock: mockUpdateServer{},
		},
		{
			name: "error_unavailable",
			mock: mockUpdateServer{
				err: status.Error(codes.Unavailable, "unavailable"),
			},
			expectedErr: core.ErrServiceUnavailable,
		},
		{
			name: "error_already_exists",
			mock: mockUpdateServer{
				err: status.Error(codes.AlreadyExists, "already running"),
			},
			expectedErr: core.ErrAlreadyExists,
		},
	}

	for _, tc := range testCases {
		t.Run(tc.name, func(t *testing.T) {
			client := setupTestClient(t, &tc.mock)

			err := client.Update(context.Background())
			if tc.expectedErr != nil {
				require.ErrorIs(t, tc.expectedErr, err)
				return
			}

			require.NoError(t, err)
		})
	}
}

func TestDrop(t *testing.T) {
	testCases := []struct {
		name        string
		mock        mockUpdateServer
		expectedErr error
	}{
		{
			name: "success",
			mock: mockUpdateServer{},
		},
		{
			name: "error_unavailable",
			mock: mockUpdateServer{
				err: status.Error(codes.Unavailable, "unavailable"),
			},
			expectedErr: core.ErrServiceUnavailable,
		},
	}

	for _, tc := range testCases {
		t.Run(tc.name, func(t *testing.T) {
			client := setupTestClient(t, &tc.mock)

			err := client.Drop(context.Background())
			if tc.expectedErr != nil {
				require.ErrorIs(t, tc.expectedErr, err)
				return
			}

			require.NoError(t, err)
		})
	}
}

// setupTestClient creates a test gRPC client with in-memory bufconn connection.
func setupTestClient(t *testing.T, mock updatepb.UpdateServer) *Client {
	// сreate in-memory listener
	lis := bufconn.Listen(1024 * 1024)
	s := grpc.NewServer()
	updatepb.RegisterUpdateServer(s, mock)

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
	t.Cleanup(func() { conn.Close() })

	return &Client{
		log:    slog.Default(),
		conn:   conn,
		client: updatepb.NewUpdateClient(conn),
	}
}
