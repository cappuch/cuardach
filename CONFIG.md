# Configuration

cuardach is configured via YAML at `~/.config/cuardach/config.yaml`.

Generate a default config:

```bash
cuardach config init
```

View current config:

```bash
cuardach config show
```

---

## Full reference

```yaml
# Search engines
engines:
  # Which engines to query (order matters for display)
  enabled:
    - duckduckgo
    - bing

  duckduckgo:
    enabled: true
    base_url: https://lite.duckduckgo.com   # DuckDuckGo lite HTML endpoint

  bing:
    enabled: true
    base_url: https://www.bing.com           # Bing search (uses headless Chromium)

# Privacy settings
privacy:
  user_agent_rotation: true    # rotate through 21 realistic browser UA strings
  strip_referrer: true         # remove Referer header from all requests

  random_delay:
    enabled: true              # (currently unused — reserved for future rate limiting)
    min_ms: 200
    max_ms: 2000

  proxy:
    enabled: false             # use a fixed proxy for all requests
    address: socks5://127.0.0.1:9050   # proxy address (socks5:// or http://)
    type: socks5               # socks5 or http
    rotate: false              # fetch and rotate through public proxy lists

  dns_over_https:
    enabled: false             # (use proxy remote DNS or system DoH client)
    resolver_url: https://dns.google/dns-query

# Cache settings
cache:
  result_ttl: 24h0m0s         # how long search results stay cached
  content_ttl: 168h0m0s       # how long page content stays cached (7 days)
  content_dir: ~/.cache/cuardach/content   # filesystem cache for page content
  max_size_mb: 500             # max cache size on disk

# Database
database:
  path: ~/.local/share/cuardach/cuardach.db   # SQLite database location

# Display
display:
  format: table                # default output: table, json, plain
  max_results: 20              # default max results per search
  color: true                  # colored terminal output
```

---

## engines

### engines.enabled

List of engine names to query. Engines run in parallel. Results are merged and deduplicated.

```yaml
enabled:
  - duckduckgo       # always works, no deps
  - bing             # needs Chromium (auto-downloaded ~130MB on first use)
```

### engines.duckduckgo

Scrapes DuckDuckGo's lite HTML page. No API key, no JavaScript, no browser needed. Most reliable engine.

| Field | Default | Description |
|-------|---------|-------------|
| `enabled` | `true` | enable/disable |
| `base_url` | `https://lite.duckduckgo.com` | DDG lite endpoint |

### engines.bing

Uses headless Chromium via [rod](https://github.com/go-rod/rod) to render Bing results. On first run, rod auto-downloads a compatible Chromium binary to `~/.cache/rod/browser/`.

| Field | Default | Description |
|-------|---------|-------------|
| `enabled` | `true` | enable/disable |
| `base_url` | `https://www.bing.com` | Bing search URL |

---

## privacy

### privacy.user_agent_rotation

Rotates through a pool of 21 realistic browser user-agent strings (Chrome, Firefox, Safari, Edge on Windows/macOS/Linux/Android). A random UA is picked for each HTTP request.

### privacy.strip_referrer

Removes the `Referer` header from all outbound requests so search engines can't see where you came from.

### privacy.proxy

**Fixed proxy** — route all traffic through a single proxy:

```yaml
proxy:
  enabled: true
  address: socks5://127.0.0.1:9050    # Tor
  type: socks5
```

Supported types: `socks5`, `http`.

When `enabled: true`, this takes precedence over proxy rotation.

**Rotating proxy pool** — route each request through a different public proxy:

```yaml
proxy:
  rotate: true
```

When enabled, cuardach will:
1. Fetch proxies from 5 public GitHub-hosted lists
2. Validate a random sample of 500 by connecting through each to httpbin.org
3. Pick a random working proxy per request
4. Remove dead proxies automatically and fall back to direct connection

Proxy sources:
- `TheSpeedX/PROXY-List` (HTTP)
- `ShiftyTR/Proxy-List` (HTTP)
- `monosans/proxy-list` (HTTP + SOCKS5)
- `hookzof/socks5_list` (SOCKS5)

Test proxies manually:

```bash
cuardach proxy test
```

### privacy.dns_over_https

Reserved for future use. When using a SOCKS5 proxy, DNS is resolved remotely by the proxy, preventing DNS leaks. For standalone DoH, use a system-level resolver like `dnscrypt-proxy`.

---

## cache

| Field | Default | Description |
|-------|---------|-------------|
| `result_ttl` | `24h` | TTL for cached search results |
| `content_ttl` | `168h` (7d) | TTL for cached page content |
| `content_dir` | `~/.cache/cuardach/content` | directory for content blobs |
| `max_size_mb` | `500` | max disk usage for content cache |

Manage the cache:

```bash
cuardach cache stats            # show counts
cuardach cache list             # list cached queries
cuardach cache purge            # remove expired entries
cuardach cache purge --all      # remove everything
```

---

## database

| Field | Default | Description |
|-------|---------|-------------|
| `path` | `~/.local/share/cuardach/cuardach.db` | SQLite database file |

The database uses WAL mode for concurrent reads and FTS5 for full-text search. Paths with `~` are expanded to your home directory.

---

## display

| Field | Default | Description |
|-------|---------|-------------|
| `format` | `table` | output format: `table`, `json`, `plain` |
| `max_results` | `20` | default max results per search |
| `color` | `true` | colored terminal output |

Override per-command:

```bash
cuardach search "query" --format json
cuardach search "query" -n 5
```

---

## File locations

| File | Path |
|------|------|
| Config | `~/.config/cuardach/config.yaml` |
| Database | `~/.local/share/cuardach/cuardach.db` |
| Content cache | `~/.cache/cuardach/content/` |
| Chromium (rod) | `~/.cache/rod/browser/` |

All paths follow XDG conventions. Override config path with `--config /path/to/config.yaml`.

---
Note: this is a LLM generated file. Contents may be inaccurate or outdated. Always refer to the source code for the most up-to-date information.