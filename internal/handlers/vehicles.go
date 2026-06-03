package handlers

import (
	"encoding/json"
	"log/slog"
	"net/http"
	"strings"

	"github.com/go-chi/chi/v5"

	"github.com/jmaguta/vehicle-service/internal/vehicles"
)

// VehicleHandler handles all /vehicles routes.
type VehicleHandler struct {
	repo vehicles.Repository
	log  *slog.Logger
}

func NewVehicleHandler(repo vehicles.Repository, log *slog.Logger) *VehicleHandler {
	return &VehicleHandler{repo: repo, log: log}
}

type vehicleRequest struct {
	CustomerID   *string `json:"customer_id,omitempty"`
	Registration *string `json:"registration,omitempty"`
	TruckNumber  *string `json:"truck_number,omitempty"`
	MakeModel    *string `json:"make_model,omitempty"`
	Mileage      *string `json:"mileage,omitempty"`
	Notes        *string `json:"notes,omitempty"`
	Active       *bool   `json:"active,omitempty"`
	// DVSA compliance fields
	VIN                     *string `json:"vin,omitempty"`
	DVSAClass               *string `json:"dvsa_class,omitempty"`
	GrossVehicleWeight      *string `json:"gross_vehicle_weight,omitempty"`
	AxleCount               *string `json:"axle_count,omitempty"`
	DateFirstRegistered     *string `json:"date_first_registered,omitempty"`
	MOTExpiry               *string `json:"mot_expiry,omitempty"`
	TachographCalDue        *string `json:"tachograph_cal_due,omitempty"`
	InspectionIntervalWeeks *string `json:"inspection_interval_weeks,omitempty"`
	LastPMIDate             *string `json:"last_pmi_date,omitempty"`
	NextPMIDue              *string `json:"next_pmi_due,omitempty"`
	HasAdBlueDPF            *bool   `json:"has_adbluedpf,omitempty"`
	VORStatus               *bool   `json:"vor_status,omitempty"`
}

func (h *VehicleHandler) List(w http.ResponseWriter, r *http.Request) {
	active, err := parseOptionalBool(r.URL.Query().Get("active"))
	if err != nil {
		writeError(w, "invalid active filter", http.StatusBadRequest)
		return
	}

	vvs, err := h.repo.ListVehicles(
		r.Context(),
		workshopIDFromClaims(r),
		strings.TrimSpace(r.URL.Query().Get("q")),
		strings.TrimSpace(r.URL.Query().Get("customer_id")),
		active,
	)
	if err != nil {
		h.log.Error("list vehicles", "error", err)
		writeError(w, "internal server error", http.StatusInternalServerError)
		return
	}

	if vvs == nil {
		vvs = []vehicles.VehicleWithCustomer{}
	}
	writeJSON(w, http.StatusOK, vvs)
}

func (h *VehicleHandler) Get(w http.ResponseWriter, r *http.Request) {
	v, err := h.repo.GetVehicle(r.Context(), workshopIDFromClaims(r), chi.URLParam(r, "id"))
	if err != nil {
		writeError(w, "not found", http.StatusNotFound)
		return
	}
	writeJSON(w, http.StatusOK, v)
}

func (h *VehicleHandler) Create(w http.ResponseWriter, r *http.Request) {
	var body vehicleRequest
	if err := json.NewDecoder(r.Body).Decode(&body); err != nil {
		writeError(w, "invalid request body", http.StatusBadRequest)
		return
	}
	if body.CustomerID == nil || strings.TrimSpace(*body.CustomerID) == "" {
		writeError(w, "customer_id is required", http.StatusBadRequest)
		return
	}
	if body.Registration == nil || strings.TrimSpace(*body.Registration) == "" {
		writeError(w, "registration is required", http.StatusBadRequest)
		return
	}

	active := true
	if body.Active != nil {
		active = *body.Active
	}
	hasAdBlueDPF := false
	if body.HasAdBlueDPF != nil {
		hasAdBlueDPF = *body.HasAdBlueDPF
	}
	vorStatus := false
	if body.VORStatus != nil {
		vorStatus = *body.VORStatus
	}

	v, err := h.repo.CreateVehicle(r.Context(), vehicles.CreateVehicleParams{
		WorkshopID:              workshopIDFromClaims(r),
		CustomerID:              strings.TrimSpace(*body.CustomerID),
		Registration:            strings.ToUpper(strings.TrimSpace(*body.Registration)),
		TruckNumber:             strings.TrimSpace(stringValue(body.TruckNumber)),
		MakeModel:               strings.TrimSpace(stringValue(body.MakeModel)),
		Mileage:                 strings.TrimSpace(stringValue(body.Mileage)),
		Notes:                   strings.TrimSpace(stringValue(body.Notes)),
		Active:                  active,
		VIN:                     strings.TrimSpace(stringValue(body.VIN)),
		DVSAClass:               strings.TrimSpace(stringValue(body.DVSAClass)),
		GrossVehicleWeight:      strings.TrimSpace(stringValue(body.GrossVehicleWeight)),
		AxleCount:               strings.TrimSpace(stringValue(body.AxleCount)),
		DateFirstRegistered:     strings.TrimSpace(stringValue(body.DateFirstRegistered)),
		MOTExpiry:               strings.TrimSpace(stringValue(body.MOTExpiry)),
		TachographCalDue:        strings.TrimSpace(stringValue(body.TachographCalDue)),
		InspectionIntervalWeeks: strings.TrimSpace(stringValue(body.InspectionIntervalWeeks)),
		LastPMIDate:             strings.TrimSpace(stringValue(body.LastPMIDate)),
		NextPMIDue:              strings.TrimSpace(stringValue(body.NextPMIDue)),
		HasAdBlueDPF:            hasAdBlueDPF,
		VORStatus:               vorStatus,
	})
	if err != nil {
		h.log.Error("create vehicle", "error", err)
		writeError(w, "internal server error", http.StatusInternalServerError)
		return
	}

	writeJSON(w, http.StatusCreated, v)
}

