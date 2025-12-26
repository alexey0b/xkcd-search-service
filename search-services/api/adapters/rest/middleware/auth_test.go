package middleware_test

import (
	"net/http"
	"net/http/httptest"
	"search-service/api/adapters/rest/middleware"
	"search-service/api/core"
	"testing"
	"time"

	"github.com/stretchr/testify/require"
)

const (
	validUser     = "admin"
	validPassword = "password"
	jwtSecret     = "your-secret-key"
)

func TestCreateToken(t *testing.T) {
	testCases := []struct {
		desc        string
		user        string
		password    string
		wantErr     bool
		expectedErr error
	}{
		{
			desc:     "success - valid credentials",
			user:     validUser,
			password: validPassword,
		},
		{
			desc:        "error - invalid user",
			user:        "invalid_user",
			password:    "password",
			wantErr:     true,
			expectedErr: core.ErrInvalidCredentials,
		},
		{
			desc:        "error - invalid password",
			user:        "admin",
			password:    "invalid_password",
			wantErr:     true,
			expectedErr: core.ErrInvalidCredentials,
		},
		{
			desc:        "error - both invalid",
			user:        "invalid_user",
			password:    "invalid_password",
			wantErr:     true,
			expectedErr: core.ErrInvalidCredentials,
		},
	}

	for _, tc := range testCases {
		t.Run(tc.desc, func(t *testing.T) {
			auth, err := middleware.NewJwtAuthenticator(validUser, validPassword, jwtSecret, time.Hour)
			require.NoError(t, err)

			token, err := auth.CreateToken(tc.user, tc.password)

			if tc.wantErr {
				require.Equal(t, tc.expectedErr, err)
				require.Empty(t, token)
			} else {
				require.NoError(t, err)
				require.NotEmpty(t, token)

				err = auth.ValidateToken(token)
				require.NoError(t, err)
			}
		})
	}
}

func TestValidateToken(t *testing.T) {
	testCases := []struct {
		desc                string
		prepareTokenAndAuth func(*testing.T) (string, *middleware.JwtAuthenticator)
		wantErr             bool
		expectedErr         error
	}{
		{
			desc: "success - valid token",
			prepareTokenAndAuth: func(t *testing.T) (string, *middleware.JwtAuthenticator) {
				auth, err := middleware.NewJwtAuthenticator(validUser, validPassword, jwtSecret, 1*time.Hour)
				require.NoError(t, err)
				token, err := auth.CreateToken(validUser, validPassword)
				require.NoError(t, err)
				return token, auth
			},
			wantErr: false,
		},
		{
			desc: "error - expired token",
			prepareTokenAndAuth: func(t *testing.T) (string, *middleware.JwtAuthenticator) {
				auth, err := middleware.NewJwtAuthenticator(validUser, validPassword, jwtSecret, 1*time.Millisecond)
				require.NoError(t, err)
				token, err := auth.CreateToken(validUser, validPassword)
				require.NoError(t, err)
				time.Sleep(10 * time.Millisecond)
				return token, auth
			},
			wantErr:     true,
			expectedErr: core.ErrInvalidCredentials,
		},
		{
			desc: "error - invalid signature",
			prepareTokenAndAuth: func(t *testing.T) (string, *middleware.JwtAuthenticator) {
				auth, err := middleware.NewJwtAuthenticator(validUser, validPassword, jwtSecret, time.Minute)
				require.NoError(t, err)
				token, err := auth.CreateToken(validUser, validPassword)
				require.NoError(t, err)

				// проверяем с помощью otherAuth, содержащий другой сгенерированный jwt secret
				otherAuth, err := middleware.NewJwtAuthenticator(validUser, validPassword, "invalid signature", time.Hour)
				require.NoError(t, err)
				return token, otherAuth
			},
			wantErr:     true,
			expectedErr: core.ErrInvalidCredentials,
		},
		{
			desc: "error - malformed token",
			prepareTokenAndAuth: func(t *testing.T) (string, *middleware.JwtAuthenticator) {
				auth, err := middleware.NewJwtAuthenticator(validUser, validPassword, jwtSecret, time.Hour)
				require.NoError(t, err)
				return "invalid.token.string", auth
			},
			wantErr:     true,
			expectedErr: core.ErrInvalidCredentials,
		},
		{
			desc: "error - empty token",
			prepareTokenAndAuth: func(_ *testing.T) (string, *middleware.JwtAuthenticator) {
				auth, err := middleware.NewJwtAuthenticator(validUser, validPassword, jwtSecret, time.Hour)
				require.NoError(t, err)
				return "", auth
			},
			wantErr:     true,
			expectedErr: core.ErrInvalidCredentials,
		},
	}

	for _, tc := range testCases {
		t.Run(tc.desc, func(t *testing.T) {
			token, auth := tc.prepareTokenAndAuth(t)
			err := auth.ValidateToken(token)

			if tc.wantErr {
				require.Equal(t, tc.expectedErr, err)
			} else {
				require.NoError(t, err)
			}
		})
	}
}

