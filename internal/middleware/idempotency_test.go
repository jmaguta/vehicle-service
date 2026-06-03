package middleware

import (
	"context"
	"errors"
	"net/http"
	"net/http/httptest"
	"testing"

	"github.com/jackc/pgx/v5"
	"github.com/jackc/pgx/v5/pgconn"

	"github.com/jmaguta/vehicle-service/internal/auth"
)

// --- mocks ---

type mockQuerier struct {
	row        pgx.Row
	execCalled bool
}

func (m *mockQuerier) QueryRow(_ context.Context, _ string, _ ...any) pgx.Row {
	return m.row
}

func (m *mockQuerier) Exec(_ context.Context, _ string, _ ...any) (pgconn.CommandTag, error) {
	m.execCalled = true
	return pgconn.CommandTag{}, nil
}

type mockRow struct {
	status int
	body   []byte
	err    error
}

func (r *mockRow) Scan(dest ...any) error {
	if r.err != nil {
		return r.err
	}
	if len(dest) >= 2 {
		if p, ok := dest[0].(*int); ok {
			*p = r.status
		}
		if p, ok := dest[1].(*[]byte); ok {
			*p = r.body
		}
	}
	return nil
}

// --- helpers ---

func requestWithMutationID(method, path, mutationID string, workshopID string) *http.Request {
	r := httptest.NewRequest(method, path, nil)
	if mutationID != "" {
		r.Header.Set("Client-Mutation-Id", mutationID)
	}
	if workshopID != "" {
		claims := &auth.Claims{}
		claims.WorkshopID = workshopID
		ctx := context.WithValue(r.Context(), ClaimsKey, claims)
		r = r.WithContext(ctx)
	}
	return r
}

// --- tests ---

func TestIdempotency_NoHeader_PassesThrough(t *testing.T) {
	called := false
	handler := Idempotency(nil)(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		called = true
		w.WriteHeader(http.StatusCreated)
		_, _ = w.Write([]byte(`{"id":"1"}`))
	}))

	rr := httptest.NewRecorder()
	handler.ServeHTTP(rr, httptest.NewRequest(http.MethodPost, "/vehicles", nil))

	if !called {
		t.Fatal("expected handler to be called when no Client-Mutation-Id header")
	}
	if rr.Code != http.StatusCreated {
		t.Errorf("expected 201, got %d", rr.Code)
	}
	if rr.Header().Get("X-Idempotent-Replayed") != "" {
		t.Error("unexpected X-Idempotent-Replayed header")
	}
}

func TestIdempotency_CacheHit_ReplaysCachedResponse(t *testing.T) {
	db := &mockQuerier{
		row: &mockRow{status: http.StatusCreated, body: []byte(`{"id":"42"}`)},
	}
	called := false
	handler := Idempotency(db)(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		called = true
		w.WriteHeader(http.StatusInternalServerError)
	}))

	rr := httptest.NewRecorder()
	r := requestWithMutationID(http.MethodPost, "/vehicles", "mut-abc", "ws-111")
	handler.ServeHTTP(rr, r)

	if called {
		t.Error("handler should not be called on a cache hit")
	}
	if rr.Code != http.StatusCreated {
		t.Errorf("expected replayed status 201, got %d", rr.Code)
	}
	if rr.Body.String() != `{"id":"42"}` {
		t.Errorf("unexpected body: %s", rr.Body.String())
	}
	if rr.Header().Get("X-Idempotent-Replayed") != "true" {
		t.Error("expected X-Idempotent-Replayed: true")
	}
	if db.execCalled {
		t.Error("Exec should not be called on a cache hit")
	}
}

func TestIdempotency_CacheMiss_ExecutesAndStores(t *testing.T) {
	db := &mockQuerier{
		row: &mockRow{err: errors.New("no rows")},
	}
	handler := Idempotency(db)(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.WriteHeader(http.StatusCreated)
		_, _ = w.Write([]byte(`{"id":"99"}`))
	}))

	rr := httptest.NewRecorder()
	r := requestWithMutationID(http.MethodPost, "/vehicles", "mut-xyz", "ws-222")
	handler.ServeHTTP(rr, r)

	if rr.Code != http.StatusCreated {
		t.Errorf("expected 201, got %d", rr.Code)
	}
	if rr.Header().Get("X-Idempotent-Replayed") != "" {
		t.Error("unexpected X-Idempotent-Replayed on cache miss")
	}
	if !db.execCalled {
		t.Error("expected Exec to be called to store the response")
	}
}

func TestIdempotency_5xxNotCached(t *testing.T) {
	db := &mockQuerier{
		row: &mockRow{err: errors.New("no rows")},
	}
	handler := Idempotency(db)(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.WriteHeader(http.StatusInternalServerError)
		_, _ = w.Write([]byte(`{"error":"boom"}`))
	}))

	rr := httptest.NewRecorder()
	r := requestWithMutationID(http.MethodPost, "/vehicles", "mut-err", "ws-333")
	handler.ServeHTTP(rr, r)

	if rr.Code != http.StatusInternalServerError {
		t.Errorf("expected 500, got %d", rr.Code)
	}
	if db.execCalled {
		t.Error("Exec must not be called for 5xx responses")
	}
}

func TestIdempotency_NoJWTClaims_UsesEmptyWorkshopID(t *testing.T) {
	db := &mockQuerier{
		row: &mockRow{err: errors.New("no rows")},
	}
	handler := Idempotency(db)(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.WriteHeader(http.StatusOK)
	}))

	rr := httptest.NewRecorder()
	r := httptest.NewRequest(http.MethodPost, "/vehicles", nil)
	r.Header.Set("Client-Mutation-Id", "mut-svc")
	handler.ServeHTTP(rr, r)

	if rr.Code != http.StatusOK {
		t.Errorf("expected 200, got %d", rr.Code)
	}
	if !db.execCalled {
		t.Error("expected Exec to be called (store with empty workshop_id)")
	}
}
