package cmd

import (
	"fmt"

	"github.com/hardhacker/podwise-cli/internal/config"
	"github.com/spf13/cobra"
)

var configCmd = &cobra.Command{
	Use:   "config",
	Short: "Manage podwise configuration",
}

// podwise config show
var configShowCmd = &cobra.Command{
	Use:   "show",
	Short: "Print current configuration",
	RunE: func(cmd *cobra.Command, args []string) error {
		cfg, err := config.Load()
		if err != nil {
			return err
		}
		path, err := config.FilePath()
		if err != nil {
			return err
		}

		maskedKey := maskAPIKey(cfg.APIKey)

		fmt.Printf("config file : %s\n", path)
		fmt.Printf("api_key     : %s\n", maskedKey)
		fmt.Printf("api_base_url: %s\n", cfg.APIBaseURL)
		return nil
	},
}

// podwise config set <key> <value>
var configSetCmd = &cobra.Command{
	Use:   "set <key> <value>",
	Short: "Set a configuration value",
	Long: `Set a configuration value and save it to the config file.

Available keys:
  api_key       Your podwise.ai API key
  api_base_url  API base URL (default: https://podwise.ai/api)

Examples:
  podwise config set api_key sk-xxxx
  podwise config set api_base_url https://podwise.ai/api`,
	Args: cobra.ExactArgs(2),
	RunE: func(cmd *cobra.Command, args []string) error {
		key, value := args[0], args[1]

		cfg, err := config.Load()
		if err != nil {
			return err
		}

		switch key {
		case "api_key":
			cfg.APIKey = value
		case "api_base_url":
			cfg.APIBaseURL = value
		default:
			return fmt.Errorf("unknown config key %q", key)
		}

		if err := config.Save(cfg); err != nil {
			return err
		}

		fmt.Printf("Saved: %s = %s\n", key, value)
		return nil
	},
}

func init() {
	configCmd.AddCommand(configShowCmd)
	configCmd.AddCommand(configSetCmd)
}

// maskAPIKey shows only the last 4 characters to avoid leaking the key.
func maskAPIKey(key string) string {
	if key == "" {
		return "(not set)"
	}
	if len(key) <= 4 {
		return "****"
	}
	return "****" + key[len(key)-4:]
}
