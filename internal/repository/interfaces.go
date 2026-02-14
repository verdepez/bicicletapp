// Package repository defines interfaces for data persistence
package repository

import (
	"context"
	"time"

	"bicicletapp/internal/domain"
)

// UserRepository defines the interface for user data operations
type UserRepository interface {
	Create(ctx context.Context, user *domain.User) error
	GetByID(ctx context.Context, id int64) (*domain.User, error)
	GetByEmail(ctx context.Context, email string) (*domain.User, error)
	Update(ctx context.Context, user *domain.User) error
	Delete(ctx context.Context, id int64) error
	List(ctx context.Context, role string, limit, offset int) ([]domain.User, error)
	Count(ctx context.Context, role string) (int, error)
}

// BrandRepository defines the interface for brand data operations
type BrandRepository interface {
	Create(ctx context.Context, brand *domain.Brand) error
	GetByID(ctx context.Context, id int64) (*domain.Brand, error)
	Update(ctx context.Context, brand *domain.Brand) error
	Delete(ctx context.Context, id int64) error
	List(ctx context.Context) ([]domain.Brand, error)
}

// ModelRepository defines the interface for model data operations
type ModelRepository interface {
	Create(ctx context.Context, model *domain.Model) error
	GetByID(ctx context.Context, id int64) (*domain.Model, error)
	GetByBrandID(ctx context.Context, brandID int64) ([]domain.Model, error)
	Update(ctx context.Context, model *domain.Model) error
	Delete(ctx context.Context, id int64) error
	List(ctx context.Context) ([]domain.Model, error)
}

// ServiceRepository defines the interface for service data operations
type ServiceRepository interface {
	Create(ctx context.Context, service *domain.Service) error
	GetByID(ctx context.Context, id int64) (*domain.Service, error)
	Update(ctx context.Context, service *domain.Service) error
	Delete(ctx context.Context, id int64) error
	List(ctx context.Context) ([]domain.Service, error)
}

// BookingRepository defines the interface for booking data operations
type BookingRepository interface {
	Create(ctx context.Context, booking *domain.Booking) error
	GetByID(ctx context.Context, id int64) (*domain.Booking, error)
	GetByCustomerID(ctx context.Context, customerID int64, limit, offset int) ([]domain.Booking, error)
	GetByDateRange(ctx context.Context, start, end time.Time) ([]domain.Booking, error)
	Update(ctx context.Context, booking *domain.Booking) error
	UpdateStatus(ctx context.Context, id int64, status string) error
	Delete(ctx context.Context, id int64) error
	List(ctx context.Context, status string, limit, offset int) ([]domain.Booking, error)
	CountByStatus(ctx context.Context, status string) (int, error)
}

// QuoteRepository defines the interface for quote data operations
type QuoteRepository interface {
	Create(ctx context.Context, quote *domain.Quote) error
	GetByID(ctx context.Context, id int64) (*domain.Quote, error)
	GetByBookingID(ctx context.Context, bookingID int64) (*domain.Quote, error)
	Update(ctx context.Context, quote *domain.Quote) error
	Approve(ctx context.Context, id int64) error
	Reject(ctx context.Context, id int64, reason string) error
	List(ctx context.Context, status string, limit, offset int) ([]domain.Quote, error)
}

// TicketRepository defines the interface for ticket data operations
type TicketRepository interface {
	Create(ctx context.Context, ticket *domain.Ticket) error
	GetByID(ctx context.Context, id int64) (*domain.Ticket, error)
	GetByTrackingCode(ctx context.Context, code string) (*domain.Ticket, error)
	GetByTechnicianID(ctx context.Context, technicianID int64, status string, limit, offset int) ([]domain.Ticket, error)
	Update(ctx context.Context, ticket *domain.Ticket) error
	UpdateStatus(ctx context.Context, id int64, status string, changedBy int64, notes string) error
	CreateStatusHistory(ctx context.Context, history *domain.TicketStatusHistory) error
	GetStatusHistory(ctx context.Context, ticketID int64) ([]domain.TicketStatusHistory, error)

	// Ticket Parts
	CreateTicketPart(ctx context.Context, part *domain.TicketPart) error
	GetTicketParts(ctx context.Context, ticketID int64) ([]domain.TicketPart, error)
	ToggleTicketPartStatus(ctx context.Context, id int64) error
	DeleteTicketPart(ctx context.Context, id int64) error
	List(ctx context.Context, status string, limit, offset int) ([]domain.Ticket, error)
	CountByStatus(ctx context.Context) (map[string]int, error)
}

// SurveyRepository defines the interface for survey data operations
type SurveyRepository interface {
	Create(ctx context.Context, survey *domain.Survey) error
	GetByTicketID(ctx context.Context, ticketID int64) (*domain.Survey, error)
	GetAverageRating(ctx context.Context, fromDate time.Time) (float64, error)
	Count(ctx context.Context) (int, error)
	GetRatingDistribution(ctx context.Context) (map[int]int, error)
	List(ctx context.Context, limit, offset int) ([]domain.Survey, error)
}

// AdRepository defines the interface for ad data operations
type AdRepository interface {
	Create(ctx context.Context, ad *domain.Ad) error
	GetByID(ctx context.Context, id int64) (*domain.Ad, error)
	GetRandomActive(ctx context.Context) (*domain.Ad, error)
	Update(ctx context.Context, ad *domain.Ad) error
	Delete(ctx context.Context, id int64) error
	List(ctx context.Context) ([]domain.Ad, error)
	IncrementImpressions(ctx context.Context, id int64) error
	IncrementClicks(ctx context.Context, id int64) error
}

// BicycleRepository defines the interface for bicycle data operations
type BicycleRepository interface {
	Create(ctx context.Context, bicycle *domain.Bicycle) error
	GetByID(ctx context.Context, id int64) (*domain.Bicycle, error)
	GetByUserID(ctx context.Context, userID int64) ([]domain.Bicycle, error)
	Update(ctx context.Context, bicycle *domain.Bicycle) error
	Delete(ctx context.Context, id int64) error
}

// SettingsRepository handles application configuration
type SettingsRepository interface {
	Get(ctx context.Context, key string) (string, error)
	Set(ctx context.Context, key, value string) error
}

// Repositories bundles all repository interfaces
type Repositories struct {
	Users    UserRepository
	Brands   BrandRepository
	Models   ModelRepository
	Services ServiceRepository
	Bicycles BicycleRepository
	Bookings BookingRepository
	Quotes   QuoteRepository
	Tickets  TicketRepository
	Surveys  SurveyRepository
	Ads      AdRepository
	Settings SettingsRepository
}
