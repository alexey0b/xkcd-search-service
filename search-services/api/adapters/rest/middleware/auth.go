package middleware

import (
	"fmt"
	"net/http"
	"search-service/api/core"
	"strings"
	"time"

	"github.com/golang-jwt/jwt/v5"
)

const (
	tokenPrefix  = "Token "
	validSubject = "superuser"
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

func (tm *JwtAuthenticator) CheckToken(next http.Handler) http.Handler {
	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		var token string

		// Приоритет 1: Authorization header
		authHeader := r.Header.Get("Authorization")
		cleanedToken, found := strings.CutPrefix(authHeader, tokenPrefix)
		if found {
			token = cleanedToken
		} else {
			// Приоритет 2: Cookie
			cookie, err := r.Cookie("jwt_token")
			if err != nil {
				http.Error(w, http.StatusText(http.StatusUnauthorized), http.StatusUnauthorized)
				return
			}
			token = cookie.Value
		}

		if err := tm.ValidateToken(token); err != nil {
			http.Error(w, http.StatusText(http.StatusUnauthorized), http.StatusUnauthorized)
			return
		}
		next.ServeHTTP(w, r)
	})
}
