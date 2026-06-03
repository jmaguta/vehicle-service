package vehicles

import (
	"context"
	"time"
)

// Customer represents a workshop customer / fleet operator.
type Customer struct {
	ID         string    `json:"id"`
	WorkshopID string    `json:"workshop_id"`
	Name       string    `json:"name"`
	Email      *string   `json:"email"`
	Phone      *string   `json:"phone"`
	Address    *string   `json:"address"`
	Notes      *string   `json:"notes"`
	Active     bool      `json:"active"`
	CreatedAt  time.Time `json:"created_at"`
	UpdatedAt  time.Time `json:"updated_at"`
	// DVSA operator fields
	OperatorLicenceNumber *string `json:"operator_licence_number"`
	TrafficArea           *string `json:"traffic_area"`
	CompanyRegNumber      *string `json:"company_reg_number"`
	LicenceType           *string `json:"licence_type"`
	TransportManagerName  *string `json:"transport_manager_name"`
	TransportManagerCPC   *string `json:"transport_manager_cpc"`
}

// Vehicle represents a fleet vehicle.
type Vehicle struct {
	ID           string    `json:"id"`
	WorkshopID   string    `json:"workshop_id"`
	CustomerID   string    `json:"customer_id"`
	Registration string    `json:"registration"`
	TruckNumber  *string   `json:"truck_number"`
	MakeModel    *string   `json:"make_model"`
	Mileage      *string   `json:"mileage"`
	Notes        *string   `json:"notes"`
	Active       bool      `json:"active"`
	CreatedAt    time.Time `json:"created_at"`
	UpdatedAt    time.Time `json:"updated_at"`
	// DVSA compliance fields
	VIN                     *string `json:"vin"`
	DVSAClass               *string `json:"dvsa_class"`
	GrossVehicleWeight      *string `json:"gross_vehicle_weight"`
	AxleCount               *string `json:"axle_count"`
	DateFirstRegistered     *string `json:"date_first_registered"`
	MOTExpiry               *string `json:"mot_expiry"`
	TachographCalDue        *string `json:"tachograph_cal_due"`
	InspectionIntervalWeeks *string `json:"inspection_interval_weeks"`
	LastPMIDate             *string `json:"last_pmi_date"`
	NextPMIDue              *string `json:"next_pmi_due"`
	HasAdBlueDPF            bool    `json:"has_adbluedpf"`
	VORStatus               bool    `json:"vor_status"`
}

// VehicleWithCustomer embeds Vehicle with the customer's name for convenience.
type VehicleWithCustomer struct {
	Vehicle
	CustomerName *string `json:"customer_name"`
}

type CreateCustomerParams struct {
	WorkshopID            string
	Name                  string
	Email                 string
	Phone                 string
	Address               string
	Notes                 string
	Active                bool
	OperatorLicenceNumber string
	TrafficArea           string
	CompanyRegNumber      string
	LicenceType           string
	TransportManagerName  string
	TransportManagerCPC   string
}

type UpdateCustomerParams struct {
	Name                  *string
	Email                 *string
	Phone                 *string
	Address               *string
	Notes                 *string
	Active                *bool
	OperatorLicenceNumber *string
	TrafficArea           *string
	CompanyRegNumber      *string
	LicenceType           *string
	TransportManagerName  *string
	TransportManagerCPC   *string
}

type CreateVehicleParams struct {
	WorkshopID              string
	CustomerID              string
	Registration            string
	TruckNumber             string
	MakeModel               string
	Mileage                 string
	Notes                   string
	Active                  bool
	VIN                     string
	DVSAClass               string
	GrossVehicleWeight      string
	AxleCount               string
	DateFirstRegistered     string
	MOTExpiry               string
	TachographCalDue        string
	InspectionIntervalWeeks string
	LastPMIDate             string
	NextPMIDue              string
	HasAdBlueDPF            bool
	VORStatus               bool
}

type UpdateVehicleParams struct {
	CustomerID              *string
	Registration            *string
	TruckNumber             *string
	MakeModel               *string
	Mileage                 *string
	Notes                   *string
	Active                  *bool
	VIN                     *string
	DVSAClass               *string
	GrossVehicleWeight      *string
	AxleCount               *string
	DateFirstRegistered     *string
	MOTExpiry               *string
	TachographCalDue        *string
	InspectionIntervalWeeks *string
	LastPMIDate             *string
	NextPMIDue              *string
	HasAdBlueDPF            *bool
	VORStatus               *bool
}

// Repository is the vehicle-service data access interface.
type Repository interface {
	ListCustomers(ctx context.Context, workshopID, query string, active *bool) ([]Customer, error)
	GetCustomer(ctx context.Context, workshopID, id string) (Customer, error)
	CreateCustomer(ctx context.Context, p CreateCustomerParams) (Customer, error)
	UpdateCustomer(ctx context.Context, workshopID, id string, p UpdateCustomerParams) (Customer, error)

	ListVehicles(ctx context.Context, workshopID, query, customerID string, active *bool) ([]VehicleWithCustomer, error)
	GetVehicle(ctx context.Context, workshopID, id string) (VehicleWithCustomer, error)
	CreateVehicle(ctx context.Context, p CreateVehicleParams) (VehicleWithCustomer, error)
	UpdateVehicle(ctx context.Context, workshopID, id string, p UpdateVehicleParams) (VehicleWithCustomer, error)
}
