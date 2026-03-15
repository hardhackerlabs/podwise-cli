package cmd

import (
	"context"
	"errors"
	"fmt"
	"io"
	"path/filepath"
	"strconv"
	"strings"
	"time"

	"github.com/hardhacker/podwise-cli/internal/api"
	"github.com/hardhacker/podwise-cli/internal/config"
	"github.com/hardhacker/podwise-cli/internal/episode"
	"github.com/modelcontextprotocol/go-sdk/mcp"
	"github.com/spf13/cobra"
)

var mcpCmd = &cobra.Command{
	Use:   "mcp",
	Short: "Start the Podwise MCP server over stdin/stdout",
	Long: `Start an MCP (Model Context Protocol) server that exposes Podwise functionality as tools.

The server communicates over stdin/stdout using the MCP protocol, allowing AI assistants
such as Cursor and Claude to search episodes, process media, and retrieve AI-generated
content (transcripts, summaries, chapters, Q&A, mind maps, highlights, keywords).`,
	Example: `  podwise mcp`,
	Args:    cobra.NoArgs,
	RunE:    runMCP,
}

func runMCP(cmd *cobra.Command, args []string) error {
	server := mcp.NewServer(&mcp.Implementation{Name: "podwise", Version: "v1.0.0"}, nil)

	mcp.AddTool(server, &mcp.Tool{
		Name:        "search",
		Description: "Search for podcast episodes by title keywords. Returns a list of matching episodes with titles, podcast names, publish dates, and episode URLs.",
	}, mcpSearch)

	mcp.AddTool(server, &mcp.Tool{
		Name:        "process",
		Description: "Submit a podcast episode, YouTube video, Xiaoyuzhou episode, or local media file for AI processing (transcription and analysis). Accepted inputs: Podwise episode URL, YouTube URL, Xiaoyuzhou URL, or local file path (mp3/wav/m4a/mp4/m4v/mov/webm). Returns the episode URL and final processing status.",
	}, mcpProcess)

	mcp.AddTool(server, &mcp.Tool{
		Name:        "get_transcript",
		Description: "Get the full AI-generated transcript of a processed episode. Requires a Podwise episode URL (https://podwise.ai/dashboard/episodes/<id>).",
	}, mcpGetTranscript)

	mcp.AddTool(server, &mcp.Tool{
		Name:        "get_summary",
		Description: "Get the AI-generated summary and key takeaways for a processed episode. Requires a Podwise episode URL.",
	}, mcpGetSummary)

	mcp.AddTool(server, &mcp.Tool{
		Name:        "get_qa",
		Description: "Get AI-extracted question-and-answer pairs from a processed episode. Requires a Podwise episode URL.",
	}, mcpGetQA)

	mcp.AddTool(server, &mcp.Tool{
		Name:        "get_chapters",
		Description: "Get the AI-generated chapter breakdown with timestamps for a processed episode. Requires a Podwise episode URL.",
	}, mcpGetChapters)

	mcp.AddTool(server, &mcp.Tool{
		Name:        "get_mindmap",
		Description: "Get the AI-generated mind map (Markdown outline) for a processed episode. Requires a Podwise episode URL.",
	}, mcpGetMindmap)

	mcp.AddTool(server, &mcp.Tool{
		Name:        "get_highlights",
		Description: "Get AI-extracted notable highlights with timestamps from a processed episode. Requires a Podwise episode URL.",
	}, mcpGetHighlights)

	mcp.AddTool(server, &mcp.Tool{
		Name:        "get_keywords",
		Description: "Get AI-extracted topic keywords with descriptions from a processed episode. Requires a Podwise episode URL.",
	}, mcpGetKeywords)

	err := server.Run(context.Background(), &mcp.StdioTransport{})
	if errors.Is(err, io.EOF) || (err != nil && strings.Contains(err.Error(), "closing")) {
		return nil
	}
	return err
}

// textResult wraps plain text as an MCP CallToolResult.
func textResult(text string) *mcp.CallToolResult {
	return &mcp.CallToolResult{
		Content: []mcp.Content{&mcp.TextContent{Text: text}},
	}
}

