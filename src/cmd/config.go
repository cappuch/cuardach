package cmd

import (
	"encoding/json"
	"fmt"

	"github.com/cappuch/cuardach/src/config"
	"github.com/spf13/cobra"
	"gopkg.in/yaml.v3"
)

func init() {
	rootCmd.AddCommand(configCmd)
	configCmd.AddCommand(configInitCmd)
	configCmd.AddCommand(configShowCmd)
}

var configCmd = &cobra.Command{
	Use:   "config",
	Short: "Manage configuration",
}

var configInitCmd = &cobra.Command{
	Use:   "init",
	Short: "Create default configuration file",
	RunE: func(cmd *cobra.Command, args []string) error {
		cfg := config.DefaultConfig()
		if err := config.Save(cfg); err != nil {
			return err
		}
		fmt.Printf("Configuration written to %s\n", config.ConfigPath())
		return nil
	},
}

var configShowCmd = &cobra.Command{
	Use:   "show",
	Short: "Print current configuration",
	RunE: func(cmd *cobra.Command, args []string) error {
		cfg, err := config.Load()
		if err != nil {
			return err
		}

		format, _ := cmd.Flags().GetString("format")
		if format == "" {
			format = "yaml"
		}

		switch format {
		case "json":
			data, err := json.MarshalIndent(cfg, "", "  ")
			if err != nil {
				return err
			}
			fmt.Println(string(data))
		default:
			data, err := yaml.Marshal(cfg)
			if err != nil {
				return err
			}
			fmt.Print(string(data))
		}
		return nil
	},
}
