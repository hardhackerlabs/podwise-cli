package episode

import (
	"context"
	"encoding/json"
	"fmt"
	"os"
	"path/filepath"
	"strings"
	"time"

	"github.com/hardhacker/podwise-cli/internal/api"
	"github.com/hardhacker/podwise-cli/internal/async"
	"github.com/hardhacker/podwise-cli/internal/utils"
)

// clipCreatedAtLayout is yyyy-mm-dd hh:mm:ss in local time for CLI readability.
const clipCreatedAtLayout = "2006-01-02 15:04:05"

// Clip is a single user clip on an episode.
type Clip struct {
	ID           int      `json:"id"`
	EpisodeID    string   `json:"episodeId"`
	EpisodeSeq   int      `json:"episodeSeq"`
	EpisodeTitle string   `json:"episodeTitle"`
	EpisodeCover *string  `json:"episodeCover"`
	PodcastName  string   `json:"podcastName"`
	PodcastCover *string  `json:"podcastCover"`
	Title        string   `json:"title"`
	Takeaways    []string `json:"takeaways"`
	Content      *string  `json:"content"`
	Point        *int     `json:"point"`
	ClipStart    *int     `json:"clipStart"`
	ClipEnd      *int     `json:"clipEnd"`
	Status       string   `json:"status"`
	Exportable   bool     `json:"exportable"`
	CreatedAt    int64    `json:"createdAt"`
}

// ClipsEpisodeInfo is the episode summary returned alongside the clips list.
type ClipsEpisodeInfo struct {
	EpisodeID   string `json:"episodeId"`
	Seq         int    `json:"seq"`
	Title       string `json:"title"`
	PodcastName string `json:"podcastName"`
}

// ClipsResult is the decoded list response for GET /episodes/{seq}/clips.
type ClipsResult struct {
	Clips                 []Clip
	ClipCount             int
	ExportableClipCount   int
	UnexportableClipCount int
	Episode               ClipsEpisodeInfo
}

type clipsAPIResponse struct {
	Success bool `json:"success"`
	Result  struct {
		Clips                 []Clip `json:"clips"`
		ClipCount             int    `json:"clipCount"`
		ExportableClipCount   int    `json:"exportableClipCount"`
		UnexportableClipCount int    `json:"unexportableClipCount"`
	} `json:"result"`
	Episode ClipsEpisodeInfo `json:"episode"`
}

// FetchEpisodeClips returns the authenticated user's clips for the episode seq.
// GET /open/v1/episodes/{seq}/clips
func FetchEpisodeClips(ctx context.Context, client *api.Client, seq int) (*ClipsResult, error) {
	path := fmt.Sprintf("/open/v1/episodes/%d/clips", seq)
	var resp clipsAPIResponse
	if err := client.Get(ctx, path, nil, &resp); err != nil {
		return nil, err
	}
	out := &ClipsResult{
		Clips:                 resp.Result.Clips,
		ClipCount:             resp.Result.ClipCount,
		ExportableClipCount:   resp.Result.ExportableClipCount,
		UnexportableClipCount: resp.Result.UnexportableClipCount,
		Episode:               resp.Episode,
	}
	return out, nil
}

// ClipsMarkdownExportOptions configures ExportClipsToMarkdown.
type ClipsMarkdownExportOptions struct {
	// OutputDir is the directory for the generated .md file (default: current directory).
	OutputDir string
}

// ClipsMarkdownExportResult is the outcome of saving the API-generated clips Markdown.
type ClipsMarkdownExportResult struct {
	FilePath              string
	SuccessCount          int
	UnexportableClipCount int
}

type clipsMarkdownExportResponse struct {
	Success bool `json:"success"`
	Result  struct {
		Markdown              string `json:"markdown"`
		Filename              string `json:"filename"`
		SuccessCount          int    `json:"successCount"`
		UnexportableClipCount int    `json:"unexportableClipCount"`
	} `json:"result"`
	Episode ClipsEpisodeInfo `json:"episode"`
}

// ClipsMarkdownFetchResult is the API payload from clips/export/markdown before writing to disk.
type ClipsMarkdownFetchResult struct {
	Markdown              string
	Filename              string
	SuccessCount          int
	UnexportableClipCount int
}

