package aaa_test

import (
	"os"
	"testing"
	"time"

	"github.com/golang-jwt/jwt/v5"
	"yadro.com/course/api/adapters/aaa"
)

const (
	testUser   = "admin"
	testPass   = "password123"
	testSecret = "supersecretkey_that_is_at_least_32_bytes!"
)

func withEnv(t *testing.T, env map[string]string) {
	t.Helper()
	for _, key := range []string{"ADMIN_USER", "ADMIN_PASSWORD", "JWT_SECRET"} {
		orig, existed := os.LookupEnv(key)
		if existed {
			t.Cleanup(func() {
				if err := os.Setenv(key, orig); err != nil {
					t.Errorf("failed to restore env %s: %v", key, err)
				}
			})
		} else {
			t.Cleanup(func() {
				if err := os.Unsetenv(key); err != nil {
					t.Errorf("failed to unset env %s: %v", key, err)
				}
			})
		}
		if err := os.Unsetenv(key); err != nil {
			t.Fatalf("failed to unset env %s: %v", key, err)
		}
	}
	for key, val := range env {
		if err := os.Setenv(key, val); err != nil {
			t.Fatalf("failed to set env %s: %v", key, err)
		}
	}
}

func newAAA(t *testing.T, ttl time.Duration) aaa.AAA {
	t.Helper()
	withEnv(t, map[string]string{
		"ADMIN_USER":     testUser,
		"ADMIN_PASSWORD": testPass,
		"JWT_SECRET":     testSecret,
	})
	a, err := aaa.New(ttl)
	if err != nil {
		t.Fatalf("aaa.New: %v", err)
	}
	return a
}

func TestNew(t *testing.T) {
	tests := []struct {
		name    string
		env     map[string]string
		wantErr bool
	}{
		{
			name:    "missing ADMIN_USER",
			env:     map[string]string{"ADMIN_PASSWORD": testPass, "JWT_SECRET": testSecret},
			wantErr: true,
		},
		{
			name:    "missing ADMIN_PASSWORD",
			env:     map[string]string{"ADMIN_USER": testUser, "JWT_SECRET": testSecret},
			wantErr: true,
		},
		{
			name:    "missing JWT_SECRET",
			env:     map[string]string{"ADMIN_USER": testUser, "ADMIN_PASSWORD": testPass},
			wantErr: true,
		},
		{
			name:    "JWT_SECRET too short",
			env:     map[string]string{"ADMIN_USER": testUser, "ADMIN_PASSWORD": testPass, "JWT_SECRET": "tooshort"},
			wantErr: true,
		},
		{
			name:    "all vars valid",
			env:     map[string]string{"ADMIN_USER": testUser, "ADMIN_PASSWORD": testPass, "JWT_SECRET": testSecret},
			wantErr: false,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			withEnv(t, tt.env)
			_, err := aaa.New(time.Hour)
			if (err != nil) != tt.wantErr {
				t.Errorf("New() error = %v, wantErr = %v", err, tt.wantErr)
			}
		})
	}
}

func TestLogin(t *testing.T) {
	a := newAAA(t, time.Hour)

	tests := []struct {
		name    string
		user    string
		pass    string
		wantErr bool
	}{
		{"valid credentials", testUser, testPass, false},
		{"wrong username", "nobody", testPass, true},
		{"wrong password", testUser, "wrongpassword", true},
		{"empty username", "", testPass, true},
		{"empty password", testUser, "", true},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			token, err := a.Login(tt.user, tt.pass)
			if (err != nil) != tt.wantErr {
				t.Errorf("Login() error = %v, wantErr = %v", err, tt.wantErr)
			}
			if !tt.wantErr && token == "" {
				t.Error("want non-empty token on success")
			}
		})
	}
}

func TestVerify(t *testing.T) {
	a := newAAA(t, time.Hour)

	validToken, err := a.Login(testUser, testPass)
	if err != nil {
		t.Fatalf("Login: %v", err)
	}

	makeToken := func(method jwt.SigningMethod, subject string, ttl time.Duration, key []byte) string {
		tok := jwt.NewWithClaims(method, jwt.RegisteredClaims{
			Subject:   subject,
			ExpiresAt: jwt.NewNumericDate(time.Now().Add(ttl)),
			IssuedAt:  jwt.NewNumericDate(time.Now()),
			NotBefore: jwt.NewNumericDate(time.Now()),
		})
		s, err := tok.SignedString(key)
		if err != nil {
			t.Fatalf("sign token: %v", err)
		}
		return s
	}

	tests := []struct {
		name    string
		token   string
		wantErr bool
	}{
		{
			name:    "valid token",
			token:   validToken,
			wantErr: false,
		},
		{
			name:    "expired token",
			token:   makeToken(jwt.SigningMethodHS256, "superuser", -time.Second, []byte(testSecret)),
			wantErr: true,
		},
		{
			name:    "wrong subject",
			token:   makeToken(jwt.SigningMethodHS256, "not_superuser", time.Hour, []byte(testSecret)),
			wantErr: true,
		},
		{
			name:    "wrong algorithm HS384",
			token:   makeToken(jwt.SigningMethodHS384, "superuser", time.Hour, []byte(testSecret)),
			wantErr: true,
		},
		{
			name:    "wrong signing key",
			token:   makeToken(jwt.SigningMethodHS256, "superuser", time.Hour, []byte("another_secret_key_that_is_32bytes!!")),
			wantErr: true,
		},
		{
			name:    "malformed token",
			token:   "not.a.valid.jwt",
			wantErr: true,
		},
		{
			name:    "empty token",
			token:   "",
			wantErr: true,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			err := a.Verify(tt.token)
			if (err != nil) != tt.wantErr {
				t.Errorf("Verify() error = %v, wantErr = %v", err, tt.wantErr)
			}
		})
	}
}
