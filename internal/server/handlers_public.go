package server

import (
	"context"
	"net/http"
	"strconv"
	"time"

	"bicicletapp/internal/domain"

	"golang.org/x/crypto/bcrypt"
)

// PageData holds common data for all page templates
type PageData struct {
	Title     string
	Config    interface{}
	Year      int
	User      *Claims
	Flash     *FlashMessage
	Data      interface{}
	CSRFToken string
}

// FlashMessage represents a flash message
type FlashMessage struct {
	Type    string // success, error, warning, info
	Message string
}

// newPageData creates a new PageData with common fields
func (s *Server) newPageData(r *http.Request, title string) *PageData {
	claims := getUserClaims(r)

	return &PageData{
		Title:  title,
		Config: s.config,
		Year:   time.Now().Year(),
		User:   claims,
	}
}

// render renders a template with the given data
func (s *Server) render(w http.ResponseWriter, r *http.Request, template string, data *PageData) {
	w.Header().Set("Content-Type", "text/html; charset=utf-8")

	if err := s.templates.Render(w, template, data); err != nil {
		http.Error(w, "Error rendering page: "+err.Error(), http.StatusInternalServerError)
	}
}

// handleHome renders the home page
func (s *Server) handleHome(w http.ResponseWriter, r *http.Request) {
	data := s.newPageData(r, "Inicio")

	ctx := r.Context()
	heroConcept, err := s.repos.Settings.Get(ctx, "hero_concept")
	if err != nil || heroConcept == "" {
		heroConcept = "bicycle workshop"
	}

	data.Data = map[string]interface{}{
		"HeroConcept": heroConcept,
	}
	s.render(w, r, "pages/public/home.html", data)
}

// handleLoginPage renders the login page
func (s *Server) handleLoginPage(w http.ResponseWriter, r *http.Request) {
	// If already logged in, redirect to dashboard
	if claims := getUserClaims(r); claims != nil {
		http.Redirect(w, r, "/dashboard", http.StatusSeeOther)
		return
	}

	data := s.newPageData(r, "Iniciar Sesión")
	s.render(w, r, "pages/public/login.html", data)
}

// handleLogin processes login form
func (s *Server) handleLogin(w http.ResponseWriter, r *http.Request) {
	if err := r.ParseForm(); err != nil {
		http.Error(w, "Error processing form", http.StatusBadRequest)
		return
	}

	email := r.FormValue("email")
	password := r.FormValue("password")

	// Get user by email
	ctx := r.Context()
	user, err := s.repos.Users.GetByEmail(ctx, email)
	if err != nil || user == nil {
		data := s.newPageData(r, "Iniciar Sesión")
		data.Flash = &FlashMessage{Type: "error", Message: "Credenciales inválidas"}
		s.render(w, r, "pages/public/login.html", data)
		return
	}

	// Check password
	if !checkPasswordHash(password, user.PasswordHash) {
		data := s.newPageData(r, "Iniciar Sesión")
		data.Flash = &FlashMessage{Type: "error", Message: "Credenciales inválidas"}
		s.render(w, r, "pages/public/login.html", data)
		return
	}

	// Generate JWT token
	token, err := s.generateToken(user)
	if err != nil {
		http.Error(w, "Error generating token", http.StatusInternalServerError)
		return
	}

	// Set auth cookie
	maxAge := s.config.JWT.ExpirationHours * 3600
	s.setAuthCookie(w, token, maxAge)

	// Redirect based on role
	switch user.Role {
	case "admin":
		http.Redirect(w, r, "/admin", http.StatusSeeOther)
	case "technician":
		http.Redirect(w, r, "/workshop", http.StatusSeeOther)
	default:
		http.Redirect(w, r, "/dashboard", http.StatusSeeOther)
	}
}

// handleRegisterPage renders the registration page
func (s *Server) handleRegisterPage(w http.ResponseWriter, r *http.Request) {
	data := s.newPageData(r, "Registrarse")
	s.render(w, r, "pages/public/register.html", data)
}

