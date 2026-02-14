package sqlite

import (
	"context"
	"database/sql"
	"encoding/base64"
	"fmt"
	"time"

	"bicicletapp/internal/domain"
	"bicicletapp/internal/repository"
)

// TicketRepo implements repository.TicketRepository
type TicketRepo struct {
	db *DB
}

// NewTicketRepo creates a new TicketRepo
func NewTicketRepo(db *DB) repository.TicketRepository {
	return &TicketRepo{db: db}
}

func (r *TicketRepo) Create(ctx context.Context, ticket *domain.Ticket) error {
	query := `
		INSERT INTO tickets (booking_id, technician_id, tracking_code, qr_code, status, notes, created_at, updated_at)
		VALUES (?, ?, ?, ?, ?, ?, ?, ?)
	`
	now := time.Now()
	result, err := r.db.ExecContext(ctx, query,
		ticket.BookingID, ticket.TechnicianID, ticket.TrackingCode, ticket.QRCode,
		ticket.Status, ticket.Notes, now, now)
	if err != nil {
		return fmt.Errorf("failed to create ticket: %w", err)
	}

	id, err := result.LastInsertId()
	if err != nil {
		return fmt.Errorf("failed to get ticket ID: %w", err)
	}
	ticket.ID = id
	ticket.CreatedAt = now
	ticket.UpdatedAt = now

	// Create initial history record
	history := &domain.TicketStatusHistory{
		TicketID:  id,
		Status:    ticket.Status,
		ChangedBy: ticket.TechnicianID,
		Notes:     "Ticket creado",
		CreatedAt: now,
	}

	if err := r.CreateStatusHistory(ctx, history); err != nil {
		fmt.Printf("failed to create initial history record for ticket %d: %v\n", id, err)
	}

	return nil
}

func (r *TicketRepo) GetByID(ctx context.Context, id int64) (*domain.Ticket, error) {
	query := `
		SELECT t.id, t.booking_id, t.technician_id, t.tracking_code, t.qr_code, 
			   t.status, t.notes, t.created_at, t.updated_at,
			   u.id, u.name, u.email
		FROM tickets t
		LEFT JOIN users u ON t.technician_id = u.id
		WHERE t.id = ?
	`
	ticket := &domain.Ticket{
		Technician: &domain.User{},
	}

	var qrCode []byte
	var techID sql.NullInt64
	var techName, techEmail sql.NullString

	err := r.db.QueryRowContext(ctx, query, id).Scan(
		&ticket.ID, &ticket.BookingID, &ticket.TechnicianID, &ticket.TrackingCode, &qrCode,
		&ticket.Status, &ticket.Notes, &ticket.CreatedAt, &ticket.UpdatedAt,
		&techID, &techName, &techEmail,
	)
	if err == sql.ErrNoRows {
		return nil, nil
	}
	if err != nil {
		return nil, fmt.Errorf("failed to get ticket: %w", err)
	}

	ticket.QRCode = qrCode
	if len(qrCode) > 0 {
		ticket.QRCodeBase64 = base64.StdEncoding.EncodeToString(qrCode)
	}

	if techID.Valid {
		ticket.Technician.ID = techID.Int64
		ticket.Technician.Name = techName.String
		ticket.Technician.Email = techEmail.String
	}

	return ticket, nil
}

func (r *TicketRepo) GetByTrackingCode(ctx context.Context, code string) (*domain.Ticket, error) {
	query := `
		SELECT t.id, t.booking_id, t.technician_id, t.tracking_code, t.qr_code, 
			   t.status, t.notes, t.created_at, t.updated_at
		FROM tickets t
		WHERE t.tracking_code = ?
	`
	ticket := &domain.Ticket{}
	var qrCode []byte

	err := r.db.QueryRowContext(ctx, query, code).Scan(
		&ticket.ID, &ticket.BookingID, &ticket.TechnicianID, &ticket.TrackingCode, &qrCode,
		&ticket.Status, &ticket.Notes, &ticket.CreatedAt, &ticket.UpdatedAt,
	)
	if err == sql.ErrNoRows {
		return nil, nil
	}
	if err != nil {
		return nil, fmt.Errorf("failed to get ticket by tracking code: %w", err)
	}

	ticket.QRCode = qrCode
	if len(qrCode) > 0 {
		ticket.QRCodeBase64 = base64.StdEncoding.EncodeToString(qrCode)
	}

	return ticket, nil
}

