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
	"github.com/jmaguta/vehicle-service/internal/auth"
	mw "github.com/jmaguta/vehicle-service/internal/middleware"
	"github.com/jmaguta/vehicle-service/internal/testhelpers"
	"github.com/jmaguta/vehicle-service/internal/vehicles"
)

// --- shared mock repository ---

type mockRepo struct {
	customers   []vehicles.Customer
	vehicleList []vehicles.VehicleWithCustomer
	createErr   error

	updateCustomerFn func(id string, p vehicles.UpdateCustomerParams) (vehicles.Customer, error)
	updateVehicleFn  func(id string, p vehicles.UpdateVehicleParams) (vehicles.VehicleWithCustomer, error)

	lastUpdateCustomerParams *vehicles.UpdateCustomerParams
	lastUpdateVehicleParams  *vehicles.UpdateVehicleParams
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
func (m *mockRepo) UpdateCustomer(_ context.Context, _, id string, p vehicles.UpdateCustomerParams) (vehicles.Customer, error) {
	m.lastUpdateCustomerParams = &p
	if m.updateCustomerFn != nil {
		return m.updateCustomerFn(id, p)
	}
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
func (m *mockRepo) UpdateVehicle(_ context.Context, _, id string, p vehicles.UpdateVehicleParams) (vehicles.VehicleWithCustomer, error) {
	m.lastUpdateVehicleParams = &p
	if m.updateVehicleFn != nil {
		return m.updateVehicleFn(id, p)
	}
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

func strPtr(s string) *string { return &s }
func boolPtr(b bool) *bool    { return &b }

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
	decodeData(t, rr.Body, &got)
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
	decodeData(t, rr.Body, &got)
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

// --- PATCH customer tests ---

func TestUpdateCustomer_Success(t *testing.T) {
	repo := &mockRepo{customers: []vehicles.Customer{{ID: "c1", Name: "Old Corp", Active: true}}}
	h := NewCustomerHandler(repo, testLog)

	body := `{"name":"New Corp"}`
	r := httptest.NewRequest(http.MethodPatch, "/customers/c1", bytes.NewBufferString(body))
	r.Header.Set("Content-Type", "application/json")
	r = withClaims(t, r, "admin", "ws-1")
	r = chiURLParam(r, "id", "c1")
	rr := httptest.NewRecorder()

	h.Patch(rr, r)

	if rr.Code != http.StatusOK {
		t.Fatalf("expected 200, got %d: %s", rr.Code, rr.Body.String())
	}
	var got vehicles.Customer
	decodeData(t, rr.Body, &got)
	if got.ID != "c1" {
		t.Errorf("expected ID c1, got %q", got.ID)
	}
}

func TestUpdateCustomer_FieldTrimming(t *testing.T) {
	repo := &mockRepo{
		updateCustomerFn: func(_ string, p vehicles.UpdateCustomerParams) (vehicles.Customer, error) {
			return vehicles.Customer{ID: "c1", Name: "Trimmed Ltd"}, nil
		},
	}
	h := NewCustomerHandler(repo, testLog)

	body := `{"name":"  Trimmed Ltd  ","email":"  info@example.com  "}`
	r := httptest.NewRequest(http.MethodPatch, "/customers/c1", bytes.NewBufferString(body))
	r.Header.Set("Content-Type", "application/json")
	r = withClaims(t, r, "admin", "ws-1")
	r = chiURLParam(r, "id", "c1")
	rr := httptest.NewRecorder()

	h.Patch(rr, r)

	if rr.Code != http.StatusOK {
		t.Fatalf("expected 200, got %d", rr.Code)
	}
	p := repo.lastUpdateCustomerParams
	if p == nil {
		t.Fatal("expected params to be captured")
	}
	if p.Name == nil || *p.Name != "Trimmed Ltd" {
		t.Errorf("expected trimmed name, got %v", p.Name)
	}
	if p.Email == nil || *p.Email != "info@example.com" {
		t.Errorf("expected trimmed email, got %v", p.Email)
	}
}

func TestUpdateCustomer_MultipleFields(t *testing.T) {
	repo := &mockRepo{
		updateCustomerFn: func(_ string, _ vehicles.UpdateCustomerParams) (vehicles.Customer, error) {
			return vehicles.Customer{ID: "c2"}, nil
		},
	}
	h := NewCustomerHandler(repo, testLog)

	body := `{"name":"DVSA Corp","email":"dvsa@example.com","phone":"01234567890","active":true}`
	r := httptest.NewRequest(http.MethodPatch, "/customers/c2", bytes.NewBufferString(body))
	r.Header.Set("Content-Type", "application/json")
	r = withClaims(t, r, "admin", "ws-1")
	r = chiURLParam(r, "id", "c2")
	rr := httptest.NewRecorder()

	h.Patch(rr, r)

	if rr.Code != http.StatusOK {
		t.Fatalf("expected 200, got %d", rr.Code)
	}
	p := repo.lastUpdateCustomerParams
	if p.Name == nil || *p.Name != "DVSA Corp" {
		t.Errorf("unexpected name: %v", p.Name)
	}
	if p.Email == nil || *p.Email != "dvsa@example.com" {
		t.Errorf("unexpected email: %v", p.Email)
	}
	if p.Phone == nil || *p.Phone != "01234567890" {
		t.Errorf("unexpected phone: %v", p.Phone)
	}
	if p.Active == nil || *p.Active != true {
		t.Errorf("unexpected active: %v", p.Active)
	}
}

func TestUpdateCustomer_ActiveFalse(t *testing.T) {
	repo := &mockRepo{
		updateCustomerFn: func(_ string, _ vehicles.UpdateCustomerParams) (vehicles.Customer, error) {
			return vehicles.Customer{ID: "c1"}, nil
		},
	}
	h := NewCustomerHandler(repo, testLog)

	body := `{"active":false}`
	r := httptest.NewRequest(http.MethodPatch, "/customers/c1", bytes.NewBufferString(body))
	r.Header.Set("Content-Type", "application/json")
	r = withClaims(t, r, "admin", "ws-1")
	r = chiURLParam(r, "id", "c1")
	rr := httptest.NewRecorder()

	h.Patch(rr, r)

	if rr.Code != http.StatusOK {
		t.Fatalf("expected 200, got %d", rr.Code)
	}
	p := repo.lastUpdateCustomerParams
	if p.Active == nil || *p.Active != false {
		t.Errorf("expected Active=false, got %v", p.Active)
	}
}

func TestUpdateCustomer_OperatorFields(t *testing.T) {
	repo := &mockRepo{
		updateCustomerFn: func(_ string, _ vehicles.UpdateCustomerParams) (vehicles.Customer, error) {
			return vehicles.Customer{ID: "c3"}, nil
		},
	}
	h := NewCustomerHandler(repo, testLog)

	body := `{
		"operator_licence_number":"OF1234567",
		"licence_type":"Standard National",
		"traffic_area":"West of England",
		"company_reg_number":"12345678",
		"transport_manager_name":"J Smith",
		"transport_manager_cpc":"CPC-001"
	}`
	r := httptest.NewRequest(http.MethodPatch, "/customers/c3", bytes.NewBufferString(body))
	r.Header.Set("Content-Type", "application/json")
	r = withClaims(t, r, "admin", "ws-1")
	r = chiURLParam(r, "id", "c3")
	rr := httptest.NewRecorder()

	h.Patch(rr, r)

	if rr.Code != http.StatusOK {
		t.Fatalf("expected 200, got %d", rr.Code)
	}
	p := repo.lastUpdateCustomerParams
	if p.OperatorLicenceNumber == nil || *p.OperatorLicenceNumber != "OF1234567" {
		t.Errorf("unexpected operator_licence_number: %v", p.OperatorLicenceNumber)
	}
	if p.LicenceType == nil || *p.LicenceType != "Standard National" {
		t.Errorf("unexpected licence_type: %v", p.LicenceType)
	}
	if p.TrafficArea == nil || *p.TrafficArea != "West of England" {
		t.Errorf("unexpected traffic_area: %v", p.TrafficArea)
	}
	if p.CompanyRegNumber == nil || *p.CompanyRegNumber != "12345678" {
		t.Errorf("unexpected company_reg_number: %v", p.CompanyRegNumber)
	}
	if p.TransportManagerName == nil || *p.TransportManagerName != "J Smith" {
		t.Errorf("unexpected transport_manager_name: %v", p.TransportManagerName)
	}
	if p.TransportManagerCPC == nil || *p.TransportManagerCPC != "CPC-001" {
		t.Errorf("unexpected transport_manager_cpc: %v", p.TransportManagerCPC)
	}
}

func TestUpdateCustomer_NotFound(t *testing.T) {
	repo := &mockRepo{} // empty customers slice
	h := NewCustomerHandler(repo, testLog)

	body := `{"name":"Ghost Corp"}`
	r := httptest.NewRequest(http.MethodPatch, "/customers/nonexistent", bytes.NewBufferString(body))
	r.Header.Set("Content-Type", "application/json")
	r = withClaims(t, r, "admin", "ws-1")
	r = chiURLParam(r, "id", "nonexistent")
	rr := httptest.NewRecorder()

	h.Patch(rr, r)

	if rr.Code != http.StatusNotFound {
		t.Fatalf("expected 404, got %d", rr.Code)
	}
	var errResp map[string]string
	if err := json.NewDecoder(rr.Body).Decode(&errResp); err != nil {
		t.Fatal(err)
	}
	if errResp["error"] != "not found" {
		t.Errorf("unexpected error body: %v", errResp)
	}
}

func TestUpdateCustomer_RepoError(t *testing.T) {
	// Handler maps ALL repo errors to 404, not just not-found.
	repo := &mockRepo{
		updateCustomerFn: func(_ string, _ vehicles.UpdateCustomerParams) (vehicles.Customer, error) {
			return vehicles.Customer{}, errors.New("db timeout")
		},
	}
	h := NewCustomerHandler(repo, testLog)

	body := `{"name":"Any Corp"}`
	r := httptest.NewRequest(http.MethodPatch, "/customers/c1", bytes.NewBufferString(body))
	r.Header.Set("Content-Type", "application/json")
	r = withClaims(t, r, "admin", "ws-1")
	r = chiURLParam(r, "id", "c1")
	rr := httptest.NewRecorder()

	h.Patch(rr, r)

	if rr.Code != http.StatusNotFound {
		t.Fatalf("expected 404, got %d", rr.Code)
	}
}

func TestUpdateCustomer_InvalidBody(t *testing.T) {
	repo := &mockRepo{}
	h := NewCustomerHandler(repo, testLog)

	r := httptest.NewRequest(http.MethodPatch, "/customers/c1", bytes.NewBufferString("{bad json"))
	r = withClaims(t, r, "admin", "ws-1")
	r = chiURLParam(r, "id", "c1")
	rr := httptest.NewRecorder()

	h.Patch(rr, r)

	if rr.Code != http.StatusBadRequest {
		t.Fatalf("expected 400, got %d", rr.Code)
	}
	var errResp map[string]string
	if err := json.NewDecoder(rr.Body).Decode(&errResp); err != nil {
		t.Fatal(err)
	}
	if errResp["error"] != "invalid request body" {
		t.Errorf("unexpected error body: %v", errResp)
	}
}

func TestUpdateCustomer_OnlySetFieldsPopulated(t *testing.T) {
	repo := &mockRepo{
		updateCustomerFn: func(_ string, _ vehicles.UpdateCustomerParams) (vehicles.Customer, error) {
			return vehicles.Customer{ID: "c1"}, nil
		},
	}
	h := NewCustomerHandler(repo, testLog)

	body := `{"name":"Just Name"}`
	r := httptest.NewRequest(http.MethodPatch, "/customers/c1", bytes.NewBufferString(body))
	r.Header.Set("Content-Type", "application/json")
	r = withClaims(t, r, "admin", "ws-1")
	r = chiURLParam(r, "id", "c1")
	rr := httptest.NewRecorder()

	h.Patch(rr, r)

	if rr.Code != http.StatusOK {
		t.Fatalf("expected 200, got %d", rr.Code)
	}
	p := repo.lastUpdateCustomerParams
	if p.Name == nil {
		t.Error("expected Name to be set")
	}
	// Fields not in the request body must be nil — not empty string.
	if p.Email != nil {
		t.Errorf("expected Email to be nil, got %v", p.Email)
	}
	if p.Active != nil {
		t.Errorf("expected Active to be nil, got %v", p.Active)
	}
	if p.OperatorLicenceNumber != nil {
		t.Errorf("expected OperatorLicenceNumber to be nil, got %v", p.OperatorLicenceNumber)
	}
}

func TestUpdateCustomer_EmptyObject(t *testing.T) {
	repo := &mockRepo{
		updateCustomerFn: func(_ string, _ vehicles.UpdateCustomerParams) (vehicles.Customer, error) {
			return vehicles.Customer{ID: "c1", Name: "Unchanged"}, nil
		},
	}
	h := NewCustomerHandler(repo, testLog)

	r := httptest.NewRequest(http.MethodPatch, "/customers/c1", bytes.NewBufferString("{}"))
	r.Header.Set("Content-Type", "application/json")
	r = withClaims(t, r, "admin", "ws-1")
	r = chiURLParam(r, "id", "c1")
	rr := httptest.NewRecorder()

	h.Patch(rr, r)

	if rr.Code != http.StatusOK {
		t.Fatalf("expected 200 on empty body, got %d", rr.Code)
	}
}
