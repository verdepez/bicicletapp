package server

import (
	"crypto/rand"
	"encoding/base64"
	"encoding/hex"
	"fmt"
	"net/http"
	"strconv"
	"strings"
	"time"

	"bicicletapp/internal/domain"

	"github.com/skip2/go-qrcode"
)

// handleDashboard shows customer dashboard
func (s *Server) handleDashboard(w http.ResponseWriter, r *http.Request) {
	claims := getUserClaims(r)
	ctx := r.Context()

	// Get recent bookings
	bookings, _ := s.repos.Bookings.GetByCustomerID(ctx, claims.UserID, 5, 0)

	data := s.newPageData(r, "Mi Panel")
	data.Data = map[string]interface{}{
		"Bookings": bookings,
	}
	s.render(w, r, "pages/customer/dashboard.html", data)
}

// handleBookingsList shows all customer bookings
func (s *Server) handleBookingsList(w http.ResponseWriter, r *http.Request) {
	claims := getUserClaims(r)
	ctx := r.Context()

	bookings, err := s.repos.Bookings.GetByCustomerID(ctx, claims.UserID, 50, 0)
	if err != nil {
		http.Error(w, "Error loading bookings", http.StatusInternalServerError)
		return
	}

	data := s.newPageData(r, "Mis Reservas")
	data.Data = bookings
	s.render(w, r, "pages/customer/bookings.html", data)
}

// handleNewBookingPage shows the new booking form
func (s *Server) handleNewBookingPage(w http.ResponseWriter, r *http.Request) {
	ctx := r.Context()

	services, _ := s.repos.Services.List(ctx)
	brands, _ := s.repos.Brands.List(ctx)
	models, _ := s.repos.Models.List(ctx)

	// Get user's bicycles
	claims := getUserClaims(r)
	bicycles, _ := s.repos.Bicycles.GetByUserID(ctx, claims.UserID)

	data := s.newPageData(r, "Nueva Reserva")
	data.Data = map[string]interface{}{
		"Services": services,
		"Brands":   brands,
		"Models":   models,
		"Bicycles": bicycles,
	}
	s.render(w, r, "pages/customer/booking_new.html", data)
}

// handleCreateBooking creates a new booking
func (s *Server) handleCreateBooking(w http.ResponseWriter, r *http.Request) {
	claims := getUserClaims(r)
	ctx := r.Context()

	if err := r.ParseForm(); err != nil {
		http.Error(w, "Error processing form", http.StatusBadRequest)
		return
	}

	serviceID, _ := strconv.ParseInt(r.FormValue("service_id"), 10, 64)
	bicycleID, _ := strconv.ParseInt(r.FormValue("bicycle_id"), 10, 64)

	// Handle new bicycle creation if selected
	if r.FormValue("new_bicycle") == "true" {
		brandID, _ := strconv.ParseInt(r.FormValue("brand_id"), 10, 64)
		modelID, _ := strconv.ParseInt(r.FormValue("model_id"), 10, 64)
		color := r.FormValue("color")
		serial := r.FormValue("serial_number")

		newBike := &domain.Bicycle{
			UserID:       claims.UserID,
			BrandID:      brandID,
			ModelID:      modelID,
			Color:        color,
			SerialNumber: serial,
		}

		if err := s.repos.Bicycles.Create(ctx, newBike); err != nil {
			http.Error(w, "Error creating bicycle", http.StatusInternalServerError)
			return
		}
		bicycleID = newBike.ID
	}

	dateStr := r.FormValue("date")
	timeStr := r.FormValue("time")
	notes := r.FormValue("notes")

	// Parse date and time
	scheduledAt, err := time.Parse("2006-01-02 15:04", dateStr+" "+timeStr)
	if err != nil {
		data := s.newPageData(r, "Nueva Reserva")
		data.Flash = &FlashMessage{Type: "error", Message: "Fecha u hora inválida"}
		s.render(w, r, "pages/customer/booking_new.html", data)
		return
	}

	booking := &domain.Booking{
		CustomerID:  claims.UserID,
		BicycleID:   bicycleID,
		ServiceID:   serviceID,
		ScheduledAt: scheduledAt,
		Status:      domain.BookingStatusPending,
		Notes:       notes,
	}

	if err := s.repos.Bookings.Create(ctx, booking); err != nil {
		http.Error(w, "Error creating booking", http.StatusInternalServerError)
		return
	}

	http.Redirect(w, r, "/bookings", http.StatusSeeOther)
}

