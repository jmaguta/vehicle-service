package middleware

import (
	"bytes"
	"context"
	"net/http"

	"github.com/jackc/pgx/v5"
	"github.com/jackc/pgx/v5/pgconn"

	"github.com/jmaguta/vehicle-service/internal/auth"
)

type idempotencyQuerier interface {
	QueryRow(ctx context.Context, sql string, args ...any) pgx.Row
	Exec(ctx context.Context, sql string, args ...any) (pgconn.CommandTag, error)
}

// Idempotency returns a middleware that deduplicates mutation requests keyed on
// the Client-Mutation-Id header. Responses are stored for 24 hours (TTL set in
// the DB). A replayed response carries X-Idempotent-Replayed: true.
// If the header is absent the request passes through unchanged.
func Idempotency(db idempotencyQuerier) func(http.Handler) http.Handler {
	return func(next http.Handler) http.Handler {
		return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
			mutationID := r.Header.Get("Client-Mutation-Id")
			if mutationID == "" {
				next.ServeHTTP(w, r)
				return
			}

			wid := workshopIDFromCtx(r.Context())

			var status int
			var body []byte
			err := db.QueryRow(r.Context(),
				`SELECT status_code, response FROM vehicle.idempotency_keys
				  WHERE mutation_id = $1 AND workshop_id = $2 AND expires_at > now()`,
				mutationID, wid,
			).Scan(&status, &body)
			if err == nil {
				w.Header().Set("Content-Type", "application/json")
				w.Header().Set("X-Idempotent-Replayed", "true")
				w.WriteHeader(status)
				_, _ = w.Write(body)
				return
			}

			rec := &teeRecorder{ResponseWriter: w, code: http.StatusOK}
			next.ServeHTTP(rec, r)

			if rec.code < 500 {
				_, _ = db.Exec(r.Context(),
					`INSERT INTO vehicle.idempotency_keys
					    (mutation_id, workshop_id, path, status_code, response)
					 VALUES ($1, $2, $3, $4, $5)
					 ON CONFLICT DO NOTHING`,
					mutationID, wid, r.URL.Path, rec.code, rec.buf.Bytes(),
				)
			}
		})
	}
}

func workshopIDFromCtx(ctx context.Context) string {
	claims, ok := ctx.Value(ClaimsKey).(*auth.Claims)
	if !ok || claims == nil {
		return ""
	}
	return claims.WorkshopID
}

type teeRecorder struct {
	http.ResponseWriter
	code int
	buf  bytes.Buffer
}

func (t *teeRecorder) WriteHeader(code int) {
	t.code = code
	t.ResponseWriter.WriteHeader(code)
}

func (t *teeRecorder) Write(b []byte) (int, error) {
	t.buf.Write(b)
	return t.ResponseWriter.Write(b)
}
