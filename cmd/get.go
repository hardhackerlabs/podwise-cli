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

// podwise get <subcommand>
var getCmd = &cobra.Command{
	Use:   "get <subcommand>",
	Short: "Fetch AI-generated content for a podcast episode or YouTube video",
	Long:  "Fetch AI-generated content (transcript, summary, chapters, Q&A, mind map, highlights, keywords) for a podcast episode or YouTube video via the podwise.ai API.",
	Example: `  podwise get transcript https://podwise.ai/dashboard/episodes/7360326
  podwise get summary    https://podwise.ai/dashboard/episodes/7360326
  podwise get qa         https://podwise.ai/dashboard/episodes/7360326
  podwise get chapters   https://podwise.ai/dashboard/episodes/7360326
  podwise get mindmap    https://podwise.ai/dashboard/episodes/7360326
  podwise get highlights https://podwise.ai/dashboard/episodes/7360326
  podwise get keywords   https://podwise.ai/dashboard/episodes/7360326`,
}

// forceRefresh, when true, bypasses the cache for any get subcommand
// but only if the cached file is older than 10 minutes.
var forceRefresh bool

// getLang is the optional translation language for any get subcommand.
var getLang string

// podwise get transcript <episode-url>
var transcriptSeconds bool
var transcriptFormat string

var getTranscriptCmd = &cobra.Command{
	Use:     "transcript <episode-url>",
	Short:   "Print the full transcript with timestamps and speaker labels",
	Long:    "Print the full AI-generated transcript of a podcast episode or YouTube video. Each line includes a timestamp; speaker names are shown when available.",
	Example: `  podwise get transcript https://podwise.ai/dashboard/episodes/7360326`,
	Args:    cobra.ExactArgs(1),
	RunE:    runGetTranscript,
}

// podwise get summary <episode-url>
var getSummaryCmd = &cobra.Command{
	Use:     "summary <episode-url>",
	Short:   "Print the AI-generated summary and key takeaways",
	Long:    "Print the AI-generated summary of a podcast episode or YouTube video, followed by a numbered list of key takeaways.",
	Example: `  podwise get summary https://podwise.ai/dashboard/episodes/7360326`,
	Args:    cobra.ExactArgs(1),
	RunE:    runGetSummary,
}

// podwise get qa <episode-url>
var getQACmd = &cobra.Command{
	Use:     "qa <episode-url>",
	Short:   "Print AI-extracted Q&A pairs with optional speaker attribution",
	Long:    "Print the question-and-answer pairs extracted by AI from a podcast episode or YouTube video. Speaker names are shown alongside each question and answer when available.",
	Example: `  podwise get qa https://podwise.ai/dashboard/episodes/7360326`,
	Args:    cobra.ExactArgs(1),
	RunE:    runGetQA,
}

// podwise get chapters <episode-url>
var getChaptersCmd = &cobra.Command{
	Use:     "chapters <episode-url>",
	Short:   "Print the AI-generated chapter breakdown with timestamps",
	Long:    "Print the time-stamped chapter breakdown of a podcast episode or YouTube video. Each chapter includes a title and a short summary; chapters containing ads are labeled [ad].",
	Example: `  podwise get chapters https://podwise.ai/dashboard/episodes/7360326`,
	Args:    cobra.ExactArgs(1),
	RunE:    runGetChapters,
}

// podwise get mindmap <episode-url>
var getMindmapCmd = &cobra.Command{
	Use:     "mindmap <episode-url>",
	Short:   "Print the AI-generated mind map in Markdown format",
	Long:    "Print an AI-generated mind map of the key topics in a podcast episode or YouTube video, formatted as a Markdown outline.",
	Example: `  podwise get mindmap https://podwise.ai/dashboard/episodes/7360326`,
	Args:    cobra.ExactArgs(1),
	RunE:    runGetMindmap,
}

// podwise get highlights <episode-url>
var getHighlightsCmd = &cobra.Command{
	Use:     "highlights <episode-url>",
	Short:   "Print AI-extracted notable highlights with timestamps",
	Long:    "Print the notable moments extracted by AI from a podcast episode or YouTube video, each with a timestamp.",
	Example: `  podwise get highlights https://podwise.ai/dashboard/episodes/7360326`,
	Args:    cobra.ExactArgs(1),
	RunE:    runGetHighlights,
}

// podwise get keywords <episode-url>
var getKeywordsCmd = &cobra.Command{
	Use:     "keywords <episode-url>",
	Short:   "Print AI-extracted topic keywords with descriptions",
	Long:    "Print the key topics extracted by AI from a podcast episode or YouTube video. Each keyword is accompanied by a short description.",
	Example: `  podwise get keywords https://podwise.ai/dashboard/episodes/7360326`,
	Args:    cobra.ExactArgs(1),
	RunE:    runGetKeywords,
}

