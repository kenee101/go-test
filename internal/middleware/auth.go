package middleware

import (
    "context"
    "net/http"
    "os"
    "strings"

    "github.com/golang-jwt/jwt/v5"
)

type ctxKey string

func AuthMiddleware(next http.Handler) http.Handler {
    return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
        auth := r.Header.Get("Authorization")
        if auth == "" {
            http.Error(w, "missing authorization", http.StatusUnauthorized)
            return
        }
        parts := strings.SplitN(auth, " ", 2)
        if len(parts) != 2 || strings.ToLower(parts[0]) != "bearer" {
            http.Error(w, "invalid authorization", http.StatusUnauthorized)
            return
        }
        tokenString := parts[1]
        secret := os.Getenv("JWT_SECRET")
        if secret == "" {
            secret = "secret"
        }
        token, err := jwt.Parse(tokenString, func(t *jwt.Token) (interface{}, error) {
            return []byte(secret), nil
        })
        if err != nil || !token.Valid {
            http.Error(w, "invalid token", http.StatusUnauthorized)
            return
        }
        claims, ok := token.Claims.(jwt.MapClaims)
        if !ok {
            http.Error(w, "invalid token claims", http.StatusUnauthorized)
            return
        }
        sub, _ := claims["sub"].(string)
        role, _ := claims["role"].(string)
        // attach to context
        ctx := context.WithValue(r.Context(), "user_id", sub)
        ctx = context.WithValue(ctx, "role", role)
        next.ServeHTTP(w, r.WithContext(ctx))
    })
}
