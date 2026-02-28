package cmd

import (
	"context"
	"fmt"
	"time"

	"github.com/cappuch/cuardach/src/cache"
	"github.com/cappuch/cuardach/src/config"
	"github.com/cappuch/cuardach/src/engine"
	"github.com/cappuch/cuardach/src/indexer"
	"github.com/cappuch/cuardach/src/privacy"
	"github.com/cappuch/cuardach/src/web"
	"github.com/spf13/cobra"
)

func init() {
	serveCmd.Flags().IntP("port", "p", 8080, "port to listen on")
	serveCmd.Flags().String("bind", "127.0.0.1", "address to bind to (use 0.0.0.0 for all interfaces)")
	rootCmd.AddCommand(serveCmd)
}

var serveCmd = &cobra.Command{
	Use:   "serve",
	Short: "Start the web server",
	Long:  "Starts an HTTP server with a search UI on localhost.",
	RunE:  runServe,
}

func runServe(cmd *cobra.Command, args []string) error {
	cfg, err := config.Load()
	if err != nil {
		return fmt.Errorf("loading config: %w", err)
	}

	ctx, cancel := context.WithTimeout(context.Background(), 10*time.Second)
	defer cancel()

	idx, err := indexer.NewSQLiteIndexer(cfg.Database.Path)
	if err != nil {
		return fmt.Errorf("opening database: %w", err)
	}
	defer idx.Close()

	if err := idx.Init(ctx); err != nil {
		return fmt.Errorf("initializing database: %w", err)
	}

	cacheStore := cache.NewStore(idx)

	pool := loadProxyPool(cfg)
	client, err := privacy.NewClient(cfg.Privacy, pool)
	if err != nil {
		return fmt.Errorf("creating privacy client: %w", err)
	}

	registry := engine.NewRegistry()
	if cfg.Engines.DuckDuckGo.Enabled {
		registry.Register(engine.NewDuckDuckGo(client, cfg.Engines.DuckDuckGo.BaseURL))
	}
	if cfg.Engines.Bing.Enabled {
		registry.Register(engine.NewBing(cfg.Engines.Bing.BaseURL))
	}

	srv := web.NewServer(cfg, idx, cacheStore, registry)

	port, _ := cmd.Flags().GetInt("port")
	bind, _ := cmd.Flags().GetString("bind")
	addr := fmt.Sprintf("%s:%d", bind, port)

	return web.ListenAndServe(addr, srv.Handler())
}
