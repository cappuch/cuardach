package cache

import (
	"context"
	"time"

	"github.com/cappuch/cuardach/src/engine"
)

type Stats struct {
	TotalQueries int    `json:"total_queries"`
	TotalResults int    `json:"total_results"`
	TotalContent int    `json:"total_content"`
	OldestEntry  string `json:"oldest_entry"`
	NewestEntry  string `json:"newest_entry"`
}

type Entry struct {
	Query      string    `json:"query"`
	CachedAt   time.Time `json:"cached_at"`
	ExpiresAt  time.Time `json:"expires_at"`
	NumResults int       `json:"num_results"`
}

type Cache interface {
	GetResults(ctx context.Context, query string) ([]engine.Result, error)
	PutResults(ctx context.Context, query string, results []engine.Result, engines []string, ttl time.Duration) error
	Purge(ctx context.Context, all bool) (int, error)
	Stats(ctx context.Context) (*Stats, error)
	List(ctx context.Context, limit int) ([]Entry, error)
}
