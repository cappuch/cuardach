package cmd

import (
	"context"
	"fmt"
	"os"
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
		pool.ClearCache()
		ctx, cancel := context.WithTimeout(context.Background(), 2*time.Minute)
		defer cancel()

		fmt.Println("  fetching proxy lists...")
		start := time.Now()
		n, err := pool.Load(ctx, func(info privacy.ProgressInfo) {
			printProgress(info, start)
		})
		fmt.Fprintln(os.Stderr)

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

// braille spinner frames
var spinFrames = []string{"⠋", "⠙", "⠹", "⠸", "⠼", "⠴", "⠦", "⠧", "⠇", "⠏"}

func printProgress(info privacy.ProgressInfo, start time.Time) {
	pct := 0
	if info.Total > 0 {
		pct = info.Tested * 100 / info.Total
	}

	// bar with [===>    ] style, 40 chars wide
	width := 40
	filled := pct * width / 100
	bar := make([]byte, width)
	for i := range bar {
		if i < filled {
			bar[i] = '='
		} else if i == filled {
			bar[i] = '>'
		} else {
			bar[i] = ' '
		}
	}

	// ETA
	eta := ""
	elapsed := time.Since(start)
	if info.Tested > 0 && info.Tested < info.Total {
		rate := elapsed / time.Duration(info.Tested)
		remaining := rate * time.Duration(info.Total-info.Tested)
		secs := int(remaining.Seconds())
		if secs > 0 {
			eta = fmt.Sprintf("eta %ds", secs)
		} else {
			eta = "eta <1s"
		}
	} else if info.Tested >= info.Total {
		eta = "done"
	}

	// braille spinner — time-based so it spins at constant speed
	spin := spinFrames[int(time.Since(start).Milliseconds()/80)%len(spinFrames)]

	// chunky percentage using block chars
	pctStr := fmt.Sprintf("%d%%", pct)

	fmt.Fprintf(os.Stderr, "\r  %s [%s] %3s  %d/%d  %d ok  %s  ",
		spin, string(bar), pctStr, info.Tested, info.Total, info.Working, eta)
}

func loadProxyPool(cfg *config.Config) *privacy.ProxyPool {
	if !cfg.Privacy.Proxy.Rotate {
		return nil
	}

	pool := privacy.NewProxyPool()
	ctx, cancel := context.WithTimeout(context.Background(), 90*time.Second)
	defer cancel()

	start := time.Now()
	fresh := false
	n, err := pool.Load(ctx, func(info privacy.ProgressInfo) {
		fresh = true
		printProgress(info, start)
	})
	if fresh {
		fmt.Fprintln(os.Stderr)
	}

	if err != nil || n == 0 {
		fmt.Fprintln(os.Stderr, "  warning: no working proxies found, using direct connection")
		return nil
	}

	if fresh {
		fmt.Fprintf(os.Stderr, "  %d proxies validated and cached\n", n)
	} else {
		fmt.Fprintf(os.Stderr, "  %d proxies loaded from cache\n", n)
	}
	return pool
}
