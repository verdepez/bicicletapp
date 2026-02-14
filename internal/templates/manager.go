// Package templates provides a template manager with dynamic reload support.
package templates

import (
	"fmt"
	"html/template"
	"io"
	"os"
	"path/filepath"
	"sync"
	"time"
)

// Manager handles template loading and caching
type Manager struct {
	dir     string
	debug   bool
	cache   map[string]*template.Template
	mu      sync.RWMutex
	funcMap template.FuncMap
}

// NewManager creates a new template manager
// If debug is true, templates are reloaded on every request
// If debug is false, templates are cached in memory
func NewManager(dir string, debug bool) (*Manager, error) {
	// Validate and clean the directory path
	cleanDir := filepath.Clean(dir)
	if _, err := os.Stat(cleanDir); os.IsNotExist(err) {
		return nil, fmt.Errorf("template directory does not exist: %s", cleanDir)
	}

	m := &Manager{
		dir:   cleanDir,
		debug: debug,
		cache: make(map[string]*template.Template),
		funcMap: template.FuncMap{
			"formatDate":        formatDate,
			"formatTime":        formatTime,
			"formatMoney":       formatMoney,
			"safeHTML":          safeHTML,
			"add":               add,
			"sub":               sub,
			"mul":               mul,
			"div":               div,
			"statusBadge":       statusBadge,
			"ticketStatusLabel": ticketStatusLabel,
			"statusLabel":       statusLabel,
			"whatsappLink":      whatsappLink,
		},
	}

	// If not in debug mode, pre-load all templates
	if !debug {
		if err := m.loadTemplates(); err != nil {
			return nil, err
		}
	}

	return m, nil
}

// loadTemplates loads all templates from the directory
func (m *Manager) loadTemplates() error {
	m.mu.Lock()
	defer m.mu.Unlock()

	// Load layout template
	layoutPath := filepath.Join(m.dir, "layouts", "base.html")
	layoutContent, err := os.ReadFile(layoutPath)
	if err != nil {
		return fmt.Errorf("failed to read layout: %w", err)
	}

	// Walk through the pages directory and parse each page with the layout
	pagesDir := filepath.Join(m.dir, "pages")
	err = filepath.Walk(pagesDir, func(path string, info os.FileInfo, err error) error {
		if err != nil {
			return err
		}

		// Skip directories and non-HTML files
		if info.IsDir() || filepath.Ext(path) != ".html" {
			return nil
		}

		// Validate path for security
		cleanPath := filepath.Clean(path)
		if !isSubPath(m.dir, cleanPath) {
			return fmt.Errorf("invalid template path detected: %s", path)
		}

		// Read page template content
		pageContent, err := os.ReadFile(cleanPath)
		if err != nil {
			return fmt.Errorf("failed to read template %s: %w", path, err)
		}

		// Get relative path for template name (e.g., "pages/public/home.html")
		relPath, err := filepath.Rel(m.dir, cleanPath)
		if err != nil {
			return err
		}
		templateName := filepath.ToSlash(relPath)

		// Create a new template combining layout + page
		tmpl := template.New("base").Funcs(m.funcMap)

		// Parse layout first
		_, err = tmpl.Parse(string(layoutContent))
		if err != nil {
			return fmt.Errorf("failed to parse layout for %s: %w", templateName, err)
		}

		// Then parse the page content
		_, err = tmpl.Parse(string(pageContent))
		if err != nil {
			return fmt.Errorf("failed to parse template %s: %w", templateName, err)
		}

		m.cache[templateName] = tmpl
		return nil
	})

	if err != nil {
		return err
	}

	return nil
}

// Render renders a template with the given data
func (m *Manager) Render(w io.Writer, name string, data interface{}) error {
	if m.debug {
		// In debug mode, reload template on every request
		if err := m.loadSingle(name); err != nil {
			return fmt.Errorf("failed to reload templates: %w", err)
		}
	}

	m.mu.RLock()
	tmpl, ok := m.cache[name]
	m.mu.RUnlock()

	if !ok {
		return fmt.Errorf("template not found: %s", name)
	}

	return tmpl.ExecuteTemplate(w, "base", data)
}

