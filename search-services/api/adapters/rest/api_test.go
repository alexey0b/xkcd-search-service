package rest_test

import (
	"encoding/json"
	"errors"
	"log/slog"
	"net/http"
	"net/http/httptest"
	"search-service/api/adapters/rest"
	"search-service/api/core"
	"strings"
	"testing"

	"github.com/stretchr/testify/require"
	"go.uber.org/mock/gomock"
)

func TestPingHandler(t *testing.T) {
	testCases := []struct {
		desc         string
		prepare      func(*core.MockPinger, *core.MockPinger, *core.MockPinger)
		expectedResp core.PingResponse
	}{
		{
			desc: "success - all services available",
			prepare: func(p1, p2, p3 *core.MockPinger) {
				p1.EXPECT().Ping(gomock.Any()).Return(nil)
				p2.EXPECT().Ping(gomock.Any()).Return(nil)
				p3.EXPECT().Ping(gomock.Any()).Return(nil)
			},
			expectedResp: core.PingResponse{
				Replies: map[string]core.PingStatus{
					"service_1": core.StatusPingOK,
					"service_2": core.StatusPingOK,
					"service_3": core.StatusPingOK,
				},
			},
		},
		{
			desc: "partial error - service_2 is unavailable",
			prepare: func(p1, p2, p3 *core.MockPinger) {
				p1.EXPECT().Ping(gomock.Any()).Return(nil)
				p2.EXPECT().Ping(gomock.Any()).Return(core.ErrServiceUnavailable)
				p3.EXPECT().Ping(gomock.Any()).Return(nil)
			},
			expectedResp: core.PingResponse{
				Replies: map[string]core.PingStatus{
					"service_1": core.StatusPingOK,
					"service_2": core.StatusPingUnavailable,
					"service_3": core.StatusPingOK,
				},
			},
		},
		{
			desc: "error - service_1 is failed",
			prepare: func(p1, p2, p3 *core.MockPinger) {
				p1.EXPECT().Ping(gomock.Any()).Return(errors.New("ping error"))
				p2.EXPECT().Ping(gomock.Any()).Return(nil)
				p3.EXPECT().Ping(gomock.Any()).Return(nil)
			},
			expectedResp: core.PingResponse{
				Replies: map[string]core.PingStatus{
					"service_1": core.StatusPingUnavailable,
					"service_2": core.StatusPingOK,
					"service_3": core.StatusPingOK,
				},
			},
		},
	}

	for _, tc := range testCases {
		t.Run(tc.desc, func(t *testing.T) {
			ctrl := gomock.NewController(t)
			defer ctrl.Finish()

			pinger1 := core.NewMockPinger(ctrl)
			pinger2 := core.NewMockPinger(ctrl)
			pinger3 := core.NewMockPinger(ctrl)

			tc.prepare(pinger1, pinger2, pinger3)

			pingers := map[string]core.Pinger{
				"service_1": pinger1,
				"service_2": pinger2,
				"service_3": pinger3,
			}

			handler := rest.NewPingHandler(slog.Default(), pingers)

			req := httptest.NewRequest(http.MethodGet, "/ping", nil)
			w := httptest.NewRecorder()

			handler(w, req)

			require.Equal(t, http.StatusOK, w.Code)
			require.Equal(t, "application/json", w.Header().Get("Content-Type"))

			var response core.PingResponse
			err := json.NewDecoder(w.Body).Decode(&response)
			require.NoError(t, err)
			require.Equal(t, tc.expectedResp, response)
		})
	}
}

