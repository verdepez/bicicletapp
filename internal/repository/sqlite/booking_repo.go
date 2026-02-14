package sqlite

import (
	"context"
	"database/sql"
	"fmt"
	"time"

	"bicicletapp/internal/domain"
	"bicicletapp/internal/repository"
)

// BookingRepo implements repository.BookingRepository
type BookingRepo struct {
	db *DB
}

// NewBookingRepo creates a new BookingRepo
func NewBookingRepo(db *DB) repository.BookingRepository {
	return &BookingRepo{db: db}
}

func (r *BookingRepo) Create(ctx context.Context, booking *domain.Booking) error {
	query := `
		INSERT INTO bookings (customer_id, bicycle_id, service_id, scheduled_at, status, notes, created_at)
		VALUES (?, ?, ?, ?, ?, ?, ?)
	`
	var bicycleID interface{}
	if booking.BicycleID != 0 {
		bicycleID = booking.BicycleID
	}

	result, err := r.db.ExecContext(ctx, query,
		booking.CustomerID, bicycleID, booking.ServiceID, booking.ScheduledAt, booking.Status, booking.Notes, time.Now())
	if err != nil {
		return fmt.Errorf("failed to create booking: %w", err)
	}

	id, err := result.LastInsertId()
	if err != nil {
		return fmt.Errorf("failed to get booking ID: %w", err)
	}
	booking.ID = id
	return nil
}

func (r *BookingRepo) GetByID(ctx context.Context, id int64) (*domain.Booking, error) {
	query := `
		SELECT b.id, b.customer_id, b.bicycle_id, b.service_id, b.scheduled_at, b.status, b.notes, b.created_at,
			   u.id, u.email, u.name, u.phone, u.role,
			   s.id, s.name, s.description, s.base_price, s.estimated_hours
		FROM bookings b
		LEFT JOIN users u ON b.customer_id = u.id
		LEFT JOIN services s ON b.service_id = s.id
		WHERE b.id = ?
	`
	booking := &domain.Booking{
		Customer: &domain.User{},
		Service:  &domain.Service{},
	}
	
	var bicycleID sql.NullInt64
	var serviceID sql.NullInt64
	var serviceName, serviceDesc sql.NullString
	var servicePrice, serviceHours sql.NullFloat64
	
	err := r.db.QueryRowContext(ctx, query, id).Scan(
		&booking.ID, &booking.CustomerID, &bicycleID, &booking.ServiceID, &booking.ScheduledAt, 
		&booking.Status, &booking.Notes, &booking.CreatedAt,
		&booking.Customer.ID, &booking.Customer.Email, &booking.Customer.Name, 
		&booking.Customer.Phone, &booking.Customer.Role,
		&serviceID, &serviceName, &serviceDesc, &servicePrice, &serviceHours,
	)
	if err == sql.ErrNoRows {
		return nil, nil
	}
	if err != nil {
		return nil, fmt.Errorf("failed to get booking: %w", err)
	}
	
	if bicycleID.Valid {
		booking.BicycleID = bicycleID.Int64
	}
	
	if serviceID.Valid {
		booking.Service.ID = serviceID.Int64
		booking.Service.Name = serviceName.String
		booking.Service.Description = serviceDesc.String
		booking.Service.BasePrice = servicePrice.Float64
		booking.Service.EstimatedHours = serviceHours.Float64
	}
	
	return booking, nil
}

func (r *BookingRepo) GetByCustomerID(ctx context.Context, customerID int64, limit, offset int) ([]domain.Booking, error) {
	query := `
		SELECT b.id, b.customer_id, b.bicycle_id, b.service_id, b.scheduled_at, b.status, b.notes, b.created_at,
			   s.name
		FROM bookings b
		LEFT JOIN services s ON b.service_id = s.id
		WHERE b.customer_id = ?
		ORDER BY b.scheduled_at DESC
		LIMIT ? OFFSET ?
	`
	rows, err := r.db.QueryContext(ctx, query, customerID, limit, offset)
	if err != nil {
		return nil, fmt.Errorf("failed to get bookings by customer: %w", err)
	}
	defer rows.Close()

	return r.scanBookings(rows)
}

