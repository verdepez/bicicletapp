package server

import (
	"encoding/json"
	"net/http"
	"strconv"
	"time"

	"bicicletapp/internal/domain"
)

// Admin handlers

// handleAdminDashboard shows admin dashboard with metrics
func (s *Server) handleAdminDashboard(w http.ResponseWriter, r *http.Request) {
	ctx := r.Context()

	// Get counts
	userCount, _ := s.repos.Users.Count(ctx, "")
	customerCount, _ := s.repos.Users.Count(ctx, domain.RoleCustomer)
	techCount, _ := s.repos.Users.Count(ctx, domain.RoleTechnician)

	bookingCount, _ := s.repos.Bookings.CountByStatus(ctx, "")
	pendingBookings, _ := s.repos.Bookings.CountByStatus(ctx, domain.BookingStatusPending)

	ticketCounts, _ := s.repos.Tickets.CountByStatus(ctx)

	// Get average rating
	thirtyDaysAgo := time.Now().AddDate(0, 0, -30)
	avgRating, _ := s.repos.Surveys.GetAverageRating(ctx, thirtyDaysAgo)

	data := s.newPageData(r, "Panel de Administración")
	data.Data = map[string]interface{}{
		"UserCount":       userCount,
		"CustomerCount":   customerCount,
		"TechnicianCount": techCount,
		"BookingCount":    bookingCount,
		"PendingBookings": pendingBookings,
		"TicketCounts":    ticketCounts,
		"AvgRating":       avgRating,
	}
	s.render(w, r, "pages/admin/dashboard.html", data)
}

// User management

func (s *Server) handleUsersList(w http.ResponseWriter, r *http.Request) {
	ctx := r.Context()

	role := r.URL.Query().Get("role")
	users, err := s.repos.Users.List(ctx, role, 100, 0)
	if err != nil {
		http.Error(w, "Error loading users", http.StatusInternalServerError)
		return
	}

	data := s.newPageData(r, "Gestión de Usuarios")
	data.Data = map[string]interface{}{
		"Users":       users,
		"CurrentRole": role,
	}
	s.render(w, r, "pages/admin/users.html", data)
}

func (s *Server) handleNewUserPage(w http.ResponseWriter, r *http.Request) {
	data := s.newPageData(r, "Nuevo Usuario")
	data.Data = map[string]interface{}{"User": nil}
	s.render(w, r, "pages/admin/user_form.html", data)
}

func (s *Server) handleCreateUser(w http.ResponseWriter, r *http.Request) {
	ctx := r.Context()

	if err := r.ParseForm(); err != nil {
		http.Error(w, "Error processing form", http.StatusBadRequest)
		return
	}

	email := r.FormValue("email")

	// Check if email already exists
	existingUser, _ := s.repos.Users.GetByEmail(ctx, email)
	if existingUser != nil {
		data := s.newPageData(r, "Nuevo Usuario")
		// Pass back the input data so user doesn't lose it
		data.Data = map[string]interface{}{
			"User": &domain.User{
				Name:  r.FormValue("name"),
				Email: email,
				Phone: r.FormValue("phone"),
				Role:  r.FormValue("role"),
			},
		}
		data.Flash = &FlashMessage{Type: "error", Message: "El email ya está registrado"}
		s.render(w, r, "pages/admin/user_form.html", data)
		return
	}

	password := r.FormValue("password")
	hashedPassword, err := hashPassword(password)
	if err != nil {
		http.Error(w, "Error processing password", http.StatusInternalServerError)
		return
	}

	user := &domain.User{
		Name:         r.FormValue("name"),
		Email:        email,
		Phone:        r.FormValue("phone"),
		Role:         r.FormValue("role"),
		PasswordHash: hashedPassword,
	}

	if err := s.repos.Users.Create(ctx, user); err != nil {
		http.Error(w, "Error creating user", http.StatusInternalServerError)
		return
	}

	http.Redirect(w, r, "/admin/users", http.StatusSeeOther)
}

