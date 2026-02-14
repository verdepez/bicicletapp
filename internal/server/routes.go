package server

import (
	"net/http"
	"os"
	"path/filepath"
	"strings"
	"time"

	"bicicletapp/internal/domain"

	"github.com/go-chi/chi/v5"
)

// setupRoutes configures all application routes
func (s *Server) setupRoutes() {
	r := s.router

	// Static files with cache headers
	r.Handle("/static/*", s.staticHandler())

	// Health check endpoint
	r.Get("/health", s.handleHealth)

	// Public routes
	r.Group(func(r chi.Router) {
		r.Get("/", s.handleHome)
		r.Get("/login", s.handleLoginPage)
		r.Post("/login", s.handleLogin)
		r.Get("/register", s.handleRegisterPage)
		r.Post("/register", s.handleRegister)
		r.Get("/logout", s.handleLogout)

		// Public tracking
		r.Get("/tracking", s.handleTrackingPage)
		r.Get("/tracking/{code}", s.handleTrackingStatus)
		r.Post("/tracking/{code}/survey", s.handlePublicSubmitSurvey)
		r.Post("/tracking/quote/{id}/approve", s.handlePublicApproveQuote)
		r.Get("/ad/{id}/click", s.handleAdClick)

		// Public services catalog
		r.Get("/services", s.handleServicesPage)
	})

	// Protected routes - Customer
	r.Group(func(r chi.Router) {
		r.Use(s.authMiddleware)

		r.Get("/dashboard", s.handleDashboard)

		// Bookings
		r.Get("/bookings", s.handleBookingsList)
		r.Get("/bookings/new", s.handleNewBookingPage)
		r.Post("/bookings", s.handleCreateBooking)
		r.Get("/bookings/{id}", s.handleBookingDetail)
		r.Post("/bookings/{id}/cancel", s.handleCancelBooking)

		// Quotes
		r.Get("/quotes", s.handleQuotesList)
		r.Get("/quotes/{id}", s.handleQuoteDetail)
		r.Post("/quotes/{id}/approve", s.handleApproveQuote)
		r.Post("/quotes/{id}/reject", s.handleRejectQuote)

		// Profile
		r.Get("/profile", s.handleProfile)
		r.Post("/profile", s.handleUpdateProfile)

		// Surveys
		r.Get("/survey/{ticketId}", s.handleSurveyPage)
		r.Post("/survey/{ticketId}", s.handleSubmitSurvey)
	})

	// Protected routes - Technician
	r.Group(func(r chi.Router) {
		r.Use(s.authMiddleware)
		r.Use(s.roleMiddleware(domain.RoleTechnician, domain.RoleAdmin))

		// Workshop routes
		r.Get("/workshop", s.handleWorkshopDashboard)

		// Direct Ticket Creation (Walk-in)
		r.Get("/tickets/new", s.handleNewTicketPage)
		r.Post("/tickets/create_direct", s.handleCreateTicketDirect)

		// Ticket management
		r.Get("/tickets", s.handleTicketsList)
		r.Get("/tickets/{id}", s.handleTicketDetail)
		r.Post("/tickets/{id}/status", s.handleUpdateTicketStatus)
		r.Post("/tickets/{id}/notes", s.handleAddTicketNotes)

		// Create ticket from booking
		r.Post("/bookings/{id}/ticket", s.handleCreateTicket)

		// Create quote
		r.Get("/quotes/new/{bookingId}", s.handleNewQuotePage)
		r.Post("/quotes/new/{bookingId}", s.handleCreateQuote)

		// Bicycle management
		r.Post("/bicycles/{id}/update", s.handleUpdateBicycle)
		r.Post("/bookings/{id}/bicycle", s.handleCreateBicycleFromBooking)

		// Ticket Parts
		r.Post("/tickets/{id}/parts", s.handleCreateTicketPart)
		r.Post("/tickets/{id}/parts/{partId}/toggle", s.handleToggleTicketPart)
		r.Post("/tickets/{id}/parts/{partId}/delete", s.handleDeleteTicketPart)

		// Label
		r.Get("/tickets/{id}/label", s.handleTicketLabel)
		r.Get("/tickets/{id}/quote", s.handleTicketQuote)
	})

	// Protected routes - Admin only
	r.Group(func(r chi.Router) {
		r.Use(s.authMiddleware)
		r.Use(s.roleMiddleware(domain.RoleAdmin))

		// Admin dashboard
		r.Get("/admin", s.handleAdminDashboard)

		// User management
		r.Get("/admin/users", s.handleUsersList)
		r.Get("/admin/users/new", s.handleNewUserPage)
		r.Post("/admin/users", s.handleCreateUser)
		r.Get("/admin/users/{id}", s.handleEditUserPage)
		r.Post("/admin/users/{id}", s.handleUpdateUser)
		r.Post("/admin/users/{id}/delete", s.handleDeleteUser)

		// Catalog management
		r.Get("/admin/brands", s.handleBrandsList)
		r.Get("/admin/brands/new", s.handleNewBrandPage)
		r.Post("/admin/brands", s.handleCreateBrand)
		r.Get("/admin/brands/{id}", s.handleEditBrandPage)
		r.Post("/admin/brands/{id}", s.handleUpdateBrand)
		r.Post("/admin/brands/{id}/delete", s.handleDeleteBrand)

		r.Get("/admin/models", s.handleModelsList)
		r.Get("/admin/models/new", s.handleNewModelPage)
		r.Post("/admin/models", s.handleCreateModel)
		r.Get("/admin/models/{id}", s.handleEditModelPage)
		r.Post("/admin/models/{id}", s.handleUpdateModel)
		r.Post("/admin/models/{id}/delete", s.handleDeleteModel)

		r.Get("/admin/services", s.handleServicesList)
		r.Get("/admin/services/new", s.handleNewServicePage)
		r.Post("/admin/services", s.handleCreateService)
		r.Get("/admin/services/{id}", s.handleEditServicePage)
		r.Post("/admin/services/{id}", s.handleUpdateService)
		r.Post("/admin/services/{id}/delete", s.handleDeleteService)

		// Reports
		r.Get("/admin/reports", s.handleReportsDashboard)
		r.Get("/admin/reports/bookings", s.handleBookingsReport)
		r.Get("/admin/reports/revenue", s.handleRevenueReport)
		r.Get("/admin/reports/surveys", s.handleSurveysReport)

		// Ticket management
		r.Get("/admin/tickets", s.handleAdminTicketsList)
		r.Post("/admin/tickets/{id}/technician", s.handleAdminUpdateTicketTechnician)

		// Settings
		r.Get("/admin/settings", s.handleSettings)
		r.Post("/admin/settings", s.handleUpdateSettings)

		// Ad management (Press Kit)
		r.Get("/admin/ads", s.handleAdsList)
		r.Post("/admin/ads", s.handleCreateAd)
		r.Post("/admin/ads/{id}/update", s.handleUpdateAd)
		r.Post("/admin/ads/{id}/delete", s.handleDeleteAd)
	})

	// API routes (for AJAX calls)
	r.Route("/api", func(r chi.Router) {
		r.Use(s.authMiddleware)

		// Models by brand (for cascading dropdowns)
		r.Get("/brands/{brandId}/models", s.apiGetModelsByBrand)

		// Available time slots
		r.Get("/bookings/slots", s.apiGetAvailableSlots)

		// Ticket status updates
		r.Get("/tickets/{id}/status", s.apiGetTicketStatus)
	})
}

