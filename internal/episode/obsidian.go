package episode

import (
	"context"
	"fmt"
	"os"
	"os/exec"
	"path/filepath"
	"regexp"
	"strings"
	"time"

	"github.com/hardhacker/podwise-cli/internal/api"
)

// ObsidianExportOptions holds parameters for exporting to Obsidian.
type ObsidianExportOptions struct {
	// Folder is the vault-relative folder where the note will be created.
	// Empty string places the note at the vault root.
	// Only used when the obsidian CLI is available.
	Folder string
	// Language is the language code for fetching a pre-translated version.
	// Empty string means use the original language.
	Language string
}

// ObsidianExportResult holds the result of an Obsidian export.
type ObsidianExportResult struct {
	// FilePath is the absolute path to the generated markdown file.
	FilePath string
	// ImportedWithCLI indicates whether obsidian-cli was used to open the file.
	ImportedWithCLI bool
}

var nonSafeFilenameRe = regexp.MustCompile(`[^\p{L}\p{N}\-_ ]+`)

// sanitizeFilename produces a filesystem-safe name from an arbitrary string.
func sanitizeFilename(s string) string {
	s = nonSafeFilenameRe.ReplaceAllString(s, "")
	s = strings.TrimSpace(s)
	s = strings.ReplaceAll(s, " ", "_")
	if len(s) > 80 {
		s = s[:80]
	}
	if s == "" {
		s = "episode"
	}
	return s
}

// yamlQuote wraps s in double-quotes, escaping any embedded double-quotes.
func yamlQuote(s string) string {
	return `"` + strings.ReplaceAll(s, `"`, `\"`) + `"`
}

// buildObsidianMarkdown renders an Obsidian-ready Markdown document from
// the episode's summary and transcript data.
func buildObsidianMarkdown(seq int, title string, summary *SummaryResult, segments []Segment) string {
	var sb strings.Builder
	episodeURL := BuildEpisodeURL(seq)
	today := time.Now().Format("2006-01-02")

	// YAML frontmatter
	ep := summary.Episode
	podcastName := ""
	publishTime := ""
	if ep != nil {
		podcastName = ep.PodcastName
		if ep.PublishTime > 0 {
			publishTime = time.Unix(ep.PublishTime, 0).UTC().Format("2006-01-02")
		}
	}

	sb.WriteString("---\n")
	fmt.Fprintf(&sb, "podcast: %s\n", yamlQuote(podcastName))
	fmt.Fprintf(&sb, "episode: %s\n", yamlQuote(title))
	fmt.Fprintf(&sb, "link: %s\n", episodeURL)
	if publishTime != "" {
		fmt.Fprintf(&sb, "publish-time: %s\n", yamlQuote(publishTime))
	}
	fmt.Fprintf(&sb, "save-time: %s\n", yamlQuote(today))
	sb.WriteString("---\n\n")

	// Heading + back-link
	fmt.Fprintf(&sb, "# %s\n\n", title)

	// section writes a h2 heading followed by content, always ending with \n\n.
	section := func(heading, content string) {
		fmt.Fprintf(&sb, "## %s\n\n", heading)
		sb.WriteString(strings.TrimRight(content, "\n"))
		sb.WriteString("\n\n")
	}

	if s := summary.FormatSummary(); s != "" {
		section("Summary", s)
	}
	if len(summary.Chapters) > 0 {
		section("Chapters", summary.FormatChapters())
	}
	if len(summary.QAs) > 0 {
		section("Q&A", summary.FormatQA())
	}
	if len(summary.Highlights) > 0 {
		section("Highlights", summary.FormatHighlights())
	}
	if len(summary.Keywords) > 0 {
		section("Keywords", summary.FormatKeywords())
	}
	if len(segments) > 0 {
		section("Transcript", FormatMergedTranscript(segments))
	}

	return sb.String()
}

// obsidianVersionRe matches output like "1.12.7 (installer 1.12.7)".
var obsidianVersionRe = regexp.MustCompile(`^\d+\.\d+\.\d+`)

// obsidianAppRunning returns true when the Obsidian app is reachable via CLI.
// It runs `obsidian version` with a short timeout and checks the output format.
func obsidianAppRunning(ctx context.Context, cliPath string) bool {
	vCtx, cancel := context.WithTimeout(ctx, 3*time.Second)
	defer cancel()
	out, err := exec.CommandContext(vCtx, cliPath, "version").Output()
	return err == nil && obsidianVersionRe.Match(out)
}

// ExportToObsidian fetches the episode's summary and transcript and renders a
// Markdown note.
//
// When the `obsidian` CLI is present in PATH the note is created directly in
// the active vault via `obsidian create` (path= is vault-relative) and opened
// in Obsidian.
//
// When the CLI is absent the note is written to the current working directory
// and ImportedWithCLI is false.
func ExportToObsidian(ctx context.Context, client *api.Client, seq int, opts ObsidianExportOptions) (*ObsidianExportResult, error) {
	summary, err := FetchSummary(ctx, client, seq, false, opts.Language)
	if err != nil {
		return nil, fmt.Errorf("fetch summary: %w", err)
	}

	transcriptResult, err := FetchTranscripts(ctx, client, seq, false, opts.Language)
	if err != nil {
		return nil, fmt.Errorf("fetch transcript: %w", err)
	}
	segments := MergeSegments(transcriptResult.Segments, 60_000)

	// Derive a human-readable title.
	title := fmt.Sprintf("Episode %d", seq)
	if transcriptResult.Episode != nil && transcriptResult.Episode.Title != "" {
		title = transcriptResult.Episode.Title
	}

	md := buildObsidianMarkdown(seq, title, summary, segments)
	filename := fmt.Sprintf("%s_%d.md", sanitizeFilename(title), seq)
	result := &ObsidianExportResult{}

	// ── Path 1: obsidian CLI available ────────────────────────────────────────
	// `obsidian create name=<file> [path=<folder>/] content=<md> open overwrite`
	// path= is vault-relative; omitted when folder is empty (vault root).
	if cliPath, lookErr := exec.LookPath("obsidian"); lookErr == nil && obsidianAppRunning(ctx, cliPath) {
		args := []string{"create", "name=" + filename}
		if opts.Folder != "" {
			args = append(args, "path="+strings.TrimSuffix(opts.Folder, "/")+"/")
		}
		// The obsidian CLI requires the app to be running and expects \n/\t literals in content.
		escapedMD := strings.ReplaceAll(md, "\t", `\t`)
		escapedMD = strings.ReplaceAll(escapedMD, "\n", `\n`)
		args = append(args, "content="+escapedMD, "overwrite")

		cliCtx, cancel := context.WithTimeout(ctx, 10*time.Second)
		defer cancel()
		if runErr := exec.CommandContext(cliCtx, cliPath, args...).Run(); runErr == nil {
			if opts.Folder != "" {
				result.FilePath = strings.TrimSuffix(opts.Folder, "/") + "/" + filename
			} else {
				result.FilePath = filename
			}
			result.ImportedWithCLI = true
			return result, nil
		}
	}

	// ── Path 2: no CLI – write to current working directory ──────────────────
	filePath := filename
	if err := os.WriteFile(filePath, []byte(md), 0o644); err != nil {
		return nil, fmt.Errorf("write markdown file: %w", err)
	}
	absPath, err := filepath.Abs(filePath)
	if err != nil {
		absPath = filePath
	}
	result.FilePath = absPath
	return result, nil
}
