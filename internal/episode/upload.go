package episode

import (
	"context"
	"fmt"
	"io"
	"net/http"
	"os"
	"path/filepath"
	"strings"

	"github.com/hardhacker/podwise-cli/internal/api"
)

// mediaMIMEType maps lowercase file extensions to MIME types accepted by the API.
// Covers common audio and video formats.
var mediaMIMEType = map[string]string{
	// audio
	".mp3": "audio/mpeg",
	".wav": "audio/wav",
	".m4a": "audio/x-m4a",
	// video
	".mp4":  "video/mp4",
	".m4v":  "video/x-m4v",
	".mov":  "video/quicktime",
	".webm": "video/webm",
}

// UploadOptions holds all parameters for the three-step audio upload flow.
type UploadOptions struct {
	// Title is the episode title (required, max 512 characters).
	Title string
	// FilePath is the local path to the audio file (required).
	FilePath string
	// Description is the optional episode description.
	Description string
	// Keywords is a comma-separated list of keywords to improve transcription.
	Keywords string
	// Authors is a comma-separated list of speaker names to improve transcription.
	Authors string
	// Duration is the audio duration in seconds as a string (e.g. "3600"). Optional.
	Duration string
}

// UploadResult holds the episode metadata returned after a successful upload.
type UploadResult struct {
	Seq       int    `json:"seq"`
	Title     string `json:"title"`
	EpisodeID string `json:"episodeId"`
	PodcastID string `json:"podcastId"`
}

// presignRequest is the body for POST /open/v1/episodes/upload-audio/presign.
type presignRequest struct {
	FileName    string `json:"fileName"`
	ContentType string `json:"contentType"`
}

type presignResult struct {
	UploadURL   string `json:"uploadUrl"`
	StoragePath string `json:"storagePath"`
}

type presignResponse struct {
	Success bool          `json:"success"`
	Result  presignResult `json:"result"`
}

// createUploadRequest is the body for POST /open/v1/episodes/upload-audio.
type createUploadRequest struct {
	Title       string `json:"title"`
	StoragePath string `json:"storagePath"`
	AudioSize   int64  `json:"audioSize"`
	AudioType   string `json:"audioType"`
	Description string `json:"description,omitempty"`
	Keywords    string `json:"keywords,omitempty"`
	Authors     string `json:"authors,omitempty"`
	Duration    string `json:"duration,omitempty"`
}

type uploadResponse struct {
	Success bool         `json:"success"`
	Result  UploadResult `json:"result"`
}

// UploadCleanupError is returned when episode creation (step 3) fails after the
// audio has already been uploaded to storage (step 2). It wraps the original
// create error and records whether the best-effort DELETE cleanup succeeded.
// StoragePath is always populated so callers can attempt manual cleanup.
type UploadCleanupError struct {
	// CreateErr is the error returned by the create-episode API call.
	CreateErr error
	// CleanupErr is non-nil when the DELETE request to storage also failed.
	CleanupErr error
	// StoragePath is the R2 object path of the orphaned audio file.
	StoragePath string
}

func (e *UploadCleanupError) Error() string {
	if e.CleanupErr != nil {
		return fmt.Sprintf(
			"create episode failed (%v); storage cleanup also failed (%v) — orphaned object: %s",
			e.CreateErr, e.CleanupErr, e.StoragePath,
		)
	}
	return fmt.Sprintf(
		"create episode failed (%v); uploaded media has been removed from storage",
		e.CreateErr,
	)
}

func (e *UploadCleanupError) Unwrap() error { return e.CreateErr }

