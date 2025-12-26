package core_test

import (
	"context"
	"errors"
	"log/slog"
	"search-service/update/core"
	"testing"

	"github.com/stretchr/testify/require"
	"go.uber.org/mock/gomock"
)

const concurrency = 10

func TestUpdate(t *testing.T) {
	testCases := []struct {
		desc    string
		prepare func(*core.MockDB, *core.MockXKCD, *core.MockWords, *core.MockPublisher)
		wantErr bool
	}{
		{
			desc: "success - no new comics",
			prepare: func(db *core.MockDB, xkcd *core.MockXKCD, words *core.MockWords, publisher *core.MockPublisher) {
				db.EXPECT().IDs(gomock.Any()).Return([]int64{1, 2, 3}, nil)
				xkcd.EXPECT().LastID(gomock.Any()).Return(int64(3), nil)
				// не ожидаем вызовов Get, Norm, Add и Publish, т.к. все комиксы уже есть
			},
			wantErr: false,
		},
		{
			desc: "success - new comics added",
			prepare: func(db *core.MockDB, xkcd *core.MockXKCD, words *core.MockWords, publisher *core.MockPublisher) {
				db.EXPECT().IDs(gomock.Any()).Return([]int64{1, 2}, nil)
				xkcd.EXPECT().LastID(gomock.Any()).Return(int64(4), nil)

				// обрабатываем только новые комиксы (3 и 4)
				xkcd.EXPECT().Get(gomock.Any(), int64(3)).Return(core.XKCDInfo{ID: 3, Title: "New"}, nil)
				xkcd.EXPECT().Get(gomock.Any(), int64(4)).Return(core.XKCDInfo{ID: 4, Title: "Newer"}, nil)
				words.EXPECT().Norm(gomock.Any(), gomock.Any()).Return([]string{"new", "comic"}, nil).Times(2)
				db.EXPECT().Add(gomock.Any(), []core.Comic{
					{ID: int64(3), Words: []string{"new", "comic"}},
					{ID: int64(4), Words: []string{"new", "comic"}},
				}).
					Return(nil)
				publisher.EXPECT().Publish(core.EventUpdate).Return(nil)
			},
			wantErr: false,
		},
		{
			desc: "error - failed to get existing IDs",
			prepare: func(db *core.MockDB, xkcd *core.MockXKCD, words *core.MockWords, publisher *core.MockPublisher) {
				db.EXPECT().IDs(gomock.Any()).Return(nil, errors.New("db error"))
			},
			wantErr: true,
		},
		{
			desc: "error - failed to get last ID",
			prepare: func(db *core.MockDB, xkcd *core.MockXKCD, words *core.MockWords, publisher *core.MockPublisher) {
				db.EXPECT().IDs(gomock.Any()).Return([]int64{1}, nil)
				xkcd.EXPECT().LastID(gomock.Any()).Return(int64(0), errors.New("xkcd error"))
			},
			wantErr: true,
		},
		{
			desc: "error - failed to add comics",
			prepare: func(db *core.MockDB, xkcd *core.MockXKCD, words *core.MockWords, pub *core.MockPublisher) {
				db.EXPECT().IDs(gomock.Any()).Return([]int64{}, nil)
				xkcd.EXPECT().LastID(gomock.Any()).Return(int64(2), nil)
				xkcd.EXPECT().Get(gomock.Any(), int64(1)).Return(core.XKCDInfo{ID: 1}, nil)
				xkcd.EXPECT().Get(gomock.Any(), int64(2)).Return(core.XKCDInfo{ID: 2}, nil)
				words.EXPECT().Norm(gomock.Any(), gomock.Any()).Return([]string{"test"}, nil).Times(2)
				db.EXPECT().Add(gomock.Any(), []core.Comic{
					{ID: int64(1), Words: []string{"test"}},
					{ID: int64(2), Words: []string{"test"}},
				}).Return(errors.New("add error"))
			},
			wantErr: true,
		},
		{
			desc: "success - publisher error ignored",
			prepare: func(db *core.MockDB, xkcd *core.MockXKCD, words *core.MockWords, pub *core.MockPublisher) {
				db.EXPECT().IDs(gomock.Any()).Return([]int64{}, nil)
				xkcd.EXPECT().LastID(gomock.Any()).Return(int64(1), nil)
				xkcd.EXPECT().Get(gomock.Any(), int64(1)).Return(core.XKCDInfo{ID: 1}, nil)
				words.EXPECT().Norm(gomock.Any(), gomock.Any()).Return([]string{"test"}, nil)
				db.EXPECT().Add(gomock.Any(), []core.Comic{{ID: int64(1), Words: []string{"test"}}}).Return(nil)
				pub.EXPECT().Publish(core.EventUpdate).Return(errors.New("publish error"))
			},
			wantErr: false,
		},
		{
			desc: "error - words normalization failed",
			prepare: func(db *core.MockDB, xkcd *core.MockXKCD, words *core.MockWords, pub *core.MockPublisher) {
				db.EXPECT().IDs(gomock.Any()).Return([]int64{}, nil)
				xkcd.EXPECT().LastID(gomock.Any()).Return(int64(2), nil)
				xkcd.EXPECT().Get(gomock.Any(), int64(1)).Return(core.XKCDInfo{ID: 1, Title: "First"}, nil)
				xkcd.EXPECT().Get(gomock.Any(), int64(2)).Return(core.XKCDInfo{ID: 2, Title: "Second"}, nil)
				words.EXPECT().Norm(gomock.Any(), gomock.Any()).Return([]string{"first"}, nil)
				words.EXPECT().Norm(gomock.Any(), gomock.Any()).Return(nil, errors.New("normalization error"))

				// Добавляется только 1 комикс (второй пропущен из-за ошибки)
				db.EXPECT().Add(gomock.Any(), []core.Comic{{ID: int64(1), Words: []string{"first"}}}).Return(nil)
				pub.EXPECT().Publish(core.EventUpdate).Return(nil)
			},
			wantErr: false,
		},
	}

	for _, tc := range testCases {
		t.Run(tc.desc, func(t *testing.T) {
			ctrl := gomock.NewController(t)
			defer ctrl.Finish()

			mockDB := core.NewMockDB(ctrl)
			mockXKCD := core.NewMockXKCD(ctrl)
			mockWords := core.NewMockWords(ctrl)
			mockPublisher := core.NewMockPublisher(ctrl)

			tc.prepare(mockDB, mockXKCD, mockWords, mockPublisher)

			service, err := core.NewService(slog.Default(), mockDB, mockXKCD, mockWords, mockPublisher, concurrency)
			require.NoError(t, err)

			err = service.Update(context.TODO())

			if tc.wantErr {
				require.Error(t, err)
			} else {
				require.NoError(t, err)
			}
		})
	}
}

