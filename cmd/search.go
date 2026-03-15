package cmd

import (
	"context"
	"fmt"
	"strings"

	"github.com/hardhacker/podwise-cli/internal/api"
	"github.com/hardhacker/podwise-cli/internal/config"
	"github.com/hardhacker/podwise-cli/internal/episode"
	"github.com/spf13/cobra"
)

const defaultSearchLimit = 10

var searchLimit int
var searchJSONOutput bool

// podwise search <query>
var searchCmd = &cobra.Command{
	Use:   "search <query>",
	Short: "Search for podcast episodes by title keywords",
	Long: `Search for podcast episodes by title keywords.

The query is matched against episode titles. Multiple words are treated as a
single phrase — wrap them in quotes or just pass them as separate arguments.
`,
	Example: `  podwise search "machine learning"
  podwise search "machine learning" --limit 20
  podwise search "machine learning" --json`,
	Args: cobra.MinimumNArgs(1),
	RunE: runSearch,
}

func init() {
	searchCmd.Flags().IntVar(&searchLimit, "limit", defaultSearchLimit, "maximum number of results to return (max 50)")
	searchCmd.Flags().BoolVar(&searchJSONOutput, "json", false, "output results as formatted JSON instead of markdown")
}

func runSearch(cmd *cobra.Command, args []string) error {
	query := strings.Join(args, " ")

	cfg, err := config.Load()
	if err != nil {
		return err
	}
	if err := config.Validate(cfg); err != nil {
		return err
	}

	client := api.New(cfg.APIBaseURL, cfg.APIKey)
	result, err := episode.Search(context.Background(), client, query, searchLimit)
	if err != nil {
		return err
	}

	if searchJSONOutput {
		data, err := result.FormatJSON()
		if err != nil {
			return err
		}
		fmt.Fprintln(cmd.OutOrStdout(), string(data))
		return nil
	}

	fmt.Print(result.FormatText(query))
	return nil
}
