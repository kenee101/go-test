package middleware_test

import (
	"net/http"
	"net/http/httptest"
	"testing"
	"time"

	"github.com/golang-jwt/jwt/v5"
	"github.com/stretchr/testify/assert"

	"github.com/kenee101/go-test/internal/middleware"
)

const testSecret = "test-secret"

func makeToken(t *testing.T, userID, role string, ttl time.Duration) string {
	t.Helper()
	claims := jwt.MapClaims{
		"sub":  userID,
		"role": role,
		"exp":  time.Now().Add(ttl).Unix(),
	}
	tok, err := jwt.NewWithClaims(jwt.SigningMethodHS256, claims).SignedString([]byte(testSecret))
	if err != nil {
		t.Fatalf("makeToken: %v", err)
	}
	return tok
}

func TestAuthMiddleware_StatusCodes(t *testing.T) {
	next := http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.WriteHeader(http.StatusOK)
	})
	mw := middleware.AuthMiddleware(testSecret)(next)

	validToken := makeToken(t, "abc123", "user", time.Hour)
	expiredToken := makeToken(t, "abc123", "user", -time.Hour)

	tests := []struct {
		name       string
		authHeader string
		wantStatus int
	}{
		{"no header", "", http.StatusUnauthorized},
		{"wrong scheme", "Token " + validToken, http.StatusUnauthorized},
		{"malformed token", "Bearer not.a.token", http.StatusUnauthorized},
		{"expired token", "Bearer " + expiredToken, http.StatusUnauthorized},
		{"valid token", "Bearer " + validToken, http.StatusOK},
	}

	for _, tc := range tests {
		t.Run(tc.name, func(t *testing.T) {
			req := httptest.NewRequest(http.MethodGet, "/", nil)
			if tc.authHeader != "" {
				req.Header.Set("Authorization", tc.authHeader)
			}
			rr := httptest.NewRecorder()
			mw.ServeHTTP(rr, req)
			assert.Equal(t, tc.wantStatus, rr.Code)
		})
	}
}

func TestAuthMiddleware_ContextValues(t *testing.T) {
	const userID = "64b1f1a2e3d4c5b6a7f8e9d0"
	const role = "admin"

	var gotUserID, gotRole string
	next := http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		gotUserID, _ = r.Context().Value(middleware.CtxUserID).(string)
		gotRole, _ = r.Context().Value(middleware.CtxRole).(string)
		w.WriteHeader(http.StatusOK)
	})

	mw := middleware.AuthMiddleware(testSecret)(next)

	token := makeToken(t, userID, role, time.Hour)
	req := httptest.NewRequest(http.MethodGet, "/", nil)
	req.Header.Set("Authorization", "Bearer "+token)
	mw.ServeHTTP(httptest.NewRecorder(), req)

	assert.Equal(t, userID, gotUserID)
	assert.Equal(t, role, gotRole)
}

func TestAuthMiddleware_WrongSecret(t *testing.T) {
	token := makeToken(t, "user1", "user", time.Hour)

	called := false
	next := http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) { called = true })
	mw := middleware.AuthMiddleware("different-secret")(next)

	req := httptest.NewRequest(http.MethodGet, "/", nil)
	req.Header.Set("Authorization", "Bearer "+token)
	rr := httptest.NewRecorder()
	mw.ServeHTTP(rr, req)

	assert.Equal(t, http.StatusUnauthorized, rr.Code)
	assert.False(t, called, "next handler should not have been called")
}

func TestAuthMiddleware_ContextKeyType(t *testing.T) {
	// Middleware uses a typed contextKey, so a plain string lookup of "user_id"
	// must return nil — the two key types are distinct in Go's context package.
	next := http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		assert.Nil(t, r.Context().Value("user_id"),
			"plain string key should not resolve the typed context key set by middleware")
		w.WriteHeader(http.StatusOK)
	})

	mw := middleware.AuthMiddleware(testSecret)(next)
	token := makeToken(t, "abc", "user", time.Hour)

	req := httptest.NewRequest(http.MethodGet, "/", nil)
	req.Header.Set("Authorization", "Bearer "+token)
	// No plain-string injection — only the middleware's typed key should be present.

	mw.ServeHTTP(httptest.NewRecorder(), req)
}
