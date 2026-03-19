package cmd

import (
	"context"
	"fmt"

	"github.com/hardhacker/podwise-cli/internal/api"
	"github.com/hardhacker/podwise-cli/internal/config"
	"github.com/hardhacker/podwise-cli/internal/episode"
	"github.com/spf13/cobra"
)

// podwise list <subcommand>
var listCmd = &cobra.Command{
	Use:   "list <subcommand>",
	Short: "List episodes from your account",
	Long:  "List episodes related to your Podwise account, such as episodes from podcasts you follow.",
	Example: `  podwise list followed-episodes --date today
  podwise list followed-episodes --date yesterday
  podwise list followed-episodes --latest 3 --json`,
}

const defaultFollowedLatest = 7

var followedDate string
var followedLatest int
var followedJSONOutput bool

// podwise list followed-episodes
var listFollowedCmd = &cobra.Command{
	Use:   "followed-episodes",
	Short: "List recent episodes from podcasts you follow",
	Long: `List episodes published by podcasts the authenticated user follows.

Episodes are sorted by publish time (newest first).

With no flags, shows today's episodes by default.
Use --date to show episodes for a specific day (today, yesterday, or YYYY-MM-DD).
Use --latest N to show the last N days ending today (max 30).
When --date is provided it takes priority and --latest is ignored.`,
	Example: `  podwise list followed-episodes --date today
  podwise list followed-episodes --date yesterday
  podwise list followed-episodes --date 2025-03-01
  podwise list followed-episodes --latest 7 --json`,
	Args: cobra.NoArgs,
	RunE: runListFollowed,
}

func init() {
	listFollowedCmd.Flags().StringVar(&followedDate, "date", "", "show episodes for a specific day: today, yesterday, or YYYY-MM-DD (takes priority over --latest)")
	listFollowedCmd.Flags().IntVar(&followedLatest, "latest", defaultFollowedLatest, "show the last N days ending today (max 30)")
	listFollowedCmd.Flags().BoolVar(&followedJSONOutput, "json", false, "output results as formatted JSON instead of markdown")
	listCmd.AddCommand(listFollowedCmd)
}

func runListFollowed(cmd *cobra.Command, args []string) error {
	var date string
	var days int

	if followedDate != "" {
		// --date takes priority: show exactly that one day
		parsed, err := episode.ParseDate(followedDate)
		if err != nil {
			return err
		}
		date = parsed
		days = 1
	} else if cmd.Flags().Changed("latest") {
		// --latest N explicitly provided: look back N days from today
		if followedLatest < 1 || followedLatest > 30 {
			return fmt.Errorf("--latest must be between 1 and 30")
		}
		date = episode.Today()
		days = followedLatest
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

	if followedJSONOutput {
		data, err := result.FormatJSON()
		if err != nil {
			return err
		}
		fmt.Fprintln(cmd.OutOrStdout(), string(data))
		return nil
	}

	fmt.Print(result.FormatText(date, days))
	return nil
}