// handleBookingDetail shows booking details
func (s *Server) handleBookingDetail(w http.ResponseWriter, r *http.Request) {
	claims := getUserClaims(r)
	ctx := r.Context()

	id, _ := strconv.ParseInt(getURLParam(r, "id"), 10, 64)
	booking, err := s.repos.Bookings.GetByID(ctx, id)
	if err != nil || booking == nil {
		http.NotFound(w, r)
		return
	}

	// Security check - customer can only see their own bookings
	if claims.Role == domain.RoleCustomer && booking.CustomerID != claims.UserID {
		http.Error(w, "Forbidden", http.StatusForbidden)
		return
	}

	// Get associated quote if exists
	quote, _ := s.repos.Quotes.GetByBookingID(ctx, id)

	data := s.newPageData(r, "Detalle de Reserva")
	data.Data = map[string]interface{}{
		"Booking": booking,
		"Quote":   quote,
	}
	s.render(w, r, "pages/customer/booking_detail.html", data)
}

// handleCancelBooking cancels a booking
func (s *Server) handleCancelBooking(w http.ResponseWriter, r *http.Request) {
	claims := getUserClaims(r)
	ctx := r.Context()

	id, _ := strconv.ParseInt(getURLParam(r, "id"), 10, 64)
	booking, err := s.repos.Bookings.GetByID(ctx, id)
	if err != nil || booking == nil {
		http.NotFound(w, r)
		return
	}

	// Security check
	if claims.Role == domain.RoleCustomer && booking.CustomerID != claims.UserID {
		http.Error(w, "Forbidden", http.StatusForbidden)
		return
	}

	if err := s.repos.Bookings.UpdateStatus(ctx, id, domain.BookingStatusCancelled); err != nil {
		http.Error(w, "Error cancelling booking", http.StatusInternalServerError)
		return
	}

	http.Redirect(w, r, "/bookings", http.StatusSeeOther)
}

// handleQuotesList shows customer quotes
func (s *Server) handleQuotesList(w http.ResponseWriter, r *http.Request) {
	ctx := r.Context()

	quotes, err := s.repos.Quotes.List(ctx, "", 50, 0)
	if err != nil {
		http.Error(w, "Error loading quotes", http.StatusInternalServerError)
		return
	}

	data := s.newPageData(r, "Mis Presupuestos")
	data.Data = map[string]interface{}{"Quotes": quotes}
	s.render(w, r, "pages/customer/quotes.html", data)
}

// handleQuoteDetail shows quote details
func (s *Server) handleQuoteDetail(w http.ResponseWriter, r *http.Request) {
	ctx := r.Context()

	id, _ := strconv.ParseInt(getURLParam(r, "id"), 10, 64)
	quote, err := s.repos.Quotes.GetByID(ctx, id)
	if err != nil || quote == nil {
		http.NotFound(w, r)
		return
	}

	data := s.newPageData(r, "Detalle de Presupuesto")
	data.Data = map[string]interface{}{"Quote": quote}
	s.render(w, r, "pages/customer/quote_detail.html", data)
}

// handleApproveQuote approves a quote
func (s *Server) handleApproveQuote(w http.ResponseWriter, r *http.Request) {
	ctx := r.Context()

	id, _ := strconv.ParseInt(getURLParam(r, "id"), 10, 64)
	if err := s.repos.Quotes.Approve(ctx, id); err != nil {
		http.Error(w, "Error approving quote", http.StatusInternalServerError)
		return
	}

	http.Redirect(w, r, "/quotes/"+getURLParam(r, "id"), http.StatusSeeOther)
}

// handleRejectQuote rejects a quote
func (s *Server) handleRejectQuote(w http.ResponseWriter, r *http.Request) {
	ctx := r.Context()

	if err := r.ParseForm(); err != nil {
		http.Error(w, "Error processing form", http.StatusBadRequest)
		return
	}

	id, _ := strconv.ParseInt(getURLParam(r, "id"), 10, 64)
	reason := r.FormValue("reason")

	if err := s.repos.Quotes.Reject(ctx, id, reason); err != nil {
		http.Error(w, "Error rejecting quote", http.StatusInternalServerError)
		return
	}

	http.Redirect(w, r, "/quotes", http.StatusSeeOther)
}

// handleProfile shows user profile
func (s *Server) handleProfile(w http.ResponseWriter, r *http.Request) {
	claims := getUserClaims(r)
	ctx := r.Context()

	user, err := s.repos.Users.GetByID(ctx, claims.UserID)
	if err != nil || user == nil {
		http.Error(w, "User not found", http.StatusInternalServerError)
		return
	}

	data := s.newPageData(r, "Mi Perfil")
	data.Data = map[string]interface{}{"User": user}
	s.render(w, r, "pages/customer/profile.html", data)
}

// handleUpdateProfile updates user profile
func (s *Server) handleUpdateProfile(w http.ResponseWriter, r *http.Request) {
	claims := getUserClaims(r)
	ctx := r.Context()

	if err := r.ParseForm(); err != nil {
		http.Error(w, "Error processing form", http.StatusBadRequest)
		return
	}

	user, _ := s.repos.Users.GetByID(ctx, claims.UserID)
	if user == nil {
		http.Error(w, "User not found", http.StatusInternalServerError)
		return
	}

	user.Name = r.FormValue("name")
	user.Phone = r.FormValue("phone")

	if err := s.repos.Users.Update(ctx, user); err != nil {
		http.Error(w, "Error updating profile", http.StatusInternalServerError)
		return
	}

	data := s.newPageData(r, "Mi Perfil")
	data.Data = map[string]interface{}{"User": user}
	data.Flash = &FlashMessage{Type: "success", Message: "Perfil actualizado"}
	s.render(w, r, "pages/customer/profile.html", data)
}

