package middleware

import (
	"context"
	"net/http"
	"strings"

	"auth-app/internal/service"
)

type ctxKey struct{}

func Auth(auth *service.AuthService) func(http.Handler) http.Handler {
	return func(next http.Handler) http.Handler {
		return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
			scheme, token, ok := strings.Cut(r.Header.Get("Authorization"), " ")
			if !ok || !strings.EqualFold(scheme, "Bearer") || token == "" {
				unauthorized(w)
				return
			}

			userID, err := auth.ParseToken(token)
			if err != nil {
				unauthorized(w)
				return
			}

			ctx := context.WithValue(r.Context(), ctxKey{}, userID)
			next.ServeHTTP(w, r.WithContext(ctx))
		})
	}
}

func UserIDFromContext(ctx context.Context) (uint64, bool) {
	id, ok := ctx.Value(ctxKey{}).(uint64)
	return id, ok
}

func unauthorized(w http.ResponseWriter) {
	w.Header().Set("Content-Type", "application/json")
	w.WriteHeader(http.StatusUnauthorized)
	w.Write([]byte(`{"error":"unauthorized"}`))
}