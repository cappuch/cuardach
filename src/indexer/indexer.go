package indexer

import (
	"context"
	"crypto/sha256"
	"database/sql"
	"fmt"
	"net/url"
	"os"
	"path/filepath"
	"strings"
	"time"

	"github.com/cappuch/cuardach/src/engine"
	_ "modernc.org/sqlite"
)

type SQLiteIndexer struct {
	db *sql.DB
}

func NewSQLiteIndexer(dbPath string) (*SQLiteIndexer, error) {
	dir := filepath.Dir(dbPath)
	if err := os.MkdirAll(dir, 0700); err != nil {
		return nil, fmt.Errorf("creating database directory: %w", err)
	}

	db, err := sql.Open("sqlite", dbPath+"?_pragma=journal_mode(WAL)&_pragma=busy_timeout(5000)")
	if err != nil {
		return nil, fmt.Errorf("opening database: %w", err)
	}

	return &SQLiteIndexer{db: db}, nil
}

func (idx *SQLiteIndexer) Init(ctx context.Context) error {
	// Check if schema already exists
	var count int
	err := idx.db.QueryRowContext(ctx, "SELECT COUNT(*) FROM sqlite_master WHERE type='table' AND name='schema_version'").Scan(&count)
	if err != nil {
		return fmt.Errorf("checking schema: %w", err)
	}

	if count > 0 {
		var version int
		err := idx.db.QueryRowContext(ctx, "SELECT COALESCE(MAX(version), 0) FROM schema_version").Scan(&version)
		if err == nil && version >= schemaVersion {
			idx.AutoPurge(ctx)
			return nil
		}
	}

	if _, err := idx.db.ExecContext(ctx, createSchema); err != nil {
		return fmt.Errorf("creating schema: %w", err)
	}

	_, err = idx.db.ExecContext(ctx,
		"INSERT INTO schema_version (version) VALUES (?)", schemaVersion)
	if err != nil {
		return fmt.Errorf("recording schema version: %w", err)
	}

	idx.AutoPurge(ctx)

	return nil
}

func (idx *SQLiteIndexer) AutoPurge(ctx context.Context) (purged, deduped int) {
	res, err := idx.db.ExecContext(ctx,
		`DELETE FROM search_results WHERE query IN (
			SELECT query FROM cache_queries WHERE expires_at <= datetime('now')
		)`)
	if err == nil {
		n, _ := res.RowsAffected()
		purged += int(n)
	}

	idx.db.ExecContext(ctx, "DELETE FROM cache_queries WHERE expires_at <= datetime('now')")

	idx.db.ExecContext(ctx, "DELETE FROM cache_content WHERE expires_at <= datetime('now')")

	res, err = idx.db.ExecContext(ctx,
		`DELETE FROM search_results WHERE id NOT IN (
			SELECT MAX(id) FROM search_results GROUP BY url_hash, source
		)`)
	if err == nil {
		n, _ := res.RowsAffected()
		deduped += int(n)
	}

	idx.db.ExecContext(ctx, "INSERT INTO search_results_fts(search_results_fts) VALUES('rebuild')")

	idx.db.ExecContext(ctx, "PRAGMA incremental_vacuum")

	return purged, deduped
}

func (idx *SQLiteIndexer) Store(ctx context.Context, query string, results []engine.Result) error {
	tx, err := idx.db.BeginTx(ctx, nil)
	if err != nil {
		return fmt.Errorf("beginning transaction: %w", err)
	}
	defer tx.Rollback()

	stmt, err := tx.PrepareContext(ctx, insertResult)
	if err != nil {
		return fmt.Errorf("preparing statement: %w", err)
	}
	defer stmt.Close()

	for _, r := range results {
		domain := r.Domain
		if domain == "" {
			domain = extractDomain(r.URL)
		}
		_, err := stmt.ExecContext(ctx,
			query,
			r.Title,
			r.URL,
			hashURL(r.URL),
			r.Snippet,
			r.Source,
			r.ContentType,
			domain,
			r.Rank,
			r.FetchedAt.Format(time.RFC3339),
		)
		if err != nil {
			return fmt.Errorf("inserting result: %w", err)
		}
	}

	return tx.Commit()
}

func (idx *SQLiteIndexer) FullTextSearch(ctx context.Context, query string, limit int) ([]engine.Result, error) {
	if limit <= 0 {
		limit = 20
	}

	rows, err := idx.db.QueryContext(ctx, fullTextSearch, query, limit)
	if err != nil {
		return nil, fmt.Errorf("full-text search: %w", err)
	}
	defer rows.Close()

	return scanResults(rows)
}

func (idx *SQLiteIndexer) QueryByQuery(ctx context.Context, query string, limit int) ([]engine.Result, error) {
	if limit <= 0 {
		limit = 20
	}

	rows, err := idx.db.QueryContext(ctx, queryByQuery, query, limit)
	if err != nil {
		return nil, fmt.Errorf("query by query: %w", err)
	}
	defer rows.Close()

	return scanResults(rows)
}

func (idx *SQLiteIndexer) DB() *sql.DB {
	return idx.db
}

func (idx *SQLiteIndexer) Close() error {
	return idx.db.Close()
}

func scanResults(rows *sql.Rows) ([]engine.Result, error) {
	var results []engine.Result
	for rows.Next() {
		var r engine.Result
		var id int
		var query, fetchedAt string
		err := rows.Scan(&id, &query, &r.Title, &r.URL, &r.Snippet, &r.Source, &r.ContentType, &r.Domain, &r.Rank, &fetchedAt)
		if err != nil {
			return nil, fmt.Errorf("scanning row: %w", err)
		}
		r.FetchedAt, _ = time.Parse(time.RFC3339, fetchedAt)
		results = append(results, r)
	}
	return results, rows.Err()
}

func hashURL(rawURL string) string {
	normalized := normalizeURL(rawURL)
	h := sha256.Sum256([]byte(normalized))
	return fmt.Sprintf("%x", h)
}

func normalizeURL(rawURL string) string {
	u, err := url.Parse(rawURL)
	if err != nil {
		return rawURL
	}
	u.Scheme = strings.ToLower(u.Scheme)
	u.Host = strings.ToLower(u.Host)
	u.Host = strings.TrimPrefix(u.Host, "www.")
	u.Fragment = ""
	u.Path = strings.TrimRight(u.Path, "/")
	return u.String()
}

func extractDomain(rawURL string) string {
	u, err := url.Parse(rawURL)
	if err != nil {
		return ""
	}
	host := u.Hostname()
	host = strings.TrimPrefix(host, "www.")
	return host
}
