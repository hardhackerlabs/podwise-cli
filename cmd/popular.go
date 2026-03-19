package cmd

import (
	"context"
	"fmt"

	"github.com/hardhacker/podwise-cli/internal/api"
	"github.com/hardhacker/podwise-cli/internal/config"
	"github.com/hardhacker/podwise-cli/internal/episode"
	"github.com/spf13/cobra"
)

const defaultPopularLimit = 10

var popularLimit int
var popularJSONOutput bool

// podwise popular
var popularCmd = &cobra.Command{
	Use:   "popular",
	Short: "List the current trending/popular podcast episodes",
	Long:  `List the current trending/popular podcast episodes across all languages.`,

	Example: `  podwise popular
  podwise popular --limit 10
  podwise popular --json`,
	Args: cobra.NoArgs,
	RunE: runPopular,
}

func init() {
	popularCmd.Flags().IntVar(&popularLimit, "limit", defaultPopularLimit, "number of results to return (max 50)")
	popularCmd.Flags().BoolVar(&popularJSONOutput, "json", false, "output results as formatted JSON instead of markdown")
}

func runPopular(cmd *cobra.Command, args []string) error {
	cfg, err := config.Load()
	if err != nil {
		return err
	}
	if err := config.Validate(cfg); err != nil {
		return err
	}

	client := api.New(cfg.APIBaseURL, cfg.APIKey)
	result, err := episode.FetchPopular(context.Background(), client, popularLimit)
	if err != nil {
		return err
	}

	if popularJSONOutput {
		data, err := result.FormatJSON()
		if err != nil {
			return err
		}
		fmt.Fprintln(cmd.OutOrStdout(), string(data))
		return nil
	}

	fmt.Print(result.FormatText())
	return nil
}