func (r *TicketRepo) GetByTechnicianID(ctx context.Context, technicianID int64, status string, limit, offset int) ([]domain.Ticket, error) {
	var query string
	var args []interface{}

	if status != "" {
		query = `
			SELECT t.id, t.booking_id, t.technician_id, t.tracking_code, 
				   t.status, t.notes, t.created_at, t.updated_at
			FROM tickets t
			WHERE t.technician_id = ? AND t.status = ?
			ORDER BY t.updated_at DESC
			LIMIT ? OFFSET ?
		`
		args = []interface{}{technicianID, status, limit, offset}
	} else {
		query = `
			SELECT t.id, t.booking_id, t.technician_id, t.tracking_code, 
				   t.status, t.notes, t.created_at, t.updated_at
			FROM tickets t
			WHERE t.technician_id = ?
			ORDER BY t.updated_at DESC
			LIMIT ? OFFSET ?
		`
		args = []interface{}{technicianID, limit, offset}
	}

	rows, err := r.db.QueryContext(ctx, query, args...)
	if err != nil {
		return nil, fmt.Errorf("failed to get tickets by technician: %w", err)
	}
	defer rows.Close()

	return r.scanTicketsSimple(rows)
}

func (r *TicketRepo) Update(ctx context.Context, ticket *domain.Ticket) error {
	query := `
		UPDATE tickets 
		SET technician_id = ?, status = ?, notes = ?, updated_at = ?
		WHERE id = ?
	`
	ticket.UpdatedAt = time.Now()
	_, err := r.db.ExecContext(ctx, query,
		ticket.TechnicianID, ticket.Status, ticket.Notes, ticket.UpdatedAt, ticket.ID)
	if err != nil {
		return fmt.Errorf("failed to update ticket: %w", err)
	}
	return nil
}

func (r *TicketRepo) UpdateStatus(ctx context.Context, id int64, status string, changedBy int64, notes string) error {
	// Start a transaction if possible, but for now we'll do sequential operations
	// TODO: implement transaction support in DB wrapper

	query := `UPDATE tickets SET status = ?, updated_at = ? WHERE id = ?`
	now := time.Now()
	_, err := r.db.ExecContext(ctx, query, status, now, id)
	if err != nil {
		return fmt.Errorf("failed to update ticket status: %w", err)
	}

	// Create history record
	history := &domain.TicketStatusHistory{
		TicketID:  id,
		Status:    status,
		ChangedBy: changedBy,
		Notes:     notes,
		CreatedAt: now,
	}

	if err := r.CreateStatusHistory(ctx, history); err != nil {
		// Log error but don't fail the operation since status was updated
		// In a real app we would rollback transaction
		fmt.Printf("failed to create history record for ticket %d: %v\n", id, err)
	}

	return nil
}

func (r *TicketRepo) List(ctx context.Context, status string, limit, offset int) ([]domain.Ticket, error) {
	var query string
	var args []interface{}

	if status != "" {
		query = `
			SELECT t.id, t.booking_id, t.technician_id, t.tracking_code, 
				   t.status, t.notes, t.created_at, t.updated_at
			FROM tickets t
			WHERE t.status = ?
			ORDER BY t.updated_at DESC
			LIMIT ? OFFSET ?
		`
		args = []interface{}{status, limit, offset}
	} else {
		query = `
			SELECT t.id, t.booking_id, t.technician_id, t.tracking_code, 
				   t.status, t.notes, t.created_at, t.updated_at
			FROM tickets t
			ORDER BY t.updated_at DESC
			LIMIT ? OFFSET ?
		`
		args = []interface{}{limit, offset}
	}

	rows, err := r.db.QueryContext(ctx, query, args...)
	if err != nil {
		return nil, fmt.Errorf("failed to list tickets: %w", err)
	}
	defer rows.Close()

	return r.scanTicketsSimple(rows)
}

func (r *TicketRepo) CountByStatus(ctx context.Context) (map[string]int, error) {
	query := `SELECT status, COUNT(*) FROM tickets GROUP BY status`
	rows, err := r.db.QueryContext(ctx, query)
	if err != nil {
		return nil, fmt.Errorf("failed to count tickets by status: %w", err)
	}
	defer rows.Close()

	counts := make(map[string]int)
	for rows.Next() {
		var status string
		var count int
		if err := rows.Scan(&status, &count); err != nil {
			return nil, fmt.Errorf("failed to scan ticket count: %w", err)
		}
		counts[status] = count
	}
	return counts, nil
}

func (r *TicketRepo) scanTicketsSimple(rows *sql.Rows) ([]domain.Ticket, error) {
	var tickets []domain.Ticket
	for rows.Next() {
		var t domain.Ticket
		var techID sql.NullInt64
		var notes sql.NullString

		if err := rows.Scan(
			&t.ID, &t.BookingID, &techID, &t.TrackingCode,
			&t.Status, &notes, &t.CreatedAt, &t.UpdatedAt,
		); err != nil {
			return nil, fmt.Errorf("failed to scan ticket: %w", err)
		}

		if techID.Valid {
			t.TechnicianID = techID.Int64
		}
		if notes.Valid {
			t.Notes = notes.String
		}

		tickets = append(tickets, t)
	}
	return tickets, nil
}