// handleSurveyPage shows the survey form
func (s *Server) handleSurveyPage(w http.ResponseWriter, r *http.Request) {
	ctx := r.Context()

	ticketID, _ := strconv.ParseInt(getURLParam(r, "ticketId"), 10, 64)
	ticket, err := s.repos.Tickets.GetByID(ctx, ticketID)
	if err != nil || ticket == nil {
		http.NotFound(w, r)
		return
	}

	// Check if survey already exists
	existingSurvey, _ := s.repos.Surveys.GetByTicketID(ctx, ticketID)
	if existingSurvey != nil {
		data := s.newPageData(r, "Encuesta ya completada")
		data.Flash = &FlashMessage{Type: "info", Message: "Ya has completado esta encuesta"}
		s.render(w, r, "pages/customer/survey_completed.html", data)
		return
	}

	data := s.newPageData(r, "Encuesta de Satisfacción")
	data.Data = ticket
	s.render(w, r, "pages/customer/survey.html", data)
}

// handleSubmitSurvey submits a survey
func (s *Server) handleSubmitSurvey(w http.ResponseWriter, r *http.Request) {
	ctx := r.Context()

	if err := r.ParseForm(); err != nil {
		http.Error(w, "Error processing form", http.StatusBadRequest)
		return
	}

	ticketID, _ := strconv.ParseInt(getURLParam(r, "ticketId"), 10, 64)
	rating, _ := strconv.Atoi(r.FormValue("rating"))
	feedback := r.FormValue("feedback")

	survey := &domain.Survey{
		TicketID: ticketID,
		Rating:   rating,
		Feedback: feedback,
	}

	if err := s.repos.Surveys.Create(ctx, survey); err != nil {
		http.Error(w, "Error submitting survey", http.StatusInternalServerError)
		return
	}

	data := s.newPageData(r, "¡Gracias!")
	data.Flash = &FlashMessage{Type: "success", Message: "¡Gracias por tu opinión!"}
	s.render(w, r, "pages/customer/survey_completed.html", data)
}

// Technician/Workshop handlers

// handleWorkshopDashboard shows technician dashboard
func (s *Server) handleWorkshopDashboard(w http.ResponseWriter, r *http.Request) {
	ctx := r.Context()

	// Get ticket counts by status
	statusCounts, _ := s.repos.Tickets.CountByStatus(ctx)

	// Get recent tickets
	tickets, _ := s.repos.Tickets.List(ctx, "", 10, 0)

	// Get pending bookings
	pendingBookings, _ := s.repos.Bookings.List(ctx, domain.BookingStatusPending, 10, 0)

	data := s.newPageData(r, "Panel de Taller")
	data.Data = map[string]interface{}{
		"StatusCounts":    statusCounts,
		"RecentTickets":   tickets,
		"PendingBookings": pendingBookings,
	}
	s.render(w, r, "pages/technician/dashboard.html", data)
}

// handleTicketsList shows all tickets
func (s *Server) handleTicketsList(w http.ResponseWriter, r *http.Request) {
	ctx := r.Context()

	status := r.URL.Query().Get("status")
	tickets, err := s.repos.Tickets.List(ctx, status, 50, 0)
	if err != nil {
		http.Error(w, "Error loading tickets", http.StatusInternalServerError)
		return
	}

	data := s.newPageData(r, "Órdenes de Trabajo")
	data.Data = map[string]interface{}{
		"Tickets":       tickets,
		"CurrentStatus": status,
	}
	s.render(w, r, "pages/technician/tickets.html", data)
}