func (s *Server) handleEditUserPage(w http.ResponseWriter, r *http.Request) {
	ctx := r.Context()

	id, _ := strconv.ParseInt(getURLParam(r, "id"), 10, 64)
	user, err := s.repos.Users.GetByID(ctx, id)
	if err != nil || user == nil {
		http.NotFound(w, r)
		return
	}

	data := s.newPageData(r, "Editar Usuario")
	data.Data = map[string]interface{}{"User": user}
	s.render(w, r, "pages/admin/user_form.html", data)
}

func (s *Server) handleUpdateUser(w http.ResponseWriter, r *http.Request) {
	ctx := r.Context()

	if err := r.ParseForm(); err != nil {
		http.Error(w, "Error processing form", http.StatusBadRequest)
		return
	}

	id, _ := strconv.ParseInt(getURLParam(r, "id"), 10, 64)
	user, _ := s.repos.Users.GetByID(ctx, id)
	if user == nil {
		http.NotFound(w, r)
		return
	}

	// Check for duplicate email if changed
	newEmail := r.FormValue("email")
	if newEmail != user.Email {
		existing, _ := s.repos.Users.GetByEmail(ctx, newEmail)
		if existing != nil {
			data := s.newPageData(r, "Editar Usuario")
			// Update user object with form values for re-rendering
			user.Name = r.FormValue("name")
			user.Email = newEmail
			user.Phone = r.FormValue("phone")
			user.Role = r.FormValue("role")

			data.Data = map[string]interface{}{"User": user}
			data.Flash = &FlashMessage{Type: "error", Message: "El email ya está registrado"}
			s.render(w, r, "pages/admin/user_form.html", data)
			return
		}
	}

	user.Name = r.FormValue("name")
	user.Email = newEmail
	user.Phone = r.FormValue("phone")
	user.Role = r.FormValue("role")

	// Update password if provided
	if newPassword := r.FormValue("password"); newPassword != "" {
		hashedPassword, err := hashPassword(newPassword)
		if err != nil {
			http.Error(w, "Error processing password", http.StatusInternalServerError)
			return
		}
		user.PasswordHash = hashedPassword
	}

	if err := s.repos.Users.Update(ctx, user); err != nil {
		http.Error(w, "Error updating user", http.StatusInternalServerError)
		return
	}

	http.Redirect(w, r, "/admin/users", http.StatusSeeOther)
}

func (s *Server) handleDeleteUser(w http.ResponseWriter, r *http.Request) {
	ctx := r.Context()

	id, _ := strconv.ParseInt(getURLParam(r, "id"), 10, 64)
	if err := s.repos.Users.Delete(ctx, id); err != nil {
		http.Error(w, "Error deleting user", http.StatusInternalServerError)
		return
	}

	http.Redirect(w, r, "/admin/users", http.StatusSeeOther)
}

// Catalog management - Brands

func (s *Server) handleBrandsList(w http.ResponseWriter, r *http.Request) {
	ctx := r.Context()

	brands, err := s.repos.Brands.List(ctx)
	if err != nil {
		http.Error(w, "Error loading brands", http.StatusInternalServerError)
		return
	}

	data := s.newPageData(r, "Gestión de Marcas")
	data.Data = brands
	s.render(w, r, "pages/admin/brands.html", data)
}

func (s *Server) handleNewBrandPage(w http.ResponseWriter, r *http.Request) {
	data := s.newPageData(r, "Nueva Marca")
	data.Data = map[string]interface{}{"Brand": nil}
	s.render(w, r, "pages/admin/brand_form.html", data)
}

func (s *Server) handleEditBrandPage(w http.ResponseWriter, r *http.Request) {
	ctx := r.Context()

	id, _ := strconv.ParseInt(getURLParam(r, "id"), 10, 64)
	brand, err := s.repos.Brands.GetByID(ctx, id)
	if err != nil || brand == nil {
		http.NotFound(w, r)
		return
	}

	data := s.newPageData(r, "Editar Marca")
	data.Data = map[string]interface{}{"Brand": brand}
	s.render(w, r, "pages/admin/brand_form.html", data)
}