func TestStats(t *testing.T) {
	testCases := []struct {
		desc          string
		prepare       func(*core.MockDB, *core.MockXKCD)
		expectedStats core.ServiceStats
		wantErr       bool
	}{
		{
			desc: "success - returns correct stats",
			prepare: func(db *core.MockDB, xkcd *core.MockXKCD) {
				db.EXPECT().Stats(gomock.Any()).Return(core.DBStats{
					WordsTotal:    2000,
					WordsUnique:   500,
					ComicsFetched: 404,
				}, nil)
				xkcd.EXPECT().LastID(gomock.Any()).Return(int64(404), nil)
			},
			expectedStats: core.ServiceStats{
				DBStats: core.DBStats{
					WordsTotal:    2000,
					WordsUnique:   500,
					ComicsFetched: 404,
				},
				ComicsTotal: 404,
			},
			wantErr: false,
		},
		{
			desc: "success - empty database and xkcd",
			prepare: func(db *core.MockDB, xkcd *core.MockXKCD) {
				db.EXPECT().Stats(gomock.Any()).Return(core.DBStats{
					WordsTotal:    0,
					WordsUnique:   0,
					ComicsFetched: 0,
				}, nil)
				xkcd.EXPECT().LastID(gomock.Any()).Return(int64(0), nil)
			},
			expectedStats: core.ServiceStats{
				DBStats: core.DBStats{
					WordsTotal:    0,
					WordsUnique:   0,
					ComicsFetched: 0,
				},
				ComicsTotal: 0,
			},
			wantErr: false,
		},
		{
			desc: "error - failed to get database stats",
			prepare: func(db *core.MockDB, xkcd *core.MockXKCD) {
				db.EXPECT().Stats(gomock.Any()).Return(core.DBStats{}, errors.New("db error"))
			},
			expectedStats: core.ServiceStats{},
			wantErr:       true,
		},
		{
			desc: "error - failed to get last ID",
			prepare: func(db *core.MockDB, xkcd *core.MockXKCD) {
				db.EXPECT().Stats(gomock.Any()).Return(core.DBStats{
					WordsTotal:    100,
					WordsUnique:   50,
					ComicsFetched: 10,
				}, nil)
				xkcd.EXPECT().LastID(gomock.Any()).Return(int64(0), errors.New("xkcd error"))
			},
			expectedStats: core.ServiceStats{},
			wantErr:       true,
		},
		{
			desc: "error - last comic not found",
			prepare: func(db *core.MockDB, xkcd *core.MockXKCD) {
				db.EXPECT().Stats(gomock.Any()).Return(core.DBStats{
					WordsTotal:    100,
					WordsUnique:   50,
					ComicsFetched: 10,
				}, nil)
				xkcd.EXPECT().LastID(gomock.Any()).Return(int64(0), core.ErrNotFound)
			},
			expectedStats: core.ServiceStats{},
			wantErr:       true,
		},
	}

	for _, tc := range testCases {
		t.Run(tc.desc, func(t *testing.T) {
			ctrl := gomock.NewController(t)
			defer ctrl.Finish()

			mockDB := core.NewMockDB(ctrl)
			mockXKCD := core.NewMockXKCD(ctrl)
			mockWords := core.NewMockWords(ctrl)
			mockPublisher := core.NewMockPublisher(ctrl)

			tc.prepare(mockDB, mockXKCD)

			service, err := core.NewService(slog.Default(), mockDB, mockXKCD, mockWords, mockPublisher, concurrency)
			require.NoError(t, err)

			stats, err := service.Stats(context.TODO())

			if tc.wantErr {
				require.Error(t, err)
				require.Equal(t, tc.expectedStats, stats)
			} else {
				require.NoError(t, err)
				require.Equal(t, tc.expectedStats, stats)
			}
		})
	}
}