// mcpLoadClient is a shared helper: loads config and returns an API client.
func mcpLoadClient() (*api.Client, error) {
	cfg, err := config.Load()
	if err != nil {
		return nil, err
	}
	if err := config.Validate(cfg); err != nil {
		return nil, err
	}
	return api.New(cfg.APIBaseURL, cfg.APIKey), nil
}

// mcpFetchSummary is a thin wrapper around episode.FetchSummary with no cache refresh.
func mcpFetchSummary(ctx context.Context, client *api.Client, rawURL string) (*episode.SummaryResult, error) {
	seq, err := episode.ParseSeq(rawURL)
	if err != nil {
		return nil, fmt.Errorf("invalid episode URL: %w", err)
	}
	return episode.FetchSummary(ctx, client, seq, false)
}

// ─── Tool: search ─────────────────────────────────────────────────────────────

type mcpSearchInput struct {
	Query string `json:"query" jsonschema:"search query string"`
	Limit int    `json:"limit,omitempty" jsonschema:"maximum number of results (1-50, default 10)"`
}

func mcpSearch(ctx context.Context, req *mcp.CallToolRequest, in mcpSearchInput) (*mcp.CallToolResult, struct{}, error) {
	if in.Query == "" {
		return nil, struct{}{}, fmt.Errorf("query must not be empty")
	}
	limit := in.Limit
	if limit <= 0 {
		limit = 10
	}
	if limit > 50 {
		limit = 50
	}

	client, err := mcpLoadClient()
	if err != nil {
		return nil, struct{}{}, err
	}

	result, err := episode.Search(ctx, client, in.Query, limit)
	if err != nil {
		return nil, struct{}{}, err
	}

	if len(result.Hits) == 0 {
		return textResult("No results found."), struct{}{}, nil
	}

	var sb strings.Builder
	fmt.Fprintf(&sb, "# Search: %q\n\n", in.Query)
	fmt.Fprintf(&sb, "**Found:** %d\n\n---\n", len(result.Hits))
	for i, hit := range result.Hits {
		publishDate := time.Unix(hit.PublishTime, 0).Format("2006-01-02")
		fmt.Fprintf(&sb, "\n## %d. %s\n\n", i+1, hit.Title)
		fmt.Fprintf(&sb, "- **Podcast:** %s\n", hit.PodcastName)
		fmt.Fprintf(&sb, "- **Published:** %s\n", publishDate)
		fmt.Fprintf(&sb, "- **Episode URL:** %s\n", episode.BuildEpisodeURL(hit.Seq))
		if hit.Content != "" {
			fmt.Fprintf(&sb, "\n> %s\n", hit.Content)
		}
		sb.WriteString("\n---\n")
	}
	return textResult(sb.String()), struct{}{}, nil
}

// ─── Tool: process ────────────────────────────────────────────────────────────

type mcpProcessInput struct {
	Input    string `json:"input"              jsonschema:"Podwise episode URL / YouTube URL / Xiaoyuzhou URL / local file path"`
	NoWait   bool   `json:"no_wait,omitempty"  jsonschema:"if true submit and return immediately without waiting for completion"`
	Title    string `json:"title,omitempty"    jsonschema:"episode title for local file uploads (defaults to filename)"`
	Hotwords string `json:"hotwords,omitempty" jsonschema:"comma-separated hotwords to improve transcription accuracy (local files only)"`
}