func TestCheckToken(t *testing.T) {
	testCases := []struct {
		desc           string
		prepareRequest func(*testing.T, *middleware.JwtAuthenticator) *http.Request
		expectedStatus int
		expectNext     bool
	}{
		{
			desc: "success - valid token",
			prepareRequest: func(t *testing.T, auth *middleware.JwtAuthenticator) *http.Request {
				token, err := auth.CreateToken(validUser, validPassword)
				require.NoError(t, err)
				req := httptest.NewRequest(http.MethodGet, "/", nil)
				req.Header.Set("Authorization", "Token "+token)
				return req
			},
			expectedStatus: http.StatusOK,
			expectNext:     true,
		},
		{
			desc: "error - missing authorization header",
			prepareRequest: func(t *testing.T, auth *middleware.JwtAuthenticator) *http.Request {
				return httptest.NewRequest(http.MethodGet, "/", nil)
			},
			expectedStatus: http.StatusUnauthorized,
			expectNext:     false,
		},
		{
			desc: "error - wrong prefix",
			prepareRequest: func(t *testing.T, auth *middleware.JwtAuthenticator) *http.Request {
				token, err := auth.CreateToken(validUser, validPassword)
				require.NoError(t, err)
				req := httptest.NewRequest(http.MethodGet, "/", nil)
				req.Header.Set("Authorization", "Bearer "+token)
				return req
			},
			expectedStatus: http.StatusUnauthorized,
			expectNext:     false,
		},
		{
			desc: "error - invalid token",
			prepareRequest: func(t *testing.T, auth *middleware.JwtAuthenticator) *http.Request {
				req := httptest.NewRequest(http.MethodGet, "/", nil)
				req.Header.Set("Authorization", "Token invalid.token.string")
				return req
			},
			expectedStatus: http.StatusUnauthorized,
			expectNext:     false,
		},
		{
			desc: "error - expired token",
			prepareRequest: func(t *testing.T, auth *middleware.JwtAuthenticator) *http.Request {
				shortAuth, err := middleware.NewJwtAuthenticator(validUser, validPassword, jwtSecret, 1*time.Millisecond)
				require.NoError(t, err)
				token, err := shortAuth.CreateToken(validUser, validPassword)
				require.NoError(t, err)
				time.Sleep(10 * time.Millisecond)
				req := httptest.NewRequest(http.MethodGet, "/", nil)
				req.Header.Set("Authorization", "Token "+token)
				return req
			},
			expectedStatus: http.StatusUnauthorized,
			expectNext:     false,
		},
	}

	for _, tc := range testCases {
		t.Run(tc.desc, func(t *testing.T) {
			auth, err := middleware.NewJwtAuthenticator(validUser, validPassword, jwtSecret, time.Hour)
			require.NoError(t, err)

			nextCalled := false
			next := http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
				nextCalled = true
				w.WriteHeader(http.StatusOK)
			})

			handler := auth.CheckToken(next)
			req := tc.prepareRequest(t, auth)
			w := httptest.NewRecorder()

			handler.ServeHTTP(w, req)

			require.Equal(t, tc.expectedStatus, w.Code)
			require.Equal(t, tc.expectNext, nextCalled)
		})
	}
}
