package indexer

const insertResult = `
INSERT OR REPLACE INTO search_results (query, title, url, url_hash, snippet, source, content_type, domain, rank, fetched_at)
VALUES (?, ?, ?, ?, ?, ?, ?, ?, ?, ?)
`

const fullTextSearch = `
SELECT sr.id, sr.query, sr.title, sr.url, sr.snippet, sr.source, sr.content_type, sr.domain, sr.rank, sr.fetched_at
FROM search_results sr
JOIN search_results_fts fts ON sr.id = fts.rowid
WHERE search_results_fts MATCH ?
ORDER BY rank
LIMIT ?
`

const queryByQuery = `
SELECT id, query, title, url, snippet, source, content_type, domain, rank, fetched_at
FROM search_results
WHERE query = ?
ORDER BY rank ASC
LIMIT ?
`

const insertCacheQuery = `
INSERT OR REPLACE INTO cache_queries (query, query_hash, num_results, cached_at, expires_at, engines)
VALUES (?, ?, ?, datetime('now'), ?, ?)
`

const getCachedQuery = `
SELECT id, query, query_hash, num_results, cached_at, expires_at, engines
FROM cache_queries
WHERE query_hash = ? AND expires_at > datetime('now')
`

const listCachedQueries = `
SELECT query, cached_at, expires_at, num_results
FROM cache_queries
ORDER BY cached_at DESC
LIMIT ?
`

const deleteCachedQuery = `
DELETE FROM cache_queries WHERE expires_at <= datetime('now')
`

const deleteCachedResults = `
DELETE FROM search_results WHERE query IN (
    SELECT query FROM cache_queries WHERE expires_at <= datetime('now')
)
`

const deleteAllCache = `
DELETE FROM cache_queries;
DELETE FROM search_results;
`

const cacheStats = `
SELECT
    (SELECT COUNT(*) FROM cache_queries) as total_queries,
    (SELECT COUNT(*) FROM search_results) as total_results,
    (SELECT COUNT(*) FROM cache_content) as total_content,
    (SELECT COALESCE(MIN(cached_at), '') FROM cache_queries) as oldest,
    (SELECT COALESCE(MAX(cached_at), '') FROM cache_queries) as newest
`
