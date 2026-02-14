// Package config handles external configuration loading from JSON and environment variables.
package config

import (
	"encoding/json"
	"fmt"
	"os"
	"path/filepath"
	"strconv"
)

// Config holds all application configuration
type Config struct {
	Debug    bool     `json:"debug"`
	Server   Server   `json:"server"`
	Database Database `json:"database"`
	Business Business `json:"business"`
	Features Features `json:"features"`
	JWT      JWT      `json:"jwt"`
}

// Server holds HTTP server configuration
type Server struct {
	Port         int    `json:"port"`
	Host         string `json:"host"`
	ReadTimeout  int    `json:"readTimeout"`
	WriteTimeout int    `json:"writeTimeout"`
}

// Database holds database configuration
type Database struct {
	Path string `json:"path"`
}

// Business holds branding and business information
type Business struct {
	Name           string `json:"name"`
	Tagline        string `json:"tagline"`
	Logo           string `json:"logo"`
	PrimaryColor   string `json:"primaryColor"`
	SecondaryColor string `json:"secondaryColor"`
	AccentColor    string `json:"accentColor"`
	ContactEmail   string `json:"contactEmail"`
	ContactPhone   string `json:"contactPhone"`
}

// Features holds feature toggles
type Features struct {
	Payments           bool `json:"payments"`
	SMS                bool `json:"sms"`
	Surveys            bool `json:"surveys"`
	EmailNotifications bool `json:"emailNotifications"`
}

// JWT holds JWT configuration
type JWT struct {
	Secret          string `json:"secret"`
	ExpirationHours int    `json:"expirationHours"`
}

// Load reads configuration from the specified JSON file and overrides with environment variables
func Load(configPath string) (*Config, error) {
	var cfg Config

	// Validate and clean the path
	cleanPath := filepath.Clean(configPath)

	// Try to read the config file
	data, err := os.ReadFile(cleanPath)
	if err == nil {
		// File exists, parse it
		if err := json.Unmarshal(data, &cfg); err != nil {
			return nil, fmt.Errorf("failed to parse config file: %w", err)
		}
	} else if !os.IsNotExist(err) {
		// Error other than "not found"
		return nil, fmt.Errorf("failed to read config file: %w", err)
	}
	// If file doesn't exist, we continue with empty config and rely on Env Vars

	// Override with environment variables
	cfg.applyEnvOverrides()

	// Set defaults if missing (e.g. for purely env-based config)
	if cfg.Server.Port == 0 {
		cfg.Server.Port = 8080 // Default port
	}
	if cfg.JWT.ExpirationHours == 0 {
		cfg.JWT.ExpirationHours = 24
	}

	// Validate configuration
	if err := cfg.validate(); err != nil {
		return nil, fmt.Errorf("invalid configuration: %w", err)
	}

	return &cfg, nil
}

// applyEnvOverrides overrides config values with environment variables if set
func (c *Config) applyEnvOverrides() {
	// Debug mode
	if debug := os.Getenv("DEBUG"); debug != "" {
		c.Debug = debug == "true" || debug == "1"
	}

	// Server port
	if port := os.Getenv("PORT"); port != "" {
		if p, err := strconv.Atoi(port); err == nil {
			c.Server.Port = p
		}
	}

	// Server host
	if host := os.Getenv("HOST"); host != "" {
		c.Server.Host = host
	}

	// Database path
	if dbPath := os.Getenv("DATABASE_PATH"); dbPath != "" {
		c.Database.Path = dbPath
	}

	// JWT secret (critical for production)
	if secret := os.Getenv("JWT_SECRET"); secret != "" {
		c.JWT.Secret = secret
	}
}

// validate checks that all required configuration values are present
func (c *Config) validate() error {
	if c.Server.Port <= 0 || c.Server.Port > 65535 {
		return fmt.Errorf("invalid server port: %d", c.Server.Port)
	}

	if c.Database.Path == "" {
		return fmt.Errorf("database path is required")
	}

	// Validate database path for security
	cleanDBPath := filepath.Clean(c.Database.Path)
	if !filepath.IsLocal(cleanDBPath) && !filepath.IsAbs(cleanDBPath) {
		return fmt.Errorf("invalid database path: potential path traversal detected")
	}

	if c.JWT.Secret == "" || c.JWT.Secret == "CHANGE_THIS_SECRET_IN_PRODUCTION" {
		if !c.Debug {
			return fmt.Errorf("JWT secret must be changed for production")
		}
	}

	if c.JWT.ExpirationHours <= 0 {
		c.JWT.ExpirationHours = 24 // Default to 24 hours
	}

	return nil
}

// Address returns the full server address (host:port)
func (c *Config) Address() string {
	return fmt.Sprintf("%s:%d", c.Server.Host, c.Server.Port)
}

// GetDatabasePath returns the cleaned and validated database path
func (c *Config) GetDatabasePath() string {
	return filepath.Clean(c.Database.Path)
}