// handleTicketDetail shows ticket details
func (s *Server) handleTicketDetail(w http.ResponseWriter, r *http.Request) {
	ctx := r.Context()

	id, _ := strconv.ParseInt(getURLParam(r, "id"), 10, 64)
	ticket, err := s.repos.Tickets.GetByID(ctx, id)
	if err != nil || ticket == nil {
		http.NotFound(w, r)
		return
	}

	// Get booking details
	booking, _ := s.repos.Bookings.GetByID(ctx, ticket.BookingID)

	// Get bicycle details if present
	if booking != nil && booking.BicycleID != 0 {
		booking.Bicycle, _ = s.repos.Bicycles.GetByID(ctx, booking.BicycleID)
	}

	// Get status history
	history, _ := s.repos.Tickets.GetStatusHistory(ctx, id)

	// Get ticket parts
	parts, _ := s.repos.Tickets.GetTicketParts(ctx, id)

	// Get quote if exists
	quote, _ := s.repos.Quotes.GetByBookingID(ctx, ticket.BookingID)

	data := s.newPageData(r, "Orden de Trabajo #"+ticket.TrackingCode)

	// Check for errors
	errorType := r.URL.Query().Get("error")
	if errorType == "invalid_transition" {
		data.Flash = &FlashMessage{Type: "error", Message: "No puedes cambiar a ese estado (solo avance permitido)"}
	} else if errorType == "update_failed" {
		data.Flash = &FlashMessage{Type: "error", Message: "Error al actualizar el estado"}
	}

	// Get technicians list for admin assignment
	claims := getUserClaims(r)
	var technicians []domain.User
	if claims.Role == domain.RoleAdmin {
		technicians, _ = s.repos.Users.List(ctx, domain.RoleTechnician, 100, 0)
	}

	data.Data = map[string]interface{}{
		"Ticket":        ticket,
		"Booking":       booking,
		"StatusHistory": history,
		"Parts":         parts,
		"Quote":         quote,
		"Technicians":   technicians,
	}
	s.render(w, r, "pages/technician/ticket_detail.html", data)
}

// handleUpdateTicketStatus updates ticket status
func (s *Server) handleUpdateTicketStatus(w http.ResponseWriter, r *http.Request) {
	ctx := r.Context()

	if err := r.ParseForm(); err != nil {
		http.Error(w, "Error processing form", http.StatusBadRequest)
		return
	}

	claims := getUserClaims(r)
	id, _ := strconv.ParseInt(getURLParam(r, "id"), 10, 64)
	status := r.FormValue("status")
	notes := r.FormValue("notes") // Optional notes for the status change

	// Fetch ticket to check current status for permissions
	ticket, err := s.repos.Tickets.GetByID(ctx, id)
	if err != nil || ticket == nil {
		http.NotFound(w, r)
		return
	}

	// Security Check: Technician can only edit assigned tickets
	if claims.Role == domain.RoleTechnician && ticket.TechnicianID != claims.UserID {
		http.Error(w, "Forbidden: You are not assigned to this ticket", http.StatusForbidden)
		return
	}

	// Restrict state changes for technicians
	if claims.Role == domain.RoleTechnician {
		valid := false
		// Define allowed transitions
		switch ticket.Status {
		case domain.TicketStatusReceived:
			if status == domain.TicketStatusDiagnosing {
				valid = true
			}
		case domain.TicketStatusDiagnosing:
			if status == domain.TicketStatusInProgress || status == domain.TicketStatusWaitingParts || status == domain.TicketStatusReady {
				valid = true
			}
		case domain.TicketStatusInProgress:
			if status == domain.TicketStatusWaitingParts || status == domain.TicketStatusReady {
				valid = true
			}
		case domain.TicketStatusWaitingParts:
			if status == domain.TicketStatusInProgress || status == domain.TicketStatusReady {
				valid = true
			}
		case domain.TicketStatusReady:
			if status == domain.TicketStatusDelivered {
				valid = true
			}
		case domain.TicketStatusDelivered:
			// No changes allowed from delivered
			valid = false
		default:
			// Allow initial transition if status is somehow unknown or empty?
			// Assuming "received" is the start, but let's be strict.
			// checking if just updating notes (same status)
			if ticket.Status == status {
				valid = true
			}
		}

		// Allow keeping same status (for adding notes)
		if ticket.Status == status {
			valid = true
		}

		if !valid {
			// Redirect back with error
			http.Redirect(w, r, "/tickets/"+strconv.FormatInt(id, 10)+"?error=invalid_transition", http.StatusSeeOther)
			return
		}
	}

	if err := s.repos.Tickets.UpdateStatus(ctx, id, status, claims.UserID, notes); err != nil {
		http.Redirect(w, r, "/tickets/"+strconv.FormatInt(id, 10)+"?error=update_failed", http.StatusSeeOther)
		return
	}

	http.Redirect(w, r, "/tickets/"+getURLParam(r, "id"), http.StatusSeeOther)
}

// handleAddTicketNotes adds notes to a ticket
func (s *Server) handleAddTicketNotes(w http.ResponseWriter, r *http.Request) {
	ctx := r.Context()

	if err := r.ParseForm(); err != nil {
		http.Error(w, "Error processing form", http.StatusBadRequest)
		return
	}

	id, _ := strconv.ParseInt(getURLParam(r, "id"), 10, 64)
	ticket, _ := s.repos.Tickets.GetByID(ctx, id)
	if ticket == nil {
		http.NotFound(w, r)
		return
	}

	// Security Check: Technician can only edit assigned tickets
	claims := getUserClaims(r)
	if claims.Role == domain.RoleTechnician && ticket.TechnicianID != claims.UserID {
		http.Error(w, "Forbidden: You are not assigned to this ticket", http.StatusForbidden)
		return
	}

	ticket.Notes = r.FormValue("notes")
	if err := s.repos.Tickets.Update(ctx, ticket); err != nil {
		http.Error(w, "Error updating ticket", http.StatusInternalServerError)
		return
	}

	http.Redirect(w, r, "/tickets/"+getURLParam(r, "id"), http.StatusSeeOther)
}

