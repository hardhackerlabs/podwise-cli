package cmd

import (
	"context"
	"errors"
	"fmt"
	"strings"
	"sync"
	"time"

	"github.com/hardhacker/podwise-cli/internal/api"
	"github.com/hardhacker/podwise-cli/internal/config"
	"github.com/hardhacker/podwise-cli/internal/episode"
	"github.com/hardhacker/podwise-cli/internal/utils"
	"github.com/spf13/cobra"
)

var processPollInterval time.Duration
var processTimeout time.Duration
var processTitle string
var processHotwords string

// podwise process <url|file>
var processCmd = &cobra.Command{
	Use:   "process <url|file>",
	Short: "Submit a podcast episode, YouTube video, or local media file for AI processing",
	Long: `Submit a podcast episode, YouTube video, or local audio/video file for AI processing
(transcription and analysis).

Accepted inputs:
  https://podwise.ai/dashboard/episodes/<id>               Podwise episode
  https://www.xiaoyuzhoufm.com/episode/<id>                Xiaoyuzhou episode
  https://www.youtube.com/watch?v=<id>                     YouTube video
  https://youtu.be/<id>                                    YouTube short URL
  /path/to/file.mp3 (or .wav .m4a .mp4 .m4v .mov .webm)    Local media file

For local files, use --title to set the episode title (defaults to
the filename without extension).

Processing consumes credits from your account. The API is asynchronous —
the request returns immediately and the command polls for status until complete.

Status values:
  waiting     episode is queued and will be picked up shortly
  processing  transcription and AI analysis is in progress
  done        processing is complete; use "podwise get" to fetch results`,

	Example: `  podwise process https://podwise.ai/dashboard/episodes/7360326
  podwise process https://www.xiaoyuzhoufm.com/episode/abc123
  podwise process https://www.youtube.com/watch?v=d0-Gn_Bxf8s
  podwise process https://youtu.be/d0-Gn_Bxf8s
  podwise process ./interview.mp3 --title "My Interview"
  podwise process ./interview.mp3 --title "My Interview" --hotwords "podwise,ai,podcast"`,
	Args: cobra.ExactArgs(1),
	RunE: runProcess,
}

func init() {
	processCmd.Flags().DurationVar(&processPollInterval, "interval", 30*time.Second, "how often to poll for status updates (min 30s)")
	processCmd.Flags().DurationVar(&processTimeout, "timeout", 30*time.Minute, "maximum time to wait for processing to complete")
	processCmd.Flags().StringVar(&processTitle, "title", "", "episode title for local file uploads (defaults to filename without extension)")
	processCmd.Flags().StringVar(&processHotwords, "hotwords", "", "comma-separated hotwords to improve transcription accuracy (local media file only)")
}

func runProcess(cmd *cobra.Command, args []string) error {
	input := args[0]

	cfg, err := config.Load()
	if err != nil {
		return err
	}
	if err := config.Validate(cfg); err != nil {
		return err
	}

	client := api.New(cfg.APIBaseURL, cfg.APIKey)
	ctx := context.Background()
	startTime := time.Now()

	// Print an action-specific preamble before the (potentially slow) API call.
	switch {
	case episode.IsYouTubeURL(input) || episode.IsXiaoyuzhouURL(input):
		fmt.Printf("Importing episode from %s ...\n", input)
	case episode.IsLocalMediaFile(input):
		fmt.Printf("Uploading %s ...\n", input)
	}

	resolved, err := episode.ResolveInput(ctx, client, input, episode.ResolveOptions{
		Title:    processTitle,
		Hotwords: processHotwords,
	})
	if err != nil {
		var cleanupErr *episode.UploadCleanupError
		if errors.As(err, &cleanupErr) && cleanupErr.CleanupErr != nil {
			fmt.Printf("warning: orphaned storage object %q — cleanup failed: %v\n", cleanupErr.StoragePath, cleanupErr.CleanupErr)
		}
		return err
	}

	switch resolved.Kind {
	case episode.KindImport:
		fmt.Printf("Imported: %q (%s) → episode: %s\n\n", resolved.Import.Title, resolved.Import.PodcastName, episode.BuildEpisodeURL(resolved.Seq))
	case episode.KindUpload:
		fmt.Printf("Uploaded: %q → episode: %s\n\n", resolved.Upload.Title, episode.BuildEpisodeURL(resolved.Seq))
	}

	seq := resolved.Seq

	if processPollInterval < 30*time.Second {
		processPollInterval = 30 * time.Second
	}

	fmt.Printf("Submitting episode %s for processing...\n", episode.BuildEpisodeURL(seq))

	result, err := episode.SubmitProcess(ctx, client, seq)
	if err != nil {
		return err
	}

	var initialProgress float64
	if result.Progress != nil {
		initialProgress = *result.Progress
	}

	if (prettyOutput || prettyNoPager) && isTTY() {
		return runProcessPretty(ctx, client, seq, result.Status, initialProgress, startTime)
	}

	// Plain (non-pretty) path.
	printProcessStatus(result, initialProgress)
	if result.Status == "done" {
		printProcessDoneHint(seq, time.Since(startTime))
		return nil
	}

	deadline := time.Now().Add(processTimeout)
	ticker := time.NewTicker(processPollInterval)
	defer ticker.Stop()

	var maxProgress float64
	if result.Progress != nil {
		maxProgress = *result.Progress
	}

	for range ticker.C {
		if time.Now().After(deadline) {
			return fmt.Errorf("timed out after %s waiting for episode %s to finish processing", processTimeout, episode.BuildEpisodeURL(seq))
		}
		status, err := episode.FetchStatus(ctx, client, seq)
		if err != nil {
			return err
		}
		if status.Progress != nil && *status.Progress > maxProgress {
			maxProgress = *status.Progress
		}
		printProcessStatus(status, maxProgress)
		switch status.Status {
		case "done":
			printProcessDoneHint(seq, time.Since(startTime))
			return nil
		case "failed", "not_requested":
			return fmt.Errorf("processing failed for episode %s", episode.BuildEpisodeURL(seq))
		}
	}
	return nil
}

