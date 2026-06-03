package vehicles

import (
	"context"
	"fmt"
	"strings"

	"github.com/jackc/pgx/v5/pgtype"
	"github.com/jackc/pgx/v5/pgxpool"
)

// PostgresRepository implements Repository against the vehicle schema.
type PostgresRepository struct {
	db *pgxpool.Pool
}

func NewPostgresRepository(db *pgxpool.Pool) *PostgresRepository {
	return &PostgresRepository{db: db}
}

func nullText(s string) pgtype.Text {
	if s == "" {
		return pgtype.Text{}
	}
	return pgtype.Text{String: s, Valid: true}
}

func nullNumeric(s string) pgtype.Numeric {
	if s == "" {
		return pgtype.Numeric{}
	}
	var n pgtype.Numeric
	_ = n.Scan(s)
	return n
}

func ptextPtr(t pgtype.Text) *string {
	if !t.Valid {
		return nil
	}
	s := t.String
	return &s
}

func pnumericPtr(n pgtype.Numeric) *string {
	if !n.Valid {
		return nil
	}
	f, err := n.Float64Value()
	if err != nil || !f.Valid {
		return nil
	}
	s := fmt.Sprintf("%g", f.Float64)
	return &s
}

type scanner interface {
	Scan(dest ...any) error
}

// --- customers ---

func (r *PostgresRepository) ListCustomers(ctx context.Context, workshopID, query string, active *bool) ([]Customer, error) {
	where := []string{"1=1"}
	args := []any{}
	n := 1

	if workshopID != "" {
		where = append(where, fmt.Sprintf("workshop_id = $%d::uuid", n))
		args = append(args, workshopID)
		n++
	}
	if query != "" {
		where = append(where, fmt.Sprintf("(name ILIKE $%d OR COALESCE(email, '') ILIKE $%d OR COALESCE(phone, '') ILIKE $%d)", n, n, n))
		args = append(args, "%"+query+"%")
		n++
	}
	if active != nil {
		where = append(where, fmt.Sprintf("active = $%d", n))
		args = append(args, *active)
		n++
	}

	rows, err := r.db.Query(ctx, fmt.Sprintf(`
		SELECT id::text, workshop_id::text, name, email, phone, address, notes, active, created_at, updated_at,
		       operator_licence_number, traffic_area, company_reg_number,
		       licence_type, transport_manager_name, transport_manager_cpc
		FROM vehicle.customers
		WHERE %s
		ORDER BY name ASC, created_at DESC`, strings.Join(where, " AND ")), args...)
	if err != nil {
		return nil, fmt.Errorf("list customers: %w", err)
	}
	defer rows.Close()

	var customers []Customer
	for rows.Next() {
		c, err := scanCustomer(rows)
		if err != nil {
			return nil, err
		}
		customers = append(customers, c)
	}
	return customers, nil
}

func (r *PostgresRepository) GetCustomer(ctx context.Context, workshopID, id string) (Customer, error) {
	row := r.db.QueryRow(ctx, `
		SELECT id::text, workshop_id::text, name, email, phone, address, notes, active, created_at, updated_at,
		       operator_licence_number, traffic_area, company_reg_number,
		       licence_type, transport_manager_name, transport_manager_cpc
		FROM vehicle.customers
		WHERE id = $1::uuid
		  AND ($2 = '' OR workshop_id = $2::uuid)`, id, workshopID)
	c, err := scanCustomer(row)
	if err != nil {
		return Customer{}, fmt.Errorf("get customer: %w", err)
	}
	return c, nil
}

func (r *PostgresRepository) CreateCustomer(ctx context.Context, p CreateCustomerParams) (Customer, error) {
	var id string
	err := r.db.QueryRow(ctx, `
		INSERT INTO vehicle.customers (
			workshop_id,
			name, email, phone, address, notes, active,
			operator_licence_number, traffic_area, company_reg_number,
			licence_type, transport_manager_name, transport_manager_cpc
		) VALUES (
			COALESCE(NULLIF($1, ''), '11111111-1111-1111-1111-111111111111')::uuid,
			$2, NULLIF($3, ''), NULLIF($4, ''), NULLIF($5, ''), NULLIF($6, ''), $7,
			NULLIF($8, ''), NULLIF($9, ''), NULLIF($10, ''),
			NULLIF($11, ''), NULLIF($12, ''), NULLIF($13, '')
		) RETURNING id::text`,
		p.WorkshopID,
		p.Name, p.Email, p.Phone, p.Address, p.Notes, p.Active,
		p.OperatorLicenceNumber, p.TrafficArea, p.CompanyRegNumber,
		p.LicenceType, p.TransportManagerName, p.TransportManagerCPC,
	).Scan(&id)
	if err != nil {
		return Customer{}, fmt.Errorf("create customer: %w", err)
	}
	return r.GetCustomer(ctx, p.WorkshopID, id)
}