// handleCreateTicket creates a ticket from a booking
func (s *Server) handleCreateTicket(w http.ResponseWriter, r *http.Request) {
	claims := getUserClaims(r)
	ctx := r.Context()

	bookingID, _ := strconv.ParseInt(getURLParam(r, "id"), 10, 64)
	booking, err := s.repos.Bookings.GetByID(ctx, bookingID)
	if err != nil || booking == nil {
		http.NotFound(w, r)
		return
	}

	// Generate tracking code
	trackingCode := generateTrackingCode()

	// Generate QR code
	baseURL := "http://localhost:" + strconv.Itoa(s.config.Server.Port)
	trackingURL := baseURL + "/tracking/" + trackingCode
	qrPNG, err := qrcode.Encode(trackingURL, qrcode.Medium, 256)
	if err != nil {
		http.Error(w, "Error generating QR code", http.StatusInternalServerError)
		return
	}

	ticket := &domain.Ticket{
		BookingID:    bookingID,
		TechnicianID: claims.UserID,
		TrackingCode: trackingCode,
		QRCode:       qrPNG,
		Status:       domain.TicketStatusReceived,
	}

	if err := s.repos.Tickets.Create(ctx, ticket); err != nil {
		http.Error(w, "Error creating ticket", http.StatusInternalServerError)
		return
	}

	// Update booking status
	s.repos.Bookings.UpdateStatus(ctx, bookingID, domain.BookingStatusConfirmed)

	http.Redirect(w, r, "/tickets/"+strconv.FormatInt(ticket.ID, 10), http.StatusSeeOther)
}

// handleNewQuotePage shows the new quote form
func (s *Server) handleNewQuotePage(w http.ResponseWriter, r *http.Request) {
	ctx := r.Context()

	bookingID, _ := strconv.ParseInt(getURLParam(r, "bookingId"), 10, 64)
	booking, err := s.repos.Bookings.GetByID(ctx, bookingID)
	if err != nil || booking == nil {
		http.NotFound(w, r)
		return
	}

	services, _ := s.repos.Services.List(ctx)

	// Get ticket ID from query param if available
	ticketID := r.URL.Query().Get("ticket_id")

	data := s.newPageData(r, "Nuevo Presupuesto")
	data.Data = map[string]interface{}{
		"Booking":  booking,
		"Services": services,
		"TicketID": ticketID,
	}
	s.render(w, r, "pages/technician/quote_new.html", data)
}

// handleCreateQuote creates a new quote
func (s *Server) handleCreateQuote(w http.ResponseWriter, r *http.Request) {
	ctx := r.Context()

	if err := r.ParseForm(); err != nil {
		http.Error(w, "Error processing form", http.StatusBadRequest)
		return
	}

	bookingID, _ := strconv.ParseInt(getURLParam(r, "bookingId"), 10, 64)

	// Parse quote items from form
	var items []domain.QuoteItem
	descriptions := r.Form["item_description[]"]
	quantities := r.Form["item_quantity[]"]
	prices := r.Form["item_price[]"]

	var total float64
	for i := range descriptions {
		qty, _ := strconv.Atoi(quantities[i])
		price, _ := strconv.ParseFloat(prices[i], 64)
		itemTotal := float64(qty) * price
		total += itemTotal

		items = append(items, domain.QuoteItem{
			Description: descriptions[i],
			Quantity:    qty,
			UnitPrice:   price,
			Total:       itemTotal,
		})
	}

	quote := &domain.Quote{
		BookingID:  bookingID,
		Items:      items,
		Total:      total,
		Status:     domain.QuoteStatusPending,
		ValidUntil: time.Now().AddDate(0, 0, 7), // 7 days validity
	}

	if err := s.repos.Quotes.Create(ctx, quote); err != nil {
		http.Error(w, "Error creating quote", http.StatusInternalServerError)
		return
	}

	ticketID := r.FormValue("ticket_id")
	if ticketID != "" {
		http.Redirect(w, r, "/tickets/"+ticketID+"?quote_created=true&quote_id="+strconv.FormatInt(quote.ID, 10), http.StatusSeeOther)
		return
	}

	http.Redirect(w, r, "/workshop", http.StatusSeeOther)
}

// generateTrackingCode generates a unique short tracking code
func generateTrackingCode() string {
	bytes := make([]byte, 4)
	rand.Read(bytes)
	return hex.EncodeToString(bytes)
}

