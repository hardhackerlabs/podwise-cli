package episode

import (
	"context"

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
		return nil, err
	}
	return &resp.Result, nil
}