func (r *PostgresRepository) UpdateCustomer(ctx context.Context, workshopID, id string, p UpdateCustomerParams) (Customer, error) {
	setClauses := []string{}
	args := []any{}
	n := 1

	if p.Name != nil {
		setClauses = append(setClauses, fmt.Sprintf("name = $%d", n))
		args = append(args, *p.Name)
		n++
	}
	if p.Email != nil {
		setClauses = append(setClauses, fmt.Sprintf("email = NULLIF($%d, '')", n))
		args = append(args, *p.Email)
		n++
	}
	if p.Phone != nil {
		setClauses = append(setClauses, fmt.Sprintf("phone = NULLIF($%d, '')", n))
		args = append(args, *p.Phone)
		n++
	}
	if p.Address != nil {
		setClauses = append(setClauses, fmt.Sprintf("address = NULLIF($%d, '')", n))
		args = append(args, *p.Address)
		n++
	}
	if p.Notes != nil {
		setClauses = append(setClauses, fmt.Sprintf("notes = NULLIF($%d, '')", n))
		args = append(args, *p.Notes)
		n++
	}
	if p.Active != nil {
		setClauses = append(setClauses, fmt.Sprintf("active = $%d", n))
		args = append(args, *p.Active)
		n++
	}
	if p.OperatorLicenceNumber != nil {
		setClauses = append(setClauses, fmt.Sprintf("operator_licence_number = NULLIF($%d, '')", n))
		args = append(args, *p.OperatorLicenceNumber)
		n++
	}
	if p.TrafficArea != nil {
		setClauses = append(setClauses, fmt.Sprintf("traffic_area = NULLIF($%d, '')", n))
		args = append(args, *p.TrafficArea)
		n++
	}
	if p.CompanyRegNumber != nil {
		setClauses = append(setClauses, fmt.Sprintf("company_reg_number = NULLIF($%d, '')", n))
		args = append(args, *p.CompanyRegNumber)
		n++
	}
	if p.LicenceType != nil {
		setClauses = append(setClauses, fmt.Sprintf("licence_type = NULLIF($%d, '')", n))
		args = append(args, *p.LicenceType)
		n++
	}
	if p.TransportManagerName != nil {
		setClauses = append(setClauses, fmt.Sprintf("transport_manager_name = NULLIF($%d, '')", n))
		args = append(args, *p.TransportManagerName)
		n++
	}
	if p.TransportManagerCPC != nil {
		setClauses = append(setClauses, fmt.Sprintf("transport_manager_cpc = NULLIF($%d, '')", n))
		args = append(args, *p.TransportManagerCPC)
		n++
	}

	if len(setClauses) == 0 {
		return r.GetCustomer(ctx, workshopID, id)
	}

	setClauses = append(setClauses, "updated_at = now()")
	args = append(args, id, workshopID)
	query := fmt.Sprintf("UPDATE vehicle.customers SET %s WHERE id = $%d::uuid AND ($%d = '' OR workshop_id = $%d::uuid)",
		strings.Join(setClauses, ", "), n, n+1, n+1)
	if _, err := r.db.Exec(ctx, query, args...); err != nil {
		return Customer{}, fmt.Errorf("update customer: %w", err)
	}
	return r.GetCustomer(ctx, workshopID, id)
}

// --- vehicles ---

