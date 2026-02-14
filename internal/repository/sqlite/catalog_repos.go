package sqlite

import (
	"context"
	"database/sql"
	"encoding/json"
	"fmt"
	"time"

	"bicicletapp/internal/domain"
	"bicicletapp/internal/repository"
)

// CatalogRepos provides brand, model, and service repositories
type CatalogRepos struct {
	db *DB
}

// NewCatalogRepos creates catalog repositories
func NewCatalogRepos(db *DB) (*BrandRepo, *ModelRepo, *ServiceRepo) {
	return &BrandRepo{db: db}, &ModelRepo{db: db}, &ServiceRepo{db: db}
}

// BrandRepo implements repository.BrandRepository
type BrandRepo struct {
	db *DB
}

func NewBrandRepo(db *DB) repository.BrandRepository {
	return &BrandRepo{db: db}
}

func (r *BrandRepo) Create(ctx context.Context, brand *domain.Brand) error {
	query := `INSERT INTO brands (name, logo_url) VALUES (?, ?)`
	result, err := r.db.ExecContext(ctx, query, brand.Name, brand.LogoURL)
	if err != nil {
		return fmt.Errorf("failed to create brand: %w", err)
	}
	id, _ := result.LastInsertId()
	brand.ID = id
	return nil
}

func (r *BrandRepo) GetByID(ctx context.Context, id int64) (*domain.Brand, error) {
	query := `SELECT id, name, logo_url FROM brands WHERE id = ?`
	brand := &domain.Brand{}
	var logoURL sql.NullString
	err := r.db.QueryRowContext(ctx, query, id).Scan(&brand.ID, &brand.Name, &logoURL)
	if err == sql.ErrNoRows {
		return nil, nil
	}
	if err != nil {
		return nil, fmt.Errorf("failed to get brand: %w", err)
	}
	brand.LogoURL = logoURL.String
	return brand, nil
}

func (r *BrandRepo) Update(ctx context.Context, brand *domain.Brand) error {
	query := `UPDATE brands SET name = ?, logo_url = ? WHERE id = ?`
	_, err := r.db.ExecContext(ctx, query, brand.Name, brand.LogoURL, brand.ID)
	return err
}

func (r *BrandRepo) Delete(ctx context.Context, id int64) error {
	query := `DELETE FROM brands WHERE id = ?`
	_, err := r.db.ExecContext(ctx, query, id)
	return err
}

func (r *BrandRepo) List(ctx context.Context) ([]domain.Brand, error) {
	query := `SELECT id, name, logo_url FROM brands ORDER BY name`
	rows, err := r.db.QueryContext(ctx, query)
	if err != nil {
		return nil, fmt.Errorf("failed to list brands: %w", err)
	}
	defer rows.Close()

	var brands []domain.Brand
	for rows.Next() {
		var b domain.Brand
		var logoURL sql.NullString
		if err := rows.Scan(&b.ID, &b.Name, &logoURL); err != nil {
			return nil, err
		}
		b.LogoURL = logoURL.String
		brands = append(brands, b)
	}
	return brands, nil
}

// ModelRepo implements repository.ModelRepository
type ModelRepo struct {
	db *DB
}

func NewModelRepo(db *DB) repository.ModelRepository {
	return &ModelRepo{db: db}
}

func (r *ModelRepo) Create(ctx context.Context, model *domain.Model) error {
	query := `INSERT INTO models (brand_id, name) VALUES (?, ?)`
	result, err := r.db.ExecContext(ctx, query, model.BrandID, model.Name)
	if err != nil {
		return fmt.Errorf("failed to create model: %w", err)
	}
	id, _ := result.LastInsertId()
	model.ID = id
	return nil
}

func (r *ModelRepo) GetByID(ctx context.Context, id int64) (*domain.Model, error) {
	query := `
		SELECT m.id, m.brand_id, m.name, b.id, b.name
		FROM models m
		LEFT JOIN brands b ON m.brand_id = b.id
		WHERE m.id = ?
	`
	model := &domain.Model{Brand: &domain.Brand{}}
	err := r.db.QueryRowContext(ctx, query, id).Scan(
		&model.ID, &model.BrandID, &model.Name,
		&model.Brand.ID, &model.Brand.Name,
	)
	if err == sql.ErrNoRows {
		return nil, nil
	}
	if err != nil {
		return nil, fmt.Errorf("failed to get model: %w", err)
	}
	return model, nil
}

