package scheduler_test

import (
	"context"
	"errors"
	"log/slog"
	"search-service/search/adapters/scheduler"
	"search-service/search/core"
	"testing"
	"time"

	"github.com/stretchr/testify/require"
	"go.uber.org/mock/gomock"
)

func TestStartInitialUpdate(t *testing.T) {
	testCases := []struct {
		desc    string
		prepare func(m *core.MockSearcher)
		wantErr bool
	}{
		{
			desc: "success - initial update succeeds",
			prepare: func(m *core.MockSearcher) {
				m.EXPECT().UpdateIndex(gomock.Any()).Return(nil)
			},
			wantErr: false,
		},
		{
			desc: "error - initial update fails",
			prepare: func(m *core.MockSearcher) {
				m.EXPECT().UpdateIndex(gomock.Any()).Return(errors.New("update failed"))
			},
			wantErr: true,
		},
	}

	for _, tc := range testCases {
		t.Run(tc.desc, func(t *testing.T) {
			ctrl := gomock.NewController(t)
			defer ctrl.Finish()

			mockSearcher := core.NewMockSearcher(ctrl)
			tc.prepare(mockSearcher)

			s := scheduler.NewSearcherScheduler(slog.Default(), mockSearcher, time.Second)

			err := s.Start(context.Background())

			if tc.wantErr {
				require.Error(t, err)
			} else {
				require.NoError(t, err)
			}
		})
	}
}

func TestStartPeriodicUpdate(t *testing.T) {
	ctrl := gomock.NewController(t)
	defer ctrl.Finish()

	mockSearcher := core.NewMockSearcher(ctrl)

	expectedCalls := 3
	callCount := 0

	mockSearcher.EXPECT().UpdateIndex(gomock.Any()).DoAndReturn(func(ctx context.Context) error {
		callCount++
		return nil
	}).MinTimes(expectedCalls)

	s := scheduler.NewSearcherScheduler(slog.Default(), mockSearcher, 50*time.Millisecond)

	ctx, cancel := context.WithCancel(context.Background())
	defer cancel()

	err := s.Start(ctx)
	require.NoError(t, err)

	time.Sleep(200 * time.Millisecond)
	cancel()

	time.Sleep(100 * time.Millisecond)

	require.GreaterOrEqual(t, callCount, expectedCalls)
}

func TestStartContextCancellation(t *testing.T) {
	ctrl := gomock.NewController(t)
	defer ctrl.Finish()

	mockSearcher := core.NewMockSearcher(ctrl)

	expectedCalls := 1
	callCount := 0

	mockSearcher.EXPECT().UpdateIndex(gomock.Any()).Do(func(ctx context.Context) {
		callCount++
	}).Return(nil)

	s := scheduler.NewSearcherScheduler(slog.Default(), mockSearcher, time.Second)

	ctx, cancel := context.WithCancel(context.Background())

	err := s.Start(ctx)
	require.NoError(t, err)

	cancel()
	time.Sleep(100 * time.Millisecond)

	require.Equal(t, expectedCalls, callCount)
}

func TestStartUpdateError(t *testing.T) {
	ctrl := gomock.NewController(t)
	defer ctrl.Finish()

	mockSearcher := core.NewMockSearcher(ctrl)

	gomock.InOrder(
		mockSearcher.EXPECT().UpdateIndex(gomock.Any()).Return(nil),
		mockSearcher.EXPECT().UpdateIndex(gomock.Any()).Return(errors.New("update failed")),
	)

	s := scheduler.NewSearcherScheduler(slog.Default(), mockSearcher, 50*time.Millisecond)

	ctx, cancel := context.WithCancel(context.Background())
	defer cancel()

	err := s.Start(ctx)
	require.NoError(t, err)

	time.Sleep(75 * time.Millisecond)
	cancel()
}
