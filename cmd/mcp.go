package cmd

import (
	"context"
	"errors"
	"fmt"
	"io"
	"strings"
	"time"

	"github.com/hardhacker/podwise-cli/internal/api"
	"github.com/hardhacker/podwise-cli/internal/ask"
	"github.com/hardhacker/podwise-cli/internal/config"
	"github.com/hardhacker/podwise-cli/internal/episode"
	"github.com/hardhacker/podwise-cli/internal/podcast"
	"github.com/modelcontextprotocol/go-sdk/mcp"
	"github.com/spf13/cobra"
)

var mcpCmd = &cobra.Command{
	Use:   "mcp",
	Short: "Start an MCP server that exposes Podwise tools over stdin/stdout",
	Long: `Start an MCP (Model Context Protocol) server that exposes Podwise functionality as tools over stdin/stdout.

The server allows MCP clients to search podcasts and episodes, process media, manage followed
podcasts, list followed podcasts or episodes, and retrieve AI-generated content (transcripts,
summaries, chapters, Q&A, mind maps, highlights, keywords).`,
	Example: `  podwise mcp`,
	Args:    cobra.NoArgs,
	RunE:    runMCP,
}

func runMCP(cmd *cobra.Command, args []string) error {
	server := mcp.NewServer(&mcp.Implementation{Name: "podwise", Version: "v1.0.0"}, nil)

	mcp.AddTool(server, &mcp.Tool{
		Name:        "search_episode",
		Description: "Search for podcast episodes by title keywords. Returns a list of matching episodes with titles, podcast names, publish dates, and episode URLs.",
	}, mcpSearchEpisode)

	mcp.AddTool(server, &mcp.Tool{
		Name:        "search_podcast",
		Description: "Search for podcasts by name. Returns a list of matching podcasts with names, last publish dates, and podcast URLs.",
	}, mcpSearchPodcast)

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

	mcp.AddTool(server, &mcp.Tool{
		Name:        "drill",
		Description: "Drill into a specific podcast and list its recent episodes within a date range, sorted by publish time (newest first). Requires a Podwise podcast URL (https://podwise.ai/dashboard/podcasts/<id>).",
	}, mcpDrill)

	mcp.AddTool(server, &mcp.Tool{
		Name:        "follow",
		Description: "Follow a podcast by its Podwise URL. The operation is idempotent — following an already-followed podcast succeeds silently. Requires a Podwise podcast URL (https://podwise.ai/dashboard/podcasts/<id>).",
	}, mcpFollow)

	mcp.AddTool(server, &mcp.Tool{
		Name:        "unfollow",
		Description: "Unfollow a podcast by its Podwise URL. The operation is idempotent — unfollowing a podcast you do not follow succeeds silently. Requires a Podwise podcast URL (https://podwise.ai/dashboard/podcasts/<id>).",
	}, mcpUnfollow)

	mcp.AddTool(server, &mcp.Tool{
		Name:        "list_episodes",
		Description: "List recent episodes from podcasts the authenticated user follows, sorted by publish time (newest first). This tool is only for followed podcasts. Use 'date' to filter by a specific day (today, yesterday, or YYYY-MM-DD), or 'latest' to look back N days ending today (max 30, default 7).",
	}, mcpListEpisodes)

	mcp.AddTool(server, &mcp.Tool{
		Name:        "list_podcasts",
		Description: "List podcasts the authenticated user follows that have new episodes within a date range, sorted by last publish time (newest first). This tool is only for followed podcasts. Use 'date' to filter by a specific day (today, yesterday, or YYYY-MM-DD), or 'latest' to look back N days ending today (max 30, default 7).",
	}, mcpListPodcasts)

	mcp.AddTool(server, &mcp.Tool{
		Name:        "popular",
		Description: "List the current trending/popular podcast episodes across all languages. Returns episode titles, podcast names, and episode URLs.",
	}, mcpPopular)

	mcp.AddTool(server, &mcp.Tool{
		Name:        "ask",
		Description: "Ask the AI a question based on podcast transcripts. The AI searches relevant podcast transcripts and generates an answer with source citations. Use 'show_sources' to include cited excerpts and episode links in the response. The daily ask limit depends on your Podwise plan.",
	}, mcpAsk)

	mcp.AddTool(server, &mcp.Tool{
		Name:        "history_read",
		Description: "List episodes you have read in Podwise, sorted by most recent first. Use 'limit' to control the number of results (max 100, default 20).",
	}, mcpHistoryRead)

	mcp.AddTool(server, &mcp.Tool{
		Name:        "history_listened",
		Description: "List episodes you have played in Podwise, sorted by most recent first. Use 'limit' to control the number of results (max 100, default 20).",
	}, mcpHistoryListened)

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

// ─── Tool: search_episode / search_podcast ────────────────────────────────────

type mcpSearchInput struct {
	Query string `json:"query"           jsonschema:"search query string"`
	Limit int    `json:"limit,omitempty" jsonschema:"maximum number of results (1-50, default 10)"`
}

func mcpSearchLimit(in int) int {
	if in <= 0 {
		return defaultSearchLimit
	}
	if in > 50 {
		return 50
	}
	return in
}

func mcpSearchEpisode(ctx context.Context, req *mcp.CallToolRequest, in mcpSearchInput) (*mcp.CallToolResult, any, error) {
	if in.Query == "" {
		return nil, nil, fmt.Errorf("query must not be empty")
	}
	client, err := mcpLoadClient()
	if err != nil {
		return nil, nil, err
	}
	result, err := episode.Search(ctx, client, in.Query, mcpSearchLimit(in.Limit))
	if err != nil {
		return nil, nil, err
	}
	return textResult(result.FormatText(in.Query)), nil, nil
}

func mcpSearchPodcast(ctx context.Context, req *mcp.CallToolRequest, in mcpSearchInput) (*mcp.CallToolResult, any, error) {
	if in.Query == "" {
		return nil, nil, fmt.Errorf("query must not be empty")
	}
	client, err := mcpLoadClient()
	if err != nil {
		return nil, nil, err
	}
	result, err := podcast.SearchPodcasts(ctx, client, in.Query, mcpSearchLimit(in.Limit))
	if err != nil {
		return nil, nil, err
	}
	return textResult(result.FormatText(in.Query)), nil, nil
}

// ─── Tool: process ────────────────────────────────────────────────────────────

type mcpProcessInput struct {
	Input    string `json:"input"              jsonschema:"Podwise episode URL / YouTube URL / Xiaoyuzhou URL / local file path"`
	Title    string `json:"title,omitempty"    jsonschema:"episode title for local file uploads (defaults to filename)"`
	Hotwords string `json:"hotwords,omitempty" jsonschema:"comma-separated hotwords to improve transcription accuracy (local files only)"`
}

func mcpProcess(ctx context.Context, req *mcp.CallToolRequest, in mcpProcessInput) (*mcp.CallToolResult, any, error) {
	if in.Input == "" {
		return nil, nil, fmt.Errorf("input must not be empty")
	}

	client, err := mcpLoadClient()
	if err != nil {
		return nil, nil, err
	}

	resolved, err := episode.ResolveInput(ctx, client, in.Input, episode.ResolveOptions{
		Title:    in.Title,
		Hotwords: in.Hotwords,
	})
	if err != nil {
		return nil, nil, err
	}

	var sb strings.Builder

	episodeURL := episode.BuildEpisodeURL(resolved.Seq)
	switch resolved.Kind {
	case episode.KindImport:
		fmt.Fprintf(&sb, "Imported: %q (%s)\nEpisode URL: %s\n\n", resolved.Import.Title, resolved.Import.PodcastName, episodeURL)
	case episode.KindUpload:
		fmt.Fprintf(&sb, "Uploaded: %q\nEpisode URL: %s\n\n", resolved.Upload.Title, episodeURL)
	case episode.KindExisting:
		fmt.Fprintf(&sb, "Episode URL: %s\n\n", episodeURL)
	}

	processResult, err := episode.SubmitProcess(ctx, client, resolved.Seq)
	if err != nil {
		return nil, struct{}{}, err
	}

	if processResult.Status == "done" {
		fmt.Fprintf(&sb, "Status: %s\n", processResult.Status)
		return textResult(sb.String()), struct{}{}, nil
	}

	// Poll until done or timeout (20 min, 30s interval).
	const pollInterval = 30 * time.Second
	const pollTimeout = 20 * time.Minute

	deadline := time.Now().Add(pollTimeout)
	ticker := time.NewTicker(pollInterval)
	defer ticker.Stop()

	var maxProgress float64
	if processResult.Progress != nil {
		maxProgress = *processResult.Progress
	}

	for range ticker.C {
		if time.Now().After(deadline) {
			return nil, struct{}{}, fmt.Errorf("timed out after %s waiting for episode %s to finish processing", pollTimeout, episode.BuildEpisodeURL(resolved.Seq))
		}
		status, err := episode.FetchStatus(ctx, client, resolved.Seq)
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
			return nil, struct{}{}, fmt.Errorf("processing failed for episode %s", episode.BuildEpisodeURL(resolved.Seq))
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

func mcpGetTranscript(ctx context.Context, req *mcp.CallToolRequest, in mcpGetTranscriptInput) (*mcp.CallToolResult, any, error) {
	seq, err := episode.ParseSeq(in.EpisodeURL)
	if err != nil {
		return nil, nil, fmt.Errorf("invalid episode URL: %w", err)
	}

	client, err := mcpLoadClient()
	if err != nil {
		return nil, nil, err
	}

	segments, err := episode.FetchTranscripts(ctx, client, seq, false)
	if err != nil {
		return nil, nil, err
	}

	switch in.Format {
	case "text", "":
		return textResult(episode.FormatTranscriptText(segments, in.Seconds)), nil, nil
	case "srt":
		return textResult(episode.FormatTranscriptSRT(segments)), nil, nil
	case "vtt":
		return textResult(episode.FormatTranscriptVTT(segments)), nil, nil
	default:
		return nil, nil, fmt.Errorf("unknown format %q: use text, srt, or vtt", in.Format)
	}
}

// ─── Tool: get_summary ────────────────────────────────────────────────────────

type mcpEpisodeInput struct {
	EpisodeURL string `json:"episode_url" jsonschema:"Podwise episode URL (https://podwise.ai/dashboard/episodes/<id>)"`
}

func mcpGetSummary(ctx context.Context, req *mcp.CallToolRequest, in mcpEpisodeInput) (*mcp.CallToolResult, any, error) {
	client, err := mcpLoadClient()
	if err != nil {
		return nil, nil, err
	}
	result, err := mcpFetchSummary(ctx, client, in.EpisodeURL)
	if err != nil {
		return nil, nil, err
	}
	return textResult(result.FormatSummary()), nil, nil
}

// ─── Tool: get_qa ─────────────────────────────────────────────────────────────

func mcpGetQA(ctx context.Context, req *mcp.CallToolRequest, in mcpEpisodeInput) (*mcp.CallToolResult, any, error) {
	client, err := mcpLoadClient()
	if err != nil {
		return nil, nil, err
	}
	result, err := mcpFetchSummary(ctx, client, in.EpisodeURL)
	if err != nil {
		return nil, nil, err
	}
	return textResult(result.FormatQA()), nil, nil
}

// ─── Tool: get_chapters ───────────────────────────────────────────────────────

func mcpGetChapters(ctx context.Context, req *mcp.CallToolRequest, in mcpEpisodeInput) (*mcp.CallToolResult, any, error) {
	client, err := mcpLoadClient()
	if err != nil {
		return nil, nil, err
	}
	result, err := mcpFetchSummary(ctx, client, in.EpisodeURL)
	if err != nil {
		return nil, nil, err
	}
	return textResult(result.FormatChapters()), nil, nil
}

// ─── Tool: get_mindmap ────────────────────────────────────────────────────────

func mcpGetMindmap(ctx context.Context, req *mcp.CallToolRequest, in mcpEpisodeInput) (*mcp.CallToolResult, any, error) {
	client, err := mcpLoadClient()
	if err != nil {
		return nil, nil, err
	}
	result, err := mcpFetchSummary(ctx, client, in.EpisodeURL)
	if err != nil {
		return nil, nil, err
	}
	return textResult(result.FormatMindmap()), nil, nil
}

// ─── Tool: get_highlights ─────────────────────────────────────────────────────

func mcpGetHighlights(ctx context.Context, req *mcp.CallToolRequest, in mcpEpisodeInput) (*mcp.CallToolResult, any, error) {
	client, err := mcpLoadClient()
	if err != nil {
		return nil, nil, err
	}
	result, err := mcpFetchSummary(ctx, client, in.EpisodeURL)
	if err != nil {
		return nil, nil, err
	}
	return textResult(result.FormatHighlights()), nil, nil
}

// ─── Tool: get_keywords ───────────────────────────────────────────────────────

func mcpGetKeywords(ctx context.Context, req *mcp.CallToolRequest, in mcpEpisodeInput) (*mcp.CallToolResult, any, error) {
	client, err := mcpLoadClient()
	if err != nil {
		return nil, nil, err
	}
	result, err := mcpFetchSummary(ctx, client, in.EpisodeURL)
	if err != nil {
		return nil, nil, err
	}
	return textResult(result.FormatKeywords()), nil, nil
}

// ─── Tool: drill ──────────────────────────────────────────────────────────────

type mcpDrillInput struct {
	PodcastURL string `json:"podcast_url"      jsonschema:"Podwise podcast URL (https://podwise.ai/dashboard/podcasts/<id>)"`
	Latest     int    `json:"latest,omitempty" jsonschema:"look back N days ending today (1-365, default 30)"`
}

func mcpDrill(ctx context.Context, req *mcp.CallToolRequest, in mcpDrillInput) (*mcp.CallToolResult, any, error) {
	if in.PodcastURL == "" {
		return nil, nil, fmt.Errorf("podcast_url must not be empty")
	}

	podcastSeq, err := podcast.ParseSeq(in.PodcastURL)
	if err != nil {
		return nil, nil, fmt.Errorf("invalid podcast URL: %w", err)
	}

	days := in.Latest
	if days <= 0 {
		days = defaultDrillLatest
	}
	if days > 365 {
		days = 365
	}

	client, err := mcpLoadClient()
	if err != nil {
		return nil, nil, err
	}

	date := episode.Today()
	result, err := podcast.FetchPodcastEpisodes(ctx, client, podcastSeq, date, days)
	if err != nil {
		return nil, nil, err
	}

	return textResult(result.FormatText(date, days)), nil, nil
}

// ─── Tool: follow ─────────────────────────────────────────────────────────────

type mcpPodcastInput struct {
	PodcastURL string `json:"podcast_url" jsonschema:"Podwise podcast URL (https://podwise.ai/dashboard/podcasts/<id>)"`
}

func mcpFollow(ctx context.Context, req *mcp.CallToolRequest, in mcpPodcastInput) (*mcp.CallToolResult, any, error) {
	if in.PodcastURL == "" {
		return nil, nil, fmt.Errorf("podcast_url must not be empty")
	}
	seq, err := podcast.ParseSeq(in.PodcastURL)
	if err != nil {
		return nil, nil, fmt.Errorf("invalid podcast URL: %w", err)
	}
	client, err := mcpLoadClient()
	if err != nil {
		return nil, nil, err
	}
	if err := podcast.Follow(ctx, client, seq); err != nil {
		return nil, nil, err
	}
	return textResult(fmt.Sprintf("Following podcast %s", podcast.BuildPodcastURL(seq))), nil, nil
}

// ─── Tool: unfollow ───────────────────────────────────────────────────────────

func mcpUnfollow(ctx context.Context, req *mcp.CallToolRequest, in mcpPodcastInput) (*mcp.CallToolResult, any, error) {
	if in.PodcastURL == "" {
		return nil, nil, fmt.Errorf("podcast_url must not be empty")
	}
	seq, err := podcast.ParseSeq(in.PodcastURL)
	if err != nil {
		return nil, nil, fmt.Errorf("invalid podcast URL: %w", err)
	}
	client, err := mcpLoadClient()
	if err != nil {
		return nil, nil, err
	}
	if err := podcast.Unfollow(ctx, client, seq); err != nil {
		return nil, nil, err
	}
	return textResult(fmt.Sprintf("Unfollowed podcast %s", podcast.BuildPodcastURL(seq))), nil, nil
}

// ─── Tool: list_episodes ──────────────────────────────────────────────────────

type mcpListInput struct {
	Date   string `json:"date,omitempty"   jsonschema:"specific day to filter by: today, yesterday, or YYYY-MM-DD (takes priority over latest)"`
	Latest int    `json:"latest,omitempty" jsonschema:"look back N days ending today (1-30, default 7); ignored when date is set"`
}

func mcpListEpisodes(ctx context.Context, req *mcp.CallToolRequest, in mcpListInput) (*mcp.CallToolResult, any, error) {
	date, days, err := resolveListDateDays(in.Date, in.Latest, 30)
	if err != nil {
		return nil, nil, err
	}
	client, err := mcpLoadClient()
	if err != nil {
		return nil, nil, err
	}
	result, err := episode.FetchFollowedEpisodes(ctx, client, date, days)
	if err != nil {
		return nil, nil, err
	}
	return textResult(result.FormatText(date, days)), nil, nil
}

// ─── Tool: list_podcasts ──────────────────────────────────────────────────────

func mcpListPodcasts(ctx context.Context, req *mcp.CallToolRequest, in mcpListInput) (*mcp.CallToolResult, any, error) {
	date, days, err := resolveListDateDays(in.Date, in.Latest, 30)
	if err != nil {
		return nil, nil, err
	}
	client, err := mcpLoadClient()
	if err != nil {
		return nil, nil, err
	}
	result, err := podcast.FetchFollowedPodcasts(ctx, client, date, days)
	if err != nil {
		return nil, nil, err
	}
	return textResult(result.FormatText(date, days)), nil, nil
}

// resolveListDateDays converts the MCP date/latest inputs into a canonical
// (date string, days int) pair, mirroring the CLI flag resolution logic.
// maxLatest caps the latest value (30 for followed resources, 365 for podcasts).
func resolveListDateDays(dateStr string, latest, maxLatest int) (string, int, error) {
	if dateStr != "" {
		parsed, err := episode.ParseDate(dateStr)
		if err != nil {
			return "", 0, err
		}
		return parsed, 1, nil
	}
	if latest <= 0 {
		latest = defaultFollowedLatest
	}
	if latest > maxLatest {
		return "", 0, fmt.Errorf("latest must be between 1 and %d", maxLatest)
	}
	return episode.Today(), latest, nil
}

// ─── Tool: popular ────────────────────────────────────────────────────────────

type mcpPopularInput struct {
	Limit int `json:"limit,omitempty" jsonschema:"number of results to return (1-50, default 10)"`
}

func mcpPopular(ctx context.Context, req *mcp.CallToolRequest, in mcpPopularInput) (*mcp.CallToolResult, any, error) {
	limit := in.Limit
	if limit <= 0 {
		limit = defaultPopularLimit
	}
	if limit > 50 {
		limit = 50
	}
	client, err := mcpLoadClient()
	if err != nil {
		return nil, nil, err
	}
	result, err := episode.FetchPopular(ctx, client, limit)
	if err != nil {
		return nil, nil, err
	}
	return textResult(result.FormatText()), nil, nil
}

// ─── Tool: ask ────────────────────────────────────────────────────────────────

type mcpAskInput struct {
	Question    string `json:"question"               jsonschema:"the question to ask the AI based on podcast transcripts"`
	ShowSources bool   `json:"show_sources,omitempty" jsonschema:"if true, include cited source excerpts and episode links in the response"`
}

func mcpAsk(ctx context.Context, req *mcp.CallToolRequest, in mcpAskInput) (*mcp.CallToolResult, any, error) {
	if in.Question == "" {
		return nil, nil, fmt.Errorf("question must not be empty")
	}
	client, err := mcpLoadClient()
	if err != nil {
		return nil, nil, err
	}
	result, err := ask.Ask(ctx, client, in.Question)
	if err != nil {
		return nil, nil, err
	}
	return textResult(result.FormatText(in.Question, in.ShowSources)), nil, nil
}

// ─── Tool: history_read ───────────────────────────────────────────────────────

type mcpHistoryInput struct {
	Limit int `json:"limit,omitempty" jsonschema:"maximum number of results to return (1-100, default 20)"`
}

func mcpHistoryRead(ctx context.Context, req *mcp.CallToolRequest, in mcpHistoryInput) (*mcp.CallToolResult, any, error) {
	limit := in.Limit
	if limit <= 0 {
		limit = 20
	}
	if limit > 100 {
		limit = 100
	}
	client, err := mcpLoadClient()
	if err != nil {
		return nil, nil, err
	}
	result, err := episode.FetchReadHistory(ctx, client, limit)
	if err != nil {
		return nil, nil, err
	}
	return textResult(result.FormatText()), nil, nil
}

// ─── Tool: history_listened ───────────────────────────────────────────────────

func mcpHistoryListened(ctx context.Context, req *mcp.CallToolRequest, in mcpHistoryInput) (*mcp.CallToolResult, any, error) {
	limit := in.Limit
	if limit <= 0 {
		limit = 20
	}
	if limit > 100 {
		limit = 100
	}
	client, err := mcpLoadClient()
	if err != nil {
		return nil, nil, err
	}
	result, err := episode.FetchPlayedHistory(ctx, client, limit)
	if err != nil {
		return nil, nil, err
	}
	return textResult(result.FormatText()), nil, nil
}
