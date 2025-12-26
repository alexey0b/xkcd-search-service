package api_test

import (
	"context"
	"encoding/json"
	"log/slog"
	"net/http"
	"net/http/httptest"
	"search-service/frontend/adapters/api"
	"search-service/frontend/core"
	"testing"
	"time"

	"github.com/stretchr/testify/require"
)

const statusPingOK core.PingStatus = "ok"

func TestPing(t *testing.T) {
	testCases := []struct {
		desc         string
		serverStatus int
		serverReply  core.PingResponse
		wantErr      bool
	}{
		{
			desc:         "success - ping ok",
			serverStatus: http.StatusOK,
			serverReply:  core.PingResponse{Replies: map[string]core.PingStatus{"api": statusPingOK}},
		},
		{
			desc:         "error - bad request",
			serverStatus: http.StatusBadRequest,
			wantErr:      true,
		},
		{
			desc:         "error - service unavailable",
			serverStatus: http.StatusServiceUnavailable,
			wantErr:      true,
		},
	}

	for _, tc := range testCases {
		t.Run(tc.desc, func(t *testing.T) {
			server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
				w.WriteHeader(tc.serverStatus)
				if tc.serverStatus == http.StatusOK {
					_ = json.NewEncoder(w).Encode(tc.serverReply)
				}
			}))
			defer server.Close()

			client := api.NewClient(server.URL, time.Second, slog.Default())
			result, err := client.Ping(context.Background())

			if tc.wantErr {
				require.Error(t, err)
			} else {
				require.NoError(t, err)
				require.Equal(t, tc.serverReply, result)
			}
		})
	}
}

func TestSearch(t *testing.T) {
	testCases := []struct {
		desc         string
		phrase       string
		serverStatus int
		serverReply  core.SearchResult
		wantErr      bool
		expectedErr  error
	}{
		{
			desc:         "success - returns comics",
			phrase:       "test",
			serverStatus: http.StatusOK,
			serverReply: core.SearchResult{
				Comics: []core.Comic{{ID: 1, URL: "url1"}},
				Total:  1,
			},
		},
		{
			desc:         "error - bad request",
			phrase:       "",
			serverStatus: http.StatusBadRequest,
			wantErr:      true,
			expectedErr:  core.ErrBadArguments,
		},
		{
			desc:         "error - service unavailable",
			phrase:       "test",
			serverStatus: http.StatusServiceUnavailable,
			wantErr:      true,
			expectedErr:  core.ErrServiceUnavailable,
		},
	}

	for _, tc := range testCases {
		t.Run(tc.desc, func(t *testing.T) {
			server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
				require.Equal(t, "/api/search", r.URL.Path)
				require.Equal(t, http.MethodGet, r.Method)
				require.Equal(t, tc.phrase, r.URL.Query().Get("phrase"))
				require.Equal(t, "10000", r.URL.Query().Get("limit"))

				w.WriteHeader(tc.serverStatus)
				if tc.serverStatus == http.StatusOK {
					_ = json.NewEncoder(w).Encode(tc.serverReply)
				}
			}))
			defer server.Close()

			client := api.NewClient(server.URL, time.Second, slog.Default())
			result, err := client.Search(context.Background(), tc.phrase)

			if tc.wantErr {
				require.ErrorIs(t, err, tc.expectedErr)
			} else {
				require.NoError(t, err)
				require.Equal(t, tc.serverReply, result)
			}
		})
	}
}

func TestGetUpdateStats(t *testing.T) {
	testCases := []struct {
		desc         string
		serverStatus int
		serverReply  core.UpdateStats
		wantErr      bool
		expectedErr  error
	}{
		{
			desc:         "success - returns stats",
			serverStatus: http.StatusOK,
			serverReply:  core.UpdateStats{ComicsTotal: 100, WordsTotal: 1000},
		},
		{
			desc:         "error - service unavailable",
			serverStatus: http.StatusServiceUnavailable,
			wantErr:      true,
			expectedErr:  core.ErrServiceUnavailable,
		},
	}

	for _, tc := range testCases {
		t.Run(tc.desc, func(t *testing.T) {
			server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
				require.Equal(t, "/api/db/stats", r.URL.Path)
				require.Equal(t, http.MethodGet, r.Method)

				w.WriteHeader(tc.serverStatus)
				if tc.serverStatus == http.StatusOK {
					_ = json.NewEncoder(w).Encode(tc.serverReply)
				}
			}))
			defer server.Close()

			client := api.NewClient(server.URL, time.Second, slog.Default())
			result, err := client.GetUpdateStats(context.Background())

			if tc.wantErr {
				require.ErrorIs(t, err, tc.expectedErr)
			} else {
				require.NoError(t, err)
				require.Equal(t, tc.serverReply, result)
			}
		})
	}
}

