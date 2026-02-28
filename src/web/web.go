package web

import (
	"context"
	"fmt"
	"html"
	"net/http"
	"strings"
	"time"

	"github.com/cappuch/cuardach/src/aggregator"
	"github.com/cappuch/cuardach/src/cache"
	"github.com/cappuch/cuardach/src/config"
	"github.com/cappuch/cuardach/src/engine"
	"github.com/cappuch/cuardach/src/indexer"
	"github.com/cappuch/cuardach/src/meta"
)

type Server struct {
	cfg      *config.Config
	idx      *indexer.SQLiteIndexer
	cache    *cache.Store
	registry *engine.Registry
	layout   string
}

func NewServer(cfg *config.Config, idx *indexer.SQLiteIndexer, c *cache.Store, reg *engine.Registry) *Server {
	data, _ := staticFS.ReadFile("static/index.html")
	return &Server{cfg: cfg, idx: idx, cache: c, registry: reg, layout: string(data)}
}

func (s *Server) Handler() http.Handler {
	mux := http.NewServeMux()
	mux.HandleFunc("/", s.handleSearch)
	mux.HandleFunc("/cache", s.handleCache)
	mux.HandleFunc("/cache/purge", s.handlePurge)
	mux.HandleFunc("/meta", s.handleMeta)
	return mux
}

func (s *Server) page(w http.ResponseWriter, body string) {
	w.Header().Set("Content-Type", "text/html; charset=utf-8")
	fmt.Fprint(w, strings.Replace(s.layout, "{{BODY}}", body, 1))
}

// GET / and GET /?q=...
func (s *Server) handleSearch(w http.ResponseWriter, r *http.Request) {
	if r.URL.Path != "/" {
		http.NotFound(w, r)
		return
	}

	q := r.URL.Query().Get("q")
	var b strings.Builder

	b.WriteString(`<form action="/" method="get">`)
	b.WriteString(`<input type="text" name="q" placeholder="search..." value="` + html.EscapeString(q) + `" autofocus> `)
	b.WriteString(`<input type="submit" value="search">`)
	b.WriteString(`</form>`)

	if q == "" {
		s.page(w, b.String())
		return
	}

	ctx, cancel := context.WithTimeout(r.Context(), 30*time.Second)
	defer cancel()

	// check cache
	cached, _ := s.cache.GetResults(ctx, q)
	if len(cached) > 0 {
		b.WriteString(fmt.Sprintf(`<p>%d results (cached)</p>`, len(cached)))
		for i, r := range cached {
			writeResult(&b, i+1, r.Title, r.URL, r.Snippet, r.Source, r.Domain)
		}
		s.page(w, b.String())
		return
	}

	// live search
	engines := s.registry.Enabled(s.cfg.Engines.Enabled)
	if len(engines) == 0 {
		b.WriteString(`<p>no engines configured</p>`)
		s.page(w, b.String())
		return
	}

	agg := aggregator.New(engines)
	results, errs := agg.Search(ctx, engine.SearchParams{
		Query:      q,
		MaxResults: s.cfg.Display.MaxResults,
	})

	// cache results
	if len(results) > 0 {
		plain := make([]engine.Result, len(results))
		var sources []string
		for i, res := range results {
			plain[i] = res.Result
			sources = append(sources, res.Sources...)
		}
		s.cache.PutResults(ctx, q, plain, sources, s.cfg.Cache.ResultTTL.Duration)
	}

	b.WriteString(fmt.Sprintf(`<p>%d results</p>`, len(results)))

	for i, r := range results {
		src := strings.Join(r.Sources, ", ")
		writeResult(&b, i+1, r.Title, r.URL, r.Snippet, src, r.Domain)
	}

	for _, e := range errs {
		b.WriteString(`<p class="warn">` + html.EscapeString(e.Error()) + `</p>`)
	}

	s.page(w, b.String())
}

