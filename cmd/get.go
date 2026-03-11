package cmd

import (
	"context"
	"encoding/json"
	"fmt"
	"os"
	"strconv"

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
	getCmd.PersistentFlags().BoolVarP(&forceRefresh, "refresh", "r", false, "bypass cache and re-fetch from API (only if cached file is older than 10 minutes)")
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

	client := api.New(cfg.APIBaseURL, cfg.APIKey)
	segments, err := episode.FetchTranscripts(context.Background(), client, seq, forceRefresh)
	if err != nil {
		return err
	}

	return printTranscript(segments, transcriptFormat, transcriptSeconds)
}

// printTranscript dispatches to the appropriate format renderer.
func printTranscript(segments []episode.Segment, format string, useSeconds bool) error {
	switch format {
	case "text", "":
		printTranscriptText(segments, useSeconds)
	case "json":
		return printTranscriptJSON(segments, useSeconds)
	case "srt":
		printTranscriptSRT(segments)
	case "vtt":
		printTranscriptVTT(segments)
	default:
		return fmt.Errorf("unknown format %q: use text, json, srt, or vtt", format)
	}
	return nil
}

// timeLabel returns the timestamp string for a segment based on the useSeconds flag.
func timeLabel(seg episode.Segment, useSeconds bool) string {
	if useSeconds {
		return strconv.FormatFloat(seg.Start/1000, 'f', -1, 64)
	}
	return seg.Time
}

// segmentEnd returns the end timestamp (ms) for a segment, falling back to start+2s.
func segmentEnd(seg episode.Segment) float64 {
	if seg.End > seg.Start {
		return seg.End
	}
	return seg.Start + 2000
}

// msToTimestamp converts milliseconds to "HH:MM:SS" + sep + "mmm".
func msToTimestamp(ms float64, sep byte) string {
	total := int(ms)
	millis := total % 1000
	total /= 1000
	secs := total % 60
	total /= 60
	mins := total % 60
	hours := total / 60
	return fmt.Sprintf("%02d:%02d:%02d%c%03d", hours, mins, secs, sep, millis)
}

func printTranscriptText(segments []episode.Segment, useSeconds bool) {
	for _, seg := range segments {
		t := timeLabel(seg, useSeconds)
		if seg.Speaker != "" {
			fmt.Printf("[%s] - %s: %s\n", t, seg.Speaker, seg.Content)
		} else {
			fmt.Printf("[%s] - %s\n", t, seg.Content)
		}
	}
}

func printTranscriptJSON(segments []episode.Segment, useSeconds bool) error {
	type jsonSegment struct {
		Start   any    `json:"start"`
		Speaker string `json:"speaker,omitempty"`
		Content string `json:"content"`
	}

	out := make([]jsonSegment, len(segments))
	for i, seg := range segments {
		var start any
		if useSeconds {
			start = seg.Start / 1000
		} else {
			start = seg.Time
		}
		out[i] = jsonSegment{Start: start, Speaker: seg.Speaker, Content: seg.Content}
	}

	enc := json.NewEncoder(os.Stdout)
	enc.SetIndent("", "  ")
	return enc.Encode(out)
}

func printTranscriptSRT(segments []episode.Segment) {
	for i, seg := range segments {
		fmt.Printf("%d\n%s --> %s\n",
			i+1,
			msToTimestamp(seg.Start, ','),
			msToTimestamp(segmentEnd(seg), ','),
		)
		if seg.Speaker != "" {
			fmt.Printf("%s: %s\n", seg.Speaker, seg.Content)
		} else {
			fmt.Println(seg.Content)
		}
		fmt.Println()
	}
}

func printTranscriptVTT(segments []episode.Segment) {
	fmt.Println("WEBVTT")
	fmt.Println()
	for _, seg := range segments {
		fmt.Printf("%s --> %s\n",
			msToTimestamp(seg.Start, '.'),
			msToTimestamp(segmentEnd(seg), '.'),
		)
		if seg.Speaker != "" {
			fmt.Printf("%s: %s\n", seg.Speaker, seg.Content)
		} else {
			fmt.Println(seg.Content)
		}
		fmt.Println()
	}
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

	client := api.New(cfg.APIBaseURL, cfg.APIKey)
	return episode.FetchSummary(context.Background(), client, seq, forceRefresh)
}
