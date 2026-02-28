package cmd

import (
	"context"
	"fmt"
	"time"

	"github.com/cappuch/cuardach/src/cache"
	"github.com/cappuch/cuardach/src/config"
	"github.com/cappuch/cuardach/src/display"
	"github.com/cappuch/cuardach/src/indexer"
	"github.com/spf13/cobra"
)

func init() {
	cacheCmd.AddCommand(cacheListCmd)
	cacheCmd.AddCommand(cachePurgeCmd)
	cacheCmd.AddCommand(cacheStatsCmd)
	cacheCmd.AddCommand(cacheGetCmd)

	cacheListCmd.Flags().Int("limit", 20, "max entries to list")
	cachePurgeCmd.Flags().Bool("all", false, "purge ALL cached data")
	cacheGetCmd.Flags().String("format", "", "output format: table, json")

	rootCmd.AddCommand(cacheCmd)
}

var cacheCmd = &cobra.Command{
	Use:   "cache",
	Short: "Manage cached search results",
}

var cacheListCmd = &cobra.Command{
	Use:   "list",
	Short: "List cached queries",
	RunE: func(cmd *cobra.Command, args []string) error {
		ctx, cancel := context.WithTimeout(context.Background(), 10*time.Second)
		defer cancel()

		cfg, err := config.Load()
		if err != nil {
			return err
		}

		idx, err := indexer.NewSQLiteIndexer(cfg.Database.Path)
		if err != nil {
			return err
		}
		defer idx.Close()
		if err := idx.Init(ctx); err != nil {
			return err
		}

		store := cache.NewStore(idx)
		limit, _ := cmd.Flags().GetInt("limit")
		entries, err := store.List(ctx, limit)
		if err != nil {
			return err
		}

		format, _ := cmd.Flags().GetString("format")
		if format == "" {
			format = cfg.Display.Format
		}
		renderer := display.NewRenderer(format, cfg.Display.Color)
		renderer.RenderCacheList(entries)
		return nil
	},
}

var cacheGetCmd = &cobra.Command{
	Use:   "get [query]",
	Short: "Retrieve cached results for a query",
	Args:  cobra.MinimumNArgs(1),
	RunE: func(cmd *cobra.Command, args []string) error {
		ctx, cancel := context.WithTimeout(context.Background(), 10*time.Second)
		defer cancel()

		cfg, err := config.Load()
		if err != nil {
			return err
		}

		idx, err := indexer.NewSQLiteIndexer(cfg.Database.Path)
		if err != nil {
			return err
		}
		defer idx.Close()
		if err := idx.Init(ctx); err != nil {
			return err
		}

		store := cache.NewStore(idx)
		results, err := store.GetResults(ctx, args[0])
		if err != nil {
			return err
		}

		format, _ := cmd.Flags().GetString("format")
		if format == "" {
			format = cfg.Display.Format
		}
		renderer := display.NewRenderer(format, cfg.Display.Color)
		renderer.RenderResults(results)
		return nil
	},
}

var cachePurgeCmd = &cobra.Command{
	Use:   "purge",
	Short: "Remove expired or all cached data",
	RunE: func(cmd *cobra.Command, args []string) error {
		ctx, cancel := context.WithTimeout(context.Background(), 30*time.Second)
		defer cancel()

		cfg, err := config.Load()
		if err != nil {
			return err
		}

		idx, err := indexer.NewSQLiteIndexer(cfg.Database.Path)
		if err != nil {
			return err
		}
		defer idx.Close()
		if err := idx.Init(ctx); err != nil {
			return err
		}

		all, _ := cmd.Flags().GetBool("all")
		store := cache.NewStore(idx)
		n, err := store.Purge(ctx, all)
		if err != nil {
			return err
		}

		if all {
			fmt.Println("  All cached data purged.")
		} else {
			fmt.Printf("  Purged %d expired entries.\n", n)
		}
		return nil
	},
}

var cacheStatsCmd = &cobra.Command{
	Use:   "stats",
	Short: "Show cache statistics",
	RunE: func(cmd *cobra.Command, args []string) error {
		ctx, cancel := context.WithTimeout(context.Background(), 10*time.Second)
		defer cancel()

		cfg, err := config.Load()
		if err != nil {
			return err
		}

		idx, err := indexer.NewSQLiteIndexer(cfg.Database.Path)
		if err != nil {
			return err
		}
		defer idx.Close()
		if err := idx.Init(ctx); err != nil {
			return err
		}

		store := cache.NewStore(idx)
		stats, err := store.Stats(ctx)
		if err != nil {
			return err
		}

		format, _ := cmd.Flags().GetString("format")
		if format == "" {
			format = cfg.Display.Format
		}
		renderer := display.NewRenderer(format, cfg.Display.Color)
		renderer.RenderCacheStats(stats)
		return nil
	},
}