// runProcessPretty runs the polling loop with an animated spinner.
// A goroutine redraws the status line at 100 ms so the spinner animates
// continuously, while API polls happen on the normal (≥30 s) interval.
func runProcessPretty(ctx context.Context, client *api.Client, seq int, initialStatus string, initialProgress float64, startTime time.Time) error {
	// Shared state between the spinner goroutine and the poll loop.
	var mu sync.Mutex
	curStatus := initialStatus
	curProgress := initialProgress
	spinIdx := 0

	// Initial render.
	prettyPrintProcessStatus(&episode.ProcessResult{Status: curStatus}, curProgress, spinIdx)

	if curStatus == "done" {
		prettyPrintProcessDoneHint(seq, time.Since(startTime))
		return nil
	}

	stopCh := make(chan struct{})
	var wg sync.WaitGroup
	wg.Add(1)

	// Spinner goroutine: redraws the current line at ~100 ms.
	go func() {
		defer wg.Done()
		t := time.NewTicker(100 * time.Millisecond)
		defer t.Stop()
		for {
			select {
			case <-stopCh:
				return
			case <-t.C:
				mu.Lock()
				spinIdx++
				prettyPrintProcessStatus(&episode.ProcessResult{Status: curStatus}, curProgress, spinIdx)
				mu.Unlock()
			}
		}
	}()

	stopSpinner := func() {
		select {
		case <-stopCh:
		default:
			close(stopCh)
		}
		wg.Wait()
	}

	deadline := time.Now().Add(processTimeout)
	pollTicker := time.NewTicker(processPollInterval)
	defer pollTicker.Stop()

	for range pollTicker.C {
		if time.Now().After(deadline) {
			stopSpinner()
			fmt.Println()
			return fmt.Errorf("timed out after %s waiting for episode %s to finish processing", processTimeout, episode.BuildEpisodeURL(seq))
		}
		status, err := episode.FetchStatus(ctx, client, seq)
		if err != nil {
			stopSpinner()
			fmt.Println()
			return err
		}

		mu.Lock()
		if status.Progress != nil && *status.Progress > curProgress {
			curProgress = *status.Progress
		}
		curStatus = status.Status
		mu.Unlock()

		switch status.Status {
		case "done":
			stopSpinner()
			prettyPrintProcessStatus(status, curProgress, 0)
			prettyPrintProcessDoneHint(seq, time.Since(startTime))
			return nil
		case "failed", "not_requested":
			stopSpinner()
			prettyPrintProcessStatus(status, curProgress, 0)
			return fmt.Errorf("processing failed for episode %s", episode.BuildEpisodeURL(seq))
		}
	}
	return nil
}

// printProcessStatus prints a single status line. maxProgress is the
// highest progress value observed so far across all polls, used to
// suppress any regressive values returned by the API.
func printProcessStatus(r *episode.ProcessResult, maxProgress float64) {
	ts := time.Now().Format("15:04:05")
	switch r.Status {
	case "waiting":
		fmt.Printf("  [%s] → waiting       episode is queued for processing\n", ts)
	case "done":
		fmt.Printf("  [%s] ✓ done          processing complete (100%%)\n", ts)
	case "failed", "not_requested":
		fmt.Printf("  [%s] ✗ failed         transcription failed\n", ts)
	default:
		// "processing" and any other transitional status are all treated as in-progress.
		fmt.Printf("  [%s] → processing    %.0f%% complete\n", ts, maxProgress)
	}
}

