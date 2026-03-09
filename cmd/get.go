package cmd

import (
	"context"
	"fmt"
	"path"
	"strconv"
	"strings"

	"github.com/hardhacker/podwise-cli/internal/api"
	"github.com/hardhacker/podwise-cli/internal/config"
	"github.com/hardhacker/podwise-cli/internal/episode"
	"github.com/spf13/cobra"
)

// podwise get <subcommand>
var getCmd = &cobra.Command{
	Use:   "get <subcommand>",
	Short: "Get AI-processed content for a podcast episode",
	Long:  "Get AI-processed content for a podcast episode from podwise.ai.",
	Example: `  podwise get transcript https://podwise.ai/dashboard/episodes/7360326
  podwise get summary    https://podwise.ai/dashboard/episodes/7360326
  podwise get qa         https://podwise.ai/dashboard/episodes/7360326
  podwise get chapters   https://podwise.ai/dashboard/episodes/7360326
  podwise get mindmap    https://podwise.ai/dashboard/episodes/7360326
  podwise get highlights https://podwise.ai/dashboard/episodes/7360326
  podwise get keywords   https://podwise.ai/dashboard/episodes/7360326`,
}

// podwise get transcript <episode-url>
var transcriptSeconds bool

var getTranscriptCmd = &cobra.Command{
	Use:     "transcript <episode-url>",
	Short:   "Get the full transcript of a podcast episode",
	Long:    "Get the full transcript of a podcast episode and print it to stdout.",
	Example: `  podwise get transcript https://podwise.ai/dashboard/episodes/7360326`,
	Args:    cobra.ExactArgs(1),
	RunE:    runGetTranscript,
}

// podwise get summary <episode-url>
var getSummaryCmd = &cobra.Command{
	Use:     "summary <episode-url>",
	Short:   "Get the AI-generated summary of a podcast episode",
	Long:    "Get the AI-generated summary of a podcast episode and print it to stdout.",
	Example: `  podwise get summary https://podwise.ai/dashboard/episodes/7360326`,
	Args:    cobra.ExactArgs(1),
	RunE:    runGetSummary,
}

// podwise get qa <episode-url>
var getQACmd = &cobra.Command{
	Use:     "qa <episode-url>",
	Short:   "Get the Q&A pairs extracted from a podcast episode",
	Long:    "Get the AI-extracted question-and-answer pairs from a podcast episode and print them to stdout.",
	Example: `  podwise get qa https://podwise.ai/dashboard/episodes/7360326`,
	Args:    cobra.ExactArgs(1),
	RunE:    runGetQA,
}

// podwise get chapters <episode-url>
var getChaptersCmd = &cobra.Command{
	Use:     "chapters <episode-url>",
	Short:   "Get the chapter breakdown of a podcast episode",
	Long:    "Get the AI-generated chapter breakdown of a podcast episode and print it to stdout.",
	Example: `  podwise get chapters https://podwise.ai/dashboard/episodes/7360326`,
	Args:    cobra.ExactArgs(1),
	RunE:    runGetChapters,
}

// podwise get mindmap <episode-url>
var getMindmapCmd = &cobra.Command{
	Use:     "mindmap <episode-url>",
	Short:   "Get the mind map of a podcast episode",
	Long:    "Get the AI-generated mind map (in Markdown) of a podcast episode and print it to stdout.",
	Example: `  podwise get mindmap https://podwise.ai/dashboard/episodes/7360326`,
	Args:    cobra.ExactArgs(1),
	RunE:    runGetMindmap,
}

// podwise get highlights <episode-url>
var getHighlightsCmd = &cobra.Command{
	Use:     "highlights <episode-url>",
	Short:   "Get the notable highlights of a podcast episode",
	Long:    "Get the AI-extracted notable highlights of a podcast episode and print them to stdout.",
	Example: `  podwise get highlights https://podwise.ai/dashboard/episodes/7360326`,
	Args:    cobra.ExactArgs(1),
	RunE:    runGetHighlights,
}

// podwise get keywords <episode-url>
var getKeywordsCmd = &cobra.Command{
	Use:     "keywords <episode-url>",
	Short:   "Get the topic keywords of a podcast episode",
	Long:    "Get the AI-extracted topic keywords of a podcast episode and print them to stdout.",
	Example: `  podwise get keywords https://podwise.ai/dashboard/episodes/7360326`,
	Args:    cobra.ExactArgs(1),
	RunE:    runGetKeywords,
}

func init() {
	getTranscriptCmd.Flags().BoolVar(&transcriptSeconds, "seconds", false, "show time as start offset in seconds instead of hh:mm:ss")
	getCmd.AddCommand(getTranscriptCmd)
	getCmd.AddCommand(getSummaryCmd)
	getCmd.AddCommand(getQACmd)
	getCmd.AddCommand(getChaptersCmd)
	getCmd.AddCommand(getMindmapCmd)
	getCmd.AddCommand(getHighlightsCmd)
	getCmd.AddCommand(getKeywordsCmd)
}