func (r *ModelRepo) GetByBrandID(ctx context.Context, brandID int64) ([]domain.Model, error) {
	query := `SELECT id, brand_id, name FROM models WHERE brand_id = ? ORDER BY name`
	rows, err := r.db.QueryContext(ctx, query, brandID)
	if err != nil {
		return nil, fmt.Errorf("failed to get models by brand: %w", err)
	}
	defer rows.Close()

	var models []domain.Model
	for rows.Next() {
		var m domain.Model
		if err := rows.Scan(&m.ID, &m.BrandID, &m.Name); err != nil {
			return nil, err
		}
		models = append(models, m)
	}
	return models, nil
}

func (r *ModelRepo) Update(ctx context.Context, model *domain.Model) error {
	query := `UPDATE models SET brand_id = ?, name = ? WHERE id = ?`
	_, err := r.db.ExecContext(ctx, query, model.BrandID, model.Name, model.ID)
	return err
}

func (r *ModelRepo) Delete(ctx context.Context, id int64) error {
	query := `DELETE FROM models WHERE id = ?`
	_, err := r.db.ExecContext(ctx, query, id)
	return err
}

func (r *ModelRepo) List(ctx context.Context) ([]domain.Model, error) {
	query := `
		SELECT m.id, m.brand_id, m.name, b.id, b.name
		FROM models m
		LEFT JOIN brands b ON m.brand_id = b.id
		ORDER BY b.name, m.name
	`
	rows, err := r.db.QueryContext(ctx, query)
	if err != nil {
		return nil, fmt.Errorf("failed to list models: %w", err)
	}
	defer rows.Close()

	var models []domain.Model
	for rows.Next() {
		var m domain.Model
		m.Brand = &domain.Brand{}
		if err := rows.Scan(&m.ID, &m.BrandID, &m.Name, &m.Brand.ID, &m.Brand.Name); err != nil {
			return nil, err
		}
		models = append(models, m)
	}
	return models, nil
}

// ServiceRepo implements repository.ServiceRepository
type ServiceRepo struct {
	db *DB
}

func NewServiceRepo(db *DB) repository.ServiceRepository {
	return &ServiceRepo{db: db}
}

func (r *ServiceRepo) Create(ctx context.Context, service *domain.Service) error {
	query := `INSERT INTO services (name, description, base_price, estimated_hours) VALUES (?, ?, ?, ?)`
	result, err := r.db.ExecContext(ctx, query,
		service.Name, service.Description, service.BasePrice, service.EstimatedHours)
	if err != nil {
		return fmt.Errorf("failed to create service: %w", err)
	}
	id, _ := result.LastInsertId()
	service.ID = id
	return nil
}

func (r *ServiceRepo) GetByID(ctx context.Context, id int64) (*domain.Service, error) {
	query := `SELECT id, name, description, base_price, estimated_hours FROM services WHERE id = ?`
	service := &domain.Service{}
	err := r.db.QueryRowContext(ctx, query, id).Scan(
		&service.ID, &service.Name, &service.Description, &service.BasePrice, &service.EstimatedHours)
	if err == sql.ErrNoRows {
		return nil, nil
	}
	if err != nil {
		return nil, fmt.Errorf("failed to get service: %w", err)
	}
	return service, nil
}

func (r *ServiceRepo) Update(ctx context.Context, service *domain.Service) error {
	query := `UPDATE services SET name = ?, description = ?, base_price = ?, estimated_hours = ? WHERE id = ?`
	_, err := r.db.ExecContext(ctx, query,
		service.Name, service.Description, service.BasePrice, service.EstimatedHours, service.ID)
	return err
}

func (r *ServiceRepo) Delete(ctx context.Context, id int64) error {
	query := `DELETE FROM services WHERE id = ?`
	_, err := r.db.ExecContext(ctx, query, id)
	return err
}

func (r *ServiceRepo) List(ctx context.Context) ([]domain.Service, error) {
	query := `SELECT id, name, description, base_price, estimated_hours FROM services ORDER BY name`
	rows, err := r.db.QueryContext(ctx, query)
	if err != nil {
		return nil, fmt.Errorf("failed to list services: %w", err)
	}
	defer rows.Close()

	var services []domain.Service
	for rows.Next() {
		var s domain.Service
		if err := rows.Scan(&s.ID, &s.Name, &s.Description, &s.BasePrice, &s.EstimatedHours); err != nil {
			return nil, err
		}
		services = append(services, s)
	}
	return services, nil
}

// QuoteRepo implements repository.QuoteRepository
type QuoteRepo struct {
	db *DB
}

func NewQuoteRepo(db *DB) repository.QuoteRepository {
	return &QuoteRepo{db: db}
}

