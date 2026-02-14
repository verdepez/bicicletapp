package sqlite

import (
	"context"
	"database/sql"
	"fmt"
	"time"

	"bicicletapp/internal/domain"
	"bicicletapp/internal/repository"
)

// BicycleRepo implements repository.BicycleRepository
type BicycleRepo struct {
	db *DB
}

// NewBicycleRepo creates a new BicycleRepo
func NewBicycleRepo(db *DB) repository.BicycleRepository {
	return &BicycleRepo{db: db}
}

func (r *BicycleRepo) Create(ctx context.Context, bicycle *domain.Bicycle) error {
	query := `
		INSERT INTO bicycles (user_id, brand_id, model_id, color, serial_number, notes, created_at)
		VALUES (?, ?, ?, ?, ?, ?, ?)
	`
	now := time.Now()
	var brandID, modelID interface{}
	
	if bicycle.BrandID != 0 {
		brandID = bicycle.BrandID
	}
	if bicycle.ModelID != 0 {
		modelID = bicycle.ModelID
	}

	result, err := r.db.ExecContext(ctx, query,
		bicycle.UserID, brandID, modelID, bicycle.Color, bicycle.SerialNumber, bicycle.Notes, now)
	if err != nil {
		return fmt.Errorf("failed to create bicycle: %w", err)
	}

	id, err := result.LastInsertId()
	if err != nil {
		return fmt.Errorf("failed to get bicycle ID: %w", err)
	}
	bicycle.ID = id
	bicycle.CreatedAt = now

	return nil
}

func (r *BicycleRepo) GetByID(ctx context.Context, id int64) (*domain.Bicycle, error) {
	query := `
		SELECT b.id, b.user_id, b.brand_id, b.model_id, b.color, b.serial_number, b.notes, b.created_at,
			   br.name, m.name
		FROM bicycles b
		LEFT JOIN brands br ON b.brand_id = br.id
		LEFT JOIN models m ON b.model_id = m.id
		WHERE b.id = ?
	`
	bicycle := &domain.Bicycle{
		Brand: &domain.Brand{},
		Model: &domain.Model{},
	}
	
	var brandID, modelID sql.NullInt64
	var brandName, modelName sql.NullString
	
	err := r.db.QueryRowContext(ctx, query, id).Scan(
		&bicycle.ID, &bicycle.UserID, &brandID, &modelID, &bicycle.Color, &bicycle.SerialNumber, &bicycle.Notes, &bicycle.CreatedAt,
		&brandName, &modelName,
	)
	if err == sql.ErrNoRows {
		return nil, nil
	}
	if err != nil {
		return nil, fmt.Errorf("failed to get bicycle: %w", err)
	}

	if brandID.Valid {
		bicycle.BrandID = brandID.Int64
		bicycle.Brand.ID = brandID.Int64
		bicycle.Brand.Name = brandName.String
	}
	if modelID.Valid {
		bicycle.ModelID = modelID.Int64
		bicycle.Model.ID = modelID.Int64
		bicycle.Model.Name = modelName.String
		bicycle.Model.BrandID = bicycle.BrandID
	}

	return bicycle, nil
}

func (r *BicycleRepo) GetByUserID(ctx context.Context, userID int64) ([]domain.Bicycle, error) {
	query := `
		SELECT b.id, b.user_id, b.brand_id, b.model_id, b.color, b.serial_number, b.notes, b.created_at,
			   br.name, m.name
		FROM bicycles b
		LEFT JOIN brands br ON b.brand_id = br.id
		LEFT JOIN models m ON b.model_id = m.id
		WHERE b.user_id = ?
		ORDER BY b.created_at DESC
	`
	rows, err := r.db.QueryContext(ctx, query, userID)
	if err != nil {
		return nil, fmt.Errorf("failed to get bicycles by user: %w", err)
	}
	defer rows.Close()

	var bicycles []domain.Bicycle
	for rows.Next() {
		b := domain.Bicycle{
			Brand: &domain.Brand{},
			Model: &domain.Model{},
		}
		var brandID, modelID sql.NullInt64
		var brandName, modelName sql.NullString

		if err := rows.Scan(
			&b.ID, &b.UserID, &brandID, &modelID, &b.Color, &b.SerialNumber, &b.Notes, &b.CreatedAt,
			&brandName, &modelName,
		); err != nil {
			return nil, fmt.Errorf("failed to scan bicycle: %w", err)
		}

		if brandID.Valid {
			b.BrandID = brandID.Int64
			b.Brand.ID = brandID.Int64
			b.Brand.Name = brandName.String
		}
		if modelID.Valid {
			b.ModelID = modelID.Int64
			b.Model.ID = modelID.Int64
			b.Model.Name = modelName.String
			b.Model.BrandID = b.BrandID
		}
		
		bicycles = append(bicycles, b)
	}
	return bicycles, nil
}

func (r *BicycleRepo) Update(ctx context.Context, bicycle *domain.Bicycle) error {
	query := `
		UPDATE bicycles 
		SET brand_id = ?, model_id = ?, color = ?, serial_number = ?, notes = ?
		WHERE id = ?
	`
	var brandID, modelID interface{}
	if bicycle.BrandID != 0 {
		brandID = bicycle.BrandID
	}
	if bicycle.ModelID != 0 {
		modelID = bicycle.ModelID
	}

	_, err := r.db.ExecContext(ctx, query, 
		brandID, modelID, bicycle.Color, bicycle.SerialNumber, bicycle.Notes, bicycle.ID)
	if err != nil {
		return fmt.Errorf("failed to update bicycle: %w", err)
	}
	return nil
}

func (r *BicycleRepo) Delete(ctx context.Context, id int64) error {
	query := `DELETE FROM bicycles WHERE id = ?`
	_, err := r.db.ExecContext(ctx, query, id)
	if err != nil {
		return fmt.Errorf("failed to delete bicycle: %w", err)
	}
	return nil
}
