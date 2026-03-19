package cmd

import (
	"fmt"

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
	if prettyOutput {
		text = render.Markdown(text, loadGlamourStyle())
	}
	fmt.Fprint(cmd.OutOrStdout(), text)
}

// printMarkdownAnswer is like printMarkdown but uses render.MarkdownAnswer,
// which normalizes AI-generated multi-paragraph list items before rendering
// so glamour does not collapse them into a single line per item.
func printMarkdownAnswer(cmd *cobra.Command, text string) {
	if prettyOutput {
		text = render.MarkdownAnswer(text, loadGlamourStyle())
	}
	fmt.Fprint(cmd.OutOrStdout(), text)
}