// handleRegister processes registration form
func (s *Server) handleRegister(w http.ResponseWriter, r *http.Request) {
	if err := r.ParseForm(); err != nil {
		http.Error(w, "Error processing form", http.StatusBadRequest)
		return
	}

	name := r.FormValue("name")
	email := r.FormValue("email")
	phone := r.FormValue("phone")
	password := r.FormValue("password")
	confirmPassword := r.FormValue("confirm_password")

	// Validate passwords match
	if password != confirmPassword {
		data := s.newPageData(r, "Registrarse")
		data.Flash = &FlashMessage{Type: "error", Message: "Las contraseñas no coinciden"}
		s.render(w, r, "pages/public/register.html", data)
		return
	}

	// Check if email already exists
	ctx := r.Context()
	existingUser, _ := s.repos.Users.GetByEmail(ctx, email)
	if existingUser != nil {
		data := s.newPageData(r, "Registrarse")
		data.Flash = &FlashMessage{Type: "error", Message: "El email ya está registrado"}
		s.render(w, r, "pages/public/register.html", data)
		return
	}

	// Hash password
	hashedPassword, err := hashPassword(password)
	if err != nil {
		http.Error(w, "Error processing registration", http.StatusInternalServerError)
		return
	}

	// Create user
	user := &domain.User{
		Name:         name,
		Email:        email,
		Phone:        phone,
		PasswordHash: hashedPassword,
		Role:         "customer",
	}

	if err := s.repos.Users.Create(ctx, user); err != nil {
		http.Error(w, "Error creating user", http.StatusInternalServerError)
		return
	}

	// Redirect to login with success message
	http.Redirect(w, r, "/login?registered=1", http.StatusSeeOther)
}

// handleLogout logs out the user
func (s *Server) handleLogout(w http.ResponseWriter, r *http.Request) {
	clearAuthCookie(w)
	http.Redirect(w, r, "/", http.StatusSeeOther)
}

// handleTrackingPage renders the tracking search page
func (s *Server) handleTrackingPage(w http.ResponseWriter, r *http.Request) {
	data := s.newPageData(r, "Consultar Estado")
	s.render(w, r, "pages/public/tracking.html", data)
}

// handleTrackingStatus shows ticket status by tracking code
func (s *Server) handleTrackingStatus(w http.ResponseWriter, r *http.Request) {
	code := getURLParam(r, "code")

	ctx := r.Context()
	ticket, err := s.repos.Tickets.GetByTrackingCode(ctx, code)
	if err != nil || ticket == nil {
		data := s.newPageData(r, "Tracking no encontrado")
		data.Flash = &FlashMessage{Type: "error", Message: "Código de seguimiento no encontrado"}
		s.render(w, r, "pages/public/tracking.html", data)
		return
	}

	// Get status history
	history, _ := s.repos.Tickets.GetStatusHistory(ctx, ticket.ID)

	// Get quote if exists
	quote, _ := s.repos.Quotes.GetByBookingID(ctx, ticket.BookingID)

	// Create a map of status -> history entry for easier lookup in template
	statusMap := make(map[string]domain.TicketStatusHistory)
	for _, h := range history {
		statusMap[h.Status] = h
	}

	// Get survey if exists
	survey, _ := s.repos.Surveys.GetByTicketID(ctx, ticket.ID)

	// Get active ad (Press Kit)
	ad, _ := s.repos.Ads.GetRandomActive(ctx)
	if ad != nil {
		// Increment impression in background
		go func(id int64) {
			s.repos.Ads.IncrementImpressions(context.Background(), id)
		}(ad.ID)
	}

	data := s.newPageData(r, "Estado de tu Reparación")
	data.Data = map[string]interface{}{
		"Ticket":        ticket,
		"StatusHistory": history,
		"StatusMap":     statusMap,
		"Quote":         quote,
		"Survey":        survey,
		"Ad":            ad,
	}
	s.render(w, r, "pages/public/tracking_result.html", data)
}

// handleServicesPage shows available services
func (s *Server) handleServicesPage(w http.ResponseWriter, r *http.Request) {
	ctx := r.Context()
	services, err := s.repos.Services.List(ctx)
	if err != nil {
		http.Error(w, "Error loading services", http.StatusInternalServerError)
		return
	}

	data := s.newPageData(r, "Nuestros Servicios")
	data.Data = services
	s.render(w, r, "pages/public/services.html", data)
}

