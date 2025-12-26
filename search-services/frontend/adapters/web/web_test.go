package web_test

import (
	"encoding/json"
	"errors"
	"log/slog"
	"net/http"
	"net/http/httptest"
	"search-service/frontend/adapters/web"
	"search-service/frontend/core"
	"strings"
	"testing"
	"time"

	"github.com/stretchr/testify/require"
	"go.uber.org/mock/gomock"
)

const statusPingOK core.PingStatus = "ok"

func TestPingHandler(t *testing.T) {
	testCases := []struct {
		desc           string
		prepare        func(*core.MockPinger)
		expectedStatus int
		wantBody       bool
		expectedBody   core.PingResponse
	}{
		{
			desc: "success - ping ok",
			prepare: func(p *core.MockPinger) {
				p.EXPECT().Ping(gomock.Any()).Return(core.PingResponse{
					Replies: map[string]core.PingStatus{"api": statusPingOK},
				}, nil)
			},
			expectedStatus: http.StatusOK,
			wantBody:       true,
			expectedBody: core.PingResponse{
				Replies: map[string]core.PingStatus{"api": statusPingOK},
			},
		},
		{
			desc: "error - service unavailable",
			prepare: func(p *core.MockPinger) {
				p.EXPECT().Ping(gomock.Any()).Return(core.PingResponse{}, core.ErrServiceUnavailable)
			},
			expectedStatus: http.StatusServiceUnavailable,
		},
		{
			desc: "error - internal error",
			prepare: func(p *core.MockPinger) {
				p.EXPECT().Ping(gomock.Any()).Return(core.PingResponse{}, errors.New("internal"))
			},
			expectedStatus: http.StatusInternalServerError,
		},
	}

	for _, tc := range testCases {
		t.Run(tc.desc, func(t *testing.T) {
			ctrl := gomock.NewController(t)
			defer ctrl.Finish()

			mockPinger := core.NewMockPinger(ctrl)
			tc.prepare(mockPinger)

			handler := web.NewPingHandler(slog.Default(), mockPinger)

			req := httptest.NewRequest(http.MethodGet, "/ping", nil)
			w := httptest.NewRecorder()

			handler(w, req)

			require.Equal(t, tc.expectedStatus, w.Code)
			if tc.wantBody {
				require.Equal(t, "application/json", w.Header().Get("Content-Type"))
				var response core.PingResponse
				err := json.NewDecoder(w.Body).Decode(&response)
				require.NoError(t, err)
				require.Equal(t, tc.expectedBody, response)
			}
		})
	}
}

func TestLoginHandler(t *testing.T) {
	testCases := []struct {
		desc           string
		body           string
		prepare        func(*core.MockAuthenticator)
		expectedStatus int
		expectCookie   bool
	}{
		{
			desc: "success - valid credentials",
			body: `{"name":"admin","password":"password"}`,
			prepare: func(auth *core.MockAuthenticator) {
				auth.EXPECT().CreateToken("admin", "password").Return("token123", nil)
			},
			expectedStatus: http.StatusOK,
			expectCookie:   true,
		},
		{
			desc:           "error - invalid json",
			body:           `{invalid}`,
			prepare:        func(auth *core.MockAuthenticator) {},
			expectedStatus: http.StatusBadRequest,
		},
		{
			desc: "error - invalid credentials",
			body: `{"name":"admin","password":"wrong"}`,
			prepare: func(auth *core.MockAuthenticator) {
				auth.EXPECT().CreateToken("admin", "wrong").Return("", core.ErrInvalidCredentials)
			},
			expectedStatus: http.StatusUnauthorized,
		},
		{
			desc: "error - token creation failed",
			body: `{"name":"admin","password":"password"}`,
			prepare: func(auth *core.MockAuthenticator) {
				auth.EXPECT().CreateToken("admin", "password").Return("", errors.New("internal"))
			},
			expectedStatus: http.StatusInternalServerError,
		},
	}

	for _, tc := range testCases {
		t.Run(tc.desc, func(t *testing.T) {
			ctrl := gomock.NewController(t)
			defer ctrl.Finish()

			mockAuth := core.NewMockAuthenticator(ctrl)
			tc.prepare(mockAuth)

			handler := web.NewLoginHandler(slog.Default(), mockAuth, 2*time.Minute)

			req := httptest.NewRequest(http.MethodPost, "/login", strings.NewReader(tc.body))
			w := httptest.NewRecorder()

			handler(w, req)

			require.Equal(t, tc.expectedStatus, w.Code)
			if tc.expectCookie {
				cookies := w.Result().Cookies()
				require.Len(t, cookies, 1)
				require.Equal(t, "jwt_token", cookies[0].Name)
				require.Equal(t, "token123", cookies[0].Value)
				require.True(t, cookies[0].HttpOnly)
			}
		})
	}
}

