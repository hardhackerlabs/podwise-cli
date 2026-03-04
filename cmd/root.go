package cmd

import (
	"fmt"
	"os"

	"github.com/spf13/cobra"
)

var rootCmd = &cobra.Command{
	Use:   "podwise",
	Short: "podwise — AI podcast insights from your terminal",
	Long: `podwise is the CLI client for podwise.ai.

Turn any podcast episode into AI-powered summaries, outlines, transcripts,
Q&A, and mind maps — then export them to Notion, Obsidian, Readwise, and more.`,
}

func Execute(version, commit, date string) {
	rootCmd.Version = fmt.Sprintf("%s (commit %s, built %s)", version, commit, date)

	if err := rootCmd.Execute(); err != nil {
		os.Exit(1)
	}
}

func init() {
	rootCmd.AddCommand(getCmd)
	rootCmd.AddCommand(configCmd)
}