func (s *Server) handleCreateBrand(w http.ResponseWriter, r *http.Request) {
	ctx := r.Context()

	if err := r.ParseForm(); err != nil {
		http.Error(w, "Error processing form", http.StatusBadRequest)
		return
	}

	brand := &domain.Brand{
		Name:    r.FormValue("name"),
		LogoURL: r.FormValue("logo_url"),
	}

	if err := s.repos.Brands.Create(ctx, brand); err != nil {
		http.Error(w, "Error creating brand", http.StatusInternalServerError)
		return
	}

	http.Redirect(w, r, "/admin/brands", http.StatusSeeOther)
}

func (s *Server) handleUpdateBrand(w http.ResponseWriter, r *http.Request) {
	ctx := r.Context()

	if err := r.ParseForm(); err != nil {
		http.Error(w, "Error processing form", http.StatusBadRequest)
		return
	}

	id, _ := strconv.ParseInt(getURLParam(r, "id"), 10, 64)
	brand, _ := s.repos.Brands.GetByID(ctx, id)
	if brand == nil {
		http.NotFound(w, r)
		return
	}

	brand.Name = r.FormValue("name")
	brand.LogoURL = r.FormValue("logo_url")

	if err := s.repos.Brands.Update(ctx, brand); err != nil {
		http.Error(w, "Error updating brand", http.StatusInternalServerError)
		return
	}

	http.Redirect(w, r, "/admin/brands", http.StatusSeeOther)
}

func (s *Server) handleDeleteBrand(w http.ResponseWriter, r *http.Request) {
	ctx := r.Context()

	id, _ := strconv.ParseInt(getURLParam(r, "id"), 10, 64)
	if err := s.repos.Brands.Delete(ctx, id); err != nil {
		http.Error(w, "Error deleting brand", http.StatusInternalServerError)
		return
	}

	http.Redirect(w, r, "/admin/brands", http.StatusSeeOther)
}

// Catalog management - Models

func (s *Server) handleModelsList(w http.ResponseWriter, r *http.Request) {
	ctx := r.Context()

	models, err := s.repos.Models.List(ctx)
	if err != nil {
		http.Error(w, "Error loading models", http.StatusInternalServerError)
		return
	}

	brands, _ := s.repos.Brands.List(ctx)

	data := s.newPageData(r, "Gestión de Modelos")
	data.Data = map[string]interface{}{
		"Models": models,
		"Brands": brands,
	}
	s.render(w, r, "pages/admin/models.html", data)
}

func (s *Server) handleNewModelPage(w http.ResponseWriter, r *http.Request) {
	ctx := r.Context()

	brands, _ := s.repos.Brands.List(ctx)

	data := s.newPageData(r, "Nuevo Modelo")
	data.Data = map[string]interface{}{
		"Model":  nil,
		"Brands": brands,
	}
	s.render(w, r, "pages/admin/model_form.html", data)
}

func (s *Server) handleEditModelPage(w http.ResponseWriter, r *http.Request) {
	ctx := r.Context()

	id, _ := strconv.ParseInt(getURLParam(r, "id"), 10, 64)
	model, err := s.repos.Models.GetByID(ctx, id)
	if err != nil || model == nil {
		http.NotFound(w, r)
		return
	}

	brands, _ := s.repos.Brands.List(ctx)

	data := s.newPageData(r, "Editar Modelo")
	data.Data = map[string]interface{}{
		"Model":  model,
		"Brands": brands,
	}
	s.render(w, r, "pages/admin/model_form.html", data)
}

func (s *Server) handleCreateModel(w http.ResponseWriter, r *http.Request) {
	ctx := r.Context()

	if err := r.ParseForm(); err != nil {
		http.Error(w, "Error processing form", http.StatusBadRequest)
		return
	}

	brandID, _ := strconv.ParseInt(r.FormValue("brand_id"), 10, 64)

	model := &domain.Model{
		BrandID: brandID,
		Name:    r.FormValue("name"),
	}

	if err := s.repos.Models.Create(ctx, model); err != nil {
		http.Error(w, "Error creating model", http.StatusInternalServerError)
		return
	}

	http.Redirect(w, r, "/admin/models", http.StatusSeeOther)
}

