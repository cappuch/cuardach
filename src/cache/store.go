package cache

import (
	"context"
	"crypto/sha256"
	"database/sql"
	"encoding/json"
	"fmt"
	"time"

	"github.com/cappuch/cuardach/src/engine"
	"github.com/cappuch/cuardach/src/indexer"
)

type Store struct {
	idx *indexer.SQLiteIndexer
}

func NewStore(idx *indexer.SQLiteIndexer) *Store {
	return &Store{idx: idx}
}

func (s *Store) GetResults(ctx context.Context, query string) ([]engine.Result, error) {
	hash := queryHash(query)

	var id int
	var q, qh, cachedAt, expiresAt, enginesJSON string
	var numResults int
	err := s.idx.DB().QueryRowContext(ctx,
		`SELECT id, query, query_hash, num_results, cached_at, expires_at, engines
		 FROM cache_queries WHERE query_hash = ? AND expires_at > datetime('now')`,
		hash,
	).Scan(&id, &q, &qh, &numResults, &cachedAt, &expiresAt, &enginesJSON)

	if err == sql.ErrNoRows {
		return nil, nil // cache miss
	}
	if err != nil {
		return nil, fmt.Errorf("cache lookup: %w", err)
	}
	return s.idx.QueryByQuery(ctx, query, numResults+10)
}

func (s *Store) PutResults(ctx context.Context, query string, results []engine.Result, engines []string, ttl time.Duration) error {
	if err := s.idx.Store(ctx, query, results); err != nil {
		return fmt.Errorf("storing results: %w", err)
	}

	hash := queryHash(query)
	enginesJSON, _ := json.Marshal(engines)
	expiresAt := time.Now().Add(ttl).Format(time.RFC3339)

	_, err := s.idx.DB().ExecContext(ctx,
		`INSERT OR REPLACE INTO cache_queries (query, query_hash, num_results, cached_at, expires_at, engines)
		 VALUES (?, ?, ?, datetime('now'), ?, ?)`,
		query, hash, len(results), expiresAt, string(enginesJSON),
	)
	if err != nil {
		return fmt.Errorf("recording cache entry: %w", err)
	}

	return nil
}

func (s *Store) Purge(ctx context.Context, all bool) (int, error) {
	if all {
		res, err := s.idx.DB().ExecContext(ctx, "DELETE FROM search_results")
		if err != nil {
			return 0, err
		}
		s.idx.DB().ExecContext(ctx, "DELETE FROM cache_queries")
		s.idx.DB().ExecContext(ctx, "DELETE FROM cache_content")
		n, _ := res.RowsAffected()
		return int(n), nil
	}

	s.idx.DB().ExecContext(ctx,
		`DELETE FROM search_results WHERE query IN (
			SELECT query FROM cache_queries WHERE expires_at <= datetime('now')
		)`)
	res, err := s.idx.DB().ExecContext(ctx,
		"DELETE FROM cache_queries WHERE expires_at <= datetime('now')")
	if err != nil {
		return 0, err
	}
	n, _ := res.RowsAffected()
	return int(n), nil
}

func (s *Store) Stats(ctx context.Context) (*Stats, error) {
	var stats Stats
	err := s.idx.DB().QueryRowContext(ctx,
		`SELECT
			(SELECT COUNT(*) FROM cache_queries) as total_queries,
			(SELECT COUNT(*) FROM search_results) as total_results,
			(SELECT COUNT(*) FROM cache_content) as total_content,
			(SELECT COALESCE(MIN(cached_at), '') FROM cache_queries) as oldest,
			(SELECT COALESCE(MAX(cached_at), '') FROM cache_queries) as newest`,
	).Scan(&stats.TotalQueries, &stats.TotalResults, &stats.TotalContent, &stats.OldestEntry, &stats.NewestEntry)
	if err != nil {
		return nil, fmt.Errorf("querying stats: %w", err)
	}
	return &stats, nil
}

func (s *Store) List(ctx context.Context, limit int) ([]Entry, error) {
	if limit <= 0 {
		limit = 20
	}

	rows, err := s.idx.DB().QueryContext(ctx,
		`SELECT query, cached_at, expires_at, num_results
		 FROM cache_queries ORDER BY cached_at DESC LIMIT ?`, limit)
	if err != nil {
		return nil, fmt.Errorf("listing cache: %w", err)
	}
	defer rows.Close()

	var entries []Entry
	for rows.Next() {
		var e Entry
		var cachedAt, expiresAt string
		if err := rows.Scan(&e.Query, &cachedAt, &expiresAt, &e.NumResults); err != nil {
			return nil, fmt.Errorf("scanning cache entry: %w", err)
		}
		e.CachedAt, _ = time.Parse("2006-01-02T15:04:05Z07:00", cachedAt)
		if e.CachedAt.IsZero() {
			e.CachedAt, _ = time.Parse("2006-01-02 15:04:05", cachedAt)
		}
		e.ExpiresAt, _ = time.Parse("2006-01-02T15:04:05Z07:00", expiresAt)
		if e.ExpiresAt.IsZero() {
			e.ExpiresAt, _ = time.Parse("2006-01-02 15:04:05", expiresAt)
		}
		entries = append(entries, e)
	}
	return entries, rows.Err()
}

func queryHash(query string) string {
	h := sha256.Sum256([]byte(query))
	return fmt.Sprintf("%x", h)
}