// handleUpdateBicycle updates bicycle details
func (s *Server) handleUpdateBicycle(w http.ResponseWriter, r *http.Request) {
	ctx := r.Context()

	if err := r.ParseForm(); err != nil {
		http.Error(w, "Error processing form", http.StatusBadRequest)
		return
	}

	id, _ := strconv.ParseInt(getURLParam(r, "id"), 10, 64)
	bicycle, err := s.repos.Bicycles.GetByID(ctx, id)
	if err != nil || bicycle == nil {
		http.NotFound(w, r)
		return
	}

	// Update fields
	bicycle.Color = r.FormValue("color")
	bicycle.SerialNumber = r.FormValue("serial_number")
	bicycle.Notes = r.FormValue("notes")

	if err := s.repos.Bicycles.Update(ctx, bicycle); err != nil {
		http.Error(w, "Error updating bicycle", http.StatusInternalServerError)
		return
	}

	redirectTo := r.FormValue("redirect_to")
	if redirectTo == "" {
		redirectTo = "/workshop"
	}
	http.Redirect(w, r, redirectTo, http.StatusSeeOther)
}

// handleCreateTicketPart adds a new part/item to the ticket checklist
func (s *Server) handleCreateTicketPart(w http.ResponseWriter, r *http.Request) {
	ticketID, _ := strconv.ParseInt(getURLParam(r, "id"), 10, 64)
	name := r.FormValue("name")

	if name == "" {
		http.Redirect(w, r, fmt.Sprintf("/tickets/%d", ticketID), http.StatusSeeOther)
		return
	}

	// Security Check
	claims := getUserClaims(r)
	if claims.Role == domain.RoleTechnician {
		ticket, _ := s.repos.Tickets.GetByID(r.Context(), ticketID)
		if ticket != nil && ticket.TechnicianID != claims.UserID {
			http.Error(w, "Forbidden", http.StatusForbidden)
			return
		}
	}

	part := &domain.TicketPart{
		TicketID: ticketID,
		Name:     name,
	}

	if err := s.repos.Tickets.CreateTicketPart(r.Context(), part); err != nil {
		// Log error
		fmt.Printf("Error creating ticket part: %v\n", err)
	}

	http.Redirect(w, r, fmt.Sprintf("/tickets/%d", ticketID), http.StatusSeeOther)
}

// handleToggleTicketPart toggles the status of a ticket part
func (s *Server) handleToggleTicketPart(w http.ResponseWriter, r *http.Request) {
	ticketID, _ := strconv.ParseInt(getURLParam(r, "id"), 10, 64)
	partID, _ := strconv.ParseInt(getURLParam(r, "partId"), 10, 64)

	// Security Check
	claims := getUserClaims(r)
	if claims.Role == domain.RoleTechnician {
		ticket, _ := s.repos.Tickets.GetByID(r.Context(), ticketID)
		if ticket != nil && ticket.TechnicianID != claims.UserID {
			http.Error(w, "Forbidden", http.StatusForbidden)
			return
		}
	}

	if err := s.repos.Tickets.ToggleTicketPartStatus(r.Context(), partID); err != nil {
		fmt.Printf("Error toggling part: %v\n", err)
	}

	// Return generic 200 OK for AJAX or redirect
	if r.Header.Get("HX-Request") != "" {
		w.WriteHeader(http.StatusOK)
		return
	}
	http.Redirect(w, r, fmt.Sprintf("/tickets/%d", ticketID), http.StatusSeeOther)
}

// handleDeleteTicketPart deletes a ticket part
func (s *Server) handleDeleteTicketPart(w http.ResponseWriter, r *http.Request) {
	ticketID, _ := strconv.ParseInt(getURLParam(r, "id"), 10, 64)
	partID, _ := strconv.ParseInt(getURLParam(r, "partId"), 10, 64)

	// Security Check
	claims := getUserClaims(r)
	if claims.Role == domain.RoleTechnician {
		ticket, _ := s.repos.Tickets.GetByID(r.Context(), ticketID)
		if ticket != nil && ticket.TechnicianID != claims.UserID {
			http.Error(w, "Forbidden", http.StatusForbidden)
			return
		}
	}

	if err := s.repos.Tickets.DeleteTicketPart(r.Context(), partID); err != nil {
		fmt.Printf("Error deleting part: %v\n", err)
	}

	http.Redirect(w, r, fmt.Sprintf("/tickets/%d", ticketID), http.StatusSeeOther)
}

// handleCreateBicycleFromBooking creates a new bicycle and links it to the booking
func (s *Server) handleCreateBicycleFromBooking(w http.ResponseWriter, r *http.Request) {
	bookingID, _ := strconv.ParseInt(getURLParam(r, "id"), 10, 64)

	// Parse form
	if err := r.ParseForm(); err != nil {
		http.Error(w, "Bad request", http.StatusBadRequest)
		return
	}

	booking, err := s.repos.Bookings.GetByID(r.Context(), bookingID)
	if err != nil {
		http.Error(w, "Booking not found", http.StatusNotFound)
		return
	}

	// Create Bicycle
	bicycle := &domain.Bicycle{
		UserID:       booking.CustomerID,
		Color:        r.FormValue("color"),
		SerialNumber: r.FormValue("serial_number"),
		Notes:        r.FormValue("notes"),
	}

	// Handle Brand/Model if passed (optional for quick registration)
	// For now we might just create it with basic info

	if err := s.repos.Bicycles.Create(r.Context(), bicycle); err != nil {
		http.Error(w, "Error creating bicycle", http.StatusInternalServerError)
		return
	}

	// Link to Booking
	booking.BicycleID = bicycle.ID
	if err := s.repos.Bookings.Update(r.Context(), booking); err != nil {
		http.Error(w, "Error linking bicycle", http.StatusInternalServerError)
		return
	}

	// Redirect back to ticket or booking
	redirectTo := r.FormValue("redirect_to")
	if redirectTo != "" {
		http.Redirect(w, r, redirectTo, http.StatusSeeOther)
		return
	}
	http.Redirect(w, r, "/workshop", http.StatusSeeOther)
}

