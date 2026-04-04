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

// podwise export <subcommand>
var exportCmd = &cobra.Command{
	Use:   "export <subcommand>",
	Short: "Export episode content to external services",
	Long:  "Export AI-generated episode content to external services like Notion, Readwise, Obsidian, and others.",
	Example: `  podwise export notion https://podwise.ai/dashboard/episodes/7360326
  podwise export readwise https://podwise.ai/dashboard/episodes/7360326
  podwise export obsidian https://podwise.ai/dashboard/episodes/7360326`,
}

// Notion export flags
var notionLang string

// Readwise export flags
var (
	readwiseLang     string
	readwiseLocation string
)

// Obsidian export flags
var (
	obsidianFolder string
	obsidianLang   string
)

// Markdown export flags
var (
	markdownOutput string
	markdownLang   string
)

// podwise export notion <episode-url>
var exportNotionCmd = &cobra.Command{
	Use:   "notion <episode-url>",
	Short: "Export episode content to Notion",
	Long: `Export a processed episode's content to your connected Notion workspace.

Requires Notion to be connected and configured in Podwise settings.
Visit https://podwise.ai/dashboard/settings to set up Notion integration.

The command creates a new page in your configured Notion database with the episode content.`,
	Example: `  podwise export notion https://podwise.ai/dashboard/episodes/7360326
  podwise export notion https://podwise.ai/dashboard/episodes/7360326 --lang Chinese`,
	Args: cobra.ExactArgs(1),
	RunE: runExportNotion,
}

func init() {
	langUsage := "export the translated version in this language: " + strings.Join(episode.LanguageNames(), ", ")

	exportNotionCmd.Flags().StringVar(&notionLang, "lang", "", langUsage)

	exportReadwiseCmd.Flags().StringVar(&readwiseLang, "lang", "", langUsage)
	exportReadwiseCmd.Flags().StringVar(&readwiseLocation, "location", "archive", "where to save in Reader: new (inbox), later, archive")

	exportObsidianCmd.Flags().StringVar(&obsidianFolder, "folder", "", "vault-relative folder to place the note in (e.g. Podcasts/2026); defaults to vault root")
	exportObsidianCmd.Flags().StringVar(&obsidianLang, "lang", "", langUsage)

	exportMarkdownCmd.Flags().StringVar(&markdownOutput, "output", "", "directory to write the .md file into (default: current directory)")
	exportMarkdownCmd.Flags().StringVar(&markdownLang, "lang", "", langUsage)

	exportCmd.AddCommand(exportNotionCmd)
	exportCmd.AddCommand(exportReadwiseCmd)
	exportCmd.AddCommand(exportObsidianCmd)
	exportCmd.AddCommand(exportMarkdownCmd)
}

func runExportNotion(cmd *cobra.Command, args []string) error {
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

	translationCode, err := episode.ResolveLangCode(notionLang)
	if err != nil {
		return err
	}

	opts := episode.NotionExportOptions{
		Transcripts:           true,
		Mindmap:               false,
		Translation:           translationCode,
		MixWithOriginLanguage: false,
	}

	client := api.New(cfg.APIBaseURL, cfg.APIKey)
	ctx := context.Background()

	fmt.Printf("Exporting episode %s to Notion...\n", episode.BuildEpisodeURL(seq))

	result, err := episode.ExportToNotion(ctx, client, seq, opts)
	if err != nil {
		return err
	}

	fmt.Printf("\n✓ Successfully exported to Notion\n")
	fmt.Printf("  Page URL: %s\n", result.URL)

	if result.Warning != "" {
		fmt.Printf("\n⚠ Warning: %s\n", result.Warning)
	}

	return nil
}

// podwise export readwise <episode-url>
var exportReadwiseCmd = &cobra.Command{
	Use:   "readwise <episode-url>",
	Short: "Export episode content to Readwise Reader",
	Long: `Export a processed episode's content to your connected Readwise Reader account.

Requires Readwise API token to be configured in Podwise settings.
Visit https://podwise.ai/dashboard/settings to set up Readwise integration.

The command creates a new document in your Readwise Reader with the episode content.`,
	Example: `  podwise export readwise https://podwise.ai/dashboard/episodes/7360326
  podwise export readwise https://podwise.ai/dashboard/episodes/7360326 --location later
  podwise export readwise https://podwise.ai/dashboard/episodes/7360326 --lang Chinese`,
	Args: cobra.ExactArgs(1),
	RunE: runExportReadwise,
}

