package cmd

import (
	"fmt"
	"os"
	"strings"

	"github.com/hardhacker/podwise-cli/internal/update"
	"github.com/spf13/cobra"
)

// prettyOutput is the global flag that enables glamour markdown rendering.
var prettyOutput bool

// prettyNoPager is the global flag that enables pretty output without pager.
var prettyNoPager bool

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

	updateCh := maybeStartUpdateCheck(version)

	if err := rootCmd.Execute(); err != nil {
		os.Exit(1)
	}

	printUpdateNotice(updateCh)
}

// maybeStartUpdateCheck starts an async update check and returns a channel that
// will receive the result. The channel is closed immediately when the check
// should be skipped (MCP mode, non-TTY, or PODWISE_NO_UPDATE_CHECK env var set).
func maybeStartUpdateCheck(version string) <-chan update.Result {
	ch := make(chan update.Result, 1)

	if isMCPCommand() || !isTerminal(os.Stderr) || os.Getenv("PODWISE_NO_UPDATE_CHECK") != "" {
		close(ch)
		return ch
	}

	go func() {
		ch <- update.Check(version)
	}()

	return ch
}

// printUpdateNotice performs a non-blocking read of the update channel and
// prints a notice to stderr when a newer version is available.
func printUpdateNotice(ch <-chan update.Result) {
	var result update.Result
	select {
	case r, ok := <-ch:
		if !ok {
			return
		}
		result = r
	default:
		return
	}

	if !result.HasUpdate {
		return
	}

	fmt.Fprintf(os.Stderr, "\n  A new version of podwise is available: v%s\n", result.LatestVersion)
	fmt.Fprintf(os.Stderr, "  To upgrade: %s\n\n", update.UpgradeHint())
}

// isMCPCommand reports whether the CLI was invoked as "podwise mcp".
// Flags (e.g. --flag) are skipped; the first positional argument is checked.
func isMCPCommand() bool {
	for _, arg := range os.Args[1:] {
		if strings.HasPrefix(arg, "-") {
			continue
		}
		return arg == "mcp"
	}
	return false
}

// isTerminal reports whether f is connected to a terminal (not a pipe/file).
func isTerminal(f *os.File) bool {
	stat, err := f.Stat()
	if err != nil {
		return false
	}
	return (stat.Mode() & os.ModeCharDevice) != 0
}

func init() {
	rootCmd.PersistentFlags().BoolVar(&prettyOutput, "pretty", false, "render markdown output with terminal styling and pager (AI Agents/LLMs should not use this flag)")
	rootCmd.PersistentFlags().BoolVar(&prettyNoPager, "pretty-no-pager", false, "render markdown output with terminal styling but without pager (AI Agents/LLMs should not use this flag)")

	rootCmd.AddCommand(getCmd)
	rootCmd.AddCommand(processCmd)
	rootCmd.AddCommand(askCmd)
	rootCmd.AddCommand(searchCmd)
	rootCmd.AddCommand(popularCmd)
	rootCmd.AddCommand(listCmd)
	rootCmd.AddCommand(drillCmd)
	rootCmd.AddCommand(followCmd)
	rootCmd.AddCommand(unfollowCmd)
	rootCmd.AddCommand(authCmd)
	rootCmd.AddCommand(configCmd)
	rootCmd.AddCommand(mcpCmd)
	rootCmd.AddCommand(exportCmd)
	rootCmd.AddCommand(historyCmd)
}