// handleTicketLabel shows a printable label for the ticket
func (s *Server) handleTicketLabel(w http.ResponseWriter, r *http.Request) {
	id, _ := strconv.ParseInt(getURLParam(r, "id"), 10, 64)
	ticket, err := s.repos.Tickets.GetByID(r.Context(), id)
	if err != nil {
		http.NotFound(w, r)
		return
	}

	booking, _ := s.repos.Bookings.GetByID(r.Context(), ticket.BookingID)
	if booking != nil && booking.BicycleID != 0 {
		booking.Bicycle, _ = s.repos.Bicycles.GetByID(r.Context(), booking.BicycleID)
	}

	// Fetch customer if needed (booking has customer ID)
	if booking != nil && booking.CustomerID != 0 {
		booking.Customer, _ = s.repos.Users.GetByID(r.Context(), booking.CustomerID)
	}

	data := s.newPageData(r, "Etiqueta Taller #"+ticket.TrackingCode)
	data.Data = map[string]interface{}{
		"Ticket":  ticket,
		"Booking": booking,
	}

	s.render(w, r, "pages/technician/ticket_label.html", data)
}

// handleTicketQuote shows a printable quote for the ticket
func (s *Server) handleTicketQuote(w http.ResponseWriter, r *http.Request) {
	id, _ := strconv.ParseInt(getURLParam(r, "id"), 10, 64)
	ticket, err := s.repos.Tickets.GetByID(r.Context(), id)
	if err != nil {
		http.NotFound(w, r)
		return
	}

	booking, _ := s.repos.Bookings.GetByID(r.Context(), ticket.BookingID)
	if booking != nil {
		if booking.BicycleID != 0 {
			booking.Bicycle, _ = s.repos.Bicycles.GetByID(r.Context(), booking.BicycleID)
			if booking.Bicycle.BrandID != 0 {
				booking.Bicycle.Brand, _ = s.repos.Brands.GetByID(r.Context(), booking.Bicycle.BrandID)
			}
			if booking.Bicycle.ModelID != 0 {
				booking.Bicycle.Model, _ = s.repos.Models.GetByID(r.Context(), booking.Bicycle.ModelID)
			}
		}
		if booking.CustomerID != 0 {
			booking.Customer, _ = s.repos.Users.GetByID(r.Context(), booking.CustomerID)
		}
	}

	quote, err := s.repos.Quotes.GetByBookingID(r.Context(), ticket.BookingID)
	if err != nil || quote == nil {
		http.Error(w, "Presupuesto no encontrado", http.StatusNotFound)
		return
	}

	data := s.newPageData(r, "Presupuesto #"+strconv.FormatInt(quote.ID, 10))
	data.Data = map[string]interface{}{
		"Ticket":  ticket,
		"Booking": booking,
		"Quote":   quote,
	}

	s.render(w, r, "pages/technician/ticket_quote.html", data)
}

// handleNewTicketPage shows the direct ticket creation form
func (s *Server) handleNewTicketPage(w http.ResponseWriter, r *http.Request) {
	services, _ := s.repos.Services.List(r.Context())

	data := s.newPageData(r, "Nuevo Ticket")
	data.Data = map[string]interface{}{
		"Services": services,
	}

	s.render(w, r, "pages/technician/tickets_new.html", data)
}

