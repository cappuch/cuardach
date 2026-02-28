package cmd

import (
	"fmt"

	"github.com/spf13/cobra"
)

var rootCmd = &cobra.Command{
	Use:   "cuardach",
	Short: "privacy-focused search aggregator",
	Long: `cuardach (Irish: search/hunt) - cache-driven, privacy-focused search aggregator.`,
	SilenceUsage:  true,
	SilenceErrors: true,
}

func Execute() error {
	if err := rootCmd.Execute(); err != nil {
		fmt.Fprintf(rootCmd.ErrOrStderr(), "Error: %v\n", err)
		return err
	}
	return nil
}

func init() {
	rootCmd.PersistentFlags().String("config", "", "config file path (default ~/.config/cuardach/config.yaml)")
	rootCmd.PersistentFlags().String("format", "", "output format: table, json, plain")
}
