package sqlite

import (
	"context"
	"database/sql"
	"fmt"
	"time"

	"bicicletapp/internal/domain"
	"bicicletapp/internal/repository"

	"golang.org/x/crypto/bcrypt"
)

// UserRepo implements repository.UserRepository
type UserRepo struct {
	db *DB
}

// NewUserRepo creates a new UserRepo
func NewUserRepo(db *DB) repository.UserRepository {
	return &UserRepo{db: db}
}

func (r *UserRepo) Create(ctx context.Context, user *domain.User) error {
	query := `
		INSERT INTO users (email, password_hash, name, phone, role, created_at)
		VALUES (?, ?, ?, ?, ?, ?)
	`
	result, err := r.db.ExecContext(ctx, query,
		user.Email, user.PasswordHash, user.Name, user.Phone, user.Role, time.Now())
	if err != nil {
		return fmt.Errorf("failed to create user: %w", err)
	}

	id, err := result.LastInsertId()
	if err != nil {
		return fmt.Errorf("failed to get user ID: %w", err)
	}
	user.ID = id
	return nil
}

func (r *UserRepo) GetByID(ctx context.Context, id int64) (*domain.User, error) {
	query := `SELECT id, email, password_hash, name, phone, role, created_at FROM users WHERE id = ?`
	user := &domain.User{}
	err := r.db.QueryRowContext(ctx, query, id).Scan(
		&user.ID, &user.Email, &user.PasswordHash, &user.Name, &user.Phone, &user.Role, &user.CreatedAt)
	if err == sql.ErrNoRows {
		return nil, nil
	}
	if err != nil {
		return nil, fmt.Errorf("failed to get user: %w", err)
	}
	return user, nil
}

func (r *UserRepo) GetByEmail(ctx context.Context, email string) (*domain.User, error) {
	query := `SELECT id, email, password_hash, name, phone, role, created_at FROM users WHERE email = ?`
	user := &domain.User{}
	err := r.db.QueryRowContext(ctx, query, email).Scan(
		&user.ID, &user.Email, &user.PasswordHash, &user.Name, &user.Phone, &user.Role, &user.CreatedAt)
	if err == sql.ErrNoRows {
		return nil, nil
	}
	if err != nil {
		return nil, fmt.Errorf("failed to get user by email: %w", err)
	}
	return user, nil
}

func (r *UserRepo) Update(ctx context.Context, user *domain.User) error {
	query := `UPDATE users SET email = ?, name = ?, phone = ?, role = ? WHERE id = ?`
	_, err := r.db.ExecContext(ctx, query, user.Email, user.Name, user.Phone, user.Role, user.ID)
	if err != nil {
		return fmt.Errorf("failed to update user: %w", err)
	}
	return nil
}

func (r *UserRepo) Delete(ctx context.Context, id int64) error {
	query := `DELETE FROM users WHERE id = ?`
	_, err := r.db.ExecContext(ctx, query, id)
	if err != nil {
		return fmt.Errorf("failed to delete user: %w", err)
	}
	return nil
}

func (r *UserRepo) List(ctx context.Context, role string, limit, offset int) ([]domain.User, error) {
	var query string
	var args []interface{}

	if role != "" {
		query = `SELECT id, email, password_hash, name, phone, role, created_at FROM users WHERE role = ? ORDER BY name LIMIT ? OFFSET ?`
		args = []interface{}{role, limit, offset}
	} else {
		query = `SELECT id, email, password_hash, name, phone, role, created_at FROM users ORDER BY name LIMIT ? OFFSET ?`
		args = []interface{}{limit, offset}
	}

	rows, err := r.db.QueryContext(ctx, query, args...)
	if err != nil {
		return nil, fmt.Errorf("failed to list users: %w", err)
	}
	defer rows.Close()

	var users []domain.User
	for rows.Next() {
		var u domain.User
		if err := rows.Scan(&u.ID, &u.Email, &u.PasswordHash, &u.Name, &u.Phone, &u.Role, &u.CreatedAt); err != nil {
			return nil, fmt.Errorf("failed to scan user: %w", err)
		}
		users = append(users, u)
	}
	return users, nil
}

func (r *UserRepo) Count(ctx context.Context, role string) (int, error) {
	var query string
	var args []interface{}

	if role != "" {
		query = `SELECT COUNT(*) FROM users WHERE role = ?`
		args = []interface{}{role}
	} else {
		query = `SELECT COUNT(*) FROM users`
	}

	var count int
	err := r.db.QueryRowContext(ctx, query, args...).Scan(&count)
	if err != nil {
		return 0, fmt.Errorf("failed to count users: %w", err)
	}
	return count, nil
}

// HashPassword hashes a password using bcrypt
func HashPassword(password string) (string, error) {
	bytes, err := bcrypt.GenerateFromPassword([]byte(password), bcrypt.DefaultCost)
	return string(bytes), err
}

// CheckPassword compares a password with a hash
func CheckPassword(password, hash string) bool {
	err := bcrypt.CompareHashAndPassword([]byte(hash), []byte(password))
	return err == nil
}