func (h *VehicleHandler) Patch(w http.ResponseWriter, r *http.Request) {
	var body vehicleRequest
	if err := json.NewDecoder(r.Body).Decode(&body); err != nil {
		writeError(w, "invalid request body", http.StatusBadRequest)
		return
	}

	params := vehicles.UpdateVehicleParams{}
	if body.CustomerID != nil {
		v := strings.TrimSpace(*body.CustomerID)
		params.CustomerID = &v
	}
	if body.Registration != nil {
		v := strings.ToUpper(strings.TrimSpace(*body.Registration))
		params.Registration = &v
	}
	if body.TruckNumber != nil {
		v := strings.TrimSpace(*body.TruckNumber)
		params.TruckNumber = &v
	}
	if body.MakeModel != nil {
		v := strings.TrimSpace(*body.MakeModel)
		params.MakeModel = &v
	}
	if body.Mileage != nil {
		v := strings.TrimSpace(*body.Mileage)
		params.Mileage = &v
	}
	if body.Notes != nil {
		v := strings.TrimSpace(*body.Notes)
		params.Notes = &v
	}
	if body.Active != nil {
		params.Active = body.Active
	}
	if body.VIN != nil {
		v := strings.TrimSpace(*body.VIN)
		params.VIN = &v
	}
	if body.DVSAClass != nil {
		v := strings.TrimSpace(*body.DVSAClass)
		params.DVSAClass = &v
	}
	if body.GrossVehicleWeight != nil {
		v := strings.TrimSpace(*body.GrossVehicleWeight)
		params.GrossVehicleWeight = &v
	}
	if body.AxleCount != nil {
		v := strings.TrimSpace(*body.AxleCount)
		params.AxleCount = &v
	}
	if body.DateFirstRegistered != nil {
		v := strings.TrimSpace(*body.DateFirstRegistered)
		params.DateFirstRegistered = &v
	}
	if body.MOTExpiry != nil {
		v := strings.TrimSpace(*body.MOTExpiry)
		params.MOTExpiry = &v
	}
	if body.TachographCalDue != nil {
		v := strings.TrimSpace(*body.TachographCalDue)
		params.TachographCalDue = &v
	}
	if body.InspectionIntervalWeeks != nil {
		v := strings.TrimSpace(*body.InspectionIntervalWeeks)
		params.InspectionIntervalWeeks = &v
	}
	if body.LastPMIDate != nil {
		v := strings.TrimSpace(*body.LastPMIDate)
		params.LastPMIDate = &v
	}
	if body.NextPMIDue != nil {
		v := strings.TrimSpace(*body.NextPMIDue)
		params.NextPMIDue = &v
	}
	if body.HasAdBlueDPF != nil {
		params.HasAdBlueDPF = body.HasAdBlueDPF
	}
	if body.VORStatus != nil {
		params.VORStatus = body.VORStatus
	}

	v, err := h.repo.UpdateVehicle(r.Context(), workshopIDFromClaims(r), chi.URLParam(r, "id"), params)
	if err != nil {
		h.log.Error("update vehicle", "error", err)
		writeError(w, "not found", http.StatusNotFound)
		return
	}

	writeJSON(w, http.StatusOK, v)
}
