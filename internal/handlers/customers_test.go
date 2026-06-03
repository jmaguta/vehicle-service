package handlers

import (
	"bytes"
	"context"
	"encoding/json"
	"errors"
	"log/slog"
	"net/http"
	"net/http/httptest"
	"os"
	"testing"

	"github.com/go-chi/chi/v5"
	mw "github.com/jmaguta/vehicle-service/internal/middleware"
	"github.com/jmaguta/vehicle-service/internal/auth"
	"github.com/jmaguta/vehicle-service/internal/testhelpers"
	"github.com/jmaguta/vehicle-service/internal/vehicles"
)

// --- shared mock repository ---

type mockRepo struct {
	customers []vehicles.Customer
	vehicleList []vehicles.VehicleWithCustomer
	createErr error
}

func (m *mockRepo) ListCustomers(_ context.Context, _, _ string, _ *bool) ([]vehicles.Customer, error) {
	return m.customers, nil
}
func (m *mockRepo) GetCustomer(_ context.Context, _, id string) (vehicles.Customer, error) {
	for _, c := range m.customers {
		if c.ID == id {
			return c, nil
		}
	}
	return vehicles.Customer{}, errors.New("not found")
}
func (m *mockRepo) CreateCustomer(_ context.Context, p vehicles.CreateCustomerParams) (vehicles.Customer, error) {
	if m.createErr != nil {
		return vehicles.Customer{}, m.createErr
	}
	return vehicles.Customer{ID: "new-id", WorkshopID: p.WorkshopID, Name: p.Name, Active: true}, nil
}
func (m *mockRepo) UpdateCustomer(_ context.Context, _, id string, _ vehicles.UpdateCustomerParams) (vehicles.Customer, error) {
	for _, c := range m.customers {
		if c.ID == id {
			return c, nil
		}
	}
	return vehicles.Customer{}, errors.New("not found")
}
func (m *mockRepo) ListVehicles(_ context.Context, _, _, _ string, _ *bool) ([]vehicles.VehicleWithCustomer, error) {
	return m.vehicleList, nil
}
func (m *mockRepo) GetVehicle(_ context.Context, _, id string) (vehicles.VehicleWithCustomer, error) {
	for _, v := range m.vehicleList {
		if v.ID == id {
			return v, nil
		}
	}
	return vehicles.VehicleWithCustomer{}, errors.New("not found")
}
func (m *mockRepo) CreateVehicle(_ context.Context, p vehicles.CreateVehicleParams) (vehicles.VehicleWithCustomer, error) {
	if m.createErr != nil {
		return vehicles.VehicleWithCustomer{}, m.createErr
	}
	return vehicles.VehicleWithCustomer{Vehicle: vehicles.Vehicle{ID: "new-v-id", Registration: p.Registration}}, nil
}
func (m *mockRepo) UpdateVehicle(_ context.Context, _, id string, _ vehicles.UpdateVehicleParams) (vehicles.VehicleWithCustomer, error) {
	for _, v := range m.vehicleList {
		if v.ID == id {
			return v, nil
		}
	}
	return vehicles.VehicleWithCustomer{}, errors.New("not found")
}

// --- test helpers ---

var testLog = slog.New(slog.NewTextHandler(os.Stderr, &slog.HandlerOptions{Level: slog.LevelError}))

func withClaims(t *testing.T, r *http.Request, role, workshopID string) *http.Request {
	t.Helper()
	t.Setenv("SUPABASE_JWT_SECRET", testhelpers.TestJWTSecret)
	claims := &auth.Claims{}
	claims.Role = role
	claims.WorkshopID = workshopID
	ctx := context.WithValue(r.Context(), mw.ClaimsKey, claims)
	return r.WithContext(ctx)
}

func chiURLParam(r *http.Request, key, val string) *http.Request {
	rctx := chi.NewRouteContext()
	rctx.URLParams.Add(key, val)
	return r.WithContext(context.WithValue(r.Context(), chi.RouteCtxKey, rctx))
}