// Helper functions
func hashPassword(password string) (string, error) {
	bytes, err := bcrypt.GenerateFromPassword([]byte(password), bcrypt.DefaultCost)
	return string(bytes), err
}

func checkPasswordHash(password, hash string) bool {
	err := bcrypt.CompareHashAndPassword([]byte(hash), []byte(password))
	return err == nil
}

// handlePublicApproveQuote allows a customer to approve a quote from tracking page
func (s *Server) handlePublicApproveQuote(w http.ResponseWriter, r *http.Request) {
	ctx := r.Context()
	id, _ := strconv.ParseInt(getURLParam(r, "id"), 10, 64)

	// We verify the quote exists
	quote, err := s.repos.Quotes.GetByID(ctx, id)
	if err != nil || quote == nil {
		http.Error(w, "Presupuesto no encontrado", http.StatusNotFound)
		return
	}

	// Approve it
	if err := s.repos.Quotes.Approve(ctx, id); err != nil {
		http.Error(w, "Error aprobando presupuesto", http.StatusInternalServerError)
		return
	}

	// Redirect back to tracking page (we need the ticket code)
	// Since we don't have the ticket code handy in the URL params of this POST,
	// we rely on the referrer or we fetch the ticket.
	// Let's create a helper or just search by BookingID on Tickets repo if possible.
	// As a fallback, we can ask the form to send the tracking code.
	code := r.FormValue("tracking_code")
	if code != "" {
		http.Redirect(w, r, "/tracking/"+code+"?quote_approved=true", http.StatusSeeOther)
		return
	}

	http.Redirect(w, r, "/", http.StatusSeeOther)
}

// handlePublicSubmitSurvey processes public survey submission from tracking page
func (s *Server) handlePublicSubmitSurvey(w http.ResponseWriter, r *http.Request) {
	if err := r.ParseForm(); err != nil {
		http.Error(w, "Error processing form", http.StatusBadRequest)
		return
	}

	code := getURLParam(r, "code")
	ctx := r.Context()

	// Verify ticket exists and matches tracking code
	ticket, err := s.repos.Tickets.GetByTrackingCode(ctx, code)
	if err != nil || ticket == nil {
		http.Error(w, "Ticket not found", http.StatusNotFound)
		return
	}

	// Verify ticket is in a state that allows survey (completed)
	if ticket.Status != domain.TicketStatusDelivered && ticket.Status != domain.TicketStatusReady {
		http.Error(w, "Survey not available for this ticket status", http.StatusBadRequest)
		return
	}

	// Check if survey already exists
	existing, _ := s.repos.Surveys.GetByTicketID(ctx, ticket.ID)
	if existing != nil {
		http.Redirect(w, r, "/tracking/"+code, http.StatusSeeOther)
		return
	}

	rating, _ := strconv.Atoi(r.FormValue("rating"))
	feedback := r.FormValue("feedback")

	survey := &domain.Survey{
		TicketID: ticket.ID,
		Rating:   rating,
		Feedback: feedback,
	}

	if err := s.repos.Surveys.Create(ctx, survey); err != nil {
		http.Error(w, "Error submitting survey", http.StatusInternalServerError)
		return
	}

	http.Redirect(w, r, "/tracking/"+code+"?survey_submitted=true", http.StatusSeeOther)
}

// handleAdClick tracks clicks and redirects
func (s *Server) handleAdClick(w http.ResponseWriter, r *http.Request) {
	ctx := r.Context()
	id, _ := strconv.ParseInt(getURLParam(r, "id"), 10, 64)

	ad, err := s.repos.Ads.GetByID(ctx, id)
	if err != nil || ad == nil {
		http.NotFound(w, r)
		return
	}

	// Increment click in background
	go func(id int64) {
		s.repos.Ads.IncrementClicks(context.Background(), id)
	}(ad.ID)

	// Redirect to ad link
	if ad.LinkURL != "" {
		http.Redirect(w, r, ad.LinkURL, http.StatusFound)
	} else {
		// If no link, just stay here or go home (shouldn't happen if validated)
		http.Redirect(w, r, "/", http.StatusSeeOther)
	}
}
