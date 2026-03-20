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

// podwise list <subcommand>
var listCmd = &cobra.Command{
	Use:   "list <subcommand>",
	Short: "List followed podcasts or episodes",
	Long:  "List the podcasts you follow, or episodes published by the podcasts you follow in Podwise.",
	Example: `  podwise list episodes --date today
  podwise list episodes --date yesterday
  podwise list episodes --latest 3 --json
  podwise list podcasts --date today
  podwise list podcasts --latest 14 --json`,
}

const defaultFollowedLatest = 7

var episodesDate string
var episodesLatest int
var episodesJSONOutput bool

// podwise list episodes
var listEpisodesCmd = &cobra.Command{
	Use:   "episodes",
	Short: "List recent episodes from followed podcasts",
	Long: `List episodes published by podcasts the authenticated user follows.

Episodes are sorted by publish time (newest first).

With no flags, shows today's episodes by default.
Use --date to show episodes for a specific day (today, yesterday, or YYYY-MM-DD).
Use --latest N to show the last N days ending today (max 30).
When --date is provided it takes priority and --latest is ignored.`,
	Example: `  podwise list episodes --date today
  podwise list episodes --date yesterday
  podwise list episodes --date 2025-03-01
  podwise list episodes --latest 7 --json`,
	Args: cobra.NoArgs,
	RunE: runListEpisodes,
}

var podcastsDate string
var podcastsLatest int
var podcastsJSONOutput bool

// podwise list podcasts
var listPodcastsCmd = &cobra.Command{
	Use:   "podcasts",
	Short: "List followed podcasts with recent episodes",
	Long: `List podcasts the authenticated user follows that have new episodes within a date range.

Podcasts are sorted by last publish time (newest first).

With no flags, shows podcasts updated today by default.
Use --date to show podcasts updated on a specific day (today, yesterday, or YYYY-MM-DD).
Use --latest N to show the last N days ending today (max 30).
When --date is provided it takes priority and --latest is ignored.`,
	Example: `  podwise list podcasts --date today
  podwise list podcasts --date yesterday
  podwise list podcasts --date 2025-03-01
  podwise list podcasts --latest 14 --json`,
	Args: cobra.NoArgs,
	RunE: runListPodcasts,
}

func init() {
	listEpisodesCmd.Flags().StringVar(&episodesDate, "date", "", "show episodes for a specific day: today, yesterday, or YYYY-MM-DD (takes priority over --latest)")
	listEpisodesCmd.Flags().IntVar(&episodesLatest, "latest", defaultFollowedLatest, "show the last N days ending today (max 30)")
	listEpisodesCmd.Flags().BoolVar(&episodesJSONOutput, "json", false, "output results as formatted JSON instead of markdown")
	listCmd.AddCommand(listEpisodesCmd)

	listPodcastsCmd.Flags().StringVar(&podcastsDate, "date", "", "show podcasts updated on a specific day: today, yesterday, or YYYY-MM-DD (takes priority over --latest)")
	listPodcastsCmd.Flags().IntVar(&podcastsLatest, "latest", defaultFollowedLatest, "show podcasts with new episodes in the last N days ending today (max 30)")
	listPodcastsCmd.Flags().BoolVar(&podcastsJSONOutput, "json", false, "output results as formatted JSON instead of markdown")
	listCmd.AddCommand(listPodcastsCmd)
}

func runListEpisodes(cmd *cobra.Command, args []string) error {
	var date string
	var days int

	if episodesDate != "" {
		// --date takes priority: show exactly that one day
		parsed, err := episode.ParseDate(episodesDate)
		if err != nil {
			return err
		}
		date = parsed
		days = 1
	} else if cmd.Flags().Changed("latest") {
		// --latest N explicitly provided: look back N days from today
		if episodesLatest < 1 || episodesLatest > 30 {
			return fmt.Errorf("--latest must be between 1 and 30")
		}
		date = episode.Today()
		days = episodesLatest
	} else {
		// no flags: default to today only
		date = episode.Today()
		days = 1
	}

	cfg, err := config.Load()
	if err != nil {
		return err
	}
	if err := config.Validate(cfg); err != nil {
		return err
	}

	client := api.New(cfg.APIBaseURL, cfg.APIKey)
	result, err := episode.FetchFollowedEpisodes(context.Background(), client, date, days)
	if err != nil {
		return err
	}

	if episodesJSONOutput {
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

func runListPodcasts(cmd *cobra.Command, args []string) error {
	var date string
	var days int

	if podcastsDate != "" {
		parsed, err := episode.ParseDate(podcastsDate)
		if err != nil {
			return err
		}
		date = parsed
		days = 1
	} else if cmd.Flags().Changed("latest") {
		if podcastsLatest < 1 || podcastsLatest > 30 {
			return fmt.Errorf("--latest must be between 1 and 30")
		}
		date = episode.Today()
		days = podcastsLatest
	} else {
		date = episode.Today()
		days = 1
	}

	cfg, err := config.Load()
	if err != nil {
		return err
	}
	if err := config.Validate(cfg); err != nil {
		return err
	}

	client := api.New(cfg.APIBaseURL, cfg.APIKey)
	result, err := podcast.FetchFollowedPodcasts(context.Background(), client, date, days)
	if err != nil {
		return err
	}

	if podcastsJSONOutput {
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
