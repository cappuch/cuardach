package cmd

import (
	"context"
	"fmt"
	"strings"
	"time"

	"github.com/cappuch/cuardach/src/aggregator"
	"github.com/cappuch/cuardach/src/cache"
	"github.com/cappuch/cuardach/src/config"
	"github.com/cappuch/cuardach/src/display"
	"github.com/cappuch/cuardach/src/engine"
	"github.com/cappuch/cuardach/src/indexer"
	"github.com/cappuch/cuardach/src/privacy"
	"github.com/spf13/cobra"
)

func init() {
	searchCmd.Flags().StringSliceP("engines", "e", nil, "engines to use (comma-separated)")
	searchCmd.Flags().IntP("max-results", "n", 0, "max results to display")
	searchCmd.Flags().Bool("no-cache", false, "bypass cache, always query live")
	searchCmd.Flags().String("format", "", "output format: table, json")
	searchCmd.Flags().IntP("page", "p", 1, "result page")
	rootCmd.AddCommand(searchCmd)
}

var searchCmd = &cobra.Command{
	Use:   "search [query]",
	Short: "Search across privacy-respecting engines",
	Long:  "Aggregates results from multiple search engines with full privacy protections.",
	Args:  cobra.MinimumNArgs(1),
	RunE:  runSearch,
}

func runSearch(cmd *cobra.Command, args []string) error {
	query := strings.Join(args, " ")
	ctx, cancel := context.WithTimeout(context.Background(), 60*time.Second)
	defer cancel()

	cfg, err := config.Load()
	if err != nil {
		return fmt.Errorf("loading config: %w", err)
	}

	format, _ := cmd.Flags().GetString("format")
	if format == "" {
		format = cfg.Display.Format
	}
	renderer := display.NewRenderer(format, cfg.Display.Color)

	// Set up database
	idx, err := indexer.NewSQLiteIndexer(cfg.Database.Path)
	if err != nil {
		return fmt.Errorf("opening database: %w", err)
	}
	defer idx.Close()

	if err := idx.Init(ctx); err != nil {
		return fmt.Errorf("initializing database: %w", err)
	}

	cacheStore := cache.NewStore(idx)

	// Check cache first
	noCache, _ := cmd.Flags().GetBool("no-cache")
	if !noCache {
		cached, err := cacheStore.GetResults(ctx, query)
		if err == nil && len(cached) > 0 {
			fmt.Fprintf(cmd.ErrOrStderr(), "  (cached results)\n")
			maxResults, _ := cmd.Flags().GetInt("max-results")
			if maxResults <= 0 {
				maxResults = cfg.Display.MaxResults
			}
			if maxResults > 0 && len(cached) > maxResults {
				cached = cached[:maxResults]
			}
			renderer.RenderResults(cached)
			return nil
		}
	}

	// Set up proxy pool + privacy client
	pool := loadProxyPool(cfg)
	client, err := privacy.NewClient(cfg.Privacy, pool)
	if err != nil {
		return fmt.Errorf("creating privacy client: %w", err)
	}

	// Set up engines
	registry := engine.NewRegistry()
	if cfg.Engines.DuckDuckGo.Enabled {
		registry.Register(engine.NewDuckDuckGo(client, cfg.Engines.DuckDuckGo.BaseURL))
	}
	if cfg.Engines.Bing.Enabled {
		registry.Register(engine.NewBing(cfg.Engines.Bing.BaseURL))
	}
	defer engine.CloseBrowser()

	// Select engines
	engineNames, _ := cmd.Flags().GetStringSlice("engines")
	if len(engineNames) == 0 {
		engineNames = cfg.Engines.Enabled
	}
	engines := registry.Enabled(engineNames)
	if len(engines) == 0 {
		return fmt.Errorf("no engines available (check config)")
	}

	// Search
	maxResults, _ := cmd.Flags().GetInt("max-results")
	if maxResults <= 0 {
		maxResults = cfg.Display.MaxResults
	}
	page, _ := cmd.Flags().GetInt("page")

	agg := aggregator.New(engines)
	results, errs := agg.Search(ctx, engine.SearchParams{
		Query:      query,
		Page:       page,
		MaxResults: maxResults,
	})

	// Cache results
	if len(results) > 0 {
		plain := make([]engine.Result, len(results))
		var sources []string
		for i, r := range results {
			plain[i] = r.Result
			sources = append(sources, r.Sources...)
		}
		cacheStore.PutResults(ctx, query, plain, sources, cfg.Cache.ResultTTL.Duration)
	}

	renderer.RenderAggregated(results, errs)
	return nil
}