func (s *Server) handleUpdateModel(w http.ResponseWriter, r *http.Request) {
	ctx := r.Context()

	if err := r.ParseForm(); err != nil {
		http.Error(w, "Error processing form", http.StatusBadRequest)
		return
	}

	id, _ := strconv.ParseInt(getURLParam(r, "id"), 10, 64)
	model, _ := s.repos.Models.GetByID(ctx, id)
	if model == nil {
		http.NotFound(w, r)
		return
	}

	brandID, _ := strconv.ParseInt(r.FormValue("brand_id"), 10, 64)
	model.BrandID = brandID
	model.Name = r.FormValue("name")

	if err := s.repos.Models.Update(ctx, model); err != nil {
		http.Error(w, "Error updating model", http.StatusInternalServerError)
		return
	}

	http.Redirect(w, r, "/admin/models", http.StatusSeeOther)
}

func (s *Server) handleDeleteModel(w http.ResponseWriter, r *http.Request) {
	ctx := r.Context()

	id, _ := strconv.ParseInt(getURLParam(r, "id"), 10, 64)
	if err := s.repos.Models.Delete(ctx, id); err != nil {
		http.Error(w, "Error deleting model", http.StatusInternalServerError)
		return
	}

	http.Redirect(w, r, "/admin/models", http.StatusSeeOther)
}

// Catalog management - Services

func (s *Server) handleServicesList(w http.ResponseWriter, r *http.Request) {
	ctx := r.Context()

	services, err := s.repos.Services.List(ctx)
	if err != nil {
		http.Error(w, "Error loading services", http.StatusInternalServerError)
		return
	}

	data := s.newPageData(r, "Gestión de Servicios")
	data.Data = services
	s.render(w, r, "pages/admin/services.html", data)
}

func (s *Server) handleNewServicePage(w http.ResponseWriter, r *http.Request) {
	data := s.newPageData(r, "Nuevo Servicio")
	data.Data = map[string]interface{}{"Service": nil}
	s.render(w, r, "pages/admin/service_form.html", data)
}

func (s *Server) handleCreateService(w http.ResponseWriter, r *http.Request) {
	ctx := r.Context()

	if err := r.ParseForm(); err != nil {
		http.Error(w, "Error processing form", http.StatusBadRequest)
		return
	}

	basePrice, _ := strconv.ParseFloat(r.FormValue("base_price"), 64)
	estimatedHours, _ := strconv.ParseFloat(r.FormValue("estimated_hours"), 64)

	service := &domain.Service{
		Name:           r.FormValue("name"),
		Description:    r.FormValue("description"),
		BasePrice:      basePrice,
		EstimatedHours: estimatedHours,
	}

	if err := s.repos.Services.Create(ctx, service); err != nil {
		http.Error(w, "Error creating service", http.StatusInternalServerError)
		return
	}

	http.Redirect(w, r, "/admin/services", http.StatusSeeOther)
}

func (s *Server) handleEditServicePage(w http.ResponseWriter, r *http.Request) {
	ctx := r.Context()

	id, _ := strconv.ParseInt(getURLParam(r, "id"), 10, 64)
	service, err := s.repos.Services.GetByID(ctx, id)
	if err != nil || service == nil {
		http.NotFound(w, r)
		return
	}

	data := s.newPageData(r, "Editar Servicio")
	data.Data = map[string]interface{}{"Service": service}
	s.render(w, r, "pages/admin/service_form.html", data)
}

func (s *Server) handleUpdateService(w http.ResponseWriter, r *http.Request) {
	ctx := r.Context()

	if err := r.ParseForm(); err != nil {
		http.Error(w, "Error processing form", http.StatusBadRequest)
		return
	}

	id, _ := strconv.ParseInt(getURLParam(r, "id"), 10, 64)
	service, _ := s.repos.Services.GetByID(ctx, id)
	if service == nil {
		http.NotFound(w, r)
		return
	}

	service.Name = r.FormValue("name")
	service.Description = r.FormValue("description")
	service.BasePrice, _ = strconv.ParseFloat(r.FormValue("base_price"), 64)
	service.EstimatedHours, _ = strconv.ParseFloat(r.FormValue("estimated_hours"), 64)

	if err := s.repos.Services.Update(ctx, service); err != nil {
		http.Error(w, "Error updating service", http.StatusInternalServerError)
		return
	}

	http.Redirect(w, r, "/admin/services", http.StatusSeeOther)
}

