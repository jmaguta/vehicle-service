package handlers

import (
	"bytes"
	"encoding/json"
	"errors"
	"io"
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
	decodeData(t, rr.Body, &got)
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
	decodeData(t, rr.Body, &got)
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

// --- PATCH vehicle tests ---

func TestUpdateVehicle_Success(t *testing.T) {
	repo := &mockRepo{vehicleList: []vehicles.VehicleWithCustomer{
		{Vehicle: vehicles.Vehicle{ID: "v1", Registration: "AB12CDE"}},
	}}
	h := NewVehicleHandler(repo, testLog)

	body := `{"registration":"XY99ZZZ"}`
	r := httptest.NewRequest(http.MethodPatch, "/vehicles/v1", bytes.NewBufferString(body))
	r.Header.Set("Content-Type", "application/json")
	r = withClaims(t, r, "admin", "ws-1")
	r = chiURLParam(r, "id", "v1")
	rr := httptest.NewRecorder()

	h.Patch(rr, r)

	if rr.Code != http.StatusOK {
		t.Fatalf("expected 200, got %d: %s", rr.Code, rr.Body.String())
	}
	var got vehicles.VehicleWithCustomer
	decodeData(t, rr.Body, &got)
	if got.ID != "v1" {
		t.Errorf("expected ID v1, got %q", got.ID)
	}
}

func TestUpdateVehicle_RegistrationUppercased(t *testing.T) {
	repo := &mockRepo{
		updateVehicleFn: func(_ string, _ vehicles.UpdateVehicleParams) (vehicles.VehicleWithCustomer, error) {
			return vehicles.VehicleWithCustomer{Vehicle: vehicles.Vehicle{ID: "v1"}}, nil
		},
	}
	h := NewVehicleHandler(repo, testLog)

	body := `{"registration":"ab12 cde"}`
	r := httptest.NewRequest(http.MethodPatch, "/vehicles/v1", bytes.NewBufferString(body))
	r.Header.Set("Content-Type", "application/json")
	r = withClaims(t, r, "admin", "ws-1")
	r = chiURLParam(r, "id", "v1")
	rr := httptest.NewRecorder()

	h.Patch(rr, r)

	if rr.Code != http.StatusOK {
		t.Fatalf("expected 200, got %d", rr.Code)
	}
	p := repo.lastUpdateVehicleParams
	if p == nil {
		t.Fatal("expected params to be captured")
	}
	if p.Registration == nil || *p.Registration != "AB12 CDE" {
		t.Errorf("expected registration uppercased to 'AB12 CDE', got %v", p.Registration)
	}
}

func TestUpdateVehicle_RegistrationAlreadyUppercase(t *testing.T) {
	repo := &mockRepo{
		updateVehicleFn: func(_ string, _ vehicles.UpdateVehicleParams) (vehicles.VehicleWithCustomer, error) {
			return vehicles.VehicleWithCustomer{Vehicle: vehicles.Vehicle{ID: "v1"}}, nil
		},
	}
	h := NewVehicleHandler(repo, testLog)

	body := `{"registration":"MN67PQR"}`
	r := httptest.NewRequest(http.MethodPatch, "/vehicles/v1", bytes.NewBufferString(body))
	r.Header.Set("Content-Type", "application/json")
	r = withClaims(t, r, "admin", "ws-1")
	r = chiURLParam(r, "id", "v1")
	rr := httptest.NewRecorder()

	h.Patch(rr, r)

	if rr.Code != http.StatusOK {
		t.Fatalf("expected 200, got %d", rr.Code)
	}
	p := repo.lastUpdateVehicleParams
	if p.Registration == nil || *p.Registration != "MN67PQR" {
		t.Errorf("expected 'MN67PQR', got %v", p.Registration)
	}
}

func TestUpdateVehicle_FieldTrimming(t *testing.T) {
	repo := &mockRepo{
		updateVehicleFn: func(_ string, _ vehicles.UpdateVehicleParams) (vehicles.VehicleWithCustomer, error) {
			return vehicles.VehicleWithCustomer{Vehicle: vehicles.Vehicle{ID: "v1"}}, nil
		},
	}
	h := NewVehicleHandler(repo, testLog)

	body := `{"make_model":"  Ford Transit  ","notes":"  check oil  "}`
	r := httptest.NewRequest(http.MethodPatch, "/vehicles/v1", bytes.NewBufferString(body))
	r.Header.Set("Content-Type", "application/json")
	r = withClaims(t, r, "admin", "ws-1")
	r = chiURLParam(r, "id", "v1")
	rr := httptest.NewRecorder()

	h.Patch(rr, r)

	if rr.Code != http.StatusOK {
		t.Fatalf("expected 200, got %d", rr.Code)
	}
	p := repo.lastUpdateVehicleParams
	if p.MakeModel == nil || *p.MakeModel != "Ford Transit" {
		t.Errorf("expected trimmed make_model, got %v", p.MakeModel)
	}
	if p.Notes == nil || *p.Notes != "check oil" {
		t.Errorf("expected trimmed notes, got %v", p.Notes)
	}
}

func TestUpdateVehicle_MultipleFields(t *testing.T) {
	repo := &mockRepo{
		updateVehicleFn: func(_ string, _ vehicles.UpdateVehicleParams) (vehicles.VehicleWithCustomer, error) {
			return vehicles.VehicleWithCustomer{Vehicle: vehicles.Vehicle{ID: "v1"}}, nil
		},
	}
	h := NewVehicleHandler(repo, testLog)

	body := `{"registration":"AA11BBB","make_model":"DAF XF","mileage":"150000","active":false}`
	r := httptest.NewRequest(http.MethodPatch, "/vehicles/v1", bytes.NewBufferString(body))
	r.Header.Set("Content-Type", "application/json")
	r = withClaims(t, r, "admin", "ws-1")
	r = chiURLParam(r, "id", "v1")
	rr := httptest.NewRecorder()

	h.Patch(rr, r)

	if rr.Code != http.StatusOK {
		t.Fatalf("expected 200, got %d", rr.Code)
	}
	p := repo.lastUpdateVehicleParams
	if p.Registration == nil || *p.Registration != "AA11BBB" {
		t.Errorf("unexpected registration: %v", p.Registration)
	}
	if p.MakeModel == nil || *p.MakeModel != "DAF XF" {
		t.Errorf("unexpected make_model: %v", p.MakeModel)
	}
	if p.Mileage == nil || *p.Mileage != "150000" {
		t.Errorf("unexpected mileage: %v", p.Mileage)
	}
	if p.Active == nil || *p.Active != false {
		t.Errorf("expected Active=false, got %v", p.Active)
	}
}

func TestUpdateVehicle_BooleanFields(t *testing.T) {
	repo := &mockRepo{
		updateVehicleFn: func(_ string, _ vehicles.UpdateVehicleParams) (vehicles.VehicleWithCustomer, error) {
			return vehicles.VehicleWithCustomer{Vehicle: vehicles.Vehicle{ID: "v1"}}, nil
		},
	}
	h := NewVehicleHandler(repo, testLog)

	body := `{"active":false,"has_adbluedpf":true,"vor_status":true}`
	r := httptest.NewRequest(http.MethodPatch, "/vehicles/v1", bytes.NewBufferString(body))
	r.Header.Set("Content-Type", "application/json")
	r = withClaims(t, r, "admin", "ws-1")
	r = chiURLParam(r, "id", "v1")
	rr := httptest.NewRecorder()

	h.Patch(rr, r)

	if rr.Code != http.StatusOK {
		t.Fatalf("expected 200, got %d", rr.Code)
	}
	p := repo.lastUpdateVehicleParams
	if p.Active == nil || *p.Active != false {
		t.Errorf("expected Active=false, got %v", p.Active)
	}
	if p.HasAdBlueDPF == nil || *p.HasAdBlueDPF != true {
		t.Errorf("expected HasAdBlueDPF=true, got %v", p.HasAdBlueDPF)
	}
	if p.VORStatus == nil || *p.VORStatus != true {
		t.Errorf("expected VORStatus=true, got %v", p.VORStatus)
	}
}

func TestUpdateVehicle_DVSAComplianceFields(t *testing.T) {
	repo := &mockRepo{
		updateVehicleFn: func(_ string, _ vehicles.UpdateVehicleParams) (vehicles.VehicleWithCustomer, error) {
			return vehicles.VehicleWithCustomer{Vehicle: vehicles.Vehicle{ID: "v1"}}, nil
		},
	}
	h := NewVehicleHandler(repo, testLog)

	body := `{
		"vin":"WBAJB0C50BCF08534",
		"dvsa_class":"C+E",
		"gross_vehicle_weight":"44000",
		"axle_count":"5",
		"inspection_interval_weeks":"6"
	}`
	r := httptest.NewRequest(http.MethodPatch, "/vehicles/v1", bytes.NewBufferString(body))
	r.Header.Set("Content-Type", "application/json")
	r = withClaims(t, r, "admin", "ws-1")
	r = chiURLParam(r, "id", "v1")
	rr := httptest.NewRecorder()

	h.Patch(rr, r)

	if rr.Code != http.StatusOK {
		t.Fatalf("expected 200, got %d", rr.Code)
	}
	p := repo.lastUpdateVehicleParams
	if p.VIN == nil || *p.VIN != "WBAJB0C50BCF08534" {
		t.Errorf("unexpected vin: %v", p.VIN)
	}
	if p.DVSAClass == nil || *p.DVSAClass != "C+E" {
		t.Errorf("unexpected dvsa_class: %v", p.DVSAClass)
	}
	if p.GrossVehicleWeight == nil || *p.GrossVehicleWeight != "44000" {
		t.Errorf("unexpected gross_vehicle_weight: %v", p.GrossVehicleWeight)
	}
	if p.AxleCount == nil || *p.AxleCount != "5" {
		t.Errorf("unexpected axle_count: %v", p.AxleCount)
	}
	if p.InspectionIntervalWeeks == nil || *p.InspectionIntervalWeeks != "6" {
		t.Errorf("unexpected inspection_interval_weeks: %v", p.InspectionIntervalWeeks)
	}
}

func TestUpdateVehicle_PMIDates(t *testing.T) {
	repo := &mockRepo{
		updateVehicleFn: func(_ string, _ vehicles.UpdateVehicleParams) (vehicles.VehicleWithCustomer, error) {
			return vehicles.VehicleWithCustomer{Vehicle: vehicles.Vehicle{ID: "v1"}}, nil
		},
	}
	h := NewVehicleHandler(repo, testLog)

	body := `{
		"mot_expiry":"2027-01-15",
		"tachograph_cal_due":"2026-12-01",
		"last_pmi_date":"2026-06-01",
		"next_pmi_due":"2026-12-01"
	}`
	r := httptest.NewRequest(http.MethodPatch, "/vehicles/v1", bytes.NewBufferString(body))
	r.Header.Set("Content-Type", "application/json")
	r = withClaims(t, r, "admin", "ws-1")
	r = chiURLParam(r, "id", "v1")
	rr := httptest.NewRecorder()

	h.Patch(rr, r)

	if rr.Code != http.StatusOK {
		t.Fatalf("expected 200, got %d", rr.Code)
	}
	p := repo.lastUpdateVehicleParams
	if p.MOTExpiry == nil || *p.MOTExpiry != "2027-01-15" {
		t.Errorf("unexpected mot_expiry: %v", p.MOTExpiry)
	}
	if p.TachographCalDue == nil || *p.TachographCalDue != "2026-12-01" {
		t.Errorf("unexpected tachograph_cal_due: %v", p.TachographCalDue)
	}
	if p.LastPMIDate == nil || *p.LastPMIDate != "2026-06-01" {
		t.Errorf("unexpected last_pmi_date: %v", p.LastPMIDate)
	}
	if p.NextPMIDue == nil || *p.NextPMIDue != "2026-12-01" {
		t.Errorf("unexpected next_pmi_due: %v", p.NextPMIDue)
	}
}

func TestUpdateVehicle_NotFound(t *testing.T) {
	repo := &mockRepo{} // empty vehicleList
	h := NewVehicleHandler(repo, testLog)

	body := `{"registration":"XX99YYY"}`
	r := httptest.NewRequest(http.MethodPatch, "/vehicles/nonexistent", bytes.NewBufferString(body))
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

func TestUpdateVehicle_RepoError(t *testing.T) {
	// Handler maps ALL repo errors to 404, not just not-found.
	repo := &mockRepo{
		updateVehicleFn: func(_ string, _ vehicles.UpdateVehicleParams) (vehicles.VehicleWithCustomer, error) {
			return vehicles.VehicleWithCustomer{}, errors.New("db timeout")
		},
	}
	h := NewVehicleHandler(repo, testLog)

	body := `{"registration":"AB12CDE"}`
	r := httptest.NewRequest(http.MethodPatch, "/vehicles/v1", bytes.NewBufferString(body))
	r.Header.Set("Content-Type", "application/json")
	r = withClaims(t, r, "admin", "ws-1")
	r = chiURLParam(r, "id", "v1")
	rr := httptest.NewRecorder()

	h.Patch(rr, r)

	if rr.Code != http.StatusNotFound {
		t.Fatalf("expected 404, got %d", rr.Code)
	}
}

func TestUpdateVehicle_InvalidBody(t *testing.T) {
	repo := &mockRepo{}
	h := NewVehicleHandler(repo, testLog)

	r := httptest.NewRequest(http.MethodPatch, "/vehicles/v1", bytes.NewBufferString("{bad json"))
	r = withClaims(t, r, "admin", "ws-1")
	r = chiURLParam(r, "id", "v1")
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

func TestUpdateVehicle_OnlySetFieldsPopulated(t *testing.T) {
	repo := &mockRepo{
		updateVehicleFn: func(_ string, _ vehicles.UpdateVehicleParams) (vehicles.VehicleWithCustomer, error) {
			return vehicles.VehicleWithCustomer{Vehicle: vehicles.Vehicle{ID: "v1"}}, nil
		},
	}
	h := NewVehicleHandler(repo, testLog)

	body := `{"registration":"CC33DDD"}`
	r := httptest.NewRequest(http.MethodPatch, "/vehicles/v1", bytes.NewBufferString(body))
	r.Header.Set("Content-Type", "application/json")
	r = withClaims(t, r, "admin", "ws-1")
	r = chiURLParam(r, "id", "v1")
	rr := httptest.NewRecorder()

	h.Patch(rr, r)

	if rr.Code != http.StatusOK {
		t.Fatalf("expected 200, got %d", rr.Code)
	}
	p := repo.lastUpdateVehicleParams
	if p.Registration == nil {
		t.Error("expected Registration to be set")
	}
	// Fields absent from the request body must remain nil — not empty string.
	if p.MakeModel != nil {
		t.Errorf("expected MakeModel to be nil, got %v", p.MakeModel)
	}
	if p.Active != nil {
		t.Errorf("expected Active to be nil, got %v", p.Active)
	}
	if p.VIN != nil {
		t.Errorf("expected VIN to be nil, got %v", p.VIN)
	}
}

func TestUpdateVehicle_EmptyObject(t *testing.T) {
	repo := &mockRepo{
		updateVehicleFn: func(_ string, _ vehicles.UpdateVehicleParams) (vehicles.VehicleWithCustomer, error) {
			return vehicles.VehicleWithCustomer{Vehicle: vehicles.Vehicle{ID: "v1", Registration: "UNCHANGED"}}, nil
		},
	}
	h := NewVehicleHandler(repo, testLog)

	r := httptest.NewRequest(http.MethodPatch, "/vehicles/v1", bytes.NewBufferString("{}"))
	r.Header.Set("Content-Type", "application/json")
	r = withClaims(t, r, "admin", "ws-1")
	r = chiURLParam(r, "id", "v1")
	rr := httptest.NewRecorder()

	h.Patch(rr, r)

	if rr.Code != http.StatusOK {
		t.Fatalf("expected 200 on empty object, got %d", rr.Code)
	}
}

// decodeData unwraps the { "data", "meta" } response envelope into out.
func decodeData(t *testing.T, body io.Reader, out any) {
	t.Helper()
	var env struct {
		Data json.RawMessage `json:"data"`
	}
	if err := json.NewDecoder(body).Decode(&env); err != nil {
		t.Fatalf("decode envelope: %v", err)
	}
	if err := json.Unmarshal(env.Data, out); err != nil {
		t.Fatalf("unmarshal data: %v", err)
	}
}
