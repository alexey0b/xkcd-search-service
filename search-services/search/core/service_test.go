package core_test

import (
	"context"
	"errors"
	"log/slog"
	"search-service/search/core"
	"testing"

	"github.com/stretchr/testify/require"
	"go.uber.org/mock/gomock"
)

func TestSearch(t *testing.T) {
	testCases := []struct {
		desc     string
		phrase   string
		limit    int64
		prepare  func(*core.MockDB, *core.MockWords)
		expected []core.Comic
		wantErr  bool
	}{
		{
			desc:   "success - returns ranked comics",
			phrase: "test phrase is unknown",
			limit:  10,
			prepare: func(db *core.MockDB, words *core.MockWords) {
				words.EXPECT().Norm(gomock.Any(), "test phrase is unknown").Do(func(ctx context.Context, phrase string) {
					require.Equal(t, "test phrase is unknown", phrase)
				}).Return([]string{"test", "phrase", "is", "unknown"}, nil)
				db.EXPECT().GetAllComicsInfo(gomock.Any()).Return([]core.ComicInfo{
					{Comic: core.Comic{ID: 1, URL: "url1"}, Words: []string{"test", "phrase"}},
					{Comic: core.Comic{ID: 2, URL: "url2"}, Words: []string{"test", "phrase", "unknown"}},
					{Comic: core.Comic{ID: 3, URL: "url3"}, Words: []string{"test"}},
				}, nil)
			},
			expected: []core.Comic{
				{ID: 2, URL: "url2"},
				{ID: 1, URL: "url1"},
				{ID: 3, URL: "url3"},
			},
			wantErr: false,
		},
		{
			desc:   "success - empty phrase returns error",
			phrase: "",
			limit:  10,
			prepare: func(db *core.MockDB, words *core.MockWords) {
			},
			expected: nil,
			wantErr:  true,
		},
		{
			desc:   "success - zero limit returns error",
			phrase: "test",
			limit:  0,
			prepare: func(db *core.MockDB, words *core.MockWords) {
			},
			expected: nil,
			wantErr:  true,
		},
		{
			desc:   "error - normalization failed",
			phrase: "test",
			limit:  10,
			prepare: func(db *core.MockDB, words *core.MockWords) {
				words.EXPECT().Norm(gomock.Any(), "test").Do(func(ctx context.Context, phrase string) {
					require.Equal(t, "test", phrase)
				}).Return(nil, errors.New("norm error"))
			},
			expected: nil,
			wantErr:  true,
		},
		{
			desc:   "error - database failed",
			phrase: "test",
			limit:  10,
			prepare: func(db *core.MockDB, words *core.MockWords) {
				words.EXPECT().Norm(gomock.Any(), "test").Do(func(ctx context.Context, phrase string) {
					require.Equal(t, "test", phrase)
				}).Return([]string{"test"}, nil)
				db.EXPECT().GetAllComicsInfo(gomock.Any()).Return(nil, errors.New("db error"))
			},
			expected: nil,
			wantErr:  true,
		},
		{
			desc:   "success - no matching comics",
			phrase: "test",
			limit:  10,
			prepare: func(db *core.MockDB, words *core.MockWords) {
				words.EXPECT().Norm(gomock.Any(), "test").Do(func(ctx context.Context, phrase string) {
					require.Equal(t, "test", phrase)
				}).Return([]string{"test"}, nil)
				db.EXPECT().GetAllComicsInfo(gomock.Any()).Return([]core.ComicInfo{
					{Comic: core.Comic{ID: 1, URL: "url"}, Words: []string{"other"}},
				}, nil)
			},
			expected: []core.Comic{},
			wantErr:  false,
		},
		{
			desc:   "success - limit applied",
			phrase: "test,phrase",
			limit:  1,
			prepare: func(db *core.MockDB, words *core.MockWords) {
				words.EXPECT().Norm(gomock.Any(), "test,phrase").Do(func(ctx context.Context, phrase string) {
					require.Equal(t, "test,phrase", phrase)
				}).Return([]string{"test", "phrase"}, nil)
				db.EXPECT().GetAllComicsInfo(gomock.Any()).Return([]core.ComicInfo{
					{Comic: core.Comic{ID: 1, URL: "url1"}, Words: []string{"test", "phrase"}},
					{Comic: core.Comic{ID: 2, URL: "url2"}, Words: []string{"test"}},
				}, nil)
			},
			expected: []core.Comic{
				{ID: 1, URL: "url1"},
			},
			wantErr: false,
		},
	}

	for _, tc := range testCases {
		t.Run(tc.desc, func(t *testing.T) {
			ctrl := gomock.NewController(t)
			defer ctrl.Finish()

			mockDB := core.NewMockDB(ctrl)
			mockWords := core.NewMockWords(ctrl)

			tc.prepare(mockDB, mockWords)

			service, err := core.NewService(slog.Default(), mockDB, mockWords)
			require.NoError(t, err)

			comics, err := service.Search(context.TODO(), tc.phrase, tc.limit)

			if tc.wantErr {
				require.Error(t, err)
			} else {
				require.NoError(t, err)
				require.Equal(t, tc.expected, comics)
			}
		})
	}
}