// loadSingle loads a single template (used in debug mode)
func (m *Manager) loadSingle(name string) error {
	m.mu.Lock()
	defer m.mu.Unlock()

	// Load layout template
	layoutPath := filepath.Join(m.dir, "layouts", "base.html")
	layoutContent, err := os.ReadFile(layoutPath)
	if err != nil {
		return fmt.Errorf("failed to read layout: %w", err)
	}

	// Load the page template
	pagePath := filepath.Join(m.dir, name)
	pageContent, err := os.ReadFile(pagePath)
	if err != nil {
		return fmt.Errorf("failed to read template %s: %w", name, err)
	}

	// Create a new template combining layout + page
	tmpl := template.New("base").Funcs(m.funcMap)

	// Parse layout first
	_, err = tmpl.Parse(string(layoutContent))
	if err != nil {
		return fmt.Errorf("failed to parse layout: %w", err)
	}

	// Then parse the page content
	_, err = tmpl.Parse(string(pageContent))
	if err != nil {
		return fmt.Errorf("failed to parse template %s: %w", name, err)
	}

	m.cache[name] = tmpl
	return nil
}

// isSubPath checks if child is a subpath of parent
func isSubPath(parent, child string) bool {
	rel, err := filepath.Rel(parent, child)
	if err != nil {
		return false
	}
	// Check that the relative path doesn't escape the parent directory
	return !filepath.IsAbs(rel) && rel != ".." && len(rel) > 0 && rel[0] != '.'
}

// Template helper functions

func formatDate(t time.Time) string {
	if t.IsZero() {
		return "-"
	}
	return t.Format("02/01/2006")
}

func formatTime(t time.Time) string {
	if t.IsZero() {
		return "-"
	}
	return t.Format("15:04")
}

func formatMoney(amount float64) string {
	return fmt.Sprintf("$%.2f", amount)
}

func safeHTML(s string) template.HTML {
	return template.HTML(s)
}

func add(a, b int) int {
	return a + b
}

func sub(a, b int) int {
	return a - b
}

func mul(a, b int) int {
	return a * b
}

func div(a, b int) int {
	if b == 0 {
		return 0
	}
	return a / b
}

func statusBadge(status string) string {
	badges := map[string]string{
		"pending":       "secondary",
		"confirmed":     "primary",
		"in_progress":   "warning",
		"waiting_parts": "warning",
		"ready":         "success",
		"delivered":     "success",
		"cancelled":     "error",
		"approved":      "success",
		"rejected":      "error",
	}
	if badge, ok := badges[status]; ok {
		return badge
	}
	return "secondary"
}

func ticketStatusLabel(status string) string {
	labels := map[string]string{
		"pending":       "⏳ Pendiente",
		"received":      "📥 Recibido",
		"diagnosing":    "🔍 Diagnosticando",
		"in_progress":   "🔧 En Progreso",
		"waiting_parts": "📦 Esperando Repuestos",
		"ready":         "✅ Listo",
		"delivered":     "🚲 Entregado",
		"cancelled":     "❌ Cancelado",
	}
	if label, ok := labels[status]; ok {
		return label
	}
	return status
}

// statusLabel translates booking and quote status to Spanish
func statusLabel(status string) string {
	labels := map[string]string{
		// Booking status
		"pending":   "Pendiente",
		"confirmed": "Confirmada",
		"cancelled": "Cancelada",
		"completed": "Completada",
		// Quote status
		"approved": "Aprobado",
		"rejected": "Rechazado",
		// Ticket status
		"received":      "Recibido",
		"in_progress":   "En Progreso",
		"waiting_parts": "Esperando Repuestos",
		"ready":         "Listo para Retirar",
		"delivered":     "Entregado",
	}
	if label, ok := labels[status]; ok {
		return label
	}
	return status
}

// whatsappLink generates a WhatsApp API link with pre-filled message
func whatsappLink(phone, message string) string {
	// Clean phone number (remove spaces, dashes, etc.)
	cleanPhone := ""
	for _, c := range phone {
		if c >= '0' && c <= '9' || c == '+' {
			cleanPhone += string(c)
		}
	}
	// If phone doesn't start with +, assume Chilean number
	if len(cleanPhone) > 0 && cleanPhone[0] != '+' {
		cleanPhone = "+56" + cleanPhone
	}
	// URL encode the message
	encodedMsg := template.URLQueryEscaper(message)
	return "https://wa.me/" + cleanPhone + "?text=" + encodedMsg
}
