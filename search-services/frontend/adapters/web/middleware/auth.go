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
	validSubject = "superuser"

	cookieName = "jwt_token"

	loginPath = "/login"
)

type JwtAuthenticator struct {
	adminUser     string
	adminPassword string
	jwtSecret     string
	ttl           time.Duration
}

func NewJwtAuthenticator(adminUser, adminPassword, jwtSecret string, ttl time.Duration) (*JwtAuthenticator, error) {
	return &JwtAuthenticator{
		adminUser:     adminUser,
		adminPassword: adminPassword,
		ttl:           ttl,
		jwtSecret:     jwtSecret,
	}, nil
}

func (tm *JwtAuthenticator) CheckToken(next http.Handler) http.Handler {
	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		token, err := r.Cookie(cookieName)
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

func (tm *JwtAuthenticator) ValidateToken(tokenString string) error {
	token, err := jwt.Parse(tokenString, func(t *jwt.Token) (interface{}, error) {
		return []byte(tm.jwtSecret), nil
	}, jwt.WithValidMethods([]string{jwt.SigningMethodHS256.Alg()}))
	if err != nil {
		return core.ErrInvalidCredentials
	}
	if !token.Valid {
		return core.ErrInvalidCredentials
	}
	subject, err := token.Claims.GetSubject()
	if err != nil {
		return core.ErrInvalidCredentials
	}
	if subject != validSubject {
		return core.ErrInvalidCredentials
	}
	return nil
}