func TestISearch(t *testing.T) {
	testCases := []struct {
		desc     string
		phrase   string
		limit    int64
		prepare  func(*core.MockDB, *core.MockWords)
		expected []core.Comic
		wantErr  bool
	}{
		{
			desc:   "success - empty index returns empty result",
			phrase: "test phrase",
			limit:  10,
			prepare: func(db *core.MockDB, words *core.MockWords) {
				words.EXPECT().Norm(gomock.Any(), "test phrase").Return([]string{"test", "phrase"}, nil)
				db.EXPECT().GetComicsByIds(gomock.Any(), []int64{}).Return([]core.Comic{}, nil)
			},
			expected: []core.Comic{},
			wantErr:  false,
		},
		{
			desc:   "error - empty phrase",
			phrase: "",
			limit:  10,
			prepare: func(db *core.MockDB, words *core.MockWords) {
			},
			expected: nil,
			wantErr:  true,
		},
		{
			desc:   "error - zero limit",
			phrase: "test",
			limit:  0,
			prepare: func(db *core.MockDB, words *core.MockWords) {
			},
			expected: nil,
			wantErr:  true,
		},
		{
			desc:   "error - negative limit",
			phrase: "test",
			limit:  -1,
			prepare: func(db *core.MockDB, words *core.MockWords) {
			},
			expected: nil,
			wantErr:  true,
		},
		{
			desc:   "error - normalization failed",
			phrase: "test",
			limit:  10,
			prepare: func(db *core.MockDB, words *core.MockWords) {
				words.EXPECT().Norm(gomock.Any(), "test").Return(nil, errors.New("norm error"))
			},
			expected: nil,
			wantErr:  true,
		},
		{
			desc:   "error - failed to get comics by ids from db",
			phrase: "test",
			limit:  10,
			prepare: func(db *core.MockDB, words *core.MockWords) {
				words.EXPECT().Norm(gomock.Any(), "test").Return([]string{"test"}, nil)
				db.EXPECT().GetComicsByIds(gomock.Any(), gomock.Any()).Return(nil, errors.New("db error"))
			},
			expected: nil,
			wantErr:  true,
		},
	}

	for _, tc := range testCases {
		t.Run(tc.desc, func(t *testing.T) {
			ctrl := gomock.NewController(t)
			defer ctrl.Finish()

			mockDB := core.NewMockDB(ctrl)
			mockWords := core.NewMockWords(ctrl)

			tc.prepare(mockDB, mockWords)

			service, err := core.NewService(slog.Default(), mockDB, mockWords)
			require.NoError(t, err)

			comics, err := service.ISearch(context.TODO(), tc.phrase, tc.limit)

			if tc.wantErr {
				require.Error(t, err)
			} else {
				require.NoError(t, err)
				require.Equal(t, tc.expected, comics)
			}
		})
	}
}

func TestUpdateIndex(t *testing.T) {
	testCases := []struct {
		desc    string
		prepare func(*core.MockDB)
		wantErr bool
	}{
		{
			desc: "success - updated index",
			prepare: func(db *core.MockDB) {
				db.EXPECT().GetAllComicsInfo(gomock.Any()).Return([]core.ComicInfo{{Comic: core.Comic{}, Words: []string{"test"}}}, nil)
			},
			wantErr: false,
		},
		{
			desc: "success - empty db",
			prepare: func(db *core.MockDB) {
				db.EXPECT().GetAllComicsInfo(gomock.Any()).Return([]core.ComicInfo{}, nil)
			},
			wantErr: false,
		},
		{
			desc: "error - failed to get all comcis info from db",
			prepare: func(db *core.MockDB) {
				db.EXPECT().GetAllComicsInfo(gomock.Any()).Return(nil, errors.New("db error"))
			},
			wantErr: true,
		},
	}

	for _, tc := range testCases {
		t.Run(tc.desc, func(t *testing.T) {
			ctrl := gomock.NewController(t)
			defer ctrl.Finish()

			mockDB := core.NewMockDB(ctrl)
			mockWords := core.NewMockWords(ctrl)

			tc.prepare(mockDB)

			service, err := core.NewService(slog.Default(), mockDB, mockWords)
			require.NoError(t, err)

			err = service.UpdateIndex(context.TODO())

			if tc.wantErr {
				require.Error(t, err)
			} else {
				require.NoError(t, err)
			}
		})
	}
}

func TestHandleEvent(t *testing.T) {
	testCases := []struct {
		desc    string
		event   core.EventType
		prepare func(*core.MockDB)
		wantErr bool
	}{
		{
			desc:  "success - handled 'update' event",
			event: "update",
			prepare: func(db *core.MockDB) {
				db.EXPECT().GetAllComicsInfo(gomock.Any()).Return([]core.ComicInfo{{Comic: core.Comic{}, Words: []string{"test"}}}, nil)
			},
			wantErr: false,
		},
		{
			desc:    "success - handled 'reset' event",
			event:   "reset",
			prepare: func(db *core.MockDB) {},
			wantErr: false,
		},
		{
			desc:    "success - unknown event is not error",
			event:   "reset",
			prepare: func(db *core.MockDB) {},
			wantErr: false,
		},
		{
			desc:  "error - failed to update index",
			event: "update",
			prepare: func(db *core.MockDB) {
				db.EXPECT().GetAllComicsInfo(gomock.Any()).Return(nil, errors.New("db error"))
			},
			wantErr: true,
		},
	}

	for _, tc := range testCases {
		t.Run(tc.desc, func(t *testing.T) {
			ctrl := gomock.NewController(t)
			defer ctrl.Finish()

			mockDB := core.NewMockDB(ctrl)
			mockWords := core.NewMockWords(ctrl)

			tc.prepare(mockDB)

			service, err := core.NewService(slog.Default(), mockDB, mockWords)
			require.NoError(t, err)

			err = service.HandleEvent(context.TODO(), tc.event)

			if tc.wantErr {
				require.Error(t, err)
			} else {
				require.NoError(t, err)
			}
		})
	}
}
