package episode

import (
	"context"
	"encoding/json"
	"fmt"
	"os"
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
	// WrittenToVault indicates whether the file was written directly into an
	// Obsidian vault directory (true) or only to the current working directory
	// as a fallback (false).
	WrittenToVault bool
}

// obsidianVaultEntry represents one vault record inside obsidian.json.
type obsidianVaultEntry struct {
	Path string `json:"path"`
	Ts   int64  `json:"ts"`
	Open bool   `json:"open"`
}

// obsidianGlobalConfig is the top-level structure of obsidian.json.
type obsidianGlobalConfig struct {
	Vaults map[string]obsidianVaultEntry `json:"vaults"`
}

// obsidianConfigPath returns the platform-appropriate path to Obsidian's
// global configuration file.
//
// macOS  : ~/Library/Application Support/obsidian/obsidian.json
// Linux  : $XDG_CONFIG_HOME/obsidian/obsidian.json  (default ~/.config/…)
// Windows: %APPDATA%\obsidian\obsidian.json
func obsidianConfigPath() (string, error) {
	dir, err := os.UserConfigDir()
	if err != nil {
		return "", err
	}
	return filepath.Join(dir, "obsidian", "obsidian.json"), nil
}

// findObsidianVault reads obsidian.json and returns the absolute path of the
// most appropriate vault using this priority:
//  1. The vault flagged open:true with the highest ts.
//  2. Otherwise the vault with the highest ts.
//
// Returns ("", nil) when obsidian.json is absent or contains no vaults.
func findObsidianVault() (string, error) {
	cfgPath, err := obsidianConfigPath()
	if err != nil {
		return "", nil // config dir unavailable – treat as not found
	}
	data, err := os.ReadFile(cfgPath)
	if err != nil {
		if os.IsNotExist(err) {
			return "", nil
		}
		return "", fmt.Errorf("read obsidian config: %w", err)
	}
	var cfg obsidianGlobalConfig
	if err := json.Unmarshal(data, &cfg); err != nil {
		return "", fmt.Errorf("parse obsidian config: %w", err)
	}

	var best obsidianVaultEntry
	for _, v := range cfg.Vaults {
		if best.Path == "" {
			best = v
			continue
		}
		// open vaults take priority; break ties by most-recently-used (ts).
		if (!best.Open && v.Open) || (best.Open == v.Open && v.Ts > best.Ts) {
			best = v
		}
	}
	return best.Path, nil
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

// ExportToObsidian fetches the episode's summary and transcript, renders a
// Markdown note, and writes it to the Obsidian vault when one can be located
// automatically.
//
// Priority:
//  1. Write directly into the Obsidian vault directory discovered via
//     Obsidian's global config file (obsidian.json). This works whether or not
//     the Obsidian app is running. WrittenToVault is set to true.
//  2. Fallback: write the .md file to the current working directory and report
//     instructions for manual import. WrittenToVault is false.
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

	// ── Path 1: write directly into the Obsidian vault ───────────────────────
	// Locate the vault from obsidian.json; no app process required.
	if vaultPath, vaultErr := findObsidianVault(); vaultErr == nil && vaultPath != "" {
		destDir := vaultPath
		if opts.Folder != "" {
			// filepath.FromSlash converts forward-slash separators to the OS
			// separator on Windows, keeping behaviour correct cross-platform.
			destDir = filepath.Join(vaultPath, filepath.FromSlash(strings.TrimSuffix(opts.Folder, "/")))
		}
		if mkErr := os.MkdirAll(destDir, 0o755); mkErr == nil {
			dest := filepath.Join(destDir, filename)
			if writeErr := os.WriteFile(dest, []byte(md), 0o644); writeErr == nil {
				result.FilePath = dest
				result.WrittenToVault = true
				return result, nil
			}
		}
	}

	// ── Path 2: fallback – write to the current working directory ────────────
	if err := os.WriteFile(filename, []byte(md), 0o644); err != nil {
		return nil, fmt.Errorf("write markdown file: %w", err)
	}
	absPath, err := filepath.Abs(filename)
	if err != nil {
		absPath = filename
	}
	result.FilePath = absPath
	return result, nil
}