func TestLoginHandler(t *testing.T) {
	testCases := []struct {
		desc           string
		body           string
		prepare        func(*core.MockAuthenticator)
		expectedStatus int
		wantBody       string
	}{
		{
			desc: "success - valid credentials",
			body: `{"name":"admin","password":"password"}`,
			prepare: func(auth *core.MockAuthenticator) {
				auth.EXPECT().CreateToken("admin", "password").Return("token123", nil)
			},
			expectedStatus: http.StatusOK,
			wantBody:       "token123",
		},
		{
			desc:           "error - invalid json",
			body:           `{invalid}`,
			prepare:        func(auth *core.MockAuthenticator) {},
			expectedStatus: http.StatusBadRequest,
		},
		{
			desc: "error - invalid login",
			body: `{"name":"invalid_login","password":"password"}`,
			prepare: func(auth *core.MockAuthenticator) {
				auth.EXPECT().CreateToken("invalid_login", "password").Return("", core.ErrInvalidCredentials)
			},
			expectedStatus: http.StatusUnauthorized,
		},
		{
			desc: "error - invalid password",
			body: `{"name":"admin","password":"invalid_password"}`,
			prepare: func(auth *core.MockAuthenticator) {
				auth.EXPECT().CreateToken("admin", "invalid_password").Return("", core.ErrInvalidCredentials)
			},
			expectedStatus: http.StatusUnauthorized,
		},
		{
			desc: "error - token creation failed",
			body: `{"name":"admin","password":"password"}`,
			prepare: func(auth *core.MockAuthenticator) {
				auth.EXPECT().CreateToken("admin", "password").Return("", errors.New("internal error"))
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

			handler := rest.NewLoginHandler(slog.Default(), mockAuth)

			req := httptest.NewRequest(http.MethodPost, "/login", strings.NewReader(tc.body))
			w := httptest.NewRecorder()

			handler(w, req)

			require.Equal(t, tc.expectedStatus, w.Code)
			if tc.wantBody != "" {
				require.Equal(t, tc.wantBody, w.Body.String())
				require.Equal(t, "text/plain", w.Header().Get("Content-Type"))
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
			url:  "/search?phrase=test&limit=5",
			prepare: func(s *core.MockSearcher) {
				s.EXPECT().Search(gomock.Any(), "test", int64(5)).Return([]core.Comic{
					{ID: 1, URL: "url1"},
					{ID: 2, URL: "url2"},
				}, nil)
			},
			expectedStatus: http.StatusOK,
			wantBody:       true,
			expectedBody: core.SearchResult{
				Comics: []core.Comic{{ID: 1, URL: "url1"}, {ID: 2, URL: "url2"}},
				Total:  2,
			},
		},
		{
			desc: "success - default limit",
			url:  "/search?phrase=test",
			prepare: func(s *core.MockSearcher) {
				s.EXPECT().Search(gomock.Any(), "test", int64(10)).Return([]core.Comic{
					{ID: 1, URL: "url1"},
					{ID: 2, URL: "url2"},
				}, nil)
			},
			expectedStatus: http.StatusOK,
			wantBody:       true,
			expectedBody: core.SearchResult{
				Comics: []core.Comic{{ID: 1, URL: "url1"}, {ID: 2, URL: "url2"}},
				Total:  2,
			},
		},
		{
			desc:           "error - no phrase",
			url:            "/search?phrase=",
			prepare:        func(s *core.MockSearcher) {},
			expectedStatus: http.StatusBadRequest,
		},
		{
			desc:           "error - alpha limit",
			url:            "/search?phrase=test&limit=abc",
			prepare:        func(s *core.MockSearcher) {},
			expectedStatus: http.StatusBadRequest,
		},
		{
			desc:           "error - zero limit",
			url:            "/search?phrase=test&limit=0",
			prepare:        func(s *core.MockSearcher) {},
			expectedStatus: http.StatusBadRequest,
		},
		{
			desc:           "error - negative limit",
			url:            "/search?phrase=test&limit=0",
			prepare:        func(s *core.MockSearcher) {},
			expectedStatus: http.StatusBadRequest,
		},
		{
			desc: "error - service unavailable",
			url:  "/search?phrase=test",
			prepare: func(s *core.MockSearcher) {
				s.EXPECT().Search(gomock.Any(), "test", int64(10)).Return(nil, core.ErrServiceUnavailable)
			},
			expectedStatus: http.StatusServiceUnavailable,
		},
		{
			desc: "error - bad arguments",
			url:  "/search?phrase=test",
			prepare: func(s *core.MockSearcher) {
				s.EXPECT().Search(gomock.Any(), "test", int64(10)).Return(nil, core.ErrBadArguments)
			},
			expectedStatus: http.StatusBadRequest,
		},
		{
			desc: "error - internal error",
			url:  "/search?phrase=test",
			prepare: func(s *core.MockSearcher) {
				s.EXPECT().Search(gomock.Any(), "test", int64(10)).Return(nil, errors.New("internal"))
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

			handler := rest.NewSearchHandler(slog.Default(), mockSearcher)

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

func TestISearchHandler(t *testing.T) {
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
			url:  "/isearch?phrase=test&limit=5",
			prepare: func(s *core.MockSearcher) {
				s.EXPECT().ISearch(gomock.Any(), "test", int64(5)).Return([]core.Comic{
					{ID: 1, URL: "url1"},
				}, nil)
			},
			expectedStatus: http.StatusOK,
			wantBody:       true,
			expectedBody:   core.SearchResult{Comics: []core.Comic{{ID: 1, URL: "url1"}}, Total: 1},
		},
		{
			desc: "success - default limit",
			url:  "/isearch?phrase=test",
			prepare: func(s *core.MockSearcher) {
				s.EXPECT().ISearch(gomock.Any(), "test", int64(10)).Return([]core.Comic{}, nil)
			},
			expectedStatus: http.StatusOK,
			wantBody:       true,
			expectedBody:   core.SearchResult{Comics: []core.Comic{}, Total: 0},
		},
		{
			desc:           "error - no phrase",
			url:            "/isearch?phrase=",
			prepare:        func(s *core.MockSearcher) {},
			expectedStatus: http.StatusBadRequest,
		},
		{
			desc:           "error - alpha limit",
			url:            "/isearch?phrase=test&limit=abc",
			prepare:        func(s *core.MockSearcher) {},
			expectedStatus: http.StatusBadRequest,
		},
		{
			desc:           "error - negative limit",
			url:            "/isearch?phrase=test&limit=-1",
			prepare:        func(s *core.MockSearcher) {},
			expectedStatus: http.StatusBadRequest,
		},
		{
			desc: "error - service unavailable",
			url:  "/isearch?phrase=test",
			prepare: func(s *core.MockSearcher) {
				s.EXPECT().ISearch(gomock.Any(), "test", int64(10)).Return(nil, core.ErrServiceUnavailable)
			},
			expectedStatus: http.StatusServiceUnavailable,
		},
		{
			desc: "error - bad arguments",
			url:  "/isearch?phrase=test",
			prepare: func(s *core.MockSearcher) {
				s.EXPECT().ISearch(gomock.Any(), "test", int64(10)).Return(nil, core.ErrBadArguments)
			},
			expectedStatus: http.StatusBadRequest,
		},
		{
			desc: "error - internal error",
			url:  "/isearch?phrase=test",
			prepare: func(s *core.MockSearcher) {
				s.EXPECT().ISearch(gomock.Any(), "test", int64(10)).Return(nil, errors.New("internal"))
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

			handler := rest.NewISearchHandler(slog.Default(), mockSearcher)

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

func TestUpdateHandler(t *testing.T) {
	testCases := []struct {
		desc           string
		prepare        func(*core.MockUpdater)
		expectedStatus int
	}{
		{
			desc: "success - update completed",
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

			handler := rest.NewUpdateHandler(slog.Default(), mockUpdater)

			req := httptest.NewRequest(http.MethodPost, "/update", nil)
			w := httptest.NewRecorder()

			handler(w, req)

			require.Equal(t, tc.expectedStatus, w.Code)
		})
	}
}

func TestUpdateStatsHandler(t *testing.T) {
	testCases := []struct {
		desc           string
		prepare        func(*core.MockUpdater)
		expectedStatus int
		wantBody       bool
		expectedBody   core.UpdateStats
	}{
		{
			desc: "success - returns stats",
			prepare: func(u *core.MockUpdater) {
				u.EXPECT().Stats(gomock.Any()).Return(core.UpdateStats{
					WordsTotal:    1000,
					WordsUnique:   500,
					ComicsFetched: 100,
					ComicsTotal:   404,
				}, nil)
			},
			expectedStatus: http.StatusOK,
			wantBody:       true,
			expectedBody: core.UpdateStats{
				WordsTotal:    1000,
				WordsUnique:   500,
				ComicsFetched: 100,
				ComicsTotal:   404,
			},
		},
		{
			desc: "error - service unavailable",
			prepare: func(u *core.MockUpdater) {
				u.EXPECT().Stats(gomock.Any()).Return(core.UpdateStats{}, core.ErrServiceUnavailable)
			},
			expectedStatus: http.StatusServiceUnavailable,
		},
		{
			desc: "error - internal error",
			prepare: func(u *core.MockUpdater) {
				u.EXPECT().Stats(gomock.Any()).Return(core.UpdateStats{}, errors.New("internal"))
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

			handler := rest.NewUpdateStatsHandler(slog.Default(), mockUpdater)

			req := httptest.NewRequest(http.MethodGet, "/update/stats", nil)
			w := httptest.NewRecorder()

			handler(w, req)

			require.Equal(t, tc.expectedStatus, w.Code)
			if tc.wantBody {
				require.Equal(t, "application/json", w.Header().Get("Content-Type"))
				var stats core.UpdateStats
				err := json.NewDecoder(w.Body).Decode(&stats)
				require.NoError(t, err)
				require.Equal(t, tc.expectedBody, stats)
			}
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

			handler := rest.NewDropHandler(slog.Default(), mockUpdater)

			req := httptest.NewRequest(http.MethodDelete, "/drop", nil)
			w := httptest.NewRecorder()

			handler(w, req)

			require.Equal(t, tc.expectedStatus, w.Code)
		})
	}
}
