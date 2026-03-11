package cmd

import (
	"context"
	"encoding/json"
	"fmt"
	"strings"
	"time"

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
	Short: "Search for podcast episodes",
	Long:  "Search for podcast episodes across the Podwise database and print results to stdout.",
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

	if len(result.Hits) == 0 {
		if searchJSONOutput {
			fmt.Println("[]")
		} else {
			fmt.Println("(no results found)")
		}
		return nil
	}

	if searchJSONOutput {
		type jsonHit struct {
			Title       string `json:"title"`
			PodcastName string `json:"podcast_name"`
			PublishDate string `json:"publish_date"`
			EpisodeURL  string `json:"episode_url"`
			Description string `json:"description,omitempty"`
		}
		hits := make([]jsonHit, 0, len(result.Hits))
		for _, hit := range result.Hits {
			hits = append(hits, jsonHit{
				Title:       hit.Title,
				PodcastName: hit.PodcastName,
				PublishDate: time.Unix(hit.PublishTime, 0).Format("2006-01-02"),
				EpisodeURL:  episode.BuildEpisodeURL(hit.Seq),
				Description: hit.Content,
			})
		}
		enc := json.NewEncoder(cmd.OutOrStdout())
		enc.SetIndent("", "  ")
		return enc.Encode(hits)
	}

	fmt.Printf("# Search: \"%s\"\n\n", query)
	fmt.Printf("**Found:** %d\n\n", len(result.Hits))
	fmt.Println("---")
	for i, hit := range result.Hits {
		publishDate := time.Unix(hit.PublishTime, 0).Format("2006-01-02")
		fmt.Printf("\n## %d. %s\n\n", i+1, hit.Title)
		fmt.Printf("- **Podcast:** %s\n", hit.PodcastName)
		fmt.Printf("- **Published:** %s\n", publishDate)
		fmt.Printf("- **Episode URL:** %s\n", episode.BuildEpisodeURL(hit.Seq))
		if hit.Content != "" {
			fmt.Printf("\n> %s\n", hit.Content)
		}
		fmt.Println("\n---")
	}
	return nil
}