// FetchClipsMarkdownExport calls POST /open/v1/episodes/{seq}/clips/export/markdown and returns the body.
func FetchClipsMarkdownExport(ctx context.Context, client *api.Client, seq int) (*ClipsMarkdownFetchResult, error) {
	apiPath := fmt.Sprintf("/open/v1/episodes/%d/clips/export/markdown", seq)
	var resp clipsMarkdownExportResponse
	if err := client.Post(ctx, apiPath, nil, &resp); err != nil {
		return nil, formatClipsMarkdownExportError(err)
	}
	return &ClipsMarkdownFetchResult{
		Markdown:              resp.Result.Markdown,
		Filename:              resp.Result.Filename,
		SuccessCount:          resp.Result.SuccessCount,
		UnexportableClipCount: resp.Result.UnexportableClipCount,
	}, nil
}

// clipsExportMarkdownFilename normalizes the API filename to a safe base name ending in .md.
func clipsExportMarkdownFilename(apiFilename string, seq int) string {
	filename := filepath.Base(apiFilename)
	if filename == "" || filename == "." {
		filename = fmt.Sprintf("clips_%d.md", seq)
	}
	if !strings.HasSuffix(strings.ToLower(filename), ".md") {
		filename += ".md"
	}
	return filename
}

// ExportClipsToMarkdown calls POST /open/v1/episodes/{seq}/clips/export/markdown and writes the file locally.
func ExportClipsToMarkdown(ctx context.Context, client *api.Client, seq int, opts ClipsMarkdownExportOptions) (*ClipsMarkdownExportResult, error) {
	fetch, err := FetchClipsMarkdownExport(ctx, client, seq)
	if err != nil {
		return nil, err
	}

	filename := clipsExportMarkdownFilename(fetch.Filename, seq)

	destDir := opts.OutputDir
	if destDir == "" {
		destDir = "."
	}
	if err := os.MkdirAll(destDir, 0o755); err != nil {
		return nil, fmt.Errorf("create output directory: %w", err)
	}

	dest := filepath.Join(destDir, filename)
	if err := os.WriteFile(dest, []byte(fetch.Markdown), 0o644); err != nil {
		return nil, fmt.Errorf("write markdown file: %w", err)
	}

	absPath, err := filepath.Abs(dest)
	if err != nil {
		absPath = dest
	}
	return &ClipsMarkdownExportResult{
		FilePath:              absPath,
		SuccessCount:          fetch.SuccessCount,
		UnexportableClipCount: fetch.UnexportableClipCount,
	}, nil
}

func formatClipsMarkdownExportError(err error) error {
	apiErr, ok := err.(*api.APIError)
	if !ok {
		return err
	}
	switch apiErr.ErrCode {
	case "no_ready_clips":
		return fmt.Errorf("no clips ready to export: episode has no clips with status done and non-empty content")
	default:
		return err
	}
}

// ClipsObsidianExportResult is the outcome of saving clips markdown into Obsidian (or cwd fallback).
type ClipsObsidianExportResult struct {
	FilePath              string
	WrittenToVault        bool
	SuccessCount          int
	UnexportableClipCount int
}

// ExportClipsToObsidian fetches clips markdown from the API (same as export markdown), then writes
// using WriteMarkdownToObsidian so vault discovery matches `podwise export obsidian`.
func ExportClipsToObsidian(ctx context.Context, client *api.Client, seq int, vaultFolder string) (*ClipsObsidianExportResult, error) {
	fetch, err := FetchClipsMarkdownExport(ctx, client, seq)
	if err != nil {
		return nil, err
	}
	filename := clipsExportMarkdownFilename(fetch.Filename, seq)
	obs, err := WriteMarkdownToObsidian(fetch.Markdown, filename, vaultFolder)
	if err != nil {
		return nil, err
	}
	return &ClipsObsidianExportResult{
		FilePath:              obs.FilePath,
		WrittenToVault:        obs.WrittenToVault,
		SuccessCount:          fetch.SuccessCount,
		UnexportableClipCount: fetch.UnexportableClipCount,
	}, nil
}

