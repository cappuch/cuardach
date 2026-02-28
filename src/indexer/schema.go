package indexer

const schemaVersion = 1

const createSchema = `
CREATE TABLE IF NOT EXISTS search_results (
    id          INTEGER PRIMARY KEY AUTOINCREMENT,
    query       TEXT    NOT NULL,
    title       TEXT    NOT NULL,
    url         TEXT    NOT NULL,
    url_hash    TEXT    NOT NULL,
    snippet     TEXT    NOT NULL DEFAULT '',
    source      TEXT    NOT NULL,
    content_type TEXT   NOT NULL DEFAULT 'web',
    domain      TEXT    NOT NULL DEFAULT '',
    rank        INTEGER NOT NULL DEFAULT 0,
    fetched_at  TEXT    NOT NULL,
    created_at  TEXT    NOT NULL DEFAULT (datetime('now')),
    UNIQUE(url_hash, query, source)
);

CREATE INDEX IF NOT EXISTS idx_results_query      ON search_results(query);
CREATE INDEX IF NOT EXISTS idx_results_source     ON search_results(source);
CREATE INDEX IF NOT EXISTS idx_results_domain     ON search_results(domain);
CREATE INDEX IF NOT EXISTS idx_results_fetched_at ON search_results(fetched_at);
CREATE INDEX IF NOT EXISTS idx_results_url_hash   ON search_results(url_hash);

CREATE VIRTUAL TABLE IF NOT EXISTS search_results_fts USING fts5(
    title,
    snippet,
    url,
    content = 'search_results',
    content_rowid = 'id',
    tokenize = 'porter unicode61'
);

CREATE TRIGGER IF NOT EXISTS search_results_ai AFTER INSERT ON search_results BEGIN
    INSERT INTO search_results_fts(rowid, title, snippet, url)
    VALUES (new.id, new.title, new.snippet, new.url);
END;

CREATE TRIGGER IF NOT EXISTS search_results_ad AFTER DELETE ON search_results BEGIN
    INSERT INTO search_results_fts(search_results_fts, rowid, title, snippet, url)
    VALUES ('delete', old.id, old.title, old.snippet, old.url);
END;

CREATE TRIGGER IF NOT EXISTS search_results_au AFTER UPDATE ON search_results BEGIN
    INSERT INTO search_results_fts(search_results_fts, rowid, title, snippet, url)
    VALUES ('delete', old.id, old.title, old.snippet, old.url);
    INSERT INTO search_results_fts(rowid, title, snippet, url)
    VALUES (new.id, new.title, new.snippet, new.url);
END;

CREATE TABLE IF NOT EXISTS cache_queries (
    id          INTEGER PRIMARY KEY AUTOINCREMENT,
    query       TEXT    NOT NULL,
    query_hash  TEXT    NOT NULL UNIQUE,
    num_results INTEGER NOT NULL DEFAULT 0,
    cached_at   TEXT    NOT NULL DEFAULT (datetime('now')),
    expires_at  TEXT    NOT NULL,
    engines     TEXT    NOT NULL DEFAULT '[]'
);

CREATE INDEX IF NOT EXISTS idx_cache_expires ON cache_queries(expires_at);

CREATE TABLE IF NOT EXISTS cache_content (
    id          INTEGER PRIMARY KEY AUTOINCREMENT,
    url         TEXT    NOT NULL UNIQUE,
    url_hash    TEXT    NOT NULL UNIQUE,
    size_bytes  INTEGER NOT NULL DEFAULT 0,
    mime_type   TEXT    NOT NULL DEFAULT 'text/html',
    cached_at   TEXT    NOT NULL DEFAULT (datetime('now')),
    expires_at  TEXT    NOT NULL,
    file_path   TEXT    NOT NULL
);

CREATE INDEX IF NOT EXISTS idx_content_expires ON cache_content(expires_at);

CREATE TABLE IF NOT EXISTS schema_version (
    version    INTEGER NOT NULL,
    applied_at TEXT    NOT NULL DEFAULT (datetime('now'))
);
`