// handleCreateTicketDirect handles the unified form for checking/creating user, bike, booking, and ticket
func (s *Server) handleCreateTicketDirect(w http.ResponseWriter, r *http.Request) {
	ctx := r.Context()

	if err := r.ParseForm(); err != nil {
		http.Error(w, "Bad request", http.StatusBadRequest)
		return
	}

	email := r.FormValue("email")
	name := r.FormValue("name")
	phone := r.FormValue("phone")

	// 1. Get or Create User
	user, err := s.repos.Users.GetByEmail(ctx, email)
	if err != nil {
		// Log error but proceed (might be just not found)
	}

	if user == nil {
		// Create new user
		// Generate placeholder password
		hashedPassword, _ := hashPassword("123456") // Simple default for walk-ins

		user = &domain.User{
			Email:        email,
			Name:         name,
			Phone:        phone,
			PasswordHash: hashedPassword,
			Role:         domain.RoleCustomer,
			CreatedAt:    time.Now(),
		}

		if err := s.repos.Users.Create(ctx, user); err != nil {
			http.Error(w, "Error creating user: "+err.Error(), http.StatusInternalServerError)
			return
		}
		// Fetch back to get ID (sqlite)
		user, _ = s.repos.Users.GetByEmail(ctx, email)
	}

	// 2. Create Bicycle (Always create new for this flow for now, or could check)
	// Simplified: Always create for this "quick" flow as per plan
	// Ideally we would search, but let's assume walk-in often brings the specific bike.
	// We can add "Select existing" later or if we had JS.

	// Check/Create Brand (Mocking dynamic creation or search would be better, but let's stick to simple text for now or existing brands)
	// The repo expects IDs for brands/models. The UI sends text.
	// We need logic to handle text input for Brand/Model.
	// For MVP Session 10: Let's check if Brand exists by name, if not create?
	// OR: Just store it as notes/text if we don't strictly enforce catalog?
	// The Bicycle entity requires BrandID/ModelID.
	// Let's quickly look up Brand by name (we need a repository method for that? or List and Iterate).
	// To keep it robust without tons of new repo methods:
	// We'll iterate all brands (cached or list) to find match.

	brands, _ := s.repos.Brands.List(ctx)
	var brandID int64
	inputBrand := strings.TrimSpace(r.FormValue("brand"))

	for _, b := range brands {
		if strings.EqualFold(b.Name, inputBrand) {
			brandID = b.ID
			break
		}
	}

	if brandID == 0 && inputBrand != "" {
		// Create Brand (Auto-learn)
		newBrand := &domain.Brand{Name: inputBrand}
		s.repos.Brands.Create(ctx, newBrand)
		brandID = newBrand.ID
	}

	// Same for Model
	var modelID int64
	inputModel := strings.TrimSpace(r.FormValue("model"))
	if brandID != 0 && inputModel != "" {
		models, _ := s.repos.Models.GetByBrandID(ctx, brandID)
		for _, m := range models {
			if strings.EqualFold(m.Name, inputModel) {
				modelID = m.ID
				break
			}
		}
		if modelID == 0 {
			newModel := &domain.Model{BrandID: brandID, Name: inputModel}
			s.repos.Models.Create(ctx, newModel)
			modelID = newModel.ID
		}
	}

	bicycle := &domain.Bicycle{
		UserID:       user.ID,
		BrandID:      brandID,
		ModelID:      modelID,
		Color:        r.FormValue("color"),
		SerialNumber: r.FormValue("serial"),
		Notes:        "Creado en recepción",
		CreatedAt:    time.Now(),
	}

	if err := s.repos.Bicycles.Create(ctx, bicycle); err != nil {
		fmt.Printf("Error creating bicycle: %v\n", err)
		// Proceed? Or Error? Let's error.
		http.Error(w, "Error creating bicycle", http.StatusInternalServerError)
		return
	}

	// 3. Create Booking (Confirmed, Now)
	serviceID, _ := strconv.ParseInt(r.FormValue("service_id"), 10, 64)
	booking := &domain.Booking{
		CustomerID:  user.ID,
		BicycleID:   bicycle.ID,
		ServiceID:   serviceID,
		ScheduledAt: time.Now(),
		Status:      domain.BookingStatusConfirmed,
		Notes:       r.FormValue("notes"),
		CreatedAt:   time.Now(),
	}

	if err := s.repos.Bookings.Create(ctx, booking); err != nil {
		http.Error(w, "Error creating booking", http.StatusInternalServerError)
		return
	}

	// 4. Create Ticket (Received)
	ticket := &domain.Ticket{
		BookingID:    booking.ID,
		TrackingCode: generateTrackingCode(), // We need to export or reuse this. It's unexported in snippets?
		// Actually generateTrackingCode is in handlers_protected.go but lower case?
		// I will assume it's available in package `server`.
		Status:    domain.TicketStatusReceived,
		Notes:     r.FormValue("notes"),
		CreatedAt: time.Now(),
		UpdatedAt: time.Now(),
	}

	// Create QR
	// Re-using logic from handleCreateTicket if possible, or copy-paste
	// Copy-pasting small QR logic to be safe and independent
	qrContent := fmt.Sprintf("https://bicicletapp.com/tracking/%s", ticket.TrackingCode)
	png, _ := qrcode.Encode(qrContent, qrcode.Medium, 256)
	ticket.QRCode = png
	ticket.QRCodeBase64 = base64.StdEncoding.EncodeToString(png)

	if err := s.repos.Tickets.Create(ctx, ticket); err != nil {
		http.Error(w, "Error creating ticket", http.StatusInternalServerError)
		return
	}

	// 5. Redirect
	http.Redirect(w, r, fmt.Sprintf("/tickets/%d", ticket.ID), http.StatusSeeOther)
}