func init() {
	langUsage := "get the translated version in this language: " + strings.Join(episode.LanguageNames(), ", ")
	getCmd.PersistentFlags().BoolVarP(&forceRefresh, "refresh", "r", false, "bypass cache and re-fetch from API (only if cached file is older than 10 minutes)")
	getCmd.PersistentFlags().StringVar(&getLang, "lang", "", langUsage)
	getTranscriptCmd.Flags().BoolVar(&transcriptSeconds, "seconds", false, "show time as start offset in seconds instead of hh:mm:ss")
	getTranscriptCmd.Flags().StringVar(&transcriptFormat, "format", "text", "output format: text, json, srt, vtt")
	getCmd.AddCommand(getTranscriptCmd)
	getCmd.AddCommand(getSummaryCmd)
	getCmd.AddCommand(getQACmd)
	getCmd.AddCommand(getChaptersCmd)
	getCmd.AddCommand(getMindmapCmd)
	getCmd.AddCommand(getHighlightsCmd)
	getCmd.AddCommand(getKeywordsCmd)
}

func runGetTranscript(cmd *cobra.Command, args []string) error {
	seq, err := episode.ParseSeq(args[0])
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

	translationName, err := resolveLangName(getLang)
	if err != nil {
		return err
	}

	client := api.New(cfg.APIBaseURL, cfg.APIKey)
	result, err := episode.FetchTranscripts(context.Background(), client, seq, forceRefresh, translationName)
	if err != nil {
		return err
	}

	return printTranscript(result.Segments, transcriptFormat, transcriptSeconds)
}

// printTranscript dispatches to the appropriate format renderer.
func printTranscript(segments []episode.Segment, format string, useSeconds bool) error {
	switch format {
	case "text", "":
		fmt.Print(episode.FormatTranscriptText(segments, useSeconds))
	case "json":
		data, err := episode.FormatTranscriptJSON(segments, useSeconds)
		if err != nil {
			return err
		}
		fmt.Println(string(data))
	case "srt":
		fmt.Print(episode.FormatTranscriptSRT(segments))
	case "vtt":
		fmt.Print(episode.FormatTranscriptVTT(segments))
	default:
		return fmt.Errorf("unknown format %q: use text, json, srt, or vtt", format)
	}
	return nil
}

func runGetSummary(cmd *cobra.Command, args []string) error {
	result, err := fetchSummaryForURL(args[0])
	if err != nil {
		return err
	}
	printMarkdown(cmd, result.FormatSummary())
	return nil
}

func runGetQA(cmd *cobra.Command, args []string) error {
	result, err := fetchSummaryForURL(args[0])
	if err != nil {
		return err
	}
	printMarkdown(cmd, result.FormatQA())
	return nil
}

func runGetChapters(cmd *cobra.Command, args []string) error {
	result, err := fetchSummaryForURL(args[0])
	if err != nil {
		return err
	}
	printMarkdown(cmd, result.FormatChapters())
	return nil
}

func runGetMindmap(cmd *cobra.Command, args []string) error {
	result, err := fetchSummaryForURL(args[0])
	if err != nil {
		return err
	}
	printMarkdown(cmd, result.FormatMindmap())
	return nil
}

func runGetHighlights(cmd *cobra.Command, args []string) error {
	result, err := fetchSummaryForURL(args[0])
	if err != nil {
		return err
	}
	printMarkdown(cmd, result.FormatHighlights())
	return nil
}

func runGetKeywords(cmd *cobra.Command, args []string) error {
	result, err := fetchSummaryForURL(args[0])
	if err != nil {
		return err
	}
	printMarkdown(cmd, result.FormatKeywords())
	return nil
}

// resolveLangName validates the language name and returns it as-is for use as
// the API translation parameter. Returns an error listing valid names when the
// name is not recognised.
func resolveLangName(name string) (string, error) {
	if name == "" {
		return "", nil
	}
	lang, ok := episode.LookupLanguage(name)
	if !ok {
		return "", fmt.Errorf("unsupported language %q: available languages are %s", name, strings.Join(episode.LanguageNames(), ", "))
	}
	return strings.ReplaceAll(lang.Name, "-", " "), nil
}

// fetchSummaryForURL is a shared helper that parses the episode URL,
// loads config, and fetches (or reads from cache) the summary result.
func fetchSummaryForURL(rawURL string) (*episode.SummaryResult, error) {
	seq, err := episode.ParseSeq(rawURL)
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

	translationName, err := resolveLangName(getLang)
	if err != nil {
		return nil, err
	}

	client := api.New(cfg.APIBaseURL, cfg.APIKey)
	return episode.FetchSummary(context.Background(), client, seq, forceRefresh, translationName)
}
