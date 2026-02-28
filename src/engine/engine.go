package engine

import (
	"context"
	"time"
)

// Result represents a single search result from any engine.
type Result struct {
	Title       string    `json:"title"`
	URL         string    `json:"url"`
	Snippet     string    `json:"snippet"`
	Source      string    `json:"source"`
	ContentType string    `json:"content_type"`
	Domain      string    `json:"domain"`
	FetchedAt   time.Time `json:"fetched_at"`
	Rank        int       `json:"rank"`
}

// SearchParams holds the parameters for a search query.
type SearchParams struct {
	Query      string
	Page       int
	MaxResults int
	SafeSearch bool
	Region     string
}

// Engine defines the interface that every search engine adapter must implement.
type Engine interface {
	Name() string
	Search(ctx context.Context, params SearchParams) ([]Result, error)
}