func (r *BookingRepo) GetByDateRange(ctx context.Context, start, end time.Time) ([]domain.Booking, error) {
	query := `
		SELECT b.id, b.customer_id, b.bicycle_id, b.service_id, b.scheduled_at, b.status, b.notes, b.created_at,
			   s.name
		FROM bookings b
		LEFT JOIN services s ON b.service_id = s.id
		WHERE b.scheduled_at BETWEEN ? AND ?
		ORDER BY b.scheduled_at
	`
	rows, err := r.db.QueryContext(ctx, query, start, end)
	if err != nil {
		return nil, fmt.Errorf("failed to get bookings by date range: %w", err)
	}
	defer rows.Close()

	return r.scanBookings(rows)
}

func (r *BookingRepo) Update(ctx context.Context, booking *domain.Booking) error {
	query := `
		UPDATE bookings 
		SET bicycle_id = ?, service_id = ?, scheduled_at = ?, status = ?, notes = ?
		WHERE id = ?
	`
	var bicycleID interface{}
	if booking.BicycleID != 0 {
		bicycleID = booking.BicycleID
	}

	_, err := r.db.ExecContext(ctx, query, 
		bicycleID, booking.ServiceID, booking.ScheduledAt, booking.Status, booking.Notes, booking.ID)
	if err != nil {
		return fmt.Errorf("failed to update booking: %w", err)
	}
	return nil
}

func (r *BookingRepo) UpdateStatus(ctx context.Context, id int64, status string) error {
	query := `UPDATE bookings SET status = ? WHERE id = ?`
	_, err := r.db.ExecContext(ctx, query, status, id)
	if err != nil {
		return fmt.Errorf("failed to update booking status: %w", err)
	}
	return nil
}

func (r *BookingRepo) Delete(ctx context.Context, id int64) error {
	query := `DELETE FROM bookings WHERE id = ?`
	_, err := r.db.ExecContext(ctx, query, id)
	if err != nil {
		return fmt.Errorf("failed to delete booking: %w", err)
	}
	return nil
}

func (r *BookingRepo) List(ctx context.Context, status string, limit, offset int) ([]domain.Booking, error) {
	var query string
	var args []interface{}

	if status != "" {
		query = `
			SELECT b.id, b.customer_id, b.bicycle_id, b.service_id, b.scheduled_at, b.status, b.notes, b.created_at,
				   s.name
			FROM bookings b
			LEFT JOIN services s ON b.service_id = s.id
			WHERE b.status = ?
			ORDER BY b.scheduled_at DESC
			LIMIT ? OFFSET ?
		`
		args = []interface{}{status, limit, offset}
	} else {
		query = `
			SELECT b.id, b.customer_id, b.bicycle_id, b.service_id, b.scheduled_at, b.status, b.notes, b.created_at,
				   s.name
			FROM bookings b
			LEFT JOIN services s ON b.service_id = s.id
			ORDER BY b.scheduled_at DESC
			LIMIT ? OFFSET ?
		`
		args = []interface{}{limit, offset}
	}

	rows, err := r.db.QueryContext(ctx, query, args...)
	if err != nil {
		return nil, fmt.Errorf("failed to list bookings: %w", err)
	}
	defer rows.Close()

	return r.scanBookings(rows)
}

func (r *BookingRepo) CountByStatus(ctx context.Context, status string) (int, error) {
	var query string
	var args []interface{}

	if status != "" {
		query = `SELECT COUNT(*) FROM bookings WHERE status = ?`
		args = []interface{}{status}
	} else {
		query = `SELECT COUNT(*) FROM bookings`
	}

	var count int
	err := r.db.QueryRowContext(ctx, query, args...).Scan(&count)
	if err != nil {
		return 0, fmt.Errorf("failed to count bookings: %w", err)
	}
	return count, nil
}

func (r *BookingRepo) scanBookings(rows *sql.Rows) ([]domain.Booking, error) {
	var bookings []domain.Booking
	for rows.Next() {
		var b domain.Booking
		var bicycleID sql.NullInt64
		var serviceName sql.NullString
		if err := rows.Scan(
			&b.ID, &b.CustomerID, &bicycleID, &b.ServiceID, &b.ScheduledAt, 
			&b.Status, &b.Notes, &b.CreatedAt, &serviceName,
		); err != nil {
			return nil, fmt.Errorf("failed to scan booking: %w", err)
		}
		if bicycleID.Valid {
			b.BicycleID = bicycleID.Int64
		}
		if serviceName.Valid {
			b.Service = &domain.Service{Name: serviceName.String}
		}
		bookings = append(bookings, b)
	}
	return bookings, nil
}