func mcpProcess(ctx context.Context, req *mcp.CallToolRequest, in mcpProcessInput) (*mcp.CallToolResult, struct{}, error) {
	if in.Input == "" {
		return nil, struct{}{}, fmt.Errorf("input must not be empty")
	}

	client, err := mcpLoadClient()
	if err != nil {
		return nil, struct{}{}, err
	}

	var sb strings.Builder
	var seq int

	switch {
	case episode.IsYouTubeURL(in.Input) || episode.IsXiaoyuzhouURL(in.Input):
		result, err := episode.Import(ctx, client, in.Input)
		if err != nil {
			var apiErr *api.APIError
			if errors.As(err, &apiErr) {
				switch apiErr.ErrCode {
				case "private_episode":
					return nil, struct{}{}, fmt.Errorf("episode is private and cannot be imported")
				case "not_found":
					return nil, struct{}{}, fmt.Errorf("video not found: %s", in.Input)
				case "conflict":
					return nil, struct{}{}, fmt.Errorf("import conflict detected, please contact support@podwise.ai")
				case "fetch_error":
					return nil, struct{}{}, fmt.Errorf("failed to fetch episode data, please try again later")
				}
			}
			return nil, struct{}{}, fmt.Errorf("import failed: %w", err)
		}
		seq = result.Seq
		fmt.Fprintf(&sb, "Imported: %q (%s)\nEpisode URL: %s\n\n", result.Title, result.PodcastName, episode.BuildEpisodeURL(seq))

	case episode.IsLocalMediaFile(in.Input):
		title := in.Title
		if title == "" {
			base := filepath.Base(in.Input)
			title = strings.TrimSuffix(base, filepath.Ext(base))
		}
		result, err := episode.Upload(ctx, client, episode.UploadOptions{
			Title:    title,
			FilePath: in.Input,
			Keywords: in.Hotwords,
		})
		if err != nil {
			var cleanupErr *episode.UploadCleanupError
			if errors.As(err, &cleanupErr) && cleanupErr.CleanupErr != nil {
				fmt.Fprintf(&sb, "Warning: orphaned storage object %q — cleanup failed: %v\n", cleanupErr.StoragePath, cleanupErr.CleanupErr)
			}
			return nil, struct{}{}, fmt.Errorf("upload failed: %w", err)
		}
		seq = result.Seq
		fmt.Fprintf(&sb, "Uploaded: %q\nEpisode URL: %s\n\n", result.Title, episode.BuildEpisodeURL(seq))

	default:
		seq, err = episode.ParseSeq(in.Input)
		if err != nil {
			return nil, struct{}{}, fmt.Errorf("invalid input: %w", err)
		}
		fmt.Fprintf(&sb, "Episode URL: %s\n\n", episode.BuildEpisodeURL(seq))
	}

	processResult, err := episode.SubmitProcess(ctx, client, seq)
	if err != nil {
		return nil, struct{}{}, err
	}

	if in.NoWait || processResult.Status == "done" {
		fmt.Fprintf(&sb, "Status: %s\n", processResult.Status)
		return textResult(sb.String()), struct{}{}, nil
	}

	// Poll until done or timeout (10 min, 30s interval).
	const pollInterval = 30 * time.Second
	const pollTimeout = 10 * time.Minute

	deadline := time.Now().Add(pollTimeout)
	ticker := time.NewTicker(pollInterval)
	defer ticker.Stop()

	var maxProgress float64
	if processResult.Progress != nil {
		maxProgress = *processResult.Progress
	}

	for range ticker.C {
		if time.Now().After(deadline) {
			return nil, struct{}{}, fmt.Errorf("timed out after %s waiting for episode %s to finish processing", pollTimeout, episode.BuildEpisodeURL(seq))
		}
		status, err := episode.FetchStatus(ctx, client, seq)
		if err != nil {
			return nil, struct{}{}, err
		}
		if status.Progress != nil && *status.Progress > maxProgress {
			maxProgress = *status.Progress
		}
		switch status.Status {
		case "done":
			fmt.Fprintf(&sb, "Status: done (100%%)\n")
			fmt.Fprintf(&sb, "\nProcessing complete. Use get_transcript, get_summary, etc. to retrieve results.")
			return textResult(sb.String()), struct{}{}, nil
		case "failed":
			return nil, struct{}{}, fmt.Errorf("processing failed for episode %s", episode.BuildEpisodeURL(seq))
		case "processing":
			// continue polling
		}
	}
	return textResult(sb.String()), struct{}{}, nil
}

// ─── Tool: get_transcript ─────────────────────────────────────────────────────

type mcpGetTranscriptInput struct {
	EpisodeURL string `json:"episode_url"          jsonschema:"Podwise episode URL (https://podwise.ai/dashboard/episodes/<id>)"`
	Format     string `json:"format,omitempty"     jsonschema:"output format: text (default), srt, or vtt"`
	Seconds    bool   `json:"seconds,omitempty"    jsonschema:"show timestamps as seconds instead of hh:mm:ss"`
}