// ClipsReadwiseExportResult is the outcome of sending clips to Readwise Highlights.
type ClipsReadwiseExportResult struct {
	SuccessCount          int
	UnexportableClipCount int
	URL                   string
}

type clipsReadwiseSendResponse struct {
	Success bool `json:"success"`
	Result  struct {
		SuccessCount          int    `json:"successCount"`
		UnexportableClipCount int    `json:"unexportableClipCount"`
		URL                   string `json:"url"`
	} `json:"result"`
	Episode ClipsEpisodeInfo `json:"episode"`
}

// ExportClipsToReadwise calls POST /open/v1/episodes/{seq}/clips/send/readwise (Readwise Highlights).
func ExportClipsToReadwise(ctx context.Context, client *api.Client, seq int) (*ClipsReadwiseExportResult, error) {
	apiPath := fmt.Sprintf("/open/v1/episodes/%d/clips/send/readwise", seq)
	var resp clipsReadwiseSendResponse
	if err := client.Post(ctx, apiPath, nil, &resp); err != nil {
		return nil, formatClipsReadwiseExportError(err)
	}

	async.Go(func() {
		_ = MarkAsRead(context.Background(), client, seq)
	})

	return &ClipsReadwiseExportResult{
		SuccessCount:          resp.Result.SuccessCount,
		UnexportableClipCount: resp.Result.UnexportableClipCount,
		URL:                   resp.Result.URL,
	}, nil
}

func formatClipsReadwiseExportError(err error) error {
	apiErr, ok := err.(*api.APIError)
	if ok && apiErr.ErrCode == "no_ready_clips" {
		return fmt.Errorf("no clips ready to export: episode has no clips with status done and non-empty content")
	}
	return formatReadwiseError(err)
}

// ClipsNotionExportResult is the outcome of sending clips to the Notion clip database.
type ClipsNotionExportResult struct {
	SuccessCount          int
	UnexportableClipCount int
	URL                   string
}

type clipsNotionSendResponse struct {
	Success bool `json:"success"`
	Result  struct {
		SuccessCount          int    `json:"successCount"`
		UnexportableClipCount int    `json:"unexportableClipCount"`
		URL                   string `json:"url"`
	} `json:"result"`
	Episode ClipsEpisodeInfo `json:"episode"`
}

// ExportClipsToNotion calls POST /open/v1/episodes/{seq}/clips/send/notion (Notion clip database).
func ExportClipsToNotion(ctx context.Context, client *api.Client, seq int) (*ClipsNotionExportResult, error) {
	apiPath := fmt.Sprintf("/open/v1/episodes/%d/clips/send/notion", seq)
	var resp clipsNotionSendResponse
	if err := client.Post(ctx, apiPath, nil, &resp); err != nil {
		return nil, formatClipsNotionExportError(err)
	}

	async.Go(func() {
		_ = MarkAsRead(context.Background(), client, seq)
	})

	return &ClipsNotionExportResult{
		SuccessCount:          resp.Result.SuccessCount,
		UnexportableClipCount: resp.Result.UnexportableClipCount,
		URL:                   resp.Result.URL,
	}, nil
}

func formatClipsNotionExportError(err error) error {
	apiErr, ok := err.(*api.APIError)
	if ok && apiErr.ErrCode == "no_ready_clips" {
		return fmt.Errorf("no clips ready to export: episode has no clips with status done and non-empty content")
	}
	return formatNotionError(err)
}

func clipIsValid(c Clip) bool {
	return c.Status == "done" && c.Exportable
}

func validClips(clips []Clip) []Clip {
	out := make([]Clip, 0, len(clips))
	for _, c := range clips {
		if clipIsValid(c) {
			out = append(out, c)
		}
	}
	return out
}

