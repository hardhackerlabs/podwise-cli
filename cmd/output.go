package cmd

import (
	"fmt"
	"os"
	"os/exec"

	"github.com/charmbracelet/x/term"
	"github.com/hardhacker/podwise-cli/internal/config"
	"github.com/hardhacker/podwise-cli/internal/render"
	"github.com/spf13/cobra"
)

// loadGlamourStyle returns the configured glamour style, falling back to the
// default when the config file is missing or the field is empty.
func loadGlamourStyle() string {
	cfg, err := config.Load()
	if err != nil || cfg.GlamourStyle == "" {
		return config.DefaultGlamourStyle
	}
	return cfg.GlamourStyle
}

// printMarkdown writes text to cmd's output stream, rendering it through
// glamour when the global --pretty flag is set.
func printMarkdown(cmd *cobra.Command, text string) {
	// --pretty-no-pager takes priority over --pretty
	if prettyNoPager {
		text = render.Markdown(text, loadGlamourStyle())
		fmt.Fprint(cmd.OutOrStdout(), text)
	} else if prettyOutput {
		text = render.Markdown(text, loadGlamourStyle())
		printWithPager(cmd, text)
	} else {
		fmt.Fprint(cmd.OutOrStdout(), text)
	}
}

// printMarkdownAnswer is like printMarkdown but uses render.MarkdownAnswer,
// which normalizes AI-generated multi-paragraph list items before rendering
// so glamour does not collapse them into a single line per item.
func printMarkdownAnswer(cmd *cobra.Command, text string) {
	// --pretty-no-pager takes priority over --pretty
	if prettyNoPager {
		text = render.MarkdownAnswer(text, loadGlamourStyle())
		fmt.Fprint(cmd.OutOrStdout(), text)
	} else if prettyOutput {
		text = render.MarkdownAnswer(text, loadGlamourStyle())
		printWithPager(cmd, text)
	} else {
		fmt.Fprint(cmd.OutOrStdout(), text)
	}
}

// printWithPager writes text to cmd's output stream, using a pager (less -R)
// for interactive terminals. The pager allows scrolling and can be exited with 'q'.
func printWithPager(cmd *cobra.Command, text string) {
	// Check if stdout is a TTY - only use pager in interactive mode
	if !isTTY() {
		fmt.Fprint(cmd.OutOrStdout(), text)
		return
	}

	// Use less -RXF pager
	// -R: raw control characters (for ANSI colors)
	// -X: don't clear screen on exit
	// -F: quit if output fits in one screen
	cmdObj := exec.Command("less", "-RXF")
	cmdObj.Stdout = os.Stdout
	cmdObj.Stderr = os.Stderr

	stdin, err := cmdObj.StdinPipe()
	if err != nil {
		fmt.Fprint(cmd.OutOrStdout(), text)
		return
	}

	if err := cmdObj.Start(); err != nil {
		fmt.Fprint(cmd.OutOrStdout(), text)
		return
	}

	stdin.Write([]byte(text))
	stdin.Close()

	cmdObj.Wait()
}

// isTTY returns true if stdout is a terminal.
func isTTY() bool {
	return term.IsTerminal(1) // fd 1 = stdout
}
