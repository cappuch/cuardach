package meta

import (
	"context"
	"database/sql"
	"fmt"
	"strings"
	"time"

	"github.com/cappuch/cuardach/src/engine"
)

type Filter struct {
	Query       string
	Source      string
	Domain      string
	ContentType string
	After       string // YYYY-MM-DD
	Before      string // YYYY-MM-DD
	Limit       int
	Offset      int
}

type Searcher struct {
	db *sql.DB
}

func NewSearcher(db *sql.DB) *Searcher {
	return &Searcher{db: db}
}

func (s *Searcher) Search(ctx context.Context, filter Filter) ([]engine.Result, int, error) {
	if filter.Limit <= 0 {
		filter.Limit = 20
	}

	var conditions []string
	var args []interface{}

	useFTS := filter.Query != ""

	if filter.Source != "" {
		conditions = append(conditions, "sr.source = ?")
		args = append(args, filter.Source)
	}
	if filter.Domain != "" {
		conditions = append(conditions, "sr.domain LIKE ?")
		args = append(args, "%"+filter.Domain+"%")
	}
	if filter.ContentType != "" {
		conditions = append(conditions, "sr.content_type = ?")
		args = append(args, filter.ContentType)
	}
	if filter.After != "" {
		conditions = append(conditions, "sr.fetched_at >= ?")
		args = append(args, filter.After+"T00:00:00Z")
	}
	if filter.Before != "" {
		conditions = append(conditions, "sr.fetched_at <= ?")
		args = append(args, filter.Before+"T23:59:59Z")
	}

	where := ""
	if len(conditions) > 0 {
		where = " AND " + strings.Join(conditions, " AND ")
	}

	var query string
	if useFTS {
		ftsArgs := []interface{}{filter.Query}
		ftsArgs = append(ftsArgs, args...)
		args = ftsArgs

		query = fmt.Sprintf(`
			SELECT sr.id, sr.query, sr.title, sr.url, sr.snippet, sr.source,
			       sr.content_type, sr.domain, sr.rank, sr.fetched_at
			FROM search_results sr
			JOIN search_results_fts fts ON sr.id = fts.rowid
			WHERE search_results_fts MATCH ?%s
			ORDER BY fts.rank
			LIMIT ? OFFSET ?`, where)
	} else {
		if where != "" {
			where = " WHERE " + where[5:] // strip leading " AND "
		}
		query = fmt.Sprintf(`
			SELECT sr.id, sr.query, sr.title, sr.url, sr.snippet, sr.source,
			       sr.content_type, sr.domain, sr.rank, sr.fetched_at
			FROM search_results sr%s
			ORDER BY sr.fetched_at DESC
			LIMIT ? OFFSET ?`, where)
	}

	args = append(args, filter.Limit, filter.Offset)

	rows, err := s.db.QueryContext(ctx, query, args...)
	if err != nil {
		return nil, 0, fmt.Errorf("meta search: %w", err)
	}
	defer rows.Close()

	var results []engine.Result
	for rows.Next() {
		var r engine.Result
		var id int
		var q, fetchedAt string
		err := rows.Scan(&id, &q, &r.Title, &r.URL, &r.Snippet, &r.Source, &r.ContentType, &r.Domain, &r.Rank, &fetchedAt)
		if err != nil {
			return nil, 0, fmt.Errorf("scanning row: %w", err)
		}
		r.FetchedAt, _ = time.Parse(time.RFC3339, fetchedAt)
		results = append(results, r)
	}

	if err := rows.Err(); err != nil {
		return nil, 0, err
	}

	total := len(results) // total count is not available without a separate count query, so we infer it based on the limit
	if total == filter.Limit {
		total = filter.Limit + 1 // indicate more results exist
	}

	return results, total, nil
}