// FormatText formats clips as Markdown for terminal output.
func (r *ClipsResult) FormatText() string {
	epTitle := r.Episode.Title
	if epTitle == "" {
		epTitle = "(untitled episode)"
	}
	clips := validClips(r.Clips)
	var sb strings.Builder
	fmt.Fprintf(&sb, "# Episode clips\n\n")
	fmt.Fprintf(&sb, "- **Episode:** %s\n", epTitle)
	fmt.Fprintf(&sb, "- **Podcast:** %s\n", r.Episode.PodcastName)
	fmt.Fprintf(&sb, "- **Episode URL:** %s\n", BuildEpisodeURL(r.Episode.Seq))
	fmt.Fprintf(&sb, "- **Valid clips:** %d\n\n", len(clips))

	if len(clips) == 0 {
		if len(r.Clips) == 0 {
			sb.WriteString("*(no clips for this episode)*\n")
		} else {
			sb.WriteString("*(no clips are ready yet — need status \"done\" and exportable)*\n")
		}
		return sb.String()
	}

	sb.WriteString("---\n")

	for i, c := range clips {
		title := c.Title
		if title == "" {
			title = fmt.Sprintf("(clip %d)", c.ID)
		}
		fmt.Fprintf(&sb, "\n## %d. %s\n\n", i+1, title)
		if c.CreatedAt > 0 {
			fmt.Fprintf(&sb, "- **Created:** %s\n", formatClipCreatedAtLocal(c.CreatedAt))
		}
		if c.ClipStart != nil || c.ClipEnd != nil {
			fmt.Fprintf(&sb, "- **Range:** %s – %s\n",
				formatClipOffsetSeconds(c.ClipStart), formatClipOffsetSeconds(c.ClipEnd))
		}
		if len(c.Takeaways) > 0 {
			sb.WriteString("\n### Takeaways\n\n")
			for _, t := range c.Takeaways {
				fmt.Fprintf(&sb, "- %s\n", t)
			}
		}
		if c.Content != nil {
			body := strings.TrimSpace(*c.Content)
			if body != "" {
				sb.WriteString("\n### Content\n\n")
				sb.WriteString(body)
				sb.WriteString("\n")
			}
		}
	}
	return sb.String()
}

// ClipJSON is the JSON-serializable view of a single clip for CLI output.
type ClipJSON struct {
	Title     string   `json:"title"`
	CreatedAt string   `json:"created_at,omitempty"`
	ClipStart *string  `json:"clip_start,omitempty"`
	ClipEnd   *string  `json:"clip_end,omitempty"`
	Takeaways []string `json:"takeaways,omitempty"`
	Content   *string  `json:"content,omitempty"`
}

// ClipsResultJSON is the top-level JSON output for `podwise clip list --json`.
type ClipsResultJSON struct {
	EpisodeTitle string     `json:"episode_title"`
	PodcastName  string     `json:"podcast_name"`
	EpisodeURL   string     `json:"episode_url"`
	ClipCount    int        `json:"clip_count"`
	Clips        []ClipJSON `json:"clips"`
}

// FormatJSON serializes clips as indented JSON.
func (r *ClipsResult) FormatJSON() ([]byte, error) {
	clips := validClips(r.Clips)
	items := make([]ClipJSON, 0, len(clips))
	for _, c := range clips {
		var created string
		if c.CreatedAt > 0 {
			created = formatClipCreatedAtLocal(c.CreatedAt)
		}
		var startStr, endStr *string
		if c.ClipStart != nil {
			s := formatClipOffsetSeconds(c.ClipStart)
			startStr = &s
		}
		if c.ClipEnd != nil {
			s := formatClipOffsetSeconds(c.ClipEnd)
			endStr = &s
		}
		cj := ClipJSON{
			Title:     c.Title,
			Takeaways: c.Takeaways,
			Content:   c.Content,
			ClipStart: startStr,
			ClipEnd:   endStr,
		}
		if created != "" {
			cj.CreatedAt = created
		}
		items = append(items, cj)
	}
	wrap := ClipsResultJSON{
		EpisodeTitle: r.Episode.Title,
		PodcastName:  r.Episode.PodcastName,
		EpisodeURL:   BuildEpisodeURL(r.Episode.Seq),
		ClipCount:    len(clips),
		Clips:        items,
	}
	return json.MarshalIndent(wrap, "", "  ")
}

func formatClipCreatedAtLocal(unixSec int64) string {
	return time.Unix(unixSec, 0).In(time.Local).Format(clipCreatedAtLayout)
}

func formatClipOffsetSeconds(sec *int) string {
	if sec == nil {
		return "—"
	}
	return utils.FormatTimestampMs(*sec * 1000)
}