func (r *QuoteRepo) Create(ctx context.Context, quote *domain.Quote) error {
	itemsJSON, err := json.Marshal(quote.Items)
	if err != nil {
		return fmt.Errorf("failed to marshal quote items: %w", err)
	}

	query := `
		INSERT INTO quotes (booking_id, items_json, total, status, valid_until, created_at)
		VALUES (?, ?, ?, ?, ?, ?)
	`
	result, err := r.db.ExecContext(ctx, query,
		quote.BookingID, itemsJSON, quote.Total, quote.Status, quote.ValidUntil, time.Now())
	if err != nil {
		return fmt.Errorf("failed to create quote: %w", err)
	}
	id, _ := result.LastInsertId()
	quote.ID = id
	return nil
}

func (r *QuoteRepo) GetByID(ctx context.Context, id int64) (*domain.Quote, error) {
	query := `
		SELECT id, booking_id, items_json, total, status, rejection_reason, valid_until, created_at
		FROM quotes WHERE id = ?
	`
	quote := &domain.Quote{}
	var itemsJSON string
	var rejectionReason sql.NullString

	err := r.db.QueryRowContext(ctx, query, id).Scan(
		&quote.ID, &quote.BookingID, &itemsJSON, &quote.Total,
		&quote.Status, &rejectionReason, &quote.ValidUntil, &quote.CreatedAt)
	if err == sql.ErrNoRows {
		return nil, nil
	}
	if err != nil {
		return nil, fmt.Errorf("failed to get quote: %w", err)
	}

	if err := json.Unmarshal([]byte(itemsJSON), &quote.Items); err != nil {
		return nil, fmt.Errorf("failed to unmarshal quote items: %w", err)
	}
	quote.RejectionReason = rejectionReason.String

	return quote, nil
}

func (r *QuoteRepo) GetByBookingID(ctx context.Context, bookingID int64) (*domain.Quote, error) {
	query := `
		SELECT id, booking_id, items_json, total, status, rejection_reason, valid_until, created_at
		FROM quotes WHERE booking_id = ? ORDER BY created_at DESC LIMIT 1
	`
	quote := &domain.Quote{}
	var itemsJSON string
	var rejectionReason sql.NullString

	err := r.db.QueryRowContext(ctx, query, bookingID).Scan(
		&quote.ID, &quote.BookingID, &itemsJSON, &quote.Total,
		&quote.Status, &rejectionReason, &quote.ValidUntil, &quote.CreatedAt)
	if err == sql.ErrNoRows {
		return nil, nil
	}
	if err != nil {
		return nil, fmt.Errorf("failed to get quote by booking: %w", err)
	}

	if err := json.Unmarshal([]byte(itemsJSON), &quote.Items); err != nil {
		return nil, fmt.Errorf("failed to unmarshal quote items: %w", err)
	}
	quote.RejectionReason = rejectionReason.String

	return quote, nil
}

func (r *QuoteRepo) Update(ctx context.Context, quote *domain.Quote) error {
	itemsJSON, err := json.Marshal(quote.Items)
	if err != nil {
		return fmt.Errorf("failed to marshal quote items: %w", err)
	}

	query := `
		UPDATE quotes SET items_json = ?, total = ?, status = ?, rejection_reason = ?, valid_until = ?
		WHERE id = ?
	`
	_, err = r.db.ExecContext(ctx, query, itemsJSON, quote.Total, quote.Status,
		quote.RejectionReason, quote.ValidUntil, quote.ID)
	return err
}

func (r *QuoteRepo) Approve(ctx context.Context, id int64) error {
	query := `UPDATE quotes SET status = 'approved' WHERE id = ?`
	_, err := r.db.ExecContext(ctx, query, id)
	return err
}

func (r *QuoteRepo) Reject(ctx context.Context, id int64, reason string) error {
	query := `UPDATE quotes SET status = 'rejected', rejection_reason = ? WHERE id = ?`
	_, err := r.db.ExecContext(ctx, query, reason, id)
	return err
}

