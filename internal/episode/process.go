package episode

import (
	"context"
	"fmt"
	"path/filepath"
	"strings"

	"github.com/hardhacker/podwise-cli/internal/api"
)

// ProcessResult holds the processing status for an episode.
// Status values: "waiting", "processing", "done", "not_requested", "failed".
type ProcessResult struct {
	Status   string   `json:"status"`
	Progress *float64 `json:"progress"`
}

type processResponse struct {
	Success bool          `json:"success"`
	Result  ProcessResult `json:"result"`
}

// SubmitProcess calls POST /open/v1/episodes/{seq}/process.
// It submits the episode for AI processing (transcription and analysis) and
// returns the initial processing status. Calling this on an already-processed
// episode returns status "done" with progress 100 without consuming credits.
func SubmitProcess(ctx context.Context, client *api.Client, seq int) (*ProcessResult, error) {
	var resp processResponse
	apiPath := fmt.Sprintf("/open/v1/episodes/%d/process", seq)
	if err := client.Post(ctx, apiPath, nil, &resp); err != nil {
		return nil, formatProcessError(err)
	}
	return &resp.Result, nil
}

// FetchStatus calls GET /open/v1/episodes/{seq}/status and returns the current
// transcription status and progress. Use this to poll for completion after
// SubmitProcess; it does not consume credits.
func FetchStatus(ctx context.Context, client *api.Client, seq int) (*ProcessResult, error) {
	var resp processResponse
	apiPath := fmt.Sprintf("/open/v1/episodes/%d/status", seq)
	if err := client.Get(ctx, apiPath, nil, &resp); err != nil {
		return nil, formatStatusError(err)
	}
	return &resp.Result, nil
}

// formatProcessError translates API errors into user-friendly messages.
func formatProcessError(err error) error {
	apiErr, ok := err.(*api.APIError)
	if !ok {
		return err
	}

	switch apiErr.ErrCode {
	case "out_of_quota":
		return fmt.Errorf("insufficient transcribe credits")
	case "not_found":
		return fmt.Errorf("episode does not exist")
	default:
		return err
	}
}

// formatStatusError translates API errors into user-friendly messages.
func formatStatusError(err error) error {
	apiErr, ok := err.(*api.APIError)
	if !ok {
		return err
	}

	switch apiErr.ErrCode {
	case "not_found":
		return fmt.Errorf("episode does not exist")
	default:
		return err
	}
}

// ─── Input resolution ─────────────────────────────────────────────────────────

// InputKind identifies how a process input string was resolved.
type InputKind int

const (
	KindImport   InputKind = iota // YouTube / Xiaoyuzhou URL → Import API
	KindUpload                    // local media file path → Upload API
	KindExisting                  // Podwise episode URL → ParseSeq
)

// ResolveOptions holds parameters for local file uploads.
type ResolveOptions struct {
	// Title is the episode title; defaults to the filename stem for local files.
	Title string
	// Hotwords is a comma-separated list of terms to improve transcription accuracy.
	Hotwords string
}

// ResolveResult holds the outcome of resolving a process input string.
type ResolveResult struct {
	Seq    int
	Kind   InputKind
	Import *ImportResult // non-nil when Kind == KindImport
	Upload *UploadResult // non-nil when Kind == KindUpload
}

// ResolveInput resolves an input string (Podwise URL, YouTube/Xiaoyuzhou URL,
// or local file path) to an episode seq. It handles:
//   - YouTube / Xiaoyuzhou URLs → Import API; well-known API error codes are
//     translated to human-readable messages.
//   - Local media file paths → Upload API; title defaults to the filename stem.
//   - Podwise episode URLs → ParseSeq.
//
// Upload errors may wrap *UploadCleanupError when storage cleanup also failed;
// callers should errors.As-check to surface that detail to the user.
func ResolveInput(ctx context.Context, client *api.Client, input string, opts ResolveOptions) (*ResolveResult, error) {
	switch {
	case IsYouTubeURL(input) || IsXiaoyuzhouURL(input):
		result, err := Import(ctx, client, input)
		if err != nil {
			return nil, fmt.Errorf("import failed: %w", err)
		}
		return &ResolveResult{Seq: result.Seq, Kind: KindImport, Import: result}, nil

	case IsLocalMediaFile(input):
		title := opts.Title
		if title == "" {
			base := filepath.Base(input)
			title = strings.TrimSuffix(base, filepath.Ext(base))
		}
		result, err := Upload(ctx, client, UploadOptions{
			Title:    title,
			FilePath: input,
			Keywords: opts.Hotwords,
		})
		if err != nil {
			return nil, err // may be *UploadCleanupError
		}
		return &ResolveResult{Seq: result.Seq, Kind: KindUpload, Upload: result}, nil

	default:
		seq, err := ParseSeq(input)
		if err != nil {
			return nil, fmt.Errorf("invalid input: %w", err)
		}
		return &ResolveResult{Seq: seq, Kind: KindExisting}, nil
	}
}