func (r *PostgresRepository) ListVehicles(ctx context.Context, workshopID, query, customerID string, active *bool) ([]VehicleWithCustomer, error) {
	where := []string{"1=1"}
	args := []any{}
	n := 1

	if workshopID != "" {
		where = append(where, fmt.Sprintf("v.workshop_id = $%d::uuid", n))
		args = append(args, workshopID)
		n++
	}
	if query != "" {
		where = append(where, fmt.Sprintf("(v.registration ILIKE $%d OR COALESCE(v.truck_number, '') ILIKE $%d OR COALESCE(v.make_model, '') ILIKE $%d)", n, n, n))
		args = append(args, "%"+query+"%")
		n++
	}
	if customerID != "" {
		where = append(where, fmt.Sprintf("v.customer_id = $%d::uuid", n))
		args = append(args, customerID)
		n++
	}
	if active != nil {
		where = append(where, fmt.Sprintf("v.active = $%d", n))
		args = append(args, *active)
		n++
	}

	rows, err := r.db.Query(ctx, fmt.Sprintf(`
		SELECT v.id::text, v.workshop_id::text, v.customer_id::text, v.registration,
		       v.truck_number, v.make_model, v.mileage, v.notes, v.active, v.created_at, v.updated_at,
		       v.vin, v.dvsa_class, v.gross_vehicle_weight,
		       v.axle_count::text, v.date_first_registered::text, v.mot_expiry::text,
		       v.tachograph_cal_due::text, v.inspection_interval_weeks::text,
		       v.last_pmi_date::text, v.next_pmi_due::text,
		       v.has_adbluedpf, v.vor_status,
		       c.name AS customer_name
		FROM vehicle.vehicles v
		LEFT JOIN vehicle.customers c ON c.id = v.customer_id
		WHERE %s
		ORDER BY v.registration ASC, v.created_at DESC`, strings.Join(where, " AND ")), args...)
	if err != nil {
		return nil, fmt.Errorf("list vehicles: %w", err)
	}
	defer rows.Close()

	var vehicles []VehicleWithCustomer
	for rows.Next() {
		v, err := scanVehicleWithCustomer(rows)
		if err != nil {
			return nil, err
		}
		vehicles = append(vehicles, v)
	}
	return vehicles, nil
}

func (r *PostgresRepository) GetVehicle(ctx context.Context, workshopID, id string) (VehicleWithCustomer, error) {
	row := r.db.QueryRow(ctx, `
		SELECT v.id::text, v.workshop_id::text, v.customer_id::text, v.registration,
		       v.truck_number, v.make_model, v.mileage, v.notes, v.active, v.created_at, v.updated_at,
		       v.vin, v.dvsa_class, v.gross_vehicle_weight,
		       v.axle_count::text, v.date_first_registered::text, v.mot_expiry::text,
		       v.tachograph_cal_due::text, v.inspection_interval_weeks::text,
		       v.last_pmi_date::text, v.next_pmi_due::text,
		       v.has_adbluedpf, v.vor_status,
		       c.name AS customer_name
		FROM vehicle.vehicles v
		LEFT JOIN vehicle.customers c ON c.id = v.customer_id
		WHERE v.id = $1::uuid
		  AND ($2 = '' OR v.workshop_id = $2::uuid)`, id, workshopID)
	v, err := scanVehicleWithCustomer(row)
	if err != nil {
		return VehicleWithCustomer{}, fmt.Errorf("get vehicle: %w", err)
	}
	return v, nil
}

func (r *PostgresRepository) CreateVehicle(ctx context.Context, p CreateVehicleParams) (VehicleWithCustomer, error) {
	var id string
	err := r.db.QueryRow(ctx, `
		INSERT INTO vehicle.vehicles (
			workshop_id,
			customer_id, registration, truck_number, make_model, mileage, notes, active,
			vin, dvsa_class, gross_vehicle_weight, axle_count,
			date_first_registered, mot_expiry, tachograph_cal_due, inspection_interval_weeks,
			last_pmi_date, next_pmi_due, has_adbluedpf, vor_status
		) VALUES (
			COALESCE(NULLIF($1, ''), '11111111-1111-1111-1111-111111111111')::uuid,
			$2::uuid, $3, NULLIF($4, ''), NULLIF($5, ''), NULLIF($6, '')::numeric, NULLIF($7, ''), $8,
			NULLIF($9, ''), NULLIF($10, ''), NULLIF($11, '')::numeric, NULLIF($12, '')::int,
			NULLIF($13, '')::date, NULLIF($14, '')::date, NULLIF($15, '')::date, NULLIF($16, '')::int,
			NULLIF($17, '')::date, NULLIF($18, '')::date, $19, $20
		) RETURNING id::text`,
		p.WorkshopID,
		p.CustomerID, p.Registration, p.TruckNumber, p.MakeModel, p.Mileage, p.Notes, p.Active,
		p.VIN, p.DVSAClass, p.GrossVehicleWeight, p.AxleCount,
		p.DateFirstRegistered, p.MOTExpiry, p.TachographCalDue, p.InspectionIntervalWeeks,
		p.LastPMIDate, p.NextPMIDue, p.HasAdBlueDPF, p.VORStatus,
	).Scan(&id)
	if err != nil {
		return VehicleWithCustomer{}, fmt.Errorf("create vehicle: %w", err)
	}
	return r.GetVehicle(ctx, p.WorkshopID, id)
}