// staticHandler serves static files with caching
func (s *Server) staticHandler() http.Handler {
	// Validate and clean static directory path
	staticDir := filepath.Clean("./static")

	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		// Extract the file path from the URL
		urlPath := strings.TrimPrefix(r.URL.Path, "/static/")

		// Clean and validate the path to prevent directory traversal
		cleanPath := filepath.Clean(urlPath)
		if strings.Contains(cleanPath, "..") {
			http.Error(w, "Forbidden", http.StatusForbidden)
			return
		}

		// Full path to the file
		fullPath := filepath.Join(staticDir, cleanPath)

		// Verify the file is within the static directory
		absStaticDir, _ := filepath.Abs(staticDir)
		absFullPath, _ := filepath.Abs(fullPath)
		if !strings.HasPrefix(absFullPath, absStaticDir) {
			http.Error(w, "Forbidden", http.StatusForbidden)
			return
		}

		// Check if file exists
		if _, err := os.Stat(fullPath); os.IsNotExist(err) {
			http.NotFound(w, r)
			return
		}

		// Set cache headers for static assets (1 week in production)
		if !s.config.Debug {
			w.Header().Set("Cache-Control", "public, max-age=604800")
		} else {
			w.Header().Set("Cache-Control", "no-cache")
		}

		http.ServeFile(w, r, fullPath)
	})
}

// handleHealth returns server health status
func (s *Server) handleHealth(w http.ResponseWriter, r *http.Request) {
	w.Header().Set("Content-Type", "application/json")
	w.WriteHeader(http.StatusOK)
	w.Write([]byte(`{"status":"healthy","timestamp":"` + time.Now().Format(time.RFC3339) + `"}`))
}
