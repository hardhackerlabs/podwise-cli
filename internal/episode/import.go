package episode

import (
	"context"
	"fmt"

	"github.com/hardhacker/podwise-cli/internal/api"
)

// ImportResult holds the episode metadata returned by the import API.
// The fields mirror SearchHit (excluding Content).
type ImportResult struct {
	Seq         int    `json:"seq"`
	Title       string `json:"title"`
	PodcastName string `json:"podcastName"`
	EpisodeID   string `json:"episodeId"`
	PodcastID   string `json:"podcastId"`
	PublishTime int64  `json:"publishTime"`
	Cover       string `json:"cover"`
}

type importResponse struct {
	Success bool         `json:"success"`
	Result  ImportResult `json:"result"`
}

// Import calls POST /open/v1/episodes/import with a Xiaoyuzhou episode URL or
// YouTube video URL. It returns the resolved episode metadata, including the
// Seq needed for subsequent process/status API calls.
func Import(ctx context.Context, client *api.Client, rawURL string) (*ImportResult, error) {
	body := struct {
		URL string `json:"url"`
	}{URL: rawURL}

	var resp importResponse
	if err := client.Post(ctx, "/open/v1/episodes/import", body, &resp); err != nil {
		return nil, formatImportError(err)
	}
	return &resp.Result, nil
}

// formatImportError translates API errors into user-friendly messages.
func formatImportError(err error) error {
	apiErr, ok := err.(*api.APIError)
	if !ok {
		return err
	}

	switch apiErr.ErrCode {
	case "private_episode":
		return fmt.Errorf("the Xiaoyuzhou episode is private and cannot be imported")
	case "not_found":
		return fmt.Errorf("the YouTube video was not found")
	case "conflict":
		return fmt.Errorf("import conflict detected: please contact support at support@podwise.ai")
	case "fetch_error":
		return fmt.Errorf("failed to fetch data from the source: please try again later")
	default:
		return err
	}
}
