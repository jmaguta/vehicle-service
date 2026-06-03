package handlers

import (
	"bytes"
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"testing"

	"github.com/jmaguta/vehicle-service/internal/vehicles"
)

func TestListVehicles_Success(t *testing.T) {
	reg := "AB12 CDE"
	repo := &mockRepo{vehicleList: []vehicles.VehicleWithCustomer{
		{Vehicle: vehicles.Vehicle{ID: "v1", Registration: reg, Active: true}},
	}}
	h := NewVehicleHandler(repo, testLog)

	r := httptest.NewRequest(http.MethodGet, "/vehicles", nil)
	r = withClaims(t, r, "admin", "ws-1")
	rr := httptest.NewRecorder()

	h.List(rr, r)

	if rr.Code != http.StatusOK {
		t.Fatalf("expected 200, got %d", rr.Code)
	}
	var got []vehicles.VehicleWithCustomer
	if err := json.NewDecoder(rr.Body).Decode(&got); err != nil {
		t.Fatal(err)
	}
	if len(got) != 1 || got[0].Registration != reg {
		t.Errorf("unexpected response: %+v", got)
	}
}

func TestListVehicles_EmptyReturnsArray(t *testing.T) {
	repo := &mockRepo{}
	h := NewVehicleHandler(repo, testLog)

	r := httptest.NewRequest(http.MethodGet, "/vehicles", nil)
	r = withClaims(t, r, "admin", "ws-1")
	rr := httptest.NewRecorder()

	h.List(rr, r)

	if rr.Code != http.StatusOK {
		t.Fatalf("expected 200, got %d", rr.Code)
	}
	var got []vehicles.VehicleWithCustomer
	if err := json.NewDecoder(rr.Body).Decode(&got); err != nil {
		t.Fatal(err)
	}
	if len(got) != 0 {
		t.Errorf("expected empty array, got %+v", got)
	}
}

func TestGetVehicle_NotFound(t *testing.T) {
	repo := &mockRepo{}
	h := NewVehicleHandler(repo, testLog)

	r := httptest.NewRequest(http.MethodGet, "/vehicles/missing", nil)
	r = withClaims(t, r, "admin", "ws-1")
	r = chiURLParam(r, "id", "missing")
	rr := httptest.NewRecorder()

	h.Get(rr, r)

	if rr.Code != http.StatusNotFound {
		t.Fatalf("expected 404, got %d", rr.Code)
	}
}

func TestCreateVehicle_Success(t *testing.T) {
	repo := &mockRepo{}
	h := NewVehicleHandler(repo, testLog)

	custID := "cust-uuid"
	body, _ := json.Marshal(map[string]string{
		"customer_id":  custID,
		"registration": "AB12 CDE",
	})
	r := httptest.NewRequest(http.MethodPost, "/vehicles", bytes.NewReader(body))
	r.Header.Set("Content-Type", "application/json")
	r = withClaims(t, r, "admin", "ws-1")
	rr := httptest.NewRecorder()

	h.Create(rr, r)

	if rr.Code != http.StatusCreated {
		t.Fatalf("expected 201, got %d: %s", rr.Code, rr.Body.String())
	}
}

func TestCreateVehicle_MissingRegistration(t *testing.T) {
	repo := &mockRepo{}
	h := NewVehicleHandler(repo, testLog)

	body := `{"customer_id":"cust-uuid"}`
	r := httptest.NewRequest(http.MethodPost, "/vehicles", bytes.NewBufferString(body))
	r.Header.Set("Content-Type", "application/json")
	r = withClaims(t, r, "admin", "ws-1")
	rr := httptest.NewRecorder()

	h.Create(rr, r)

	if rr.Code != http.StatusBadRequest {
		t.Fatalf("expected 400, got %d", rr.Code)
	}
}

func TestCreateVehicle_MissingCustomerID(t *testing.T) {
	repo := &mockRepo{}
	h := NewVehicleHandler(repo, testLog)

	body := `{"registration":"AB12 CDE"}`
	r := httptest.NewRequest(http.MethodPost, "/vehicles", bytes.NewBufferString(body))
	r.Header.Set("Content-Type", "application/json")
	r = withClaims(t, r, "admin", "ws-1")
	rr := httptest.NewRecorder()

	h.Create(rr, r)

	if rr.Code != http.StatusBadRequest {
		t.Fatalf("expected 400, got %d", rr.Code)
	}
}

func TestCreateVehicle_InvalidBody(t *testing.T) {
	repo := &mockRepo{}
	h := NewVehicleHandler(repo, testLog)

	r := httptest.NewRequest(http.MethodPost, "/vehicles", bytes.NewBufferString("not-json"))
	r = withClaims(t, r, "admin", "ws-1")
	rr := httptest.NewRecorder()

	h.Create(rr, r)

	if rr.Code != http.StatusBadRequest {
		t.Fatalf("expected 400, got %d", rr.Code)
	}
}
