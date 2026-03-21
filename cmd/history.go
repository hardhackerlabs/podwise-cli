package cmd

import (
	"context"
	"fmt"

	"github.com/hardhacker/podwise-cli/internal/api"
	"github.com/hardhacker/podwise-cli/internal/config"
	"github.com/hardhacker/podwise-cli/internal/episode"
	"github.com/spf13/cobra"
)

// podwise history <subcommand>
var historyCmd = &cobra.Command{
	Use:   "history <subcommand>",
	Short: "List recently listened and read episodes",
	Long:  "List your recently listened and read episodes in Podwise.",
	Example: `  podwise history read
  podwise history read --limit 50 --json
  podwise history listened
  podwise history listened --limit 50 --json`,
}

var historyLimit int
var historyJSONOutput bool

// podwise history read
var historyReadCmd = &cobra.Command{
	Use:   "read",
	Short: "List recently read episodes",
	Long:  `List episodes you have read in Podwise, sorted by most recent first.`,

	Example: `  podwise history read
  podwise history read --limit 50 --json`,
	Args: cobra.NoArgs,
	RunE: runHistoryRead,
}

// podwise history listened
var historyListenedCmd = &cobra.Command{
	Use:   "listened",
	Short: "List recently listened episodes",
	Long:  `List episodes you have listened in Podwise, sorted by most recent first.`,

	Example: `  podwise history listened
  podwise history listened --limit 50 --json`,
	Args: cobra.NoArgs,
	RunE: runHistoryListened,
}

func init() {
	historyReadCmd.Flags().IntVar(&historyLimit, "limit", 20, "maximum number of results to return (max 100)")
	historyReadCmd.Flags().BoolVar(&historyJSONOutput, "json", false, "output results as formatted JSON instead of markdown")
	historyCmd.AddCommand(historyReadCmd)

	historyListenedCmd.Flags().IntVar(&historyLimit, "limit", 20, "maximum number of results to return (max 100)")
	historyListenedCmd.Flags().BoolVar(&historyJSONOutput, "json", false, "output results as formatted JSON instead of markdown")
	historyCmd.AddCommand(historyListenedCmd)
}

func runHistoryRead(cmd *cobra.Command, args []string) error {
	if historyLimit < 1 || historyLimit > 100 {
		return fmt.Errorf("--limit must be between 1 and 100")
	}

	cfg, err := config.Load()
	if err != nil {
		return err
	}
	if err := config.Validate(cfg); err != nil {
		return err
	}

	client := api.New(cfg.APIBaseURL, cfg.APIKey)
	result, err := episode.FetchReadHistory(context.Background(), client, historyLimit)
	if err != nil {
		return err
	}

	if historyJSONOutput {
		data, err := result.FormatJSON()
		if err != nil {
			return err
		}
		fmt.Fprintln(cmd.OutOrStdout(), string(data))
		return nil
	}

	printMarkdown(cmd, result.FormatText())
	return nil
}

func runHistoryListened(cmd *cobra.Command, args []string) error {
	if historyLimit < 1 || historyLimit > 100 {
		return fmt.Errorf("--limit must be between 1 and 100")
	}

	cfg, err := config.Load()
	if err != nil {
		return err
	}
	if err := config.Validate(cfg); err != nil {
		return err
	}

	client := api.New(cfg.APIBaseURL, cfg.APIKey)
	result, err := episode.FetchPlayedHistory(context.Background(), client, historyLimit)
	if err != nil {
		return err
	}

	if historyJSONOutput {
		data, err := result.FormatJSON()
		if err != nil {
			return err
		}
		fmt.Fprintln(cmd.OutOrStdout(), string(data))
		return nil
	}

	printMarkdown(cmd, result.FormatText())
	return nil
}
