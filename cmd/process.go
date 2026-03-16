package cmd

import (
	"context"
	"errors"
	"fmt"
	"time"

	"github.com/hardhacker/podwise-cli/internal/api"
	"github.com/hardhacker/podwise-cli/internal/config"
	"github.com/hardhacker/podwise-cli/internal/episode"
	"github.com/hardhacker/podwise-cli/internal/utils"
	"github.com/spf13/cobra"
)

var processNoWait bool
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
	processCmd.Flags().BoolVar(&processNoWait, "no-wait", false, "submit and return immediately without polling for completion")
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
	printProcessStatus(result, initialProgress)

	if processNoWait || result.Status == "done" {
		if result.Status == "done" {
			printProcessDoneHint(seq, time.Since(startTime))
		}
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
		case "failed":
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
	case "processing":
		if maxProgress >= 0.0 {
			fmt.Printf("  [%s] → processing    %.0f%% complete\n", ts, maxProgress)
		}
	case "done":
		fmt.Printf("  [%s] ✓ done          processing complete (100%%)\n", ts)
	case "not_requested":
		fmt.Printf("  [%s] → not_requested  transcription has not been requested yet\n", ts)
	case "failed":
		fmt.Printf("  [%s] ✗ failed         transcription failed\n", ts)
	default:
		fmt.Printf("  [%s] ? %s\n", ts, r.Status)
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
