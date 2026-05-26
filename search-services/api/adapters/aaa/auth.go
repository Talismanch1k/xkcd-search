package aaa

import (
	"crypto/subtle"
	"errors"
	"fmt"
	"os"
	"time"

	"github.com/golang-jwt/jwt/v5"
)

const (
	adminRole = "superuser" // token subject
)

var errInvalidCredentials = errors.New("invalid credentials")

// Authentication, Authorization, Accounting
type AAA struct {
	users    map[string]string
	tokenTTL time.Duration
	secret   []byte
}

func New(tokenTTL time.Duration) (AAA, error) {
	user, ok := os.LookupEnv("ADMIN_USER")
	if !ok {
		return AAA{}, errors.New("could not get admin user from enviroment")
	}

	password, ok := os.LookupEnv("ADMIN_PASSWORD")
	if !ok {
		return AAA{}, errors.New("could not get admin password from enviroment")
	}

	secret, ok := os.LookupEnv("JWT_SECRET")
	if !ok {
		return AAA{}, errors.New("could not get secret from enviroment")
	}
	if len(secret) < 32 {
		return AAA{}, errors.New("JWT_SECRET must be at least 32 bytes")
	}

	return AAA{
		users:    map[string]string{user: password},
		tokenTTL: tokenTTL,
		secret:   []byte(secret),
	}, nil
}

func (a AAA) Login(name, password string) (string, error) {
	storedPassword, ok := a.users[name]

	if !ok || subtle.ConstantTimeCompare([]byte(storedPassword), []byte(password)) != 1 {
		return "", errInvalidCredentials
	}

	claims := jwt.RegisteredClaims{
		Issuer:    "api-service",
		Subject:   adminRole,
		ExpiresAt: jwt.NewNumericDate(time.Now().Add(a.tokenTTL)),
		IssuedAt:  jwt.NewNumericDate(time.Now()),
		NotBefore: jwt.NewNumericDate(time.Now()),
	}

	token := jwt.NewWithClaims(jwt.SigningMethodHS256, claims)

	tokenString, err := token.SignedString([]byte(a.secret))
	if err != nil {
		return "", fmt.Errorf("sign token: %w", err)
	}

	return tokenString, nil
}

func (a AAA) Verify(tokenString string) error {
	token, err := jwt.ParseWithClaims(
		tokenString,
		&jwt.RegisteredClaims{},
		func(t *jwt.Token) (any, error) {
			if t.Method.Alg() != jwt.SigningMethodHS256.Alg() {
				return nil, fmt.Errorf("unexpected signing method: %v", t.Method.Alg())
			}

			return []byte(a.secret), nil
		},
	)
	if err != nil {
		return fmt.Errorf("parse token: %w", err)
	}

	claims, ok := token.Claims.(*jwt.RegisteredClaims)
	if !ok {
		return errors.New("invalid token claims")
	}

	if claims.Subject != adminRole {
		return errors.New("invalid token subject")
	}

	return nil
}