const (
	statusUpdateIdle    core.UpdateStatus = "idle"
	statusUpdateRunning core.UpdateStatus = "running"
)

func TestGetUpdateStatus(t *testing.T) {
	testCases := []struct {
		desc         string
		serverStatus int
		serverReply  struct {
			Status core.UpdateStatus `json:"status"`
		}
		wantErr      bool
		expectedErr  error
		expectedResp core.UpdateStatus
	}{
		{
			desc:         "success - idle status",
			serverStatus: http.StatusOK,
			serverReply: struct {
				Status core.UpdateStatus `json:"status"`
			}{Status: statusUpdateIdle},
			expectedResp: statusUpdateIdle,
		},
		{
			desc:         "success - running status",
			serverStatus: http.StatusOK,
			serverReply: struct {
				Status core.UpdateStatus `json:"status"`
			}{Status: statusUpdateRunning},
			expectedResp: statusUpdateRunning,
		},
		{
			desc:         "error - service unavailable",
			serverStatus: http.StatusServiceUnavailable,
			wantErr:      true,
			expectedErr:  core.ErrServiceUnavailable,
		},
	}

	for _, tc := range testCases {
		t.Run(tc.desc, func(t *testing.T) {
			server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
				require.Equal(t, "/api/db/status", r.URL.Path)
				require.Equal(t, http.MethodGet, r.Method)

				w.WriteHeader(tc.serverStatus)
				if tc.serverStatus == http.StatusOK {
					_ = json.NewEncoder(w).Encode(tc.serverReply)
				}
			}))
			defer server.Close()

			client := api.NewClient(server.URL, time.Second, slog.Default())
			result, err := client.GetUpdateStatus(context.Background())

			if tc.wantErr {
				require.ErrorIs(t, err, tc.expectedErr)
			} else {
				require.NoError(t, err)
				require.Equal(t, tc.expectedResp, result)
			}
		})
	}
}

func TestUpdate(t *testing.T) {
	testCases := []struct {
		desc         string
		token        string
		serverStatus int
		wantErr      bool
		expectedErr  error
	}{
		{
			desc:         "success - update started",
			token:        "valid-token",
			serverStatus: http.StatusOK,
		},
		{
			desc:         "error - already running",
			token:        "valid-token",
			serverStatus: http.StatusAccepted,
			wantErr:      true,
			expectedErr:  core.ErrAlreadyExists,
		},
		{
			desc:         "error - service unavailable",
			token:        "valid-token",
			serverStatus: http.StatusServiceUnavailable,
			wantErr:      true,
			expectedErr:  core.ErrServiceUnavailable,
		},
	}

	for _, tc := range testCases {
		t.Run(tc.desc, func(t *testing.T) {
			server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
				require.Equal(t, "/api/db/update", r.URL.Path)
				require.Equal(t, http.MethodPost, r.Method)
				require.Equal(t, "Token "+tc.token, r.Header.Get("Authorization"))

				w.WriteHeader(tc.serverStatus)
			}))
			defer server.Close()

			client := api.NewClient(server.URL, time.Second, slog.Default())
			ctx := context.WithValue(context.Background(), core.JwtTokenContextKey, tc.token)
			err := client.Update(ctx)

			if tc.wantErr {
				require.ErrorIs(t, err, tc.expectedErr)
			} else {
				require.NoError(t, err)
			}
		})
	}
}

func TestDrop(t *testing.T) {
	testCases := []struct {
		desc         string
		token        string
		serverStatus int
		wantErr      bool
		expectedErr  error
	}{
		{
			desc:         "success - drop completed",
			token:        "valid-token",
			serverStatus: http.StatusOK,
		},
		{
			desc:         "error - service unavailable",
			token:        "valid-token",
			serverStatus: http.StatusServiceUnavailable,
			wantErr:      true,
			expectedErr:  core.ErrServiceUnavailable,
		},
	}

	for _, tc := range testCases {
		t.Run(tc.desc, func(t *testing.T) {
			server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
				require.Equal(t, "/api/db", r.URL.Path)
				require.Equal(t, http.MethodDelete, r.Method)
				require.Equal(t, "Token "+tc.token, r.Header.Get("Authorization"))

				w.WriteHeader(tc.serverStatus)
			}))
			defer server.Close()

			client := api.NewClient(server.URL, time.Second, slog.Default())
			ctx := context.WithValue(context.Background(), core.JwtTokenContextKey, tc.token)
			err := client.Drop(ctx)

			if tc.wantErr {
				require.ErrorIs(t, err, tc.expectedErr)
			} else {
				require.NoError(t, err)
			}
		})
	}
}
