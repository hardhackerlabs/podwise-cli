package cmd

import (
	"context"
	"fmt"
	"os"
	"strings"
	"time"

	"github.com/hardhacker/podwise-cli/internal/api"
	"github.com/hardhacker/podwise-cli/internal/ask"
	"github.com/hardhacker/podwise-cli/internal/config"
	"github.com/hardhacker/podwise-cli/internal/utils"
	"github.com/spf13/cobra"
)

var askShowSources bool

// podwise ask <question>
var askCmd = &cobra.Command{
	Use:   "ask <question>",
	Short: "Ask the AI a question based on podcast transcripts",
	Long: `Ask the AI a question based on podcast transcripts.

The AI searches relevant podcast transcripts and generates an answer with source
citations. Each [citation:N] in the answer corresponds to a numbered source.

Use --sources to print the cited excerpts and episode links below the answer.

The daily ask limit depends on your Podwise plan.`,
	Example: `  podwise ask "the future of AI Agents"
  podwise ask "How does retrieval augmented generation work?" --sources`,
	Args: cobra.MinimumNArgs(1),
	RunE: runAsk,
}

func init() {
	askCmd.Flags().BoolVar(&askShowSources, "sources", false, "print cited source excerpts and episode links below the answer")
}

func runAsk(cmd *cobra.Command, args []string) error {
	question := strings.Join(args, " ")

	cfg, err := config.Load()
	if err != nil {
		return err
	}
	if err := config.Validate(cfg); err != nil {
		return err
	}

	client := api.New(cfg.APIBaseURL, cfg.APIKey)

	var stopSpinner chan struct{}
	var spinnerDone chan struct{}
	if prettyOutput {
		stopSpinner = make(chan struct{})
		spinnerDone = make(chan struct{})
		go func() {
			frames := []string{"⠋", "⠙", "⠹", "⠸", "⠼", "⠴", "⠦", "⠧", "⠇", "⠏"}
			i := 0
			fmt.Fprintf(os.Stderr, "\r\033[36m%s\033[0m Thinking... (AI is searching podcast transcripts, this may take up to 60s)", frames[i])
			i++
			ticker := time.NewTicker(100 * time.Millisecond)
			defer ticker.Stop()
			for {
				select {
				case <-stopSpinner:
					fmt.Fprintf(os.Stderr, "\r\033[K")
					close(spinnerDone)
					return
				case <-ticker.C:
					fmt.Fprintf(os.Stderr, "\r\033[36m%s\033[0m Thinking... (AI is searching podcast transcripts, this may take up to 60s)", frames[i])
					i = (i + 1) % len(frames)
				}
			}
		}()
	} else {
		fmt.Fprintf(os.Stderr, "Thinking... (AI is searching podcast transcripts, this may take up to 60s)\n")
	}

	start := time.Now()

	result, err := ask.Ask(context.Background(), client, question)

	if prettyOutput {
		close(stopSpinner)
		<-spinnerDone
	}

	if err != nil {
		return err
	}

	fmt.Fprintf(os.Stderr, "Done in %s\n\n", utils.FormatDuration(time.Since(start)))
	printMarkdownAnswer(cmd, result.FormatText(question, askShowSources))
	return nil
}