func (r *QuoteRepo) List(ctx context.Context, status string, limit, offset int) ([]domain.Quote, error) {
	var query string
	var args []interface{}

	if status != "" {
		query = `
			SELECT id, booking_id, items_json, total, status, rejection_reason, valid_until, created_at
			FROM quotes WHERE status = ? ORDER BY created_at DESC LIMIT ? OFFSET ?
		`
		args = []interface{}{status, limit, offset}
	} else {
		query = `
			SELECT id, booking_id, items_json, total, status, rejection_reason, valid_until, created_at
			FROM quotes ORDER BY created_at DESC LIMIT ? OFFSET ?
		`
		args = []interface{}{limit, offset}
	}

	rows, err := r.db.QueryContext(ctx, query, args...)
	if err != nil {
		return nil, fmt.Errorf("failed to list quotes: %w", err)
	}
	defer rows.Close()

	var quotes []domain.Quote
	for rows.Next() {
		var q domain.Quote
		var itemsJSON string
		var rejectionReason sql.NullString
		if err := rows.Scan(&q.ID, &q.BookingID, &itemsJSON, &q.Total,
			&q.Status, &rejectionReason, &q.ValidUntil, &q.CreatedAt); err != nil {
			return nil, err
		}
		if err := json.Unmarshal([]byte(itemsJSON), &q.Items); err != nil {
			return nil, err
		}
		q.RejectionReason = rejectionReason.String
		quotes = append(quotes, q)
	}
	return quotes, nil
}

// SurveyRepo implements repository.SurveyRepository
type SurveyRepo struct {
	db *DB
}

func NewSurveyRepo(db *DB) repository.SurveyRepository {
	return &SurveyRepo{db: db}
}

func (r *SurveyRepo) Create(ctx context.Context, survey *domain.Survey) error {
	query := `INSERT INTO surveys (ticket_id, rating, feedback, created_at) VALUES (?, ?, ?, ?)`
	result, err := r.db.ExecContext(ctx, query, survey.TicketID, survey.Rating, survey.Feedback, time.Now())
	if err != nil {
		return fmt.Errorf("failed to create survey: %w", err)
	}
	id, _ := result.LastInsertId()
	survey.ID = id
	return nil
}

func (r *SurveyRepo) GetByTicketID(ctx context.Context, ticketID int64) (*domain.Survey, error) {
	query := `SELECT id, ticket_id, rating, feedback, created_at FROM surveys WHERE ticket_id = ?`
	survey := &domain.Survey{}
	err := r.db.QueryRowContext(ctx, query, ticketID).Scan(
		&survey.ID, &survey.TicketID, &survey.Rating, &survey.Feedback, &survey.CreatedAt)
	if err == sql.ErrNoRows {
		return nil, nil
	}
	if err != nil {
		return nil, fmt.Errorf("failed to get survey: %w", err)
	}
	return survey, nil
}

func (r *SurveyRepo) GetAverageRating(ctx context.Context, fromDate time.Time) (float64, error) {
	query := `SELECT COALESCE(AVG(rating), 0) FROM surveys WHERE created_at >= ?`
	var avg float64
	err := r.db.QueryRowContext(ctx, query, fromDate).Scan(&avg)
	if err != nil {
		return 0, fmt.Errorf("failed to get average rating: %w", err)
	}
	return avg, nil
}

func (r *SurveyRepo) Count(ctx context.Context) (int, error) {
	query := `SELECT COUNT(*) FROM surveys`
	var count int
	err := r.db.QueryRowContext(ctx, query).Scan(&count)
	if err != nil {
		return 0, fmt.Errorf("failed to count surveys: %w", err)
	}
	return count, nil
}

func (r *SurveyRepo) GetRatingDistribution(ctx context.Context) (map[int]int, error) {
	query := `SELECT rating, COUNT(*) FROM surveys GROUP BY rating`
	rows, err := r.db.QueryContext(ctx, query)
	if err != nil {
		return nil, fmt.Errorf("failed to get rating distribution: %w", err)
	}
	defer rows.Close()

	dist := make(map[int]int)
	// Initialize all ratings to 0
	for i := 1; i <= 5; i++ {
		dist[i] = 0
	}

	for rows.Next() {
		var rating, count int
		if err := rows.Scan(&rating, &count); err != nil {
			return nil, err
		}
		dist[rating] = count
	}
	return dist, nil
}

func (r *SurveyRepo) List(ctx context.Context, limit, offset int) ([]domain.Survey, error) {
	query := `SELECT id, ticket_id, rating, feedback, created_at FROM surveys ORDER BY created_at DESC LIMIT ? OFFSET ?`
	rows, err := r.db.QueryContext(ctx, query, limit, offset)
	if err != nil {
		return nil, fmt.Errorf("failed to list surveys: %w", err)
	}
	defer rows.Close()

	var surveys []domain.Survey
	for rows.Next() {
		var s domain.Survey
		if err := rows.Scan(&s.ID, &s.TicketID, &s.Rating, &s.Feedback, &s.CreatedAt); err != nil {
			return nil, err
		}
		surveys = append(surveys, s)
	}
	return surveys, nil
}
