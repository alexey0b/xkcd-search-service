package middleware_test

import (
	"net/http"
	"net/http/httptest"
	"search-service/api/adapters/rest/middleware"
	"sync"
	"testing"
	"time"

	"github.com/stretchr/testify/require"
)

func TestConcurrencyLimit(t *testing.T) {
	testCases := []struct {
		desc            string
		concurrency     int
		requests        int
		expectedSuccess int
		expectedReject  int
	}{
		{
			desc:            "requests > limit",
			concurrency:     2,
			requests:        5,
			expectedSuccess: 2,
			expectedReject:  3,
		},
		{
			desc:            "requests < limit",
			concurrency:     3,
			requests:        2,
			expectedSuccess: 2,
			expectedReject:  0,
		},
		{
			desc:            "requests = limit",
			concurrency:     3,
			requests:        3,
			expectedSuccess: 3,
			expectedReject:  0,
		},
	}

	for _, tc := range testCases {
		t.Run(tc.desc, func(t *testing.T) {
			limiter := middleware.NewConcurrencyLimiter(tc.concurrency)

			handler := limiter.Limit(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
				time.Sleep(100 * time.Millisecond)
			}))

			var wg sync.WaitGroup
			var mu sync.Mutex
			successCount := 0
			rejectCount := 0

			for i := 0; i < tc.requests; i++ {
				wg.Add(1)
				go func() {
					defer wg.Done()

					req := httptest.NewRequest(http.MethodGet, "/", nil)
					rec := httptest.NewRecorder()

					handler.ServeHTTP(rec, req)

					mu.Lock()
					switch rec.Code {
					case http.StatusOK:
						successCount++
					case http.StatusServiceUnavailable:
						rejectCount++
					}
					mu.Unlock()
				}()
			}

			wg.Wait()

			require.Equal(t, tc.expectedSuccess, successCount)
			require.Equal(t, tc.expectedReject, rejectCount)
		})
	}
}

func TestConcurrencyLimitReleasesSlot(t *testing.T) {
	limiter := middleware.NewConcurrencyLimiter(1)

	handler := limiter.Limit(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		time.Sleep(100 * time.Millisecond)
	}))

	req1 := httptest.NewRequest(http.MethodGet, "/", nil)
	rec1 := httptest.NewRecorder()
	handler.ServeHTTP(rec1, req1)
	require.Equal(t, http.StatusOK, rec1.Code)

	req2 := httptest.NewRequest(http.MethodGet, "/", nil)
	rec2 := httptest.NewRecorder()
	handler.ServeHTTP(rec2, req2)
	require.Equal(t, http.StatusOK, rec2.Code)
}
