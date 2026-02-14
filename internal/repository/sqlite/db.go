// Package sqlite provides SQLite implementation of repository interfaces
package sqlite

import (
	"database/sql"
	"fmt"
	"os"
	"path/filepath"
	"strings"

	_ "modernc.org/sqlite"
)

// DB wraps the sql.DB with SQLite-specific optimizations
type DB struct {
	*sql.DB
}

// New creates a new SQLite database connection with optimizations for shared hosting
func New(dbPath string) (*DB, error) {
	// Validate and clean the path to prevent path traversal
	cleanPath := filepath.Clean(dbPath)

	// Check if path tries to escape current directory
	if !filepath.IsLocal(cleanPath) && !filepath.IsAbs(cleanPath) {
		return nil, fmt.Errorf("invalid database path: potential path traversal detected")
	}

	// Ensure the directory exists
	dir := filepath.Dir(cleanPath)
	if err := os.MkdirAll(dir, 0755); err != nil {
		return nil, fmt.Errorf("failed to create database directory: %w", err)
	}

	// Open SQLite with optimized settings for shared hosting
	// WAL mode for better concurrent read performance
	// busy_timeout to handle lock contention gracefully
	dsn := fmt.Sprintf("%s?_pragma=journal_mode(WAL)&_pragma=busy_timeout(5000)&_pragma=synchronous(NORMAL)&_pragma=cache_size(2000)", cleanPath)

	db, err := sql.Open("sqlite", dsn)
	if err != nil {
		return nil, fmt.Errorf("failed to open database: %w", err)
	}

	// Optimize connection pool for memory-constrained environments
	db.SetMaxOpenConns(1)    // Single connection to minimize memory
	db.SetMaxIdleConns(1)    // Keep one connection ready
	db.SetConnMaxLifetime(0) // Connections don't expire

	// Verify the connection
	if err := db.Ping(); err != nil {
		db.Close()
		return nil, fmt.Errorf("failed to connect to database: %w", err)
	}

	return &DB{db}, nil
}

