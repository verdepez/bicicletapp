// BicicletAPP - Technical Service Management WebApp
// Optimized for shared hosting with limited resources
package main

import (
	"log"
	"os"
	"runtime"

	"bicicletapp/internal/config"
	"bicicletapp/internal/repository"
	"bicicletapp/internal/repository/sqlite"
	"bicicletapp/internal/server"
	"bicicletapp/internal/templates"
)

func main() {
	// Limit CPU usage for shared hosting
	runtime.GOMAXPROCS(1)

	// Load configuration
	cfg, err := config.Load("config.json")
	if err != nil {
		log.Fatalf("❌ Failed to load configuration: %v", err)
	}

	log.Printf("🚲 Starting %s...", cfg.Business.Name)
	log.Printf("📋 Debug mode: %v", cfg.Debug)

	// Initialize database
	db, err := sqlite.New(cfg.GetDatabasePath())
	if err != nil {
		log.Fatalf("❌ Failed to connect to database: %v", err)
	}
	defer db.Close()

	// Run migrations
	if err := db.Migrate(); err != nil {
		log.Fatalf("❌ Failed to run migrations: %v", err)
	}
	log.Println("✅ Database initialized")

	// Create admin user if none exists
	if err := createDefaultAdmin(db); err != nil {
		log.Printf("⚠️ Could not create default admin: %v", err)
	}

	// Initialize repositories
	repos := &repository.Repositories{
		Users:    sqlite.NewUserRepo(db),
		Brands:   sqlite.NewBrandRepo(db),
		Models:   sqlite.NewModelRepo(db),
		Services: sqlite.NewServiceRepo(db),
		Bicycles: sqlite.NewBicycleRepo(db),
		Bookings: sqlite.NewBookingRepo(db),
		Quotes:   sqlite.NewQuoteRepo(db),
		Tickets:  sqlite.NewTicketRepo(db),
		Surveys:  sqlite.NewSurveyRepo(db),
		Ads:      sqlite.NewAdRepo(db),
		Settings: sqlite.NewSettingsRepo(db),
	}

	// Initialize template manager
	tmpl, err := templates.NewManager("./templates", cfg.Debug)
	if err != nil {
		log.Fatalf("❌ Failed to initialize templates: %v", err)
	}
	log.Println("✅ Templates loaded")

	// Create and run the server
	srv := server.New(cfg, repos, tmpl)

	log.Printf("🌐 Server listening on http://%s", cfg.Address())

	if err := srv.Run(); err != nil {
		log.Fatalf("❌ Server error: %v", err)
	}
}

// createDefaultAdmin creates a default admin user if no users exist
func createDefaultAdmin(db *sqlite.DB) error {
	// Check if any users exist
	var count int
	err := db.QueryRow("SELECT COUNT(*) FROM users").Scan(&count)
	if err != nil {
		return err
	}

	if count > 0 {
		return nil // Users already exist
	}

	// Create default admin
	// Password: admin123 (CHANGE IN PRODUCTION!)
	hashedPassword, err := sqlite.HashPassword("admin123")
	if err != nil {
		return err
	}

	_, err = db.Exec(`
		INSERT INTO users (email, password_hash, name, phone, role)
		VALUES (?, ?, ?, ?, ?)
	`, "admin@bicicletapp.com", hashedPassword, "Administrador", "", "admin")

	if err != nil {
		return err
	}

	log.Println("✅ Default admin user created:")
	log.Println("   Email: admin@bicicletapp.com")
	log.Println("   Password: admin123")
	log.Println("   ⚠️ CHANGE THIS PASSWORD IN PRODUCTION!")

	// Create sample data for testing
	if os.Getenv("SEED_DATA") == "true" {
		createSampleData(db)
	}

	return nil
}

// createSampleData creates sample data for testing
func createSampleData(db *sqlite.DB) {
	log.Println("🌱 Creating sample data...")

	// Sample brands
	sampleBrands := []string{"Trek", "Specialized", "Giant", "Cannondale", "Scott"}
	for _, name := range sampleBrands {
		db.Exec("INSERT INTO brands (name) VALUES (?)", name)
	}

	// Sample services
	sampleServices := []struct {
		name  string
		desc  string
		price float64
		hours float64
	}{
		{"Revisión General", "Inspección completa de todos los componentes", 2500, 1.5},
		{"Cambio de Cámara", "Cambio de cámara en cualquier rueda", 800, 0.5},
		{"Ajuste de Frenos", "Ajuste y regulación del sistema de frenos", 1200, 0.75},
		{"Cambio de Cadena", "Reemplazo de cadena desgastada", 1500, 0.5},
		{"Servicio Completo", "Mantenimiento preventivo completo", 5000, 3},
		{"Centrado de Ruedas", "Alineación y tensado de radios", 1800, 1},
		{"Cambio de Cubiertas", "Instalación de cubiertas nuevas", 1000, 0.5},
		{"Ajuste de Cambios", "Regulación del sistema de transmisión", 1500, 1},
	}
	for _, s := range sampleServices {
		db.Exec("INSERT INTO services (name, description, base_price, estimated_hours) VALUES (?, ?, ?, ?)",
			s.name, s.desc, s.price, s.hours)
	}

	log.Println("✅ Sample data created")
}
