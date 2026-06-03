package handlers

import (
	"encoding/json"
	"log/slog"
	"net/http"
	"strings"

	"github.com/go-chi/chi/v5"

	"github.com/jmaguta/vehicle-service/internal/vehicles"
)

// CustomerHandler handles all /customers routes.
type CustomerHandler struct {
	repo vehicles.Repository
	log  *slog.Logger
}

func NewCustomerHandler(repo vehicles.Repository, log *slog.Logger) *CustomerHandler {
	return &CustomerHandler{repo: repo, log: log}
}

type customerRequest struct {
	Name    *string `json:"name,omitempty"`
	Email   *string `json:"email,omitempty"`
	Phone   *string `json:"phone,omitempty"`
	Address *string `json:"address,omitempty"`
	Notes   *string `json:"notes,omitempty"`
	Active  *bool   `json:"active,omitempty"`
	// DVSA operator fields
	OperatorLicenceNumber *string `json:"operator_licence_number,omitempty"`
	TrafficArea           *string `json:"traffic_area,omitempty"`
	CompanyRegNumber      *string `json:"company_reg_number,omitempty"`
	LicenceType           *string `json:"licence_type,omitempty"`
	TransportManagerName  *string `json:"transport_manager_name,omitempty"`
	TransportManagerCPC   *string `json:"transport_manager_cpc,omitempty"`
}

func (h *CustomerHandler) List(w http.ResponseWriter, r *http.Request) {
	active, err := parseOptionalBool(r.URL.Query().Get("active"))
	if err != nil {
		writeError(w, "invalid active filter", http.StatusBadRequest)
		return
	}

	customers, err := h.repo.ListCustomers(r.Context(), workshopIDFromClaims(r), strings.TrimSpace(r.URL.Query().Get("q")), active)
	if err != nil {
		h.log.Error("list customers", "error", err)
		writeError(w, "internal server error", http.StatusInternalServerError)
		return
	}

	if customers == nil {
		customers = []vehicles.Customer{}
	}
	writeJSON(w, http.StatusOK, customers)
}

func (h *CustomerHandler) Get(w http.ResponseWriter, r *http.Request) {
	customer, err := h.repo.GetCustomer(r.Context(), workshopIDFromClaims(r), chi.URLParam(r, "id"))
	if err != nil {
		writeError(w, "not found", http.StatusNotFound)
		return
	}
	writeJSON(w, http.StatusOK, customer)
}

func (h *CustomerHandler) Create(w http.ResponseWriter, r *http.Request) {
	var body customerRequest
	if err := json.NewDecoder(r.Body).Decode(&body); err != nil {
		writeError(w, "invalid request body", http.StatusBadRequest)
		return
	}
	if body.Name == nil || strings.TrimSpace(*body.Name) == "" {
		writeError(w, "name is required", http.StatusBadRequest)
		return
	}

	active := true
	if body.Active != nil {
		active = *body.Active
	}

	customer, err := h.repo.CreateCustomer(r.Context(), vehicles.CreateCustomerParams{
		WorkshopID:            workshopIDFromClaims(r),
		Name:                  strings.TrimSpace(*body.Name),
		Email:                 strings.TrimSpace(stringValue(body.Email)),
		Phone:                 strings.TrimSpace(stringValue(body.Phone)),
		Address:               strings.TrimSpace(stringValue(body.Address)),
		Notes:                 strings.TrimSpace(stringValue(body.Notes)),
		Active:                active,
		OperatorLicenceNumber: strings.TrimSpace(stringValue(body.OperatorLicenceNumber)),
		TrafficArea:           strings.TrimSpace(stringValue(body.TrafficArea)),
		CompanyRegNumber:      strings.TrimSpace(stringValue(body.CompanyRegNumber)),
		LicenceType:           strings.TrimSpace(stringValue(body.LicenceType)),
		TransportManagerName:  strings.TrimSpace(stringValue(body.TransportManagerName)),
		TransportManagerCPC:   strings.TrimSpace(stringValue(body.TransportManagerCPC)),
	})
	if err != nil {
		h.log.Error("create customer", "error", err)
		writeError(w, "internal server error", http.StatusInternalServerError)
		return
	}

	writeJSON(w, http.StatusCreated, customer)
}

func (h *CustomerHandler) Patch(w http.ResponseWriter, r *http.Request) {
	var body customerRequest
	if err := json.NewDecoder(r.Body).Decode(&body); err != nil {
		writeError(w, "invalid request body", http.StatusBadRequest)
		return
	}

	params := vehicles.UpdateCustomerParams{}
	if body.Name != nil {
		v := strings.TrimSpace(*body.Name)
		params.Name = &v
	}
	if body.Email != nil {
		v := strings.TrimSpace(*body.Email)
		params.Email = &v
	}
	if body.Phone != nil {
		v := strings.TrimSpace(*body.Phone)
		params.Phone = &v
	}
	if body.Address != nil {
		v := strings.TrimSpace(*body.Address)
		params.Address = &v
	}
	if body.Notes != nil {
		v := strings.TrimSpace(*body.Notes)
		params.Notes = &v
	}
	if body.Active != nil {
		params.Active = body.Active
	}
	if body.OperatorLicenceNumber != nil {
		v := strings.TrimSpace(*body.OperatorLicenceNumber)
		params.OperatorLicenceNumber = &v
	}
	if body.TrafficArea != nil {
		v := strings.TrimSpace(*body.TrafficArea)
		params.TrafficArea = &v
	}
	if body.CompanyRegNumber != nil {
		v := strings.TrimSpace(*body.CompanyRegNumber)
		params.CompanyRegNumber = &v
	}
	if body.LicenceType != nil {
		v := strings.TrimSpace(*body.LicenceType)
		params.LicenceType = &v
	}
	if body.TransportManagerName != nil {
		v := strings.TrimSpace(*body.TransportManagerName)
		params.TransportManagerName = &v
	}
	if body.TransportManagerCPC != nil {
		v := strings.TrimSpace(*body.TransportManagerCPC)
		params.TransportManagerCPC = &v
	}

	customer, err := h.repo.UpdateCustomer(r.Context(), workshopIDFromClaims(r), chi.URLParam(r, "id"), params)
	if err != nil {
		h.log.Error("update customer", "error", err)
		writeError(w, "not found", http.StatusNotFound)
		return
	}

	writeJSON(w, http.StatusOK, customer)
}
