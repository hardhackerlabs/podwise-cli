package cmd

import (
	"context"
	"fmt"
	"strings"

	"github.com/hardhacker/podwise-cli/internal/api"
	"github.com/hardhacker/podwise-cli/internal/config"
	"github.com/hardhacker/podwise-cli/internal/episode"
	"github.com/hardhacker/podwise-cli/internal/podcast"
	"github.com/spf13/cobra"
)

const defaultSearchLimit = 10

var searchLimit int
var searchJSONOutput bool

// searchCmd is the parent; running it directly searches episodes (backward compat).
var searchCmd = &cobra.Command{
	Use:   "search <subcommand> <query>",
	Short: "Search for episodes or podcasts by keywords",
	Long: `Search for episodes or podcasts by keywords.

Running "search <query>" without a subcommand is equivalent to "search episode <query>".`,
	Example: `  podwise search "machine learning"
  podwise search episode "machine learning" --limit 20
  podwise search podcast "Lex Fridman" --json`,
	Args: cobra.MinimumNArgs(1),
	RunE: runSearchEpisode,
}

var searchEpisodeCmd = &cobra.Command{
	Use:   "episode <query>",
	Short: "Search for episodes by title keywords",
	Long: `Search for episodes by title keywords.

The query is matched against episode titles. Multiple words are treated as a
single phrase — wrap them in quotes or just pass them as separate arguments.`,
	Example: `  podwise search episode "machine learning"
  podwise search episode "machine learning" --limit 20
  podwise search episode "machine learning" --json`,
	Args: cobra.MinimumNArgs(1),
	RunE: runSearchEpisode,
}

var searchPodcastCmd = &cobra.Command{
	Use:   "podcast <query>",
	Short: "Search for podcasts by name",
	Long: `Search for podcasts by name.

The query is matched against podcast names. Multiple words are treated as a
single phrase — wrap them in quotes or just pass them as separate arguments.`,
	Example: `  podwise search podcast "Lex Fridman" --limit 10
  podwise search podcast "AI" --json`,
	Args: cobra.MinimumNArgs(1),
	RunE: runSearchPodcast,
}

func init() {
	searchCmd.PersistentFlags().IntVar(&searchLimit, "limit", defaultSearchLimit, "maximum number of results to return (max 50)")
	searchCmd.PersistentFlags().BoolVar(&searchJSONOutput, "json", false, "output results as formatted JSON instead of markdown")

	searchCmd.AddCommand(searchEpisodeCmd)
	searchCmd.AddCommand(searchPodcastCmd)
}

func loadClient() (*api.Client, error) {
	cfg, err := config.Load()
	if err != nil {
		return nil, err
	}
	if err := config.Validate(cfg); err != nil {
		return nil, err
	}
	return api.New(cfg.APIBaseURL, cfg.APIKey), nil
}

func runSearchEpisode(cmd *cobra.Command, args []string) error {
	query := strings.Join(args, " ")

	client, err := loadClient()
	if err != nil {
		return err
	}

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

func runSearchPodcast(cmd *cobra.Command, args []string) error {
	query := strings.Join(args, " ")

	client, err := loadClient()
	if err != nil {
		return err
	}

	result, err := podcast.SearchPodcasts(context.Background(), client, query, searchLimit)
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