// --- customer tests ---

func TestListCustomers_Success(t *testing.T) {
	repo := &mockRepo{customers: []vehicles.Customer{{ID: "c1", Name: "Acme", Active: true}}}
	h := NewCustomerHandler(repo, testLog)

	r := httptest.NewRequest(http.MethodGet, "/customers", nil)
	r = withClaims(t, r, "admin", "ws-1")
	rr := httptest.NewRecorder()

	h.List(rr, r)

	if rr.Code != http.StatusOK {
		t.Fatalf("expected 200, got %d", rr.Code)
	}
	var got []vehicles.Customer
	if err := json.NewDecoder(rr.Body).Decode(&got); err != nil {
		t.Fatal(err)
	}
	if len(got) != 1 || got[0].Name != "Acme" {
		t.Errorf("unexpected response: %+v", got)
	}
}

func TestListCustomers_EmptyReturnsArray(t *testing.T) {
	repo := &mockRepo{}
	h := NewCustomerHandler(repo, testLog)

	r := httptest.NewRequest(http.MethodGet, "/customers", nil)
	r = withClaims(t, r, "admin", "ws-1")
	rr := httptest.NewRecorder()

	h.List(rr, r)

	if rr.Code != http.StatusOK {
		t.Fatalf("expected 200, got %d", rr.Code)
	}
	var got []vehicles.Customer
	if err := json.NewDecoder(rr.Body).Decode(&got); err != nil {
		t.Fatal(err)
	}
	if len(got) != 0 {
		t.Errorf("expected empty array, got %+v", got)
	}
}

func TestGetCustomer_NotFound(t *testing.T) {
	repo := &mockRepo{}
	h := NewCustomerHandler(repo, testLog)

	r := httptest.NewRequest(http.MethodGet, "/customers/missing", nil)
	r = withClaims(t, r, "admin", "ws-1")
	r = chiURLParam(r, "id", "missing")
	rr := httptest.NewRecorder()

	h.Get(rr, r)

	if rr.Code != http.StatusNotFound {
		t.Fatalf("expected 404, got %d", rr.Code)
	}
}

func TestCreateCustomer_Success(t *testing.T) {
	repo := &mockRepo{}
	h := NewCustomerHandler(repo, testLog)

	body := `{"name":"New Corp"}`
	r := httptest.NewRequest(http.MethodPost, "/customers", bytes.NewBufferString(body))
	r.Header.Set("Content-Type", "application/json")
	r = withClaims(t, r, "admin", "ws-1")
	rr := httptest.NewRecorder()

	h.Create(rr, r)

	if rr.Code != http.StatusCreated {
		t.Fatalf("expected 201, got %d: %s", rr.Code, rr.Body.String())
	}
}

func TestCreateCustomer_MissingName(t *testing.T) {
	repo := &mockRepo{}
	h := NewCustomerHandler(repo, testLog)

	body := `{"email":"test@example.com"}`
	r := httptest.NewRequest(http.MethodPost, "/customers", bytes.NewBufferString(body))
	r.Header.Set("Content-Type", "application/json")
	r = withClaims(t, r, "admin", "ws-1")
	rr := httptest.NewRecorder()

	h.Create(rr, r)

	if rr.Code != http.StatusBadRequest {
		t.Fatalf("expected 400, got %d", rr.Code)
	}
}

func TestCreateCustomer_InvalidBody(t *testing.T) {
	repo := &mockRepo{}
	h := NewCustomerHandler(repo, testLog)

	r := httptest.NewRequest(http.MethodPost, "/customers", bytes.NewBufferString("not-json"))
	r = withClaims(t, r, "admin", "ws-1")
	rr := httptest.NewRecorder()

	h.Create(rr, r)

	if rr.Code != http.StatusBadRequest {
		t.Fatalf("expected 400, got %d", rr.Code)
	}
}
