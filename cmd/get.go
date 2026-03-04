package cmd

import (
	"github.com/spf13/cobra"
)

var getCmd = &cobra.Command{
	Use:   "get <episode-url>",
	Short: "Get AI insights for a podcast episode",
	Long: `Fetch AI-processed content for a podcast episode from podwise.ai.

By default the summary is printed to stdout. Use flags to choose a different
output type or to export directly to a note-taking tool.

Examples:
  podwise get https://podwise.ai/dashboard/episodes/7360326
  podwise get <episode-url> --type transcript
  podwise get <episode-url> --type mindmap --output ./notes/
  podwise get <episode-url> --export notion`,
	Args: cobra.ExactArgs(1),
	RunE: runGet,
}

func init() {
	// TODO: --type    string   output type: summary | outline | transcript | qa | mindmap (default "summary")
	// TODO: --lang    string   output language: en | zh | ja | ko | fr | de | es | pt (default: episode language)
	// TODO: --output  string   write output to this file or directory instead of stdout
	// TODO: --export  string   export to a tool: notion | obsidian | readwise | logseq
	// TODO: --format  string   file format when exporting: md | pdf | srt | xmind (default "md")
}

func runGet(cmd *cobra.Command, args []string) error {
	_ = args[0] // episode URL or podwise episode ID

	// TODO: resolve args[0] — accept podwise episode URL, raw RSS/audio URL, or episode ID
	// TODO: call podwise.ai API to fetch the requested content type
	// TODO: apply language translation if --lang differs from source
	// TODO: write result to stdout, --output path, or trigger --export integration

	return nil
}
