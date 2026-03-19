package cmd

import (
	"context"
	"fmt"

	"github.com/hardhacker/podwise-cli/internal/api"
	"github.com/hardhacker/podwise-cli/internal/config"
	"github.com/hardhacker/podwise-cli/internal/episode"
	"github.com/hardhacker/podwise-cli/internal/podcast"
	"github.com/spf13/cobra"
)

const defaultDrillLatest = 30

var drillLatest int
var drillJSONOutput bool

// podwise drill <podcast-url>
var drillCmd = &cobra.Command{
	Use:   "drill <podcast-url>",
	Short: "Drill into a podcast and list its recent episodes",
	Long: `Drill into a specific podcast and list its episodes within a date range, sorted by publish time (newest first).

The podcast-url must be a Podwise podcast URL, e.g. https://podwise.ai/dashboard/podcasts/386.

With no flags, shows episodes from the last 30 days ending today by default.
Use --latest N to look back N days ending today (max 365).`,
	Example: `  podwise drill https://podwise.ai/dashboard/podcasts/386
  podwise drill https://podwise.ai/dashboard/podcasts/386 --latest 90
  podwise drill https://podwise.ai/dashboard/podcasts/386 --latest 90 --json`,
	Args: cobra.ExactArgs(1),
	RunE: runDrill,
}

func init() {
	drillCmd.Flags().IntVar(&drillLatest, "latest", defaultDrillLatest, "show episodes from the last N days ending today (max 365)")
	drillCmd.Flags().BoolVar(&drillJSONOutput, "json", false, "output results as formatted JSON instead of markdown")
}

func runDrill(cmd *cobra.Command, args []string) error {
	podcastSeq, err := podcast.ParseSeq(args[0])
	if err != nil {
		return fmt.Errorf("invalid podcast: %w", err)
	}

	if drillLatest < 1 || drillLatest > 365 {
		return fmt.Errorf("--latest must be between 1 and 365")
	}
	date := episode.Today()
	days := drillLatest

	cfg, err := config.Load()
	if err != nil {
		return err
	}
	if err := config.Validate(cfg); err != nil {
		return err
	}

	client := api.New(cfg.APIBaseURL, cfg.APIKey)
	result, err := podcast.FetchPodcastEpisodes(context.Background(), client, podcastSeq, date, days)
	if err != nil {
		return err
	}

	if drillJSONOutput {
		data, err := result.FormatJSON()
		if err != nil {
			return err
		}
		fmt.Fprintln(cmd.OutOrStdout(), string(data))
		return nil
	}

	printMarkdown(cmd, result.FormatText(date, days))
	return nil
}