// GET /cache
func (s *Server) handleCache(w http.ResponseWriter, r *http.Request) {
	ctx, cancel := context.WithTimeout(r.Context(), 5*time.Second)
	defer cancel()

	var b strings.Builder

	stats, _ := s.cache.Stats(ctx)
	if stats != nil {
		b.WriteString(fmt.Sprintf(`<p>%d queries &middot; %d results indexed</p>`, stats.TotalQueries, stats.TotalResults))
	}

	b.WriteString(`<form action="/cache/purge" method="post"><button type="submit">purge expired</button></form>`)

	entries, _ := s.cache.List(ctx, 50)
	if len(entries) == 0 {
		b.WriteString(`<p>cache is empty</p>`)
	} else {
		b.WriteString(`<table><tr><th>query</th><th>results</th><th>cached</th></tr>`)
		for _, e := range entries {
			t := e.CachedAt.Format("2006-01-02 15:04")
			b.WriteString(fmt.Sprintf(`<tr><td><a href="/?q=%s">%s</a></td><td>%d</td><td>%s</td></tr>`,
				html.EscapeString(e.Query), html.EscapeString(e.Query), e.NumResults, t))
		}
		b.WriteString(`</table>`)
	}

	s.page(w, b.String())
}

// POST /cache/purge
func (s *Server) handlePurge(w http.ResponseWriter, r *http.Request) {
	if r.Method != http.MethodPost {
		http.Redirect(w, r, "/cache", http.StatusSeeOther)
		return
	}
	ctx, cancel := context.WithTimeout(r.Context(), 10*time.Second)
	defer cancel()
	s.cache.Purge(ctx, false)
	http.Redirect(w, r, "/cache", http.StatusSeeOther)
}

// GET /meta and GET /meta?q=...&source=...&domain=...&after=...&before=...
func (s *Server) handleMeta(w http.ResponseWriter, r *http.Request) {
	ctx, cancel := context.WithTimeout(r.Context(), 5*time.Second)
	defer cancel()

	params := r.URL.Query()
	fq := params.Get("q")
	fsrc := params.Get("source")
	fdom := params.Get("domain")
	fafter := params.Get("after")
	fbefore := params.Get("before")

	var b strings.Builder
	b.WriteString(`<form action="/meta" method="get">`)
	b.WriteString(`<input type="text" name="q" placeholder="text search" value="` + html.EscapeString(fq) + `"> `)
	b.WriteString(`<select name="source"><option value="">source</option>`)
	for _, name := range []string{"duckduckgo", "bing"} {
		sel := ""
		if fsrc == name {
			sel = ` selected`
		}
		b.WriteString(`<option` + sel + `>` + name + `</option>`)
	}
	b.WriteString(`</select> `)
	b.WriteString(`<input type="text" name="domain" placeholder="domain" value="` + html.EscapeString(fdom) + `" style="width:120px"> `)
	b.WriteString(`<input type="date" name="after" value="` + html.EscapeString(fafter) + `"> `)
	b.WriteString(`<input type="date" name="before" value="` + html.EscapeString(fbefore) + `"> `)
	b.WriteString(`<input type="submit" value="filter">`)
	b.WriteString(`</form>`)

	hasFilter := fq != "" || fsrc != "" || fdom != "" || fafter != "" || fbefore != ""
	if !hasFilter {
		s.page(w, b.String())
		return
	}

	searcher := meta.NewSearcher(s.idx.DB())
	results, _, err := searcher.Search(ctx, meta.Filter{
		Query:  fq,
		Source: fsrc,
		Domain: fdom,
		After:  fafter,
		Before: fbefore,
		Limit:  50,
	})
	if err != nil {
		b.WriteString(`<p class="warn">` + html.EscapeString(err.Error()) + `</p>`)
		s.page(w, b.String())
		return
	}

	b.WriteString(fmt.Sprintf(`<p>%d results</p>`, len(results)))
	for i, r := range results {
		writeResult(&b, i+1, r.Title, r.URL, r.Snippet, r.Source, r.Domain)
	}

	s.page(w, b.String())
}

func writeResult(b *strings.Builder, rank int, title, url, snippet, source, domain string) {
	b.WriteString(`<div class="r">`)
	b.WriteString(fmt.Sprintf(`<div class="t">%d. <a href="%s" rel="noopener">%s</a></div>`,
		rank, html.EscapeString(url), html.EscapeString(title)))
	b.WriteString(`<div class="u">` + html.EscapeString(url) + `</div>`)
	if snippet != "" {
		s := snippet
		if len(s) > 200 {
			s = s[:200] + "..."
		}
		b.WriteString(`<div class="s">` + html.EscapeString(s) + `</div>`)
	}
	b.WriteString(`<div class="m">[` + html.EscapeString(source) + `] ` + html.EscapeString(domain) + `</div>`)
	b.WriteString(`</div>`)
}

func ListenAndServe(addr string, handler http.Handler) error {
	fmt.Printf("  cuardach listening on http://%s\n", addr)
	return http.ListenAndServe(addr, handler)
}