func (s *Server) handleDeleteService(w http.ResponseWriter, r *http.Request) {
	ctx := r.Context()

	id, _ := strconv.ParseInt(getURLParam(r, "id"), 10, 64)
	if err := s.repos.Services.Delete(ctx, id); err != nil {
		http.Error(w, "Error deleting service", http.StatusInternalServerError)
		return
	}

	http.Redirect(w, r, "/admin/services", http.StatusSeeOther)
}

// Reports

func (s *Server) handleReportsDashboard(w http.ResponseWriter, r *http.Request) {
	ctx := r.Context()

	// Get this month's bookings count
	now := time.Now()
	startOfMonth := time.Date(now.Year(), now.Month(), 1, 0, 0, 0, 0, now.Location())
	monthlyBookings, _ := s.repos.Bookings.GetByDateRange(ctx, startOfMonth, now)

	// Get ticket counts
	ticketCounts, _ := s.repos.Tickets.CountByStatus(ctx)

	// Get average rating
	thirtyDaysAgo := now.AddDate(0, 0, -30)
	avgRating, _ := s.repos.Surveys.GetAverageRating(ctx, thirtyDaysAgo)

	// Get approved quotes for revenue
	quotes, _ := s.repos.Quotes.List(ctx, domain.QuoteStatusApproved, 100, 0)
	var totalRevenue float64
	for _, q := range quotes {
		totalRevenue += q.Total
	}

	data := s.newPageData(r, "Reportes")
	data.Data = map[string]interface{}{
		"MonthlyBookings": len(monthlyBookings),
		"TicketCounts":    ticketCounts,
		"AvgRating":       avgRating,
		"TotalRevenue":    totalRevenue,
		"CurrentMonth":    now.Month().String(),
	}
	s.render(w, r, "pages/admin/reports.html", data)
}

func (s *Server) handleBookingsReport(w http.ResponseWriter, r *http.Request) {
	ctx := r.Context()

	// Get bookings for last 30 days
	endDate := time.Now()
	startDate := endDate.AddDate(0, 0, -30)

	bookings, _ := s.repos.Bookings.GetByDateRange(ctx, startDate, endDate)

	data := s.newPageData(r, "Reporte de Reservas")
	data.Data = map[string]interface{}{
		"Bookings":  bookings,
		"StartDate": startDate,
		"EndDate":   endDate,
	}
	s.render(w, r, "pages/admin/report_bookings.html", data)
}

func (s *Server) handleRevenueReport(w http.ResponseWriter, r *http.Request) {
	ctx := r.Context()

	// Get approved quotes for revenue calculation
	quotes, _ := s.repos.Quotes.List(ctx, domain.QuoteStatusApproved, 100, 0)

	var totalRevenue float64
	for _, q := range quotes {
		totalRevenue += q.Total
	}

	data := s.newPageData(r, "Reporte de Ingresos")
	data.Data = map[string]interface{}{
		"Quotes":       quotes,
		"TotalRevenue": totalRevenue,
	}
	s.render(w, r, "pages/admin/report_revenue.html", data)
}

func (s *Server) handleSurveysReport(w http.ResponseWriter, r *http.Request) {
	ctx := r.Context()

	surveys, _ := s.repos.Surveys.List(ctx, 100, 0)
	thirtyDaysAgo := time.Now().AddDate(0, 0, -30)
	avgRating, _ := s.repos.Surveys.GetAverageRating(ctx, thirtyDaysAgo)
	totalSurveys, _ := s.repos.Surveys.Count(ctx)
	ratingDist, _ := s.repos.Surveys.GetRatingDistribution(ctx)

	// Calculate response rate
	// Total tickets that could have a survey (delivered or ready)
	counts, _ := s.repos.Tickets.CountByStatus(ctx)
	eligibleTickets := counts[domain.TicketStatusDelivered] + counts[domain.TicketStatusReady]

	var responseRate float64
	if eligibleTickets > 0 {
		responseRate = (float64(totalSurveys) / float64(eligibleTickets)) * 100
	} else if totalSurveys > 0 {
		// If we have surveys but 0 eligible tickets (maybe status changed back?), cap at 100% or just use 100
		responseRate = 100
	}

	// Calculate rating percentages for the chart
	ratingPercentages := make(map[int]int)
	if totalSurveys > 0 {
		for i := 1; i <= 5; i++ {
			count := ratingDist[i]
			percentage := float64(count) / float64(totalSurveys) * 100
			ratingPercentages[i] = int(percentage + 0.5) // Round to nearest integer
		}
	} else {
		for i := 1; i <= 5; i++ {
			ratingPercentages[i] = 0
		}
	}

	data := s.newPageData(r, "Reporte de Encuestas")
	data.Data = map[string]interface{}{
		"Surveys":           surveys,
		"AvgRating":         avgRating,
		"TotalSurveys":      totalSurveys,
		"RatingDist":        ratingDist,
		"RatingPercentages": ratingPercentages,
		"ResponseRate":      responseRate,
		"StarLevels":        []int{5, 4, 3, 2, 1},
	}
	s.render(w, r, "pages/admin/report_surveys.html", data)
}

// Settings

func (s *Server) handleSettings(w http.ResponseWriter, r *http.Request) {
	ctx := r.Context()

	// Get current hero concept
	heroConcept, err := s.repos.Settings.Get(ctx, "hero_concept")
	if err != nil {
		// Log error but don't fail, just use active default
		heroConcept = "bicycle workshop"
	}
	if heroConcept == "" {
		heroConcept = "bicycle workshop"
	}

	data := s.newPageData(r, "Configuración")
	data.Data = map[string]interface{}{
		"Config":      s.config,
		"HeroConcept": heroConcept,
	}
	s.render(w, r, "pages/admin/settings.html", data)
}

func (s *Server) handleUpdateSettings(w http.ResponseWriter, r *http.Request) {
	ctx := r.Context()

	if err := r.ParseForm(); err != nil {
		http.Error(w, "Error processing form", http.StatusBadRequest)
		return
	}

	heroConcept := r.FormValue("hero_concept")
	if heroConcept != "" {
		if err := s.repos.Settings.Set(ctx, "hero_concept", heroConcept); err != nil {
			data := s.newPageData(r, "Configuración")
			data.Flash = &FlashMessage{Type: "error", Message: "Error al guardar la configuración"}
			s.render(w, r, "pages/admin/settings.html", data)
			return
		}
	}

	data := s.newPageData(r, "Configuración")
	data.Flash = &FlashMessage{Type: "success", Message: "Configuración actualizada correctamente"}

	// Re-fetch to show updated state
	data.Data = map[string]interface{}{
		"Config":      s.config,
		"HeroConcept": heroConcept,
	}
	s.render(w, r, "pages/admin/settings.html", data)
}

// API handlers

func (s *Server) apiGetModelsByBrand(w http.ResponseWriter, r *http.Request) {
	ctx := r.Context()

	brandID, _ := strconv.ParseInt(getURLParam(r, "brandId"), 10, 64)
	models, err := s.repos.Models.GetByBrandID(ctx, brandID)
	if err != nil {
		http.Error(w, "Error loading models", http.StatusInternalServerError)
		return
	}

	w.Header().Set("Content-Type", "application/json")
	json.NewEncoder(w).Encode(models)
}

func (s *Server) apiGetAvailableSlots(w http.ResponseWriter, r *http.Request) {
	ctx := r.Context()

	dateStr := r.URL.Query().Get("date")
	date, err := time.Parse("2006-01-02", dateStr)
	if err != nil {
		http.Error(w, "Invalid date", http.StatusBadRequest)
		return
	}

	// Get existing bookings for the date
	startOfDay := time.Date(date.Year(), date.Month(), date.Day(), 0, 0, 0, 0, date.Location())
	endOfDay := startOfDay.Add(24 * time.Hour)

	existingBookings, _ := s.repos.Bookings.GetByDateRange(ctx, startOfDay, endOfDay)

	// Define available slots (9am to 6pm, 1 hour each)
	allSlots := []string{"09:00", "10:00", "11:00", "12:00", "14:00", "15:00", "16:00", "17:00"}

	// Filter out booked slots
	bookedSlots := make(map[string]bool)
	for _, b := range existingBookings {
		bookedSlots[b.ScheduledAt.Format("15:04")] = true
	}

	var availableSlots []string
	for _, slot := range allSlots {
		if !bookedSlots[slot] {
			availableSlots = append(availableSlots, slot)
		}
	}

	w.Header().Set("Content-Type", "application/json")
	json.NewEncoder(w).Encode(availableSlots)
}