func TestSearchHandler(t *testing.T) {
	testCases := []struct {
		desc           string
		url            string
		prepare        func(*core.MockSearcher)
		expectedStatus int
		wantBody       bool
		expectedBody   core.SearchResult
	}{
		{
			desc: "success - returns comics",
			url:  "/search?phrase=test",
			prepare: func(s *core.MockSearcher) {
				s.EXPECT().Search(gomock.Any(), "test").Return(core.SearchResult{
					Comics: []core.Comic{{ID: 1, URL: "url1"}},
					Total:  1,
				}, nil)
			},
			expectedStatus: http.StatusOK,
			wantBody:       true,
			expectedBody: core.SearchResult{
				Comics: []core.Comic{{ID: 1, URL: "url1"}},
				Total:  1,
			},
		},
		{
			desc:           "error - empty phrase",
			url:            "/search?phrase=",
			prepare:        func(s *core.MockSearcher) {},
			expectedStatus: http.StatusBadRequest,
		},
		{
			desc:           "error - missing phrase",
			url:            "/search",
			prepare:        func(s *core.MockSearcher) {},
			expectedStatus: http.StatusBadRequest,
		},
		{
			desc: "error - bad arguments",
			url:  "/search?phrase=test",
			prepare: func(s *core.MockSearcher) {
				s.EXPECT().Search(gomock.Any(), "test").Return(core.SearchResult{}, core.ErrBadArguments)
			},
			expectedStatus: http.StatusBadRequest,
		},
		{
			desc: "error - service unavailable",
			url:  "/search?phrase=test",
			prepare: func(s *core.MockSearcher) {
				s.EXPECT().Search(gomock.Any(), "test").Return(core.SearchResult{}, core.ErrServiceUnavailable)
			},
			expectedStatus: http.StatusServiceUnavailable,
		},
		{
			desc: "error - internal error",
			url:  "/search?phrase=test",
			prepare: func(s *core.MockSearcher) {
				s.EXPECT().Search(gomock.Any(), "test").Return(core.SearchResult{}, errors.New("internal"))
			},
			expectedStatus: http.StatusInternalServerError,
		},
	}

	for _, tc := range testCases {
		t.Run(tc.desc, func(t *testing.T) {
			ctrl := gomock.NewController(t)
			defer ctrl.Finish()

			mockSearcher := core.NewMockSearcher(ctrl)
			tc.prepare(mockSearcher)

			handler := web.NewSearchHandler(slog.Default(), mockSearcher)

			req := httptest.NewRequest(http.MethodGet, tc.url, nil)
			w := httptest.NewRecorder()

			handler(w, req)

			require.Equal(t, tc.expectedStatus, w.Code)
			if tc.wantBody {
				require.Equal(t, "application/json", w.Header().Get("Content-Type"))
				var result core.SearchResult
				err := json.NewDecoder(w.Body).Decode(&result)
				require.NoError(t, err)
				require.Equal(t, tc.expectedBody, result)
			}
		})
	}
}

const statusUpdateIdle core.UpdateStatus = "idle"

func TestStatisticsHandler(t *testing.T) {
	testCases := []struct {
		desc                 string
		prepare              func(*core.MockUpdateStatsProvider)
		expectedStatus       int
		wantBody             bool
		expectedStats        core.UpdateStats
		expectedUpdateStatus core.UpdateStatus
	}{
		{
			desc: "success - returns stats and status",
			prepare: func(p *core.MockUpdateStatsProvider) {
				p.EXPECT().GetUpdateStats(gomock.Any()).Return(core.UpdateStats{
					ComicsTotal: 100,
					WordsTotal:  1000,
				}, nil)
				p.EXPECT().GetUpdateStatus(gomock.Any()).Return(statusUpdateIdle, nil)
			},
			expectedStatus:       http.StatusOK,
			wantBody:             true,
			expectedStats:        core.UpdateStats{ComicsTotal: 100, WordsTotal: 1000},
			expectedUpdateStatus: statusUpdateIdle,
		},
		{
			desc: "error - stats unavailable",
			prepare: func(p *core.MockUpdateStatsProvider) {
				p.EXPECT().GetUpdateStats(gomock.Any()).Return(core.UpdateStats{}, core.ErrServiceUnavailable)
			},
			expectedStatus: http.StatusServiceUnavailable,
		},
		{
			desc: "error - stats internal error",
			prepare: func(p *core.MockUpdateStatsProvider) {
				p.EXPECT().GetUpdateStats(gomock.Any()).Return(core.UpdateStats{}, errors.New("internal"))
			},
			expectedStatus: http.StatusInternalServerError,
		},
		{
			desc: "error - status unavailable",
			prepare: func(p *core.MockUpdateStatsProvider) {
				p.EXPECT().GetUpdateStats(gomock.Any()).Return(core.UpdateStats{ComicsTotal: 100}, nil)
				p.EXPECT().GetUpdateStatus(gomock.Any()).Return(core.UpdateStatus(""), core.ErrServiceUnavailable)
			},
			expectedStatus: http.StatusServiceUnavailable,
		},
		{
			desc: "error - status internal error",
			prepare: func(p *core.MockUpdateStatsProvider) {
				p.EXPECT().GetUpdateStats(gomock.Any()).Return(core.UpdateStats{ComicsTotal: 100}, nil)
				p.EXPECT().GetUpdateStatus(gomock.Any()).Return(core.UpdateStatus(""), errors.New("internal"))
			},
			expectedStatus: http.StatusInternalServerError,
		},
	}

	for _, tc := range testCases {
		t.Run(tc.desc, func(t *testing.T) {
			ctrl := gomock.NewController(t)
			defer ctrl.Finish()

			mockProvider := core.NewMockUpdateStatsProvider(ctrl)
			tc.prepare(mockProvider)

			handler := web.NewStatisticsHandler(slog.Default(), mockProvider)

			req := httptest.NewRequest(http.MethodGet, "/statistics", nil)
			w := httptest.NewRecorder()

			handler(w, req)

			require.Equal(t, tc.expectedStatus, w.Code)
			if tc.wantBody {
				require.Equal(t, "application/json", w.Header().Get("Content-Type"))
				var result struct {
					Stats  core.UpdateStats  `json:"stats"`
					Status core.UpdateStatus `json:"status"`
				}
				err := json.NewDecoder(w.Body).Decode(&result)
				require.NoError(t, err)
				require.Equal(t, tc.expectedStats, result.Stats)
				require.Equal(t, tc.expectedUpdateStatus, result.Status)
			}
		})
	}
}

