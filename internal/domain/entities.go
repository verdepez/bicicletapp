// Package domain defines core business entities
package domain

import (
	"time"
)

// User represents a system user (customer, technician, or admin)
type User struct {
	ID           int64     `json:"id"`
	Email        string    `json:"email"`
	PasswordHash string    `json:"-"`
	Name         string    `json:"name"`
	Phone        string    `json:"phone,omitempty"`
	Role         string    `json:"role"` // customer, technician, admin
	CreatedAt    time.Time `json:"createdAt"`
}

// Brand represents a bicycle brand
type Brand struct {
	ID      int64  `json:"id"`
	Name    string `json:"name"`
	LogoURL string `json:"logoUrl,omitempty"`
}

// Model represents a bicycle model
type Model struct {
	ID      int64  `json:"id"`
	BrandID int64  `json:"brandId"`
	Brand   *Brand `json:"brand,omitempty"`
	Name    string `json:"name"`
}

// Service represents a service offered by the workshop
type Service struct {
	ID             int64   `json:"id"`
	Name           string  `json:"name"`
	Description    string  `json:"description"`
	BasePrice      float64 `json:"basePrice"`
	EstimatedHours float64 `json:"estimatedHours"`
}

// Bicycle represents a customer's bicycle
type Bicycle struct {
	ID           int64     `json:"id"`
	UserID       int64     `json:"userId"`
	BrandID      int64     `json:"brandId"`
	Brand        *Brand    `json:"brand,omitempty"`
	ModelID      int64     `json:"modelId"`
	Model        *Model    `json:"model,omitempty"`
	Color        string    `json:"color"`
	SerialNumber string    `json:"serialNumber,omitempty"`
	Notes        string    `json:"notes,omitempty"`
	CreatedAt    time.Time `json:"createdAt"`
}

// Booking represents a customer appointment
type Booking struct {
	ID          int64     `json:"id"`
	CustomerID  int64     `json:"customerId"`
	Customer    *User     `json:"customer,omitempty"`
	BicycleID   int64     `json:"bicycleId,omitempty"` // New field
	Bicycle     *Bicycle  `json:"bicycle,omitempty"`   // New field
	ServiceID   int64     `json:"serviceId"`
	Service     *Service  `json:"service,omitempty"`
	ScheduledAt time.Time `json:"scheduledAt"`
	Status      string    `json:"status"` // pending, confirmed, completed, cancelled
	Notes       string    `json:"notes,omitempty"`
	CreatedAt   time.Time `json:"createdAt"`
}

// QuoteItem represents a line item in a quote
type QuoteItem struct {
	Description string  `json:"description"`
	Quantity    int     `json:"quantity"`
	UnitPrice   float64 `json:"unitPrice"`
	Total       float64 `json:"total"`
}

// Quote represents a cost estimate for a service
type Quote struct {
	ID              int64       `json:"id"`
	BookingID       int64       `json:"bookingId"`
	Booking         *Booking    `json:"booking,omitempty"`
	Items           []QuoteItem `json:"items"`
	Total           float64     `json:"total"`
	Status          string      `json:"status"` // pending, approved, rejected
	RejectionReason string      `json:"rejectionReason,omitempty"`
	ValidUntil      time.Time   `json:"validUntil"`
	CreatedAt       time.Time   `json:"createdAt"`
}

// Ticket represents a work order
type Ticket struct {
	ID           int64     `json:"id"`
	BookingID    int64     `json:"bookingId"`
	Booking      *Booking  `json:"booking,omitempty"`
	TechnicianID int64     `json:"technicianId"`
	Technician   *User     `json:"technician,omitempty"`
	TrackingCode string    `json:"trackingCode"`
	QRCode       []byte    `json:"-"`
	QRCodeBase64 string    `json:"qrCode,omitempty"`
	Status       string    `json:"status"` // received, diagnosing, in_progress, waiting_parts, ready, delivered
	Notes        string    `json:"notes,omitempty"`
	CreatedAt    time.Time `json:"createdAt"`
	UpdatedAt    time.Time `json:"updatedAt"`
}

// Survey represents a post-service feedback survey
type Survey struct {
	ID        int64     `json:"id"`
	TicketID  int64     `json:"ticketId"`
	Ticket    *Ticket   `json:"ticket,omitempty"`
	Rating    int       `json:"rating"` // 1-5
	Feedback  string    `json:"feedback,omitempty"`
	CreatedAt time.Time `json:"createdAt"`
}

// Status constants
const (
	// Booking statuses
	BookingStatusPending   = "pending"
	BookingStatusConfirmed = "confirmed"
	BookingStatusCompleted = "completed"
	BookingStatusCancelled = "cancelled"

	// Quote statuses
	QuoteStatusPending  = "pending"
	QuoteStatusApproved = "approved"
	QuoteStatusRejected = "rejected"

	// Ticket statuses
	TicketStatusReceived     = "received"
	TicketStatusDiagnosing   = "diagnosing"
	TicketStatusInProgress   = "in_progress"
	TicketStatusWaitingParts = "waiting_parts"
	TicketStatusReady        = "ready"
	TicketStatusDelivered    = "delivered"

	// User roles
	RoleCustomer   = "customer"
	RoleTechnician = "technician"
	RoleAdmin      = "admin"
)

// TicketStatusLabel returns a human-readable label for a ticket status
func TicketStatusLabel(status string) string {
	labels := map[string]string{
		TicketStatusReceived:     "Recibido",
		TicketStatusDiagnosing:   "Diagnosticando",
		TicketStatusInProgress:   "En Progreso",
		TicketStatusWaitingParts: "Esperando Repuestos",
		TicketStatusReady:        "Listo para Retirar",
		TicketStatusDelivered:    "Entregado",
	}
	if label, ok := labels[status]; ok {
		return label
	}
	return status
}

// TicketStatusHistory represents a record of a ticket status change
type TicketStatusHistory struct {
	ID        int64     `json:"id"`
	TicketID  int64     `json:"ticketId"`
	Status    string    `json:"status"`
	ChangedBy int64     `json:"changedBy,omitempty"`
	User      *User     `json:"user,omitempty"`
	Notes     string    `json:"notes,omitempty"`
	CreatedAt time.Time `json:"createdAt"`
}

// TicketPart represents a part or checklist item for a ticket
type TicketPart struct {
	ID        int64     `json:"id"`
	TicketID  int64     `json:"ticketId"`
	Name      string    `json:"name"`
	Status    string    `json:"status"` // pending, done
	CreatedAt time.Time `json:"createdAt"`
}