func (r *PostgresRepository) UpdateVehicle(ctx context.Context, workshopID, id string, p UpdateVehicleParams) (VehicleWithCustomer, error) {
	setClauses := []string{}
	args := []any{}
	n := 1

	if p.CustomerID != nil {
		setClauses = append(setClauses, fmt.Sprintf("customer_id = $%d::uuid", n))
		args = append(args, *p.CustomerID)
		n++
	}
	if p.Registration != nil {
		setClauses = append(setClauses, fmt.Sprintf("registration = $%d", n))
		args = append(args, *p.Registration)
		n++
	}
	if p.TruckNumber != nil {
		setClauses = append(setClauses, fmt.Sprintf("truck_number = NULLIF($%d, '')", n))
		args = append(args, *p.TruckNumber)
		n++
	}
	if p.MakeModel != nil {
		setClauses = append(setClauses, fmt.Sprintf("make_model = NULLIF($%d, '')", n))
		args = append(args, *p.MakeModel)
		n++
	}
	if p.Mileage != nil {
		setClauses = append(setClauses, fmt.Sprintf("mileage = NULLIF($%d, '')::numeric", n))
		args = append(args, *p.Mileage)
		n++
	}
	if p.Notes != nil {
		setClauses = append(setClauses, fmt.Sprintf("notes = NULLIF($%d, '')", n))
		args = append(args, *p.Notes)
		n++
	}
	if p.Active != nil {
		setClauses = append(setClauses, fmt.Sprintf("active = $%d", n))
		args = append(args, *p.Active)
		n++
	}
	if p.VIN != nil {
		setClauses = append(setClauses, fmt.Sprintf("vin = NULLIF($%d, '')", n))
		args = append(args, *p.VIN)
		n++
	}
	if p.DVSAClass != nil {
		setClauses = append(setClauses, fmt.Sprintf("dvsa_class = NULLIF($%d, '')", n))
		args = append(args, *p.DVSAClass)
		n++
	}
	if p.GrossVehicleWeight != nil {
		setClauses = append(setClauses, fmt.Sprintf("gross_vehicle_weight = NULLIF($%d, '')::numeric", n))
		args = append(args, *p.GrossVehicleWeight)
		n++
	}
	if p.AxleCount != nil {
		setClauses = append(setClauses, fmt.Sprintf("axle_count = NULLIF($%d, '')::int", n))
		args = append(args, *p.AxleCount)
		n++
	}
	if p.DateFirstRegistered != nil {
		setClauses = append(setClauses, fmt.Sprintf("date_first_registered = NULLIF($%d, '')::date", n))
		args = append(args, *p.DateFirstRegistered)
		n++
	}
	if p.MOTExpiry != nil {
		setClauses = append(setClauses, fmt.Sprintf("mot_expiry = NULLIF($%d, '')::date", n))
		args = append(args, *p.MOTExpiry)
		n++
	}
	if p.TachographCalDue != nil {
		setClauses = append(setClauses, fmt.Sprintf("tachograph_cal_due = NULLIF($%d, '')::date", n))
		args = append(args, *p.TachographCalDue)
		n++
	}
	if p.InspectionIntervalWeeks != nil {
		setClauses = append(setClauses, fmt.Sprintf("inspection_interval_weeks = NULLIF($%d, '')::int", n))
		args = append(args, *p.InspectionIntervalWeeks)
		n++
	}
	if p.LastPMIDate != nil {
		setClauses = append(setClauses, fmt.Sprintf("last_pmi_date = NULLIF($%d, '')::date", n))
		args = append(args, *p.LastPMIDate)
		n++
	}
	if p.NextPMIDue != nil {
		setClauses = append(setClauses, fmt.Sprintf("next_pmi_due = NULLIF($%d, '')::date", n))
		args = append(args, *p.NextPMIDue)
		n++
	}
	if p.HasAdBlueDPF != nil {
		setClauses = append(setClauses, fmt.Sprintf("has_adbluedpf = $%d", n))
		args = append(args, *p.HasAdBlueDPF)
		n++
	}
	if p.VORStatus != nil {
		setClauses = append(setClauses, fmt.Sprintf("vor_status = $%d", n))
		args = append(args, *p.VORStatus)
		n++
	}

	if len(setClauses) == 0 {
		return r.GetVehicle(ctx, workshopID, id)
	}

	setClauses = append(setClauses, "updated_at = now()")
	args = append(args, id, workshopID)
	query := fmt.Sprintf("UPDATE vehicle.vehicles SET %s WHERE id = $%d::uuid AND ($%d = '' OR workshop_id = $%d::uuid)",
		strings.Join(setClauses, ", "), n, n+1, n+1)
	if _, err := r.db.Exec(ctx, query, args...); err != nil {
		return VehicleWithCustomer{}, fmt.Errorf("update vehicle: %w", err)
	}
	return r.GetVehicle(ctx, workshopID, id)
}

