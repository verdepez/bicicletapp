package domain

import "time"

// MediaType constants
const (
	MediaTypeImage = "image"
	MediaTypeVideo = "video"
)

// Ad represents a promotional banner or video
type Ad struct {
	ID          int64     `json:"id"`
	Title       string    `json:"title"`
	MediaURL    string    `json:"media_url"`
	MediaType   string    `json:"media_type"` // "image" or "video"
	LinkURL     string    `json:"link_url"`
	Active      bool      `json:"active"`
	Impressions int       `json:"impressions"`
	Clicks      int       `json:"clicks"`
	CreatedAt   time.Time `json:"created_at"`
	UpdatedAt   time.Time `json:"updated_at"`
}
