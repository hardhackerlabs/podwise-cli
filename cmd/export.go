package cmd

import (
	"context"
	"fmt"

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
var (
	notionMindmap     bool
	notionTranslation string
)

// Readwise export flags
var (
	readwiseMindmap     bool
	readwiseTranslation string
	readwiseLocation    string
)

// Obsidian export flags
var obsidianPath string

// podwise export notion <episode-url>
var exportNotionCmd = &cobra.Command{
	Use:   "notion <episode-url>",
	Short: "Export episode content to Notion",
	Long: `Export a processed episode's content to your connected Notion workspace.

Requires Notion to be connected and configured in Podwise settings.
Visit https://podwise.ai/dashboard/settings to set up Notion integration.

The command creates a new page in your configured Notion database with the episode content.`,
	Example: `  podwise export notion https://podwise.ai/dashboard/episodes/7360326
  podwise export notion https://podwise.ai/dashboard/episodes/7360326 --translation zh`,
	Args: cobra.ExactArgs(1),
	RunE: runExportNotion,
}

func init() {
	exportNotionCmd.Flags().BoolVar(&notionMindmap, "mindmap", false, "include mind map (limited to 3 nesting levels)")
	exportNotionCmd.Flags().StringVar(&notionTranslation, "translation", "", "translation language code (e.g., zh, ja)")

	exportReadwiseCmd.Flags().BoolVar(&readwiseMindmap, "mindmap", false, "include mind map as nested list")
	exportReadwiseCmd.Flags().StringVar(&readwiseTranslation, "translation", "", "translation language code (e.g., zh, ja)")
	exportReadwiseCmd.Flags().StringVar(&readwiseLocation, "location", "archive", "where to save in Reader: new (inbox), later, archive")

	exportObsidianCmd.Flags().StringVar(&obsidianPath, "path", "Podwise", "vault-relative folder path for the note (e.g. Podwise/2026)")

	exportCmd.AddCommand(exportNotionCmd)
	exportCmd.AddCommand(exportReadwiseCmd)
	exportCmd.AddCommand(exportObsidianCmd)
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

	opts := episode.NotionExportOptions{
		Transcripts:           true,
		Mindmap:               notionMindmap,
		Translation:           notionTranslation,
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
  podwise export readwise https://podwise.ai/dashboard/episodes/7360326 --translation zh`,
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

	opts := episode.ReadwiseExportOptions{
		Mindmap:               readwiseMindmap,
		Translation:           readwiseTranslation,
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
	Long: `Export episode summary and transcript as a Markdown note.

If the obsidian CLI is found in PATH, the note is created in the active vault
under the folder specified by --path (default: Podwise) and opened immediately.

If not, the .md file is written to the current directory with instructions for
manual import (drag into File Explorer or copy to vault folder).

  obsidian CLI: https://obsidian.md/help/cli`,
	Example: `  podwise export obsidian https://podwise.ai/dashboard/episodes/7360326`,
	Args:    cobra.ExactArgs(1),
	RunE:    runExportObsidian,
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

	opts := episode.ObsidianExportOptions{
		VaultPath: obsidianPath,
	}

	client := api.New(cfg.APIBaseURL, cfg.APIKey)
	ctx := context.Background()

	fmt.Printf("Fetching episode %s for Obsidian export...\n", episode.BuildEpisodeURL(seq))

	result, err := episode.ExportToObsidian(ctx, client, seq, opts)
	if err != nil {
		return err
	}

	if result.ImportedWithCLI {
		fmt.Printf("\n✓ Imported to Obsidian vault and opened\n")
		fmt.Printf("  Vault path: %s\n", result.FilePath)
	} else {
		fmt.Printf("\n✓ Markdown file saved\n")
		fmt.Printf("  File: %s\n", result.FilePath)

		fmt.Printf("\n  obsidian CLI not found — import the file manually:\n")
		fmt.Printf("  • Drag and drop %s into the Obsidian File Explorer, or\n", result.FilePath)
		fmt.Printf("  • Copy the file directly into your Obsidian vault folder using Finder / Explorer\n")
	}

	return nil
}