// --- scanners ---

func scanCustomer(s scanner) (Customer, error) {
	var (
		c                                                            Customer
		email, phone, address, notes                                pgtype.Text
		operatorLicenceNumber, trafficArea, companyRegNumber        pgtype.Text
		licenceType, transportManagerName, transportManagerCPC      pgtype.Text
	)
	if err := s.Scan(
		&c.ID, &c.WorkshopID, &c.Name,
		&email, &phone, &address, &notes,
		&c.Active, &c.CreatedAt, &c.UpdatedAt,
		&operatorLicenceNumber, &trafficArea, &companyRegNumber,
		&licenceType, &transportManagerName, &transportManagerCPC,
	); err != nil {
		return Customer{}, fmt.Errorf("scan customer: %w", err)
	}
	c.Email = ptextPtr(email)
	c.Phone = ptextPtr(phone)
	c.Address = ptextPtr(address)
	c.Notes = ptextPtr(notes)
	c.OperatorLicenceNumber = ptextPtr(operatorLicenceNumber)
	c.TrafficArea = ptextPtr(trafficArea)
	c.CompanyRegNumber = ptextPtr(companyRegNumber)
	c.LicenceType = ptextPtr(licenceType)
	c.TransportManagerName = ptextPtr(transportManagerName)
	c.TransportManagerCPC = ptextPtr(transportManagerCPC)
	return c, nil
}

func scanVehicleWithCustomer(s scanner) (VehicleWithCustomer, error) {
	var (
		v            VehicleWithCustomer
		truckNumber  pgtype.Text
		makeModel    pgtype.Text
		mileage      pgtype.Numeric
		notes        pgtype.Text
		vin          pgtype.Text
		dvsaClass    pgtype.Text
		gvw          pgtype.Numeric
		axleCount    pgtype.Text
		dateFirstReg pgtype.Text
		motExpiry    pgtype.Text
		tachCalDue   pgtype.Text
		inspInterval pgtype.Text
		lastPMI      pgtype.Text
		nextPMI      pgtype.Text
		customerName pgtype.Text
	)
	if err := s.Scan(
		&v.ID, &v.WorkshopID, &v.CustomerID, &v.Registration,
		&truckNumber, &makeModel, &mileage, &notes,
		&v.Active, &v.CreatedAt, &v.UpdatedAt,
		&vin, &dvsaClass, &gvw,
		&axleCount, &dateFirstReg, &motExpiry,
		&tachCalDue, &inspInterval,
		&lastPMI, &nextPMI,
		&v.HasAdBlueDPF, &v.VORStatus,
		&customerName,
	); err != nil {
		return VehicleWithCustomer{}, fmt.Errorf("scan vehicle: %w", err)
	}
	v.TruckNumber = ptextPtr(truckNumber)
	v.MakeModel = ptextPtr(makeModel)
	v.Mileage = pnumericPtr(mileage)
	v.Notes = ptextPtr(notes)
	v.VIN = ptextPtr(vin)
	v.DVSAClass = ptextPtr(dvsaClass)
	v.GrossVehicleWeight = pnumericPtr(gvw)
	v.AxleCount = ptextPtr(axleCount)
	v.DateFirstRegistered = ptextPtr(dateFirstReg)
	v.MOTExpiry = ptextPtr(motExpiry)
	v.TachographCalDue = ptextPtr(tachCalDue)
	v.InspectionIntervalWeeks = ptextPtr(inspInterval)
	v.LastPMIDate = ptextPtr(lastPMI)
	v.NextPMIDue = ptextPtr(nextPMI)
	v.CustomerName = ptextPtr(customerName)
	return v, nil
}

// suppress unused warning for nullText/nullNumeric (used in future or tests)
var _ = nullText
var _ = nullNumeric
