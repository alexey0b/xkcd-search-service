package grpc_test

import (
	"context"
	"errors"
	updatepb "search-service/proto/update"
	"search-service/update/adapters/grpc"
	"search-service/update/core"
	"testing"

	"github.com/stretchr/testify/require"
	"go.uber.org/mock/gomock"
	"google.golang.org/grpc/codes"
	"google.golang.org/grpc/status"
	"google.golang.org/protobuf/types/known/emptypb"
)

func TestPing(t *testing.T) {
	ctrl := gomock.NewController(t)
	defer ctrl.Finish()

	mockUpdater := core.NewMockUpdater(ctrl)
	server := grpc.NewServer(mockUpdater)

	_, err := server.Ping(context.Background(), &emptypb.Empty{})
	require.NoError(t, err)
}

func TestStatus(t *testing.T) {
	testCases := []struct {
		desc           string
		serviceStatus  core.ServiceStatus
		expectedStatus updatepb.Status
	}{
		{
			desc:           "running status",
			serviceStatus:  core.StatusRunning,
			expectedStatus: updatepb.Status_STATUS_RUNNING,
		},
		{
			desc:           "idle status",
			serviceStatus:  core.StatusIdle,
			expectedStatus: updatepb.Status_STATUS_IDLE,
		},
		{
			desc:           "unspecified status",
			serviceStatus:  core.ServiceStatus("unknown"),
			expectedStatus: updatepb.Status_STATUS_UNSPECIFIED,
		},
	}

	for _, tc := range testCases {
		t.Run(tc.desc, func(t *testing.T) {
			ctrl := gomock.NewController(t)
			defer ctrl.Finish()

			mockUpdater := core.NewMockUpdater(ctrl)
			mockUpdater.EXPECT().Status(gomock.Any()).Return(tc.serviceStatus)

			server := grpc.NewServer(mockUpdater)

			reply, err := server.Status(context.Background(), &emptypb.Empty{})
			require.NoError(t, err)
			require.Equal(t, tc.expectedStatus, reply.Status)
		})
	}
}

func TestUpdate(t *testing.T) {
	testCases := []struct {
		desc         string
		serviceError error
		expectedCode codes.Code
		wantErr      bool
	}{
		{
			desc:         "success - success update ",
			serviceError: nil,
			wantErr:      false,
		},
		{
			desc:         "error - already exists error",
			serviceError: core.ErrAlreadyExists,
			expectedCode: codes.AlreadyExists,
			wantErr:      true,
		},
		{
			desc:         "error - internal error",
			serviceError: errors.New("internal error"),
			expectedCode: codes.Internal,
			wantErr:      true,
		},
	}

	for _, tc := range testCases {
		t.Run(tc.desc, func(t *testing.T) {
			ctrl := gomock.NewController(t)
			defer ctrl.Finish()

			mockUpdater := core.NewMockUpdater(ctrl)
			mockUpdater.EXPECT().Update(gomock.Any()).Return(tc.serviceError)

			server := grpc.NewServer(mockUpdater)

			_, err := server.Update(context.Background(), &emptypb.Empty{})

			if tc.wantErr {
				require.Error(t, err)
				require.Equal(t, tc.expectedCode, status.Code(err))
			} else {
				require.NoError(t, err)
			}
		})
	}
}

func TestStats(t *testing.T) {
	testCases := []struct {
		desc          string
		serviceStats  core.ServiceStats
		serviceError  error
		expectedStats *updatepb.StatsReply
		expectedCode  codes.Code
		wantErr       bool
	}{
		{
			desc: "success - success stats",
			serviceStats: core.ServiceStats{
				DBStats: core.DBStats{
					WordsTotal:    100,
					WordsUnique:   50,
					ComicsFetched: 10,
				},
				ComicsTotal: 20,
			},
			expectedStats: &updatepb.StatsReply{
				WordsTotal:    100,
				WordsUnique:   50,
				ComicsFetched: 10,
				ComicsTotal:   20,
			},
			wantErr: false,
		},
		{
			desc:         "error - internal error",
			serviceError: errors.New("database error"),
			expectedCode: codes.Internal,
			wantErr:      true,
		},
	}

	for _, tc := range testCases {
		t.Run(tc.desc, func(t *testing.T) {
			ctrl := gomock.NewController(t)
			defer ctrl.Finish()

			mockUpdater := core.NewMockUpdater(ctrl)
			mockUpdater.EXPECT().Stats(gomock.Any()).Return(tc.serviceStats, tc.serviceError)

			server := grpc.NewServer(mockUpdater)

			reply, err := server.Stats(context.Background(), &emptypb.Empty{})

			if tc.wantErr {
				require.Error(t, err)
				require.Equal(t, tc.expectedCode, status.Code(err))
			} else {
				require.NoError(t, err)
				require.Equal(t, tc.expectedStats, reply)
			}
		})
	}
}

func TestDrop(t *testing.T) {
	testCases := []struct {
		desc         string
		serviceError error
		expectedCode codes.Code
		wantErr      bool
	}{
		{
			desc:         "success - success drop",
			serviceError: nil,
			wantErr:      false,
		},
		{
			desc:         "error - internal error",
			serviceError: errors.New("drop failed"),
			expectedCode: codes.Internal,
			wantErr:      true,
		},
	}

	for _, tc := range testCases {
		t.Run(tc.desc, func(t *testing.T) {
			ctrl := gomock.NewController(t)
			defer ctrl.Finish()

			mockUpdater := core.NewMockUpdater(ctrl)
			mockUpdater.EXPECT().Drop(gomock.Any()).Return(tc.serviceError)

			server := grpc.NewServer(mockUpdater)

			_, err := server.Drop(context.Background(), &emptypb.Empty{})

			if tc.wantErr {
				require.Error(t, err)
				require.Equal(t, tc.expectedCode, status.Code(err))
			} else {
				require.NoError(t, err)
			}
		})
	}
}