func runExportReadwise(cmd *cobra.Command, args []string) error {
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

	// Validate location value
	if readwiseLocation != "" && readwiseLocation != "new" && readwiseLocation != "later" && readwiseLocation != "archive" {
		return fmt.Errorf("invalid location %q: must be one of: new, later, archive", readwiseLocation)
	}

	translationCode, err := episode.ResolveLangCode(readwiseLang)
	if err != nil {
		return err
	}

	opts := episode.ReadwiseExportOptions{
		Mindmap:               false,
		Translation:           translationCode,
		Location:              readwiseLocation,
		Shownotes:             false,
		MixWithOriginLanguage: false,
	}

	client := api.New(cfg.APIBaseURL, cfg.APIKey)
	ctx := context.Background()

	fmt.Printf("Exporting episode %s to Readwise Reader...\n", episode.BuildEpisodeURL(seq))

	result, err := episode.ExportToReadwise(ctx, client, seq, opts)
	if err != nil {
		return err
	}

	fmt.Printf("\n✓ Successfully exported to Readwise Reader\n")
	fmt.Printf("  Document URL: %s\n", result.URL)

	return nil
}

// podwise export obsidian <episode-url>
var exportObsidianCmd = &cobra.Command{
	Use:   "obsidian <episode-url>",
	Short: "Export episode content to Obsidian",
	Long: `Export a processed episode's content to your Obsidian vault.

The vault is located automatically from Obsidian's configuration file
(obsidian.json). 

If no vault can be found the .md file is written to the current directory
with instructions for manual import.`,
	Example: `  podwise export obsidian https://podwise.ai/dashboard/episodes/7360326
  podwise export obsidian https://podwise.ai/dashboard/episodes/7360326 --lang Chinese
  podwise export obsidian https://podwise.ai/dashboard/episodes/7360326 --folder Podcasts/2026`,
	Args: cobra.ExactArgs(1),
	RunE: runExportObsidian,
}

func runExportObsidian(cmd *cobra.Command, args []string) error {
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

	lang, err := episode.ResolveLangName(obsidianLang)
	if err != nil {
		return err
	}

	opts := episode.ObsidianExportOptions{
		Folder:   obsidianFolder,
		Language: lang,
	}

	client := api.New(cfg.APIBaseURL, cfg.APIKey)
	ctx := context.Background()

	fmt.Printf("Fetching episode %s for Obsidian export...\n", episode.BuildEpisodeURL(seq))

	result, err := episode.ExportToObsidian(ctx, client, seq, opts)
	if err != nil {
		return err
	}

	if result.WrittenToVault {
		fmt.Printf("\n✓ Saved to Obsidian vault\n")
		fmt.Printf("  Path: %s\n", result.FilePath)
	} else {
		fmt.Printf("\n✓ Markdown file saved (Obsidian vault not found)\n")
		fmt.Printf("  File: %s\n", result.FilePath)
		fmt.Printf("\n  To import manually:\n")
		fmt.Printf("  • Drag and drop %s into the Obsidian File Explorer, or\n", result.FilePath)
		fmt.Printf("  • Copy the file directly into your Obsidian vault folder\n")
	}

	return nil
}

// podwise export markdown <episode-url>
var exportMarkdownCmd = &cobra.Command{
	Use:   "markdown <episode-url>",
	Short: "Export episode content to a local Markdown file",
	Long: `Export a processed episode's content to a local Markdown file.

The file is written to the current directory by default. Use --output to
specify a different destination directory.`,
	Example: `  podwise export markdown https://podwise.ai/dashboard/episodes/7360326
  podwise export markdown https://podwise.ai/dashboard/episodes/7360326 --lang Chinese
  podwise export markdown https://podwise.ai/dashboard/episodes/7360326 --output ~/notes/podcasts`,
	Args: cobra.ExactArgs(1),
	RunE: runExportMarkdown,
}

func runExportMarkdown(cmd *cobra.Command, args []string) error {
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

	lang, err := episode.ResolveLangName(markdownLang)
	if err != nil {
		return err
	}

	opts := episode.MarkdownExportOptions{
		OutputDir: markdownOutput,
		Language:  lang,
	}

	client := api.New(cfg.APIBaseURL, cfg.APIKey)
	ctx := context.Background()

	fmt.Printf("Fetching episode %s for Markdown export...\n", episode.BuildEpisodeURL(seq))

	result, err := episode.ExportToMarkdown(ctx, client, seq, opts)
	if err != nil {
		return err
	}

	fmt.Printf("\n✓ Markdown file saved\n")
	fmt.Printf("  File: %s\n", result.FilePath)

	return nil
}
