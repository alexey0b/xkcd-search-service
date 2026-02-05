package middleware

import (
	"context"
	"fmt"
	"net/http"
	"search-service/frontend/core"
	"time"

	"github.com/golang-jwt/jwt/v5"
)

const (
	cookieJwtName = "jwt_token"
	loginPath     = "/login"
	validSubject  = "superuser" // JWT subject for admin users
)

// JwtAuthenticator handles JWT token creation and validation for admin authentication.
type JwtAuthenticator struct {
	adminUser     string
	adminPassword string
	jwtSecret     string
	ttl           time.Duration
}

// NewJwtAuthenticator creates a new JWT authenticator with admin credentials.
func NewJwtAuthenticator(adminUser, adminPassword, jwtSecret string, ttl time.Duration) (*JwtAuthenticator, error) {
	return &JwtAuthenticator{
		adminUser:     adminUser,
		adminPassword: adminPassword,
		ttl:           ttl,
		jwtSecret:     jwtSecret,
	}, nil
}

// CreateToken generates a new JWT token after validating admin credentials.
func (tm *JwtAuthenticator) CreateToken(name, password string) (string, error) {
	if name != tm.adminUser || password != tm.adminPassword {
		return "", core.ErrInvalidCredentials
	}
	token := jwt.NewWithClaims(jwt.SigningMethodHS256, jwt.RegisteredClaims{
		Subject:   validSubject,
		IssuedAt:  jwt.NewNumericDate(time.Now()),
		ExpiresAt: jwt.NewNumericDate(time.Now().Add(tm.ttl)),
	})
	signedToken, err := token.SignedString([]byte(tm.jwtSecret))
	if err != nil {
		return "", fmt.Errorf("failed to sign token: %w", err)
	}
	return signedToken, nil
}

// ValidateToken verifies JWT token signature, expiration, and subject.
func (tm *JwtAuthenticator) ValidateToken(tokenString string) error {
	token, err := jwt.Parse(tokenString, func(t *jwt.Token) (any, error) {
		return []byte(tm.jwtSecret), nil
	}, jwt.WithValidMethods([]string{jwt.SigningMethodHS256.Alg()}))
	if err != nil {
		return core.ErrInvalidCredentials
	}
	if !token.Valid {
		return core.ErrInvalidCredentials
	}

	// verify subject claim
	subject, err := token.Claims.GetSubject()
	if err != nil {
		return core.ErrInvalidCredentials
	}
	if subject != validSubject {
		return core.ErrInvalidCredentials
	}
	return nil
}

// CheckToken is a middleware that validates JWT token from cookie.
func (tm *JwtAuthenticator) CheckToken(next http.Handler) http.Handler {
	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		token, err := r.Cookie(cookieJwtName)
		if err != nil {
			http.Redirect(w, r, loginPath, http.StatusSeeOther)
			return
		}
		if err := tm.ValidateToken(token.Value); err != nil {
			http.Redirect(w, r, loginPath, http.StatusSeeOther)
			return
		}
		r = r.WithContext(context.WithValue(r.Context(), core.JwtTokenContextKey, token.Value))
		next.ServeHTTP(w, r)
	})
}
