package middleware

import (
	"context"
	"log/slog"
	"net/http"
	"os"
	"time"

	chimw "github.com/go-chi/chi/v5/middleware"

	"github.com/jmaguta/vehicle-service/internal/auth"
)

// Logger returns a structured slog request logging middleware.
func Logger(log *slog.Logger) func(http.Handler) http.Handler {
	return func(next http.Handler) http.Handler {
		return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
			start := time.Now()
			ww := chimw.NewWrapResponseWriter(w, r.ProtoMajor)
			next.ServeHTTP(ww, r)
			if r.URL.Path == "/healthz" && ww.Status() == http.StatusOK {
				return
			}
			log.Info("request",
				"method", r.Method,
				"path", r.URL.Path,
				"status", ww.Status(),
				"bytes", ww.BytesWritten(),
				"duration_ms", time.Since(start).Milliseconds(),
				"request_id", chimw.GetReqID(r.Context()),
			)
		})
	}
}

type contextKey string

const ClaimsKey contextKey = "claims"

// RequireAuth validates the Bearer JWT and stores claims in context.
func RequireAuth(next http.Handler) http.Handler {
	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		token, err := auth.BearerToken(r.Header.Get("Authorization"))
		if err != nil {
			http.Error(w, `{"error":"unauthorized"}`, http.StatusUnauthorized)
			return
		}

		claims, err := auth.ValidateJWT(token)
		if err != nil {
			http.Error(w, `{"error":"unauthorized"}`, http.StatusUnauthorized)
			return
		}

		ctx := context.WithValue(r.Context(), ClaimsKey, claims)
		next.ServeHTTP(w, r.WithContext(ctx))
	})
}

// RequireAuthOrServiceKey accepts either a valid Bearer JWT or a valid X-Service-Key.
func RequireAuthOrServiceKey(next http.Handler) http.Handler {
	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		expected := os.Getenv("SERVICE_KEY")
		if expected != "" && r.Header.Get("X-Service-Key") == expected {
			next.ServeHTTP(w, r)
			return
		}

		token, err := auth.BearerToken(r.Header.Get("Authorization"))
		if err != nil {
			http.Error(w, `{"error":"unauthorized"}`, http.StatusUnauthorized)
			return
		}

		claims, err := auth.ValidateJWT(token)
		if err != nil {
			http.Error(w, `{"error":"unauthorized"}`, http.StatusUnauthorized)
			return
		}

		ctx := context.WithValue(r.Context(), ClaimsKey, claims)
		next.ServeHTTP(w, r.WithContext(ctx))
	})
}

// RequireAdminOrServiceKey accepts either a valid X-Service-Key or a Bearer JWT with
// role "admin" or "service_role".
func RequireAdminOrServiceKey(next http.Handler) http.Handler {
	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		expected := os.Getenv("SERVICE_KEY")
		if expected != "" && r.Header.Get("X-Service-Key") == expected {
			next.ServeHTTP(w, r)
			return
		}

		token, err := auth.BearerToken(r.Header.Get("Authorization"))
		if err != nil {
			http.Error(w, `{"error":"unauthorized"}`, http.StatusUnauthorized)
			return
		}

		claims, err := auth.ValidateJWT(token)
		if err != nil {
			http.Error(w, `{"error":"unauthorized"}`, http.StatusUnauthorized)
			return
		}
		if claims.Role != "admin" && claims.Role != "service_role" {
			http.Error(w, `{"error":"forbidden"}`, http.StatusForbidden)
			return
		}

		ctx := context.WithValue(r.Context(), ClaimsKey, claims)
		next.ServeHTTP(w, r.WithContext(ctx))
	})
}
