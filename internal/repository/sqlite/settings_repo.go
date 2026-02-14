package sqlite

import (
	"context"
	"database/sql"
	"fmt"
)

// SettingsRepo implements repository.SettingsRepository
type SettingsRepo struct {
	db *DB
}

// NewSettingsRepo creates a new SettingsRepo
func NewSettingsRepo(db *DB) *SettingsRepo {
	return &SettingsRepo{db: db}
}

// Get retrieves a setting value by key
func (r *SettingsRepo) Get(ctx context.Context, key string) (string, error) {
	query := `SELECT value FROM settings WHERE key = ?`
	var value string
	err := r.db.QueryRowContext(ctx, query, key).Scan(&value)
	if err == sql.ErrNoRows {
		return "", nil // Return empty string if not found, not an error
	}
	if err != nil {
		return "", fmt.Errorf("failed to get setting %s: %w", key, err)
	}
	return value, nil
}

// Set updates or creates a setting value
func (r *SettingsRepo) Set(ctx context.Context, key, value string) error {
	query := `INSERT INTO settings (key, value, updated_at) VALUES (?, ?, CURRENT_TIMESTAMP)
			  ON CONFLICT(key) DO UPDATE SET value = excluded.value, updated_at = CURRENT_TIMESTAMP`
	_, err := r.db.ExecContext(ctx, query, key, value)
	if err != nil {
		return fmt.Errorf("failed to set setting %s: %w", key, err)
	}
	return nil
}