func (r *TicketRepo) CreateStatusHistory(ctx context.Context, history *domain.TicketStatusHistory) error {
	query := `
		INSERT INTO ticket_status_history (ticket_id, status, changed_by, notes, created_at)
		VALUES (?, ?, ?, ?, ?)
	`
	var changedBy interface{}
	if history.ChangedBy != 0 {
		changedBy = history.ChangedBy
	} else {
		changedBy = nil
	}

	_, err := r.db.ExecContext(ctx, query,
		history.TicketID, history.Status, changedBy, history.Notes, time.Now())
	if err != nil {
		return fmt.Errorf("failed to create ticket status history: %w", err)
	}
	return nil
}

func (r *TicketRepo) GetStatusHistory(ctx context.Context, ticketID int64) ([]domain.TicketStatusHistory, error) {
	query := `
		SELECT h.id, h.ticket_id, h.status, h.changed_by, h.notes, h.created_at,
			   u.id, u.name
		FROM ticket_status_history h
		LEFT JOIN users u ON h.changed_by = u.id
		WHERE h.ticket_id = ?
		ORDER BY h.created_at ASC
	`
	rows, err := r.db.QueryContext(ctx, query, ticketID)
	if err != nil {
		return nil, fmt.Errorf("failed to get ticket status history: %w", err)
	}
	defer rows.Close()

	var history []domain.TicketStatusHistory
	for rows.Next() {
		var h domain.TicketStatusHistory
		var changedBy sql.NullInt64
		var userID sql.NullInt64
		var userName sql.NullString

		if err := rows.Scan(
			&h.ID, &h.TicketID, &h.Status, &changedBy, &h.Notes, &h.CreatedAt,
			&userID, &userName,
		); err != nil {
			fmt.Printf("DEBUG: GetStatusHistory Scan Error: %v\n", err)
			return nil, err
		}

		if changedBy.Valid {
			h.ChangedBy = changedBy.Int64
		}

		if userID.Valid {
			h.User = &domain.User{
				ID:   userID.Int64,
				Name: userName.String,
			}
		}
		history = append(history, h)
	}
	return history, nil
}

// CreateTicketPart creates a new ticket part
func (r *TicketRepo) CreateTicketPart(ctx context.Context, part *domain.TicketPart) error {
	query := `INSERT INTO ticket_parts (ticket_id, name, status, created_at) VALUES (?, ?, ?, ?)`
	now := time.Now()

	result, err := r.db.ExecContext(ctx, query, part.TicketID, part.Name, "pending", now)
	if err != nil {
		return fmt.Errorf("failed to create ticket part: %w", err)
	}

	id, err := result.LastInsertId()
	if err != nil {
		return fmt.Errorf("failed to get ticket part ID: %w", err)
	}

	part.ID = id
	part.Status = "pending"
	part.CreatedAt = now
	return nil
}

// GetTicketParts returns all parts for a ticket
func (r *TicketRepo) GetTicketParts(ctx context.Context, ticketID int64) ([]domain.TicketPart, error) {
	query := `SELECT id, ticket_id, name, status, created_at FROM ticket_parts WHERE ticket_id = ? ORDER BY created_at ASC`

	rows, err := r.db.QueryContext(ctx, query, ticketID)
	if err != nil {
		return nil, fmt.Errorf("failed to get ticket parts: %w", err)
	}
	defer rows.Close()

	var parts []domain.TicketPart
	for rows.Next() {
		var p domain.TicketPart
		if err := rows.Scan(&p.ID, &p.TicketID, &p.Name, &p.Status, &p.CreatedAt); err != nil {
			return nil, fmt.Errorf("failed to scan ticket part: %w", err)
		}
		parts = append(parts, p)
	}
	return parts, nil
}

// ToggleTicketPartStatus toggles the status of a ticket part
func (r *TicketRepo) ToggleTicketPartStatus(ctx context.Context, id int64) error {
	// First get current status
	var status string
	err := r.db.QueryRowContext(ctx, "SELECT status FROM ticket_parts WHERE id = ?", id).Scan(&status)
	if err != nil {
		return fmt.Errorf("failed to get ticket part status: %w", err)
	}

	newStatus := "done"
	if status == "done" {
		newStatus = "pending"
	}

	_, err = r.db.ExecContext(ctx, "UPDATE ticket_parts SET status = ? WHERE id = ?", newStatus, id)
	if err != nil {
		return fmt.Errorf("failed to update ticket part status: %w", err)
	}
	return nil
}

// DeleteTicketPart deletes a ticket part
func (r *TicketRepo) DeleteTicketPart(ctx context.Context, id int64) error {
	_, err := r.db.ExecContext(ctx, "DELETE FROM ticket_parts WHERE id = ?", id)
	if err != nil {
		return fmt.Errorf("failed to delete ticket part: %w", err)
	}
	return nil
}