func TestStatus(t *testing.T) {
	testCases := []struct {
		desc           string
		prepare        func(*core.Service)
		expectedStatus core.ServiceStatus
	}{
		{
			desc:           "idle - no update in progress",
			prepare:        func(s *core.Service) {},
			expectedStatus: core.StatusIdle,
		},
	}

	for _, tc := range testCases {
		t.Run(tc.desc, func(t *testing.T) {
			ctrl := gomock.NewController(t)
			defer ctrl.Finish()

			mockDB := core.NewMockDB(ctrl)
			mockXKCD := core.NewMockXKCD(ctrl)
			mockWords := core.NewMockWords(ctrl)
			mockPublisher := core.NewMockPublisher(ctrl)

			service, err := core.NewService(slog.Default(), mockDB, mockXKCD, mockWords, mockPublisher, concurrency)
			require.NoError(t, err)

			tc.prepare(service)

			status := service.Status(context.TODO())
			require.Equal(t, tc.expectedStatus, status)
		})
	}
}

func TestDrop(t *testing.T) {
	testCases := []struct {
		desc    string
		prepare func(*core.MockDB, *core.MockPublisher)
		wantErr bool
	}{
		{
			desc: "success - drops database and publishes event",
			prepare: func(db *core.MockDB, pub *core.MockPublisher) {
				db.EXPECT().Drop(gomock.Any()).Return(nil)
				pub.EXPECT().Publish(core.EventReset).Return(nil)
			},
			wantErr: false,
		},
		{
			desc: "error - failed to drop database",
			prepare: func(db *core.MockDB, pub *core.MockPublisher) {
				db.EXPECT().Drop(gomock.Any()).Return(errors.New("drop error"))
			},
			wantErr: true,
		},
		{
			desc: "success - publisher error ignored",
			prepare: func(db *core.MockDB, pub *core.MockPublisher) {
				db.EXPECT().Drop(gomock.Any()).Return(nil)
				pub.EXPECT().Publish(core.EventReset).Return(errors.New("publish error"))
			},
			wantErr: false,
		},
	}

	for _, tc := range testCases {
		t.Run(tc.desc, func(t *testing.T) {
			ctrl := gomock.NewController(t)
			defer ctrl.Finish()

			mockDB := core.NewMockDB(ctrl)
			mockXKCD := core.NewMockXKCD(ctrl)
			mockWords := core.NewMockWords(ctrl)
			mockPublisher := core.NewMockPublisher(ctrl)

			tc.prepare(mockDB, mockPublisher)

			service, err := core.NewService(slog.Default(), mockDB, mockXKCD, mockWords, mockPublisher, concurrency)
			require.NoError(t, err)

			err = service.Drop(context.TODO())

			if tc.wantErr {
				require.Error(t, err)
			} else {
				require.NoError(t, err)
			}
		})
	}
}