func mcpGetTranscript(ctx context.Context, req *mcp.CallToolRequest, in mcpGetTranscriptInput) (*mcp.CallToolResult, struct{}, error) {
	seq, err := episode.ParseSeq(in.EpisodeURL)
	if err != nil {
		return nil, struct{}{}, fmt.Errorf("invalid episode URL: %w", err)
	}

	client, err := mcpLoadClient()
	if err != nil {
		return nil, struct{}{}, err
	}

	segments, err := episode.FetchTranscripts(ctx, client, seq, false)
	if err != nil {
		return nil, struct{}{}, err
	}

	format := in.Format
	if format == "" {
		format = "text"
	}

	var sb strings.Builder
	switch format {
	case "text", "":
		for _, seg := range segments {
			t := mcpTimeLabel(seg, in.Seconds)
			if seg.Speaker != "" {
				fmt.Fprintf(&sb, "[%s] - %s: %s\n", t, seg.Speaker, seg.Content)
			} else {
				fmt.Fprintf(&sb, "[%s] - %s\n", t, seg.Content)
			}
		}
	case "srt":
		for i, seg := range segments {
			fmt.Fprintf(&sb, "%d\n%s --> %s\n",
				i+1,
				msToTimestamp(seg.Start, ','),
				msToTimestamp(segmentEnd(seg), ','),
			)
			if seg.Speaker != "" {
				fmt.Fprintf(&sb, "%s: %s\n", seg.Speaker, seg.Content)
			} else {
				sb.WriteString(seg.Content)
				sb.WriteByte('\n')
			}
			sb.WriteByte('\n')
		}
	case "vtt":
		sb.WriteString("WEBVTT\n\n")
		for _, seg := range segments {
			fmt.Fprintf(&sb, "%s --> %s\n",
				msToTimestamp(seg.Start, '.'),
				msToTimestamp(segmentEnd(seg), '.'),
			)
			if seg.Speaker != "" {
				fmt.Fprintf(&sb, "%s: %s\n", seg.Speaker, seg.Content)
			} else {
				sb.WriteString(seg.Content)
				sb.WriteByte('\n')
			}
			sb.WriteByte('\n')
		}
	default:
		return nil, struct{}{}, fmt.Errorf("unknown format %q: use text, srt, or vtt", format)
	}

	return textResult(sb.String()), struct{}{}, nil
}

func mcpTimeLabel(seg episode.Segment, useSeconds bool) string {
	if useSeconds {
		return strconv.FormatFloat(seg.Start/1000, 'f', -1, 64)
	}
	return seg.Time
}

// ─── Tool: get_summary ────────────────────────────────────────────────────────

type mcpEpisodeInput struct {
	EpisodeURL string `json:"episode_url" jsonschema:"Podwise episode URL (https://podwise.ai/dashboard/episodes/<id>)"`
}

func mcpGetSummary(ctx context.Context, req *mcp.CallToolRequest, in mcpEpisodeInput) (*mcp.CallToolResult, struct{}, error) {
	client, err := mcpLoadClient()
	if err != nil {
		return nil, struct{}{}, err
	}
	result, err := mcpFetchSummary(ctx, client, in.EpisodeURL)
	if err != nil {
		return nil, struct{}{}, err
	}

	var sb strings.Builder
	if result.Summary != "" {
		sb.WriteString(result.Summary)
	}
	if len(result.Takeaways) > 0 {
		sb.WriteString("\n\nTakeaways:\n")
		for i, t := range result.Takeaways {
			fmt.Fprintf(&sb, "%d. %s\n", i+1, t)
		}
	}
	return textResult(sb.String()), struct{}{}, nil
}

// ─── Tool: get_qa ─────────────────────────────────────────────────────────────

func mcpGetQA(ctx context.Context, req *mcp.CallToolRequest, in mcpEpisodeInput) (*mcp.CallToolResult, struct{}, error) {
	client, err := mcpLoadClient()
	if err != nil {
		return nil, struct{}{}, err
	}
	result, err := mcpFetchSummary(ctx, client, in.EpisodeURL)
	if err != nil {
		return nil, struct{}{}, err
	}

	if len(result.QAs) == 0 {
		return textResult("(no Q&A available)"), struct{}{}, nil
	}

	var sb strings.Builder
	for i, qa := range result.QAs {
		if qa.QuestionSpeaker != "" {
			fmt.Fprintf(&sb, "Q%d [%s]: %s\n", i+1, qa.QuestionSpeaker, qa.Question)
		} else {
			fmt.Fprintf(&sb, "Q%d: %s\n", i+1, qa.Question)
		}
		if qa.AnswerSpeaker != "" {
			fmt.Fprintf(&sb, "A%d [%s]: %s\n\n", i+1, qa.AnswerSpeaker, qa.Answer)
		} else {
			fmt.Fprintf(&sb, "A%d: %s\n\n", i+1, qa.Answer)
		}
	}
	return textResult(sb.String()), struct{}{}, nil
}