func (s *Server) apiGetTicketStatus(w http.ResponseWriter, r *http.Request) {
	ctx := r.Context()

	id, _ := strconv.ParseInt(getURLParam(r, "id"), 10, 64)
	ticket, err := s.repos.Tickets.GetByID(ctx, id)
	if err != nil || ticket == nil {
		http.Error(w, "Ticket not found", http.StatusNotFound)
		return
	}

	response := map[string]interface{}{
		"id":           ticket.ID,
		"trackingCode": ticket.TrackingCode,
		"status":       ticket.Status,
		"statusLabel":  domain.TicketStatusLabel(ticket.Status),
		"updatedAt":    ticket.UpdatedAt,
	}

	w.Header().Set("Content-Type", "application/json")
	json.NewEncoder(w).Encode(response)
}

// Ticket management

func (s *Server) handleAdminTicketsList(w http.ResponseWriter, r *http.Request) {
	ctx := r.Context()

	status := r.URL.Query().Get("status")
	tickets, err := s.repos.Tickets.List(ctx, status, 100, 0)
	if err != nil {
		http.Error(w, "Error loading tickets", http.StatusInternalServerError)
		return
	}

	// Enrich tickets with technician info if not already present (List sometimes does simple fetch)
	// The current List implementation in sqlite repo does simple fetch.
	// We might need to fetch technicians.
	// Actually List implementation does fetch basic fields.
	// Let's get all technicians for the dropdown
	technicians, _ := s.repos.Users.List(ctx, domain.RoleTechnician, 100, 0)

	// We need to fetch customer info for each ticket... this is N+1 but ok for now or we update repo.
	// For now, let's just show the ticket and technician.
	// Ideally we should update the List method to join users (technicians) and bookings->customers.
	// But let's work with what we have or do a quick loop if needed.
	// The repo `List` method returns `domain.Ticket` struct.
	// Let's check if we need to manually populate anything.
	// The `scanTicketsSimple` does `LEFT JOIN`? No, `List` query in repo is simple.
	// It does NOT join technician details in `List`.
	// We should probably update the repo or just fetch details here.
	// Given the constraints, let's just fetch technicans list for the dropdown
	// and maybe we can live with just technician ID or name if we had it.
	// Update: `scanTicketsSimple` fetches `technician_id`. It does NOT fetch names.
	// So we will need to enrich them or update repo.
	// Let's update the repo query in a separate step if needed, or just iterate.
	// Iterating 100 tickets is fast enough for this scale.
	for i := range tickets {
		if tickets[i].TechnicianID != 0 {
			tech, _ := s.repos.Users.GetByID(ctx, tickets[i].TechnicianID)
			if tech != nil {
				tickets[i].Technician = tech
			}
		}
		// Also fetch booking to get customer?
		if tickets[i].BookingID != 0 {
			booking, _ := s.repos.Bookings.GetByID(ctx, tickets[i].BookingID)
			if booking != nil {
				tickets[i].Booking = booking
			}
		}
	}

	data := s.newPageData(r, "Gestión de Tickets")
	data.Data = map[string]interface{}{
		"Tickets":       tickets,
		"Technicians":   technicians,
		"CurrentStatus": status,
	}
	s.render(w, r, "pages/admin/tickets.html", data)
}