func TestUpdateHandler(t *testing.T) {
	testCases := []struct {
		desc           string
		prepare        func(*core.MockUpdater)
		expectedStatus int
	}{
		{
			desc: "success - update started",
			prepare: func(u *core.MockUpdater) {
				u.EXPECT().Update(gomock.Any()).Return(nil)
			},
			expectedStatus: http.StatusOK,
		},
		{
			desc: "error - service unavailable",
			prepare: func(u *core.MockUpdater) {
				u.EXPECT().Update(gomock.Any()).Return(core.ErrServiceUnavailable)
			},
			expectedStatus: http.StatusServiceUnavailable,
		},
		{
			desc: "error - already running",
			prepare: func(u *core.MockUpdater) {
				u.EXPECT().Update(gomock.Any()).Return(core.ErrAlreadyExists)
			},
			expectedStatus: http.StatusAccepted,
		},
		{
			desc: "error - internal error",
			prepare: func(u *core.MockUpdater) {
				u.EXPECT().Update(gomock.Any()).Return(errors.New("internal"))
			},
			expectedStatus: http.StatusInternalServerError,
		},
	}

	for _, tc := range testCases {
		t.Run(tc.desc, func(t *testing.T) {
			ctrl := gomock.NewController(t)
			defer ctrl.Finish()

			mockUpdater := core.NewMockUpdater(ctrl)
			tc.prepare(mockUpdater)

			handler := web.NewUpdateHandler(slog.Default(), mockUpdater)

			req := httptest.NewRequest(http.MethodPost, "/update", nil)
			w := httptest.NewRecorder()

			handler(w, req)

			require.Equal(t, tc.expectedStatus, w.Code)
		})
	}
}

func TestDropHandler(t *testing.T) {
	testCases := []struct {
		desc           string
		prepare        func(*core.MockUpdater)
		expectedStatus int
	}{
		{
			desc: "success - drop completed",
			prepare: func(u *core.MockUpdater) {
				u.EXPECT().Drop(gomock.Any()).Return(nil)
			},
			expectedStatus: http.StatusOK,
		},
		{
			desc: "error - service unavailable",
			prepare: func(u *core.MockUpdater) {
				u.EXPECT().Drop(gomock.Any()).Return(core.ErrServiceUnavailable)
			},
			expectedStatus: http.StatusServiceUnavailable,
		},
		{
			desc: "error - internal error",
			prepare: func(u *core.MockUpdater) {
				u.EXPECT().Drop(gomock.Any()).Return(errors.New("internal"))
			},
			expectedStatus: http.StatusInternalServerError,
		},
	}

	for _, tc := range testCases {
		t.Run(tc.desc, func(t *testing.T) {
			ctrl := gomock.NewController(t)
			defer ctrl.Finish()

			mockUpdater := core.NewMockUpdater(ctrl)
			tc.prepare(mockUpdater)

			handler := web.NewDropHandler(slog.Default(), mockUpdater)

			req := httptest.NewRequest(http.MethodDelete, "/drop", nil)
			w := httptest.NewRecorder()

			handler(w, req)

			require.Equal(t, tc.expectedStatus, w.Code)
		})
	}
}
