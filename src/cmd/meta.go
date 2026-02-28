package cmd

import (
	"context"
	"fmt"
	"time"

	"github.com/cappuch/cuardach/src/config"
	"github.com/cappuch/cuardach/src/display"
	"github.com/cappuch/cuardach/src/indexer"
	"github.com/cappuch/cuardach/src/meta"
	"github.com/spf13/cobra"
)

func init() {
	metaCmd.AddCommand(metaSearchCmd)

	metaSearchCmd.Flags().StringP("query", "q", "", "full-text search within cached results")
	metaSearchCmd.Flags().StringP("source", "s", "", "filter by source engine")
	metaSearchCmd.Flags().StringP("domain", "d", "", "filter by URL domain")
	metaSearchCmd.Flags().String("content-type", "", "filter by content type")
	metaSearchCmd.Flags().String("after", "", "results fetched after YYYY-MM-DD")
	metaSearchCmd.Flags().String("before", "", "results fetched before YYYY-MM-DD")
	metaSearchCmd.Flags().String("format", "", "output format: table, json")
	metaSearchCmd.Flags().Int("limit", 20, "max results")
	metaSearchCmd.Flags().Int("offset", 0, "pagination offset")

	rootCmd.AddCommand(metaCmd)
}

var metaCmd = &cobra.Command{
	Use:   "meta",
	Short: "Search over locally indexed metadata",
}

var metaSearchCmd = &cobra.Command{
	Use:   "search",
	Short: "Search cached results by metadata filters",
	Long:  "Query the local index using source, domain, date range, and content type filters.",
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

		query, _ := cmd.Flags().GetString("query")
		source, _ := cmd.Flags().GetString("source")
		domain, _ := cmd.Flags().GetString("domain")
		contentType, _ := cmd.Flags().GetString("content-type")
		after, _ := cmd.Flags().GetString("after")
		before, _ := cmd.Flags().GetString("before")
		limit, _ := cmd.Flags().GetInt("limit")
		offset, _ := cmd.Flags().GetInt("offset")

		if query == "" && source == "" && domain == "" && contentType == "" && after == "" && before == "" {
			return fmt.Errorf("at least one filter required (--query, --source, --domain, --content-type, --after, --before)")
		}

		searcher := meta.NewSearcher(idx.DB())
		results, total, err := searcher.Search(ctx, meta.Filter{
			Query:       query,
			Source:      source,
			Domain:      domain,
			ContentType: contentType,
			After:       after,
			Before:      before,
			Limit:       limit,
			Offset:      offset,
		})
		if err != nil {
			return err
		}

		format, _ := cmd.Flags().GetString("format")
		if format == "" {
			format = cfg.Display.Format
		}
		renderer := display.NewRenderer(format, cfg.Display.Color)
		renderer.RenderResults(results)

		if total > len(results) {
			fmt.Fprintf(cmd.ErrOrStderr(), "  (more results available, use --offset %d)\n", offset+limit)
		}

		return nil
	},
}