func runGetTranscript(cmd *cobra.Command, args []string) error {
	seq, err := parseSeq(args[0])
	if err != nil {
		return fmt.Errorf("invalid episode: %w", err)
	}

	cfg, err := config.Load()
	if err != nil {
		return err
	}
	if err := config.Validate(cfg); err != nil {
		return err
	}

	client := api.New(cfg.APIBaseURL, cfg.APIKey)
	segments, err := episode.FetchTranscripts(context.Background(), client, seq)
	if err != nil {
		return err
	}

	for _, seg := range segments {
		var timeLabel string
		if transcriptSeconds {
			timeLabel = strconv.FormatFloat(seg.Start/1000, 'f', -1, 64)
		} else {
			timeLabel = seg.Time
		}

		if seg.Speaker != "" {
			fmt.Printf("[%s] - %s: %s\n", timeLabel, seg.Speaker, seg.Content)
		} else {
			fmt.Printf("[%s] - %s\n", timeLabel, seg.Content)
		}
	}
	return nil
}

func runGetSummary(cmd *cobra.Command, args []string) error {
	result, err := fetchSummaryForURL(args[0])
	if err != nil {
		return err
	}
	if result.Summary != "" {
		fmt.Println(result.Summary)
	}
	if len(result.Takeaways) > 0 {
		fmt.Println("\nTakeaways:")
		for i, t := range result.Takeaways {
			fmt.Printf("%d. %s\n", i+1, t)
		}
	}
	return nil
}

func runGetQA(cmd *cobra.Command, args []string) error {
	result, err := fetchSummaryForURL(args[0])
	if err != nil {
		return err
	}
	if len(result.QAs) == 0 {
		fmt.Println("(no Q&A available)")
		return nil
	}
	for i, qa := range result.QAs {
		if qa.QuestionSpeaker != "" {
			fmt.Printf("Q%d [%s]: %s\n", i+1, qa.QuestionSpeaker, qa.Question)
		} else {
			fmt.Printf("Q%d: %s\n", i+1, qa.Question)
		}
		if qa.AnswerSpeaker != "" {
			fmt.Printf("A%d [%s]: %s\n", i+1, qa.AnswerSpeaker, qa.Answer)
		} else {
			fmt.Printf("A%d: %s\n", i+1, qa.Answer)
		}
		fmt.Println()
	}
	return nil
}

func runGetChapters(cmd *cobra.Command, args []string) error {
	result, err := fetchSummaryForURL(args[0])
	if err != nil {
		return err
	}
	if len(result.Chapters) == 0 {
		fmt.Println("(no chapters available)")
		return nil
	}
	for i, ch := range result.Chapters {
		adLabel := ""
		if ch.HasAds {
			adLabel = " [ad]"
		}
		fmt.Printf("### [%s] Chapter %d: %s%s\n\n", ch.Time, i+1, ch.Title, adLabel)
		if ch.Summary != "" {
			fmt.Printf("%s\n", ch.Summary)
		}
		fmt.Println()
	}
	return nil
}

func runGetMindmap(cmd *cobra.Command, args []string) error {
	result, err := fetchSummaryForURL(args[0])
	if err != nil {
		return err
	}
	if result.Mindmap == "" {
		fmt.Println("(no mind map available)")
		return nil
	}
	fmt.Println(result.Mindmap)
	return nil
}

func runGetHighlights(cmd *cobra.Command, args []string) error {
	result, err := fetchSummaryForURL(args[0])
	if err != nil {
		return err
	}
	if len(result.Highlights) == 0 {
		fmt.Println("(no highlights available)")
		return nil
	}
	for i, h := range result.Highlights {
		fmt.Printf("%d. [%s] %s\n", i+1, h.Time, h.Content)
	}
	return nil
}

func runGetKeywords(cmd *cobra.Command, args []string) error {
	result, err := fetchSummaryForURL(args[0])
	if err != nil {
		return err
	}
	if len(result.Keywords) == 0 {
		fmt.Println("(no keywords available)")
		return nil
	}
	for i, kw := range result.Keywords {
		if kw.Desc != "" {
			fmt.Printf("%d. **%s**: %s\n", i+1, kw.Key, kw.Desc)
		}
	}
	return nil
}

// fetchSummaryForURL is a shared helper that parses the episode URL,
// loads config, and fetches (or reads from cache) the summary result.
func fetchSummaryForURL(rawURL string) (*episode.SummaryResult, error) {
	seq, err := parseSeq(rawURL)
	if err != nil {
		return nil, fmt.Errorf("invalid episode: %w", err)
	}

	cfg, err := config.Load()
	if err != nil {
		return nil, err
	}
	if err := config.Validate(cfg); err != nil {
		return nil, err
	}

	client := api.New(cfg.APIBaseURL, cfg.APIKey)
	return episode.FetchSummary(context.Background(), client, seq)
}

// parseSeq extracts the integer episode seq from a podwise episode URL.
// Expected format: https://podwise.ai/dashboard/episodes/<seq>
func parseSeq(input string) (int, error) {
	if !strings.HasPrefix(input, "http://") && !strings.HasPrefix(input, "https://") {
		return 0, fmt.Errorf("%q is not a valid episode URL", input)
	}
	raw := path.Base(strings.TrimRight(input, "/"))
	seq, err := strconv.Atoi(raw)
	if err != nil || seq <= 0 {
		return 0, fmt.Errorf("%q does not contain a valid episode ID", input)
	}
	return seq, nil
}
