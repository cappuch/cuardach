package cmd

import (
	"context"
	"fmt"
	"time"

	"github.com/cappuch/cuardach/src/config"
	"github.com/cappuch/cuardach/src/privacy"
	"github.com/spf13/cobra"
)

func init() {
	proxyCmd.AddCommand(proxyListCmd)
	proxyCmd.AddCommand(proxyTestCmd)
	rootCmd.AddCommand(proxyCmd)
}

var proxyCmd = &cobra.Command{
	Use:   "proxy",
	Short: "Manage rotating proxy pool",
}

var proxyTestCmd = &cobra.Command{
	Use:   "test",
	Short: "Fetch and validate public proxies",
	RunE: func(cmd *cobra.Command, args []string) error {
		pool := privacy.NewProxyPool()
		ctx, cancel := context.WithTimeout(context.Background(), 2*time.Minute)
		defer cancel()

		fmt.Println("  fetching proxy lists...")
		n, err := pool.Load(ctx, func(tested, working int) {
			fmt.Printf("\r  tested: %d  working: %d", tested, working)
		})
		fmt.Println()

		if err != nil {
			return err
		}

		fmt.Printf("  %d working proxies found\n\n", n)
		for _, p := range pool.List() {
			fmt.Printf("  %-6s  %-22s  %dms\n", p.Type, p.Address, p.Latency.Milliseconds())
		}
		return nil
	},
}

var proxyListCmd = &cobra.Command{
	Use:   "list",
	Short: "Show proxy list sources",
	Run: func(cmd *cobra.Command, args []string) {
		fmt.Println("  proxy list sources:")
		fmt.Println("    https://raw.githubusercontent.com/TheSpeedX/PROXY-List/master/http.txt")
		fmt.Println("    https://raw.githubusercontent.com/ShiftyTR/Proxy-List/master/http.txt")
		fmt.Println("    https://raw.githubusercontent.com/monosans/proxy-list/main/proxies/http.txt")
		fmt.Println("    https://raw.githubusercontent.com/hookzof/socks5_list/master/proxy.txt")
		fmt.Println("    https://raw.githubusercontent.com/monosans/proxy-list/main/proxies/socks5.txt")
	},
}

// loadProxyPool fetches and validates proxies if rotation is enabled.
func loadProxyPool(cfg *config.Config) *privacy.ProxyPool {
	if !cfg.Privacy.Proxy.Rotate {
		return nil
	}

	pool := privacy.NewProxyPool()
	ctx, cancel := context.WithTimeout(context.Background(), 90*time.Second)
	defer cancel()

	fresh := false
	n, err := pool.Load(ctx, func(tested, working int) {
		if !fresh {
			fresh = true
			fmt.Print("  fetching proxies...")
		}
		fmt.Printf("\r  validating proxies... tested %d, %d working", tested, working)
	})
	if fresh {
		fmt.Println()
	}

	if err != nil || n == 0 {
		fmt.Println("  warning: no working proxies found, using direct connection")
		return nil
	}

	if fresh {
		fmt.Printf("  %d proxies validated and cached\n", n)
	} else {
		fmt.Printf("  %d proxies loaded from cache\n", n)
	}
	return pool
}
