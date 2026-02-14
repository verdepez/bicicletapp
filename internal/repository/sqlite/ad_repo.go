package sqlite

import (
	"context"
	"database/sql"
	"fmt"

	"bicicletapp/internal/domain"
	"bicicletapp/internal/repository"
)

// AdRepo implements repository.AdRepository
type AdRepo struct {
	db *DB
}

func NewAdRepo(db *DB) repository.AdRepository {
	return &AdRepo{db: db}
}

func (r *AdRepo) Create(ctx context.Context, ad *domain.Ad) error {
	query := `INSERT INTO ads (title, media_url, media_type, link_url, active) VALUES (?, ?, ?, ?, ?)`
	result, err := r.db.ExecContext(ctx, query, ad.Title, ad.MediaURL, ad.MediaType, ad.LinkURL, ad.Active)
	if err != nil {
		return fmt.Errorf("failed to create ad: %w", err)
	}
	id, _ := result.LastInsertId()
	ad.ID = id
	return nil
}

func (r *AdRepo) GetByID(ctx context.Context, id int64) (*domain.Ad, error) {
	query := `SELECT id, title, media_url, media_type, link_url, active, impressions, clicks, created_at FROM ads WHERE id = ?`
	ad := &domain.Ad{}
	err := r.db.QueryRowContext(ctx, query, id).Scan(
		&ad.ID, &ad.Title, &ad.MediaURL, &ad.MediaType, &ad.LinkURL, &ad.Active, &ad.Impressions, &ad.Clicks, &ad.CreatedAt)
	if err == sql.ErrNoRows {
		return nil, nil
	}
	if err != nil {
		return nil, fmt.Errorf("failed to get ad: %w", err)
	}
	return ad, nil
}

func (r *AdRepo) GetRandomActive(ctx context.Context) (*domain.Ad, error) {
	// Select a random active ad
	query := `SELECT id, title, media_url, media_type, link_url, active, impressions, clicks, created_at FROM ads WHERE active = 1 ORDER BY RANDOM() LIMIT 1`
	ad := &domain.Ad{}
	err := r.db.QueryRowContext(ctx, query).Scan(
		&ad.ID, &ad.Title, &ad.MediaURL, &ad.MediaType, &ad.LinkURL, &ad.Active, &ad.Impressions, &ad.Clicks, &ad.CreatedAt)
	if err == sql.ErrNoRows {
		return nil, nil
	}
	if err != nil {
		return nil, fmt.Errorf("failed to get random ad: %w", err)
	}
	return ad, nil
}

func (r *AdRepo) Update(ctx context.Context, ad *domain.Ad) error {
	query := `UPDATE ads SET title = ?, media_url = ?, media_type = ?, link_url = ?, active = ? WHERE id = ?`
	_, err := r.db.ExecContext(ctx, query, ad.Title, ad.MediaURL, ad.MediaType, ad.LinkURL, ad.Active, ad.ID)
	return err
}

func (r *AdRepo) Delete(ctx context.Context, id int64) error {
	query := `DELETE FROM ads WHERE id = ?`
	_, err := r.db.ExecContext(ctx, query, id)
	return err
}

func (r *AdRepo) List(ctx context.Context) ([]domain.Ad, error) {
	query := `SELECT id, title, media_url, media_type, link_url, active, impressions, clicks, created_at FROM ads ORDER BY created_at DESC`
	rows, err := r.db.QueryContext(ctx, query)
	if err != nil {
		return nil, fmt.Errorf("failed to list ads: %w", err)
	}
	defer rows.Close()

	var ads []domain.Ad
	for rows.Next() {
		var a domain.Ad
		if err := rows.Scan(&a.ID, &a.Title, &a.MediaURL, &a.MediaType, &a.LinkURL, &a.Active, &a.Impressions, &a.Clicks, &a.CreatedAt); err != nil {
			return nil, err
		}
		ads = append(ads, a)
	}
	return ads, nil
}

func (r *AdRepo) IncrementImpressions(ctx context.Context, id int64) error {
	query := `UPDATE ads SET impressions = impressions + 1 WHERE id = ?`
	_, err := r.db.ExecContext(ctx, query, id)
	return err
}

func (r *AdRepo) IncrementClicks(ctx context.Context, id int64) error {
	query := `UPDATE ads SET clicks = clicks + 1 WHERE id = ?`
	_, err := r.db.ExecContext(ctx, query, id)
	return err
}