// Migrate runs database migrations
func (db *DB) Migrate() error {
	migrations := []string{
		`CREATE TABLE IF NOT EXISTS users (
			id INTEGER PRIMARY KEY AUTOINCREMENT,
			email TEXT UNIQUE NOT NULL,
			password_hash TEXT NOT NULL,
			name TEXT NOT NULL,
			phone TEXT,
			role TEXT DEFAULT 'customer',
			created_at DATETIME DEFAULT CURRENT_TIMESTAMP
		)`,

		`CREATE TABLE IF NOT EXISTS brands (
			id INTEGER PRIMARY KEY AUTOINCREMENT,
			name TEXT NOT NULL,
			logo_url TEXT
		)`,

		`CREATE TABLE IF NOT EXISTS models (
			id INTEGER PRIMARY KEY AUTOINCREMENT,
			brand_id INTEGER REFERENCES brands(id) ON DELETE CASCADE,
			name TEXT NOT NULL
		)`,

		`CREATE TABLE IF NOT EXISTS services (
			id INTEGER PRIMARY KEY AUTOINCREMENT,
			name TEXT NOT NULL,
			description TEXT,
			base_price REAL,
			estimated_hours REAL
		)`,

		`CREATE TABLE IF NOT EXISTS bookings (
			id INTEGER PRIMARY KEY AUTOINCREMENT,
			customer_id INTEGER REFERENCES users(id) ON DELETE CASCADE,
			service_id INTEGER REFERENCES services(id) ON DELETE SET NULL,
			scheduled_at DATETIME NOT NULL,
			status TEXT DEFAULT 'pending',
			notes TEXT,
			created_at DATETIME DEFAULT CURRENT_TIMESTAMP
		)`,

		`CREATE TABLE IF NOT EXISTS quotes (
			id INTEGER PRIMARY KEY AUTOINCREMENT,
			booking_id INTEGER REFERENCES bookings(id) ON DELETE CASCADE,
			items_json TEXT,
			total REAL,
			status TEXT DEFAULT 'pending',
			rejection_reason TEXT,
			valid_until DATETIME,
			created_at DATETIME DEFAULT CURRENT_TIMESTAMP
		)`,

		`CREATE TABLE IF NOT EXISTS tickets (
			id INTEGER PRIMARY KEY AUTOINCREMENT,
			booking_id INTEGER REFERENCES bookings(id) ON DELETE CASCADE,
			technician_id INTEGER REFERENCES users(id) ON DELETE SET NULL,
			tracking_code TEXT UNIQUE NOT NULL,
			qr_code BLOB,
			status TEXT DEFAULT 'received',
			notes TEXT,
			created_at DATETIME DEFAULT CURRENT_TIMESTAMP,
			updated_at DATETIME
		)`,

		`CREATE TABLE IF NOT EXISTS surveys (
			id INTEGER PRIMARY KEY AUTOINCREMENT,
			ticket_id INTEGER REFERENCES tickets(id) ON DELETE CASCADE,
			rating INTEGER CHECK(rating BETWEEN 1 AND 5),
			feedback TEXT,
			created_at DATETIME DEFAULT CURRENT_TIMESTAMP
		)`,

		`CREATE TABLE IF NOT EXISTS ticket_status_history (
			id INTEGER PRIMARY KEY AUTOINCREMENT,
			ticket_id INTEGER REFERENCES tickets(id) ON DELETE CASCADE,
			status TEXT NOT NULL,
			changed_by INTEGER REFERENCES users(id) ON DELETE SET NULL,
			notes TEXT,
			created_at DATETIME DEFAULT CURRENT_TIMESTAMP
		)`,

		// Indexes for performance
		`CREATE INDEX IF NOT EXISTS idx_users_email ON users(email)`,
		`CREATE INDEX IF NOT EXISTS idx_users_role ON users(role)`,
		`CREATE INDEX IF NOT EXISTS idx_bookings_customer ON bookings(customer_id)`,
		`CREATE INDEX IF NOT EXISTS idx_bookings_date ON bookings(scheduled_at)`,
		`CREATE INDEX IF NOT EXISTS idx_bookings_status ON bookings(status)`,
		`CREATE INDEX IF NOT EXISTS idx_tickets_tracking ON tickets(tracking_code)`,
		`CREATE INDEX IF NOT EXISTS idx_tickets_status ON tickets(status)`,
		`CREATE INDEX IF NOT EXISTS idx_tickets_technician ON tickets(technician_id)`,
		`CREATE INDEX IF NOT EXISTS idx_quotes_booking ON quotes(booking_id)`,
		`CREATE INDEX IF NOT EXISTS idx_quotes_status ON quotes(status)`,
		`CREATE INDEX IF NOT EXISTS idx_ticket_history_ticket ON ticket_status_history(ticket_id)`,

		// New Bicycles table
		`CREATE TABLE IF NOT EXISTS bicycles (
			id INTEGER PRIMARY KEY AUTOINCREMENT,
			user_id INTEGER REFERENCES users(id) ON DELETE CASCADE,
			brand_id INTEGER REFERENCES brands(id) ON DELETE SET NULL,
			model_id INTEGER REFERENCES models(id) ON DELETE SET NULL,
			color TEXT,
			serial_number TEXT,
			notes TEXT,
			created_at DATETIME DEFAULT CURRENT_TIMESTAMP
		)`,
		`CREATE INDEX IF NOT EXISTS idx_bicycles_user ON bicycles(user_id)`,

		// Add bicycle_id to bookings if not exists
		`ALTER TABLE bookings ADD COLUMN bicycle_id INTEGER REFERENCES bicycles(id) ON DELETE SET NULL`,

		// Ticket Parts / Checklist
		`CREATE TABLE IF NOT EXISTS ticket_parts (
			id INTEGER PRIMARY KEY AUTOINCREMENT,
			ticket_id INTEGER REFERENCES tickets(id) ON DELETE CASCADE,
			name TEXT NOT NULL,
			status TEXT DEFAULT 'pending',
			created_at DATETIME DEFAULT CURRENT_TIMESTAMP
		)`,
		`CREATE INDEX IF NOT EXISTS idx_ticket_parts_ticket ON ticket_parts(ticket_id)`,

		// Ads (Press Kit)
		`CREATE TABLE IF NOT EXISTS ads (
			id INTEGER PRIMARY KEY AUTOINCREMENT,
			title TEXT NOT NULL,
			media_url TEXT NOT NULL,
			media_type TEXT NOT NULL,
			link_url TEXT,
			active BOOLEAN DEFAULT 1,
			impressions INTEGER DEFAULT 0,
			clicks INTEGER DEFAULT 0,
			created_at DATETIME DEFAULT CURRENT_TIMESTAMP,
			updated_at DATETIME
		)`,
		`CREATE INDEX IF NOT EXISTS idx_ads_active ON ads(active)`,

		// Settings (Key-Value Store)
		`CREATE TABLE IF NOT EXISTS settings (
			key TEXT PRIMARY KEY,
			value TEXT,
			updated_at DATETIME DEFAULT CURRENT_TIMESTAMP
		)`,
	}

	for _, migration := range migrations {
		if _, err := db.Exec(migration); err != nil {
			// Ignore "duplicate column name" error for idempotent migrations
			if strings.Contains(err.Error(), "duplicate column name") {
				continue
			}
			return fmt.Errorf("migration failed: %w\nSQL: %s", err, migration)
		}
	}

	return nil
}

// Close closes the database connection
func (db *DB) Close() error {
	return db.DB.Close()
}