func printProcessDoneHint(seq int, elapsed time.Duration) {
	episodeURL := episode.BuildEpisodeURL(seq)
	sep := "─────────────────────────────────────────────────────────"
	fmt.Printf("\n%s\n", sep)
	fmt.Printf("  ✓  Processing Complete\n")
	fmt.Printf("%s\n", sep)
	fmt.Printf("  Episode URL:   %s\n", episodeURL)
	fmt.Printf("  Duration   :   %s\n", utils.FormatDuration(elapsed))
	fmt.Printf("\n")
	fmt.Printf("  Next steps:\n")
	fmt.Printf("    podwise get transcript %s\n", episodeURL)
	fmt.Printf("    podwise get summary    %s\n", episodeURL)
	fmt.Printf("    podwise get --help     to see all available commands\n")
}

// prettySpinnerFrames are the braille spinner frames used in pretty mode.
var prettySpinnerFrames = []string{"⠋", "⠙", "⠹", "⠸", "⠼", "⠴", "⠦", "⠧", "⠇", "⠏"}

// prettyPrintProcessStatus overwrites the current terminal line with a styled
// status. spinnerIdx is incremented by the caller on each poll to advance the
// spinner. done/failed states end the line with \n so subsequent output is clean.
func prettyPrintProcessStatus(r *episode.ProcessResult, maxProgress float64, spinnerIdx int) {
	const (
		reset  = "\033[0m"
		bold   = "\033[1m"
		dim    = "\033[2m"
		green  = "\033[32m"
		yellow = "\033[33m"
		blue   = "\033[34m"
		red    = "\033[31m"
		cyan   = "\033[36m"
		barW   = 24
	)
	spinner := prettySpinnerFrames[spinnerIdx%len(prettySpinnerFrames)]

	buildBar := func(pct float64) string {
		n := int(pct / 100.0 * barW)
		if n > barW {
			n = barW
		}
		return strings.Repeat("█", n) + strings.Repeat("░", barW-n)
	}

	var line string
	switch r.Status {
	case "waiting":
		line = fmt.Sprintf("\r\033[K  %s%s%s %swaiting%s    queued for processing",
			yellow, spinner, reset, yellow, reset)
	case "done":
		bar := buildBar(100)
		line = fmt.Sprintf("\r\033[K  %s✓%s %sdone%s  %s[%s]%s %s100%%%s\n",
			green, reset, green+bold, reset, dim, bar, reset, bold, reset)
	case "failed", "not_requested":
		line = fmt.Sprintf("\r\033[K  %s✗%s %sfailed%s  transcription failed\n",
			red, reset, red+bold, reset)
	default:
		// "processing" and any other transitional status are all treated as in-progress.
		bar := buildBar(maxProgress)
		line = fmt.Sprintf("\r\033[K  %s%s%s %sprocessing%s  %s[%s]%s %s%.0f%%%s",
			blue, spinner, reset, blue, reset, dim, bar, reset, bold, maxProgress, reset)
	}
	fmt.Print(line)
}

func prettyPrintProcessDoneHint(seq int, elapsed time.Duration) {
	const (
		reset = "\033[0m"
		bold  = "\033[1m"
		dim   = "\033[2m"
		green = "\033[32m"
		cyan  = "\033[36m"
	)
	episodeURL := episode.BuildEpisodeURL(seq)
	sep := "─────────────────────────────────────────────────────────"
	fmt.Printf("\n%s%s%s\n", dim, sep, reset)
	fmt.Printf("  %s✓  Processing Complete%s\n", green+bold, reset)
	fmt.Printf("%s%s%s\n", dim, sep, reset)
	fmt.Printf("  %sEpisode URL:%s   %s%s%s\n", dim, reset, cyan, episodeURL, reset)
	fmt.Printf("  %sDuration   :%s   %s\n", dim, reset, utils.FormatDuration(elapsed))
	fmt.Printf("\n")
	fmt.Printf("  %sNext steps:%s\n", bold, reset)
	fmt.Printf("    %spodwise get transcript%s %s\n", cyan, reset, episodeURL)
	fmt.Printf("    %spodwise get summary%s    %s\n", cyan, reset, episodeURL)
	fmt.Printf("    %spodwise get --help%s     to see all available commands\n", cyan, reset)
}
