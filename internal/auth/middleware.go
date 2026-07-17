package auth

import (
	"context"
	"net/http"
	"strings"
)

type contextKey struct{}

var userIDKey contextKey
var apiKeyKey contextKey

func UserIDFromContext(ctx context.Context) string {
	id := ctx.Value(userIDKey).(string)
	return id
}

func KeyFromContext(ctx context.Context) (Key, bool) {
	key, ok := ctx.Value(apiKeyKey).(Key)
	return key, ok
}

func RequireJWT(svc *Service) func(http.Handler) http.HandlerFunc {
	return func(next http.Handler) http.HandlerFunc {
		return func(w http.ResponseWriter, r *http.Request) {
			token, err := bearerToken(r)
			if err != nil {
				http.Error(w, ErrUnauthorized.Error(), http.StatusUnauthorized)
				return
			}

			userID, err := svc.ParseJWT(token)
			if err != nil {
				http.Error(w, ErrUnauthorized.Error(), http.StatusUnauthorized)
				return
			}

			ctx := context.WithValue(r.Context(), userIDKey, userID)
			next.ServeHTTP(w, r.WithContext(ctx))
		}
	}
}

func RequireAPIKey(svc *Service, role KeyRole) func(http.Handler) http.HandlerFunc {
	return func(next http.Handler) http.HandlerFunc {
		return func(w http.ResponseWriter, r *http.Request) {
			plainKey, err := apiKey(r)
			if err != nil {
				http.Error(w, ErrUnauthorized.Error(), http.StatusUnauthorized)
				return
			}

			key, err := svc.ValidateKey(r.Context(), plainKey)
			if err != nil {
				http.Error(w, ErrUnauthorized.Error(), http.StatusUnauthorized)
				return
			}
			if key.Role != role {
				http.Error(w, ErrUnauthorized.Error(), http.StatusUnauthorized)
				return
			}

			ctx := context.WithValue(r.Context(), apiKeyKey, key)
			next.ServeHTTP(w, r.WithContext(ctx))
		}
	}
}

func bearerToken(r *http.Request) (string, error) {
	header := r.Header.Get("Authorization")
	if header == "" {
		return "", ErrUnauthorized
	}
	scheme, token, found := strings.Cut(header, " ")
	if !found || !strings.EqualFold(scheme, "Bearer") || token == "" {
		return "", ErrUnauthorized
	}
	return token, nil
}

func apiKey(r *http.Request) (string, error) {
	header := r.Header.Get("X-API-KEY")
	if header == "" {
		return "", ErrUnauthorized
	}
	return header, nil
}