// ─── Tool: get_chapters ───────────────────────────────────────────────────────

func mcpGetChapters(ctx context.Context, req *mcp.CallToolRequest, in mcpEpisodeInput) (*mcp.CallToolResult, struct{}, error) {
	client, err := mcpLoadClient()
	if err != nil {
		return nil, struct{}{}, err
	}
	result, err := mcpFetchSummary(ctx, client, in.EpisodeURL)
	if err != nil {
		return nil, struct{}{}, err
	}

	if len(result.Chapters) == 0 {
		return textResult("(no chapters available)"), struct{}{}, nil
	}

	var sb strings.Builder
	for i, ch := range result.Chapters {
		adLabel := ""
		if ch.HasAds {
			adLabel = " [ad]"
		}
		fmt.Fprintf(&sb, "### [%s] Chapter %d: %s%s\n\n", ch.Time, i+1, ch.Title, adLabel)
		if ch.Summary != "" {
			fmt.Fprintf(&sb, "%s\n", ch.Summary)
		}
		sb.WriteByte('\n')
	}
	return textResult(sb.String()), struct{}{}, nil
}

// ─── Tool: get_mindmap ────────────────────────────────────────────────────────

func mcpGetMindmap(ctx context.Context, req *mcp.CallToolRequest, in mcpEpisodeInput) (*mcp.CallToolResult, struct{}, error) {
	client, err := mcpLoadClient()
	if err != nil {
		return nil, struct{}{}, err
	}
	result, err := mcpFetchSummary(ctx, client, in.EpisodeURL)
	if err != nil {
		return nil, struct{}{}, err
	}

	if result.Mindmap == "" {
		return textResult("(no mind map available)"), struct{}{}, nil
	}
	return textResult(result.Mindmap), struct{}{}, nil
}

// ─── Tool: get_highlights ─────────────────────────────────────────────────────

func mcpGetHighlights(ctx context.Context, req *mcp.CallToolRequest, in mcpEpisodeInput) (*mcp.CallToolResult, struct{}, error) {
	client, err := mcpLoadClient()
	if err != nil {
		return nil, struct{}{}, err
	}
	result, err := mcpFetchSummary(ctx, client, in.EpisodeURL)
	if err != nil {
		return nil, struct{}{}, err
	}

	if len(result.Highlights) == 0 {
		return textResult("(no highlights available)"), struct{}{}, nil
	}

	var sb strings.Builder
	for i, h := range result.Highlights {
		fmt.Fprintf(&sb, "%d. [%s] %s\n", i+1, h.Time, h.Content)
	}
	return textResult(sb.String()), struct{}{}, nil
}

// ─── Tool: get_keywords ───────────────────────────────────────────────────────

func mcpGetKeywords(ctx context.Context, req *mcp.CallToolRequest, in mcpEpisodeInput) (*mcp.CallToolResult, struct{}, error) {
	client, err := mcpLoadClient()
	if err != nil {
		return nil, struct{}{}, err
	}
	result, err := mcpFetchSummary(ctx, client, in.EpisodeURL)
	if err != nil {
		return nil, struct{}{}, err
	}

	if len(result.Keywords) == 0 {
		return textResult("(no keywords available)"), struct{}{}, nil
	}

	var sb strings.Builder
	for i, kw := range result.Keywords {
		if kw.Desc != "" {
			fmt.Fprintf(&sb, "%d. **%s**: %s\n", i+1, kw.Key, kw.Desc)
		} else {
			fmt.Fprintf(&sb, "%d. **%s**\n", i+1, kw.Key)
		}
	}
	return textResult(sb.String()), struct{}{}, nil
}
