package middleware_test

import (
	"context"
	"net/http"
	"net/http/httptest"
	"search-service/api/adapters/rest/middleware"
	"sync"
	"sync/atomic"
	"testing"
	"time"

	"github.com/stretchr/testify/require"
)

func TestRateLimit(t *testing.T) {
	testCases := []struct {
		desc     string
		rps      int
		requests int
	}{
		{
			desc:     "requests < rate limit",
			rps:      10,
			requests: 5,
		},
		{
			desc:     "requests > rate limit",
			rps:      5,
			requests: 10,
		},
		{
			desc:     "requests = rate limit",
			rps:      50,
			requests: 50,
		},
	}

	for _, tc := range testCases {
		t.Run(tc.desc, func(t *testing.T) {
			limiter := middleware.NewRateLimiter(tc.rps)

			handler := limiter.Limit(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {}))

			var wg sync.WaitGroup
			var reqCount atomic.Int32

			start := time.Now()
			for i := 0; i < tc.requests; i++ {
				wg.Go(func() {
					req := httptest.NewRequest(http.MethodGet, "/", nil)
					rec := httptest.NewRecorder()

					handler.ServeHTTP(rec, req)

					if rec.Code == http.StatusOK {
						reqCount.Add(1)
					}
				})
			}

			wg.Wait()

			elapsed := time.Since(start)
			successReq := reqCount.Load()
			require.Equal(t, tc.requests, int(successReq))
			actualRPS := float64(successReq) / elapsed.Seconds()
			require.LessOrEqual(t, actualRPS, float64(tc.rps)*1.3)
		})
	}
}

func TestRateLimitZeroOrNegativeRate(t *testing.T) {
	testCases := []struct {
		desc               string
		rps                int
		requests           int
		expectedSuccessReq int
	}{
		{
			desc:               "rate limit is zero",
			rps:                0,
			requests:           10,
			expectedSuccessReq: 0,
		},
		{
			desc:               "rate limit is negative",
			rps:                -1,
			requests:           10,
			expectedSuccessReq: 0,
		},
	}

	for _, tc := range testCases {
		t.Run(tc.desc, func(t *testing.T) {
			limiter := middleware.NewRateLimiter(tc.rps)

			handler := limiter.Limit(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {}))

			var wg sync.WaitGroup
			var successReq atomic.Int32

			for i := 0; i < tc.requests; i++ {
				wg.Go(func() {
					ctx, cancel := context.WithTimeout(context.Background(), 1*time.Second)
					defer cancel()
					req := httptest.NewRequest(http.MethodGet, "/", nil).WithContext(ctx)
					rec := httptest.NewRecorder()

					handler.ServeHTTP(rec, req)

					if rec.Code == http.StatusOK {
						successReq.Add(1)
					}
				})
			}

			wg.Wait()

			require.Equal(t, tc.expectedSuccessReq, int(successReq.Load()))
		})
	}
}