// Upload performs the full three-step audio upload flow:
//  1. Request a presigned PUT URL from the Podwise API.
//  2. Upload the audio file directly to Cloudflare R2 via HTTP PUT.
//  3. Create the episode using the returned storage path.
//
// If step 3 fails, Upload attempts a best-effort DELETE of the already-uploaded
// object and returns an *UploadCleanupError that wraps the original error.
func Upload(ctx context.Context, client *api.Client, opts UploadOptions) (*UploadResult, error) {
	// --- resolve file metadata ---
	info, err := os.Stat(opts.FilePath)
	if err != nil {
		return nil, fmt.Errorf("open audio file: %w", err)
	}
	if info.IsDir() {
		return nil, fmt.Errorf("%q is a directory, not an audio file", opts.FilePath)
	}

	fileName := filepath.Base(opts.FilePath)
	ext := strings.ToLower(filepath.Ext(fileName))

	contentType, ok := mediaMIMEType[ext]
	if !ok {
		return nil, fmt.Errorf(
			"unsupported media extension %q (audio: mp3 wav m4a — video: mp4 m4v mov webm)",
			ext,
		)
	}

	// audioType is the bare extension without the leading dot.
	audioType := strings.TrimPrefix(ext, ".")
	audioSize := info.Size()

	const maxFileSize = 500 << 20 // 500 MiB
	if audioSize > maxFileSize {
		return nil, fmt.Errorf("file %q is %.1f MiB, exceeds the 500 MiB upload limit", opts.FilePath, float64(audioSize)/(1<<20))
	}

	// --- step 1: get presigned upload URL ---
	var presignResp presignResponse
	if err := client.Post(ctx, "/open/v1/episodes/upload-audio/presign", presignRequest{
		FileName:    fileName,
		ContentType: contentType,
	}, &presignResp); err != nil {
		return nil, fmt.Errorf("presign upload URL: %w", err)
	}

	uploadURL := presignResp.Result.UploadURL
	storagePath := presignResp.Result.StoragePath

	if uploadURL == "" || storagePath == "" {
		return nil, fmt.Errorf("presign response missing uploadUrl or storagePath")
	}

	// --- step 2: PUT audio to Cloudflare R2 ---
	if err := putAudioFile(ctx, opts.FilePath, uploadURL, contentType, audioSize); err != nil {
		return nil, fmt.Errorf("upload audio to storage: %w", err)
	}

	// --- step 3: create episode ---
	createReq := createUploadRequest{
		Title:       opts.Title,
		StoragePath: storagePath,
		AudioSize:   audioSize,
		AudioType:   audioType,
		Description: opts.Description,
		Keywords:    opts.Keywords,
		Authors:     opts.Authors,
		Duration:    opts.Duration,
	}

	var uploadResp uploadResponse
	if err := client.Post(ctx, "/open/v1/episodes/upload-audio", createReq, &uploadResp); err != nil {
		// Step 3 failed — attempt to clean up the orphaned object in R2.
		// Use a background context so a cancelled caller context does not
		// prevent the DELETE from being sent.
		cleanupErr := deleteStorageObject(context.Background(), uploadURL)
		return nil, &UploadCleanupError{
			CreateErr:   err,
			CleanupErr:  cleanupErr,
			StoragePath: storagePath,
		}
	}

	return &uploadResp.Result, nil
}

// deleteStorageObject sends an HTTP DELETE to uploadURL to remove a previously
// uploaded object from Cloudflare R2. This is a best-effort call; errors are
// returned but do not affect the primary error path.
func deleteStorageObject(ctx context.Context, uploadURL string) error {
	req, err := http.NewRequestWithContext(ctx, http.MethodDelete, uploadURL, nil)
	if err != nil {
		return fmt.Errorf("build DELETE request: %w", err)
	}

	resp, err := http.DefaultClient.Do(req)
	if err != nil {
		return fmt.Errorf("DELETE %s: %w", uploadURL, err)
	}
	defer resp.Body.Close()

	// 2xx and 404 are both acceptable outcomes.
	if resp.StatusCode == http.StatusNotFound {
		return nil
	}
	if resp.StatusCode < 200 || resp.StatusCode >= 300 {
		body, _ := io.ReadAll(resp.Body)
		return fmt.Errorf("storage DELETE returned %d: %s", resp.StatusCode, strings.TrimSpace(string(body)))
	}
	return nil
}

// putAudioFile streams the local file to uploadURL via an authenticated PUT request.
// No API key is needed here; the URL itself carries the presigned credentials.
func putAudioFile(ctx context.Context, filePath, uploadURL, contentType string, size int64) error {
	f, err := os.Open(filePath)
	if err != nil {
		return fmt.Errorf("open %q: %w", filePath, err)
	}
	defer f.Close()

	req, err := http.NewRequestWithContext(ctx, http.MethodPut, uploadURL, f)
	if err != nil {
		return fmt.Errorf("build PUT request: %w", err)
	}
	req.ContentLength = size
	req.Header.Set("Content-Type", contentType)

	resp, err := http.DefaultClient.Do(req)
	if err != nil {
		return fmt.Errorf("PUT %s: %w", uploadURL, err)
	}
	defer resp.Body.Close()

	if resp.StatusCode < 200 || resp.StatusCode >= 300 {
		body, _ := io.ReadAll(resp.Body)
		return fmt.Errorf("storage PUT returned %d: %s", resp.StatusCode, strings.TrimSpace(string(body)))
	}

	return nil
}
