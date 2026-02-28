package display

import (
	"encoding/json"
	"fmt"
	"io"
	"os"
	"strings"

	"github.com/cappuch/cuardach/src/aggregator"
	"github.com/cappuch/cuardach/src/cache"
	"github.com/cappuch/cuardach/src/engine"
	"github.com/fatih/color"
)

type Renderer struct {
	out       io.Writer
	format    string
	useColor  bool
}

func NewRenderer(format string, useColor bool) *Renderer {
	if !useColor {
		color.NoColor = true
	}
	return &Renderer{
		out:      os.Stdout,
		format:   format,
		useColor: useColor,
	}
}

func (r *Renderer) RenderAggregated(results []aggregator.AggregatedResult, errs []error) {
	if r.format == "json" {
		data, _ := json.MarshalIndent(results, "", "  ")
		fmt.Fprintln(r.out, string(data))
		return
	}

	if len(results) == 0 && len(errs) == 0 {
		fmt.Fprintln(r.out, "No results found.")
		return
	}

	title := color.New(color.FgCyan, color.Bold)
	urlColor := color.New(color.FgGreen)
	source := color.New(color.FgYellow)
	dim := color.New(color.Faint)

	for i, res := range results {
		fmt.Fprintf(r.out, "\n")
		title.Fprintf(r.out, " %d. %s\n", i+1, res.Title)
		urlColor.Fprintf(r.out, "    %s\n", res.URL)
		if res.Snippet != "" {
			snippet := res.Snippet
			if len(snippet) > 200 {
				snippet = snippet[:200] + "..."
			}
			fmt.Fprintf(r.out, "    %s\n", snippet)
		}
		source.Fprintf(r.out, "    [%s]", strings.Join(res.Sources, ", "))
		dim.Fprintf(r.out, " score: %.4f", res.Score)
		fmt.Fprintln(r.out)
	}

	if len(errs) > 0 {
		fmt.Fprintln(r.out)
		dim.Fprintln(r.out, "  Warnings:")
		for _, err := range errs {
			dim.Fprintf(r.out, "    - %v\n", err)
		}
	}

	fmt.Fprintf(r.out, "\n  %d results\n", len(results))
}

func (r *Renderer) RenderResults(results []engine.Result) {
	if r.format == "json" {
		data, _ := json.MarshalIndent(results, "", "  ")
		fmt.Fprintln(r.out, string(data))
		return
	}

	if len(results) == 0 {
		fmt.Fprintln(r.out, "No results found.")
		return
	}

	title := color.New(color.FgCyan, color.Bold)
	urlColor := color.New(color.FgGreen)
	source := color.New(color.FgYellow)

	for i, res := range results {
		fmt.Fprintf(r.out, "\n")
		title.Fprintf(r.out, " %d. %s\n", i+1, res.Title)
		urlColor.Fprintf(r.out, "    %s\n", res.URL)
		if res.Snippet != "" {
			snippet := res.Snippet
			if len(snippet) > 200 {
				snippet = snippet[:200] + "..."
			}
			fmt.Fprintf(r.out, "    %s\n", snippet)
		}
		source.Fprintf(r.out, "    [%s] %s\n", res.Source, res.Domain)
	}

	fmt.Fprintf(r.out, "\n  %d results\n", len(results))
}

func (r *Renderer) RenderCacheList(entries []cache.Entry) {
	if r.format == "json" {
		data, _ := json.MarshalIndent(entries, "", "  ")
		fmt.Fprintln(r.out, string(data))
		return
	}

	if len(entries) == 0 {
		fmt.Fprintln(r.out, "Cache is empty.")
		return
	}

	header := color.New(color.Bold)
	header.Fprintf(r.out, "\n  %-40s  %-20s  %s\n", "QUERY", "CACHED AT", "RESULTS")
	fmt.Fprintf(r.out, "  %s\n", strings.Repeat("-", 75))

	for _, e := range entries {
		query := e.Query
		if len(query) > 38 {
			query = query[:38] + ".."
		}
		fmt.Fprintf(r.out, "  %-40s  %-20s  %d\n",
			query,
			e.CachedAt.Format("2006-01-02 15:04:05"),
			e.NumResults,
		)
	}
	fmt.Fprintln(r.out)
}

func (r *Renderer) RenderCacheStats(stats *cache.Stats) {
	if r.format == "json" {
		data, _ := json.MarshalIndent(stats, "", "  ")
		fmt.Fprintln(r.out, string(data))
		return
	}

	header := color.New(color.Bold)
	fmt.Fprintln(r.out)
	header.Fprintln(r.out, "  Cache Statistics")
	fmt.Fprintf(r.out, "  Cached queries:  %d\n", stats.TotalQueries)
	fmt.Fprintf(r.out, "  Indexed results: %d\n", stats.TotalResults)
	fmt.Fprintf(r.out, "  Cached content:  %d\n", stats.TotalContent)
	if stats.OldestEntry != "" {
		fmt.Fprintf(r.out, "  Oldest entry:    %s\n", stats.OldestEntry)
	}
	if stats.NewestEntry != "" {
		fmt.Fprintf(r.out, "  Newest entry:    %s\n", stats.NewestEntry)
	}
	fmt.Fprintln(r.out)
}
