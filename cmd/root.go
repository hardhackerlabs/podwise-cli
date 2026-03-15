package cmd

import (
	"fmt"
	"os"

	"github.com/spf13/cobra"
)

var rootCmd = &cobra.Command{
	Use:          "podwise",
	Short:        "podwise — AI podcast & YouTube insights from your terminal",
	SilenceUsage: true,
	Long: `podwise is the CLI client for podwise.ai.

Turn any podcast episode or YouTube video into AI-powered transcripts, summaries, chapters, Q&A,
mind maps, highlights and more.`,
}

func Execute(version, commit, date string) {
	rootCmd.Version = fmt.Sprintf("%s (commit %s, built %s)", version, commit, date)

	if err := rootCmd.Execute(); err != nil {
		os.Exit(1)
	}
}

func init() {
	rootCmd.AddCommand(getCmd)
	rootCmd.AddCommand(processCmd)
	rootCmd.AddCommand(searchCmd)
	rootCmd.AddCommand(configCmd)
	rootCmd.AddCommand(mcpCmd)
}