func (s *Server) handleAdminUpdateTicketTechnician(w http.ResponseWriter, r *http.Request) {
	ctx := r.Context()

	if err := r.ParseForm(); err != nil {
		http.Error(w, "Error processing form", http.StatusBadRequest)
		return
	}

	id, _ := strconv.ParseInt(getURLParam(r, "id"), 10, 64)
	ticket, err := s.repos.Tickets.GetByID(ctx, id)
	if err != nil || ticket == nil {
		http.NotFound(w, r)
		return
	}

	techID, _ := strconv.ParseInt(r.FormValue("technician_id"), 10, 64)

	// Update technician
	ticket.TechnicianID = techID

	if err := s.repos.Tickets.Update(ctx, ticket); err != nil {
		http.Error(w, "Error updating ticket", http.StatusInternalServerError)
		return
	}

	// Add history record for reassignment
	// We can use the UpdateStatus logic or just insert history manually
	// Let's insert a history record manually since status didn't change
	// Or even better, we check if status changed too? Admin might want to change both.
	// For now, just technician.

	history := &domain.TicketStatusHistory{
		TicketID:  id,
		Status:    ticket.Status,
		ChangedBy: getUserClaims(r).UserID,
		Notes:     "Técnico reasignado por administrador",
	}
	s.repos.Tickets.CreateStatusHistory(ctx, history)

	http.Redirect(w, r, "/admin/tickets", http.StatusSeeOther)
}

// Ad Management (Press Kit)

func (s *Server) handleAdsList(w http.ResponseWriter, r *http.Request) {
	ctx := r.Context()
	ads, err := s.repos.Ads.List(ctx)
	if err != nil {
		http.Error(w, "Error listing ads", http.StatusInternalServerError)
		return
	}

	data := s.newPageData(r, "Gestión de Anuncios")
	data.Data = map[string]interface{}{
		"Ads": ads,
	}
	s.render(w, r, "pages/admin/ads.html", data)
}

func (s *Server) handleCreateAd(w http.ResponseWriter, r *http.Request) {
	ctx := r.Context()
	if err := r.ParseForm(); err != nil {
		http.Error(w, "Error processing form", http.StatusBadRequest)
		return
	}

	ad := &domain.Ad{
		Title:     r.FormValue("title"),
		MediaURL:  r.FormValue("media_url"),
		MediaType: r.FormValue("media_type"),
		LinkURL:   r.FormValue("link_url"),
		Active:    r.FormValue("active") == "on",
	}

	if err := s.repos.Ads.Create(ctx, ad); err != nil {
		http.Error(w, "Error creating ad", http.StatusInternalServerError)
		return
	}

	http.Redirect(w, r, "/admin/ads", http.StatusSeeOther)
}

func (s *Server) handleUpdateAd(w http.ResponseWriter, r *http.Request) {
	ctx := r.Context()
	if err := r.ParseForm(); err != nil {
		http.Error(w, "Error processing form", http.StatusBadRequest)
		return
	}

	id, _ := strconv.ParseInt(getURLParam(r, "id"), 10, 64)
	ad, err := s.repos.Ads.GetByID(ctx, id)
	if err != nil || ad == nil {
		http.NotFound(w, r)
		return
	}

	// Update fields if provided (for simple toggle from list, we might just look at form)
	// If it's a full update form, we'd have all fields.
	// The plan says "Update/Toggle Active". Let's assume the form provides all or we handle specific actions.
	// For simplicity, let's assume it's a full update or just active toggle.
	// If "action" param is "toggle", just flip active.

	if r.FormValue("action") == "toggle" {
		ad.Active = !ad.Active
	} else {
		// Full update
		ad.Title = r.FormValue("title")
		ad.MediaURL = r.FormValue("media_url")
		ad.MediaType = r.FormValue("media_type")
		ad.LinkURL = r.FormValue("link_url")
		ad.Active = r.FormValue("active") == "on"
	}

	if err := s.repos.Ads.Update(ctx, ad); err != nil {
		http.Error(w, "Error updating ad", http.StatusInternalServerError)
		return
	}

	http.Redirect(w, r, "/admin/ads", http.StatusSeeOther)
}

func (s *Server) handleDeleteAd(w http.ResponseWriter, r *http.Request) {
	ctx := r.Context()
	id, _ := strconv.ParseInt(getURLParam(r, "id"), 10, 64)

	if err := s.repos.Ads.Delete(ctx, id); err != nil {
		http.Error(w, "Error deleting ad", http.StatusInternalServerError)
		return
	}

	http.Redirect(w, r, "/admin/ads", http.StatusSeeOther)
}
