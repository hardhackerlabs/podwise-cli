package episode

import (
	"context"
	"encoding/json"
	"fmt"
	"net/url"
	"strings"
	"time"

	"github.com/hardhacker/podwise-cli/internal/api"
	"github.com/hardhacker/podwise-cli/internal/utils"
)

// ReadEpisode represents an episode in the user's read history.
type ReadEpisode struct {
	EpisodeID   string  `json:"episodeId"`
	Seq         int64   `json:"seq"`
	Title       string  `json:"title"`
	PodcastName string  `json:"podcastName"`
	Cover       string  `json:"cover"`
	EpCover     *string `json:"epCover"`
	Description *string `json:"description"`
	PublishTime int64   `json:"publishTime"`
	Link        string  `json:"link"`
	LinkType    string  `json:"linkType"`
	Duration    *int    `json:"duration"`
	Transcribed bool    `json:"transcribed"`
	Language    *string `json:"language"`
}

// ReadHistoryResult holds the list of read episodes returned by the API.
type ReadHistoryResult struct {
	Episodes []ReadEpisode
}

// FormatText formats the read episodes as a Markdown document.
// Returns "(no episodes found)" when the list is empty.
func (r *ReadHistoryResult) FormatText() string {
	if len(r.Episodes) == 0 {
		return "(no read episodes found)\n"
	}
	var sb strings.Builder
	fmt.Fprintf(&sb, "# Read Episodes\n\n")
	fmt.Fprintf(&sb, "**Total:** %d\n\n---\n", len(r.Episodes))
	for i, ep := range r.Episodes {
		publishDate := time.Unix(ep.PublishTime, 0).Format("2006-01-02")
		fmt.Fprintf(&sb, "\n%d. %s\n\n", i+1, ep.Title)
		fmt.Fprintf(&sb, "- **Podcast:** %s\n", ep.PodcastName)
		fmt.Fprintf(&sb, "- **Published:** %s\n", publishDate)
		if ep.Duration != nil && *ep.Duration > 0 {
			fmt.Fprintf(&sb, "- **Duration:** %s\n", utils.FormatDuration(time.Duration(*ep.Duration)*time.Second))
		}
		if ep.Language != nil && *ep.Language != "" {
			fmt.Fprintf(&sb, "- **Language:** %s\n", *ep.Language)
		}
		fmt.Fprintf(&sb, "- **Episode URL:** %s\n", BuildEpisodeURL(int(ep.Seq)))
		sb.WriteString("\n")
	}
	return sb.String()
}

// ReadEpisodeJSON is the JSON-serializable view of a single read episode.
type ReadEpisodeJSON struct {
	Title       string  `json:"title"`
	PodcastName string  `json:"podcast_name"`
	PublishDate string  `json:"publish_date"`
	EpisodeURL  string  `json:"episode_url"`
	Duration    *string `json:"duration,omitempty"`
	Language    *string `json:"language,omitempty"`
}

// FormatJSON serializes the read episodes as indented JSON.
func (r *ReadHistoryResult) FormatJSON() ([]byte, error) {
	items := make([]ReadEpisodeJSON, 0, len(r.Episodes))
	for _, ep := range r.Episodes {
		var duration *string
		if ep.Duration != nil && *ep.Duration > 0 {
			dur := utils.FormatDuration(time.Duration(*ep.Duration) * time.Second)
			duration = &dur
		}
		items = append(items, ReadEpisodeJSON{
			Title:       ep.Title,
			PodcastName: ep.PodcastName,
			PublishDate: time.Unix(ep.PublishTime, 0).Format("2006-01-02"),
			EpisodeURL:  BuildEpisodeURL(int(ep.Seq)),
			Duration:    duration,
			Language:    ep.Language,
		})
	}
	return json.MarshalIndent(items, "", "  ")
}

// ReadHistoryResponse represents the API response for read history.
type ReadHistoryResponse struct {
	Success  bool          `json:"success"`
	Result   []ReadEpisode `json:"result"`
	Page     int           `json:"page"`
	PageSize int           `json:"pageSize"`
}

// FetchReadHistory retrieves the user's read episodes history.
// It handles pagination internally to return up to limit records.
func FetchReadHistory(ctx context.Context, client *api.Client, limit int) (*ReadHistoryResult, error) {
	if limit <= 0 {
		limit = 20
	}
	if limit > 100 {
		limit = 100
	}

	pageSize := 50 // API max is 50
	var allEpisodes []ReadEpisode
	page := 0

	for len(allEpisodes) < limit {
		query := url.Values{
			"page":     []string{fmt.Sprintf("%d", page)},
			"pageSize": []string{fmt.Sprintf("%d", pageSize)},
		}

		var resp ReadHistoryResponse
		err := client.Get(ctx, "/open/v1/user/episodes/read", query, &resp)
		if err != nil {
			return nil, err
		}

		if len(resp.Result) == 0 {
			break
		}

		allEpisodes = append(allEpisodes, resp.Result...)

		// Check if we've reached the end
		if len(resp.Result) < pageSize {
			break
		}
		page++
	}

	// Trim to the requested limit
	if len(allEpisodes) > limit {
		allEpisodes = allEpisodes[:limit]
	}

	return &ReadHistoryResult{Episodes: allEpisodes}, nil
}

// PlayedEpisode represents an episode in the user's play history.
type PlayedEpisode = ReadEpisode

// PlayedHistoryResult holds the list of played episodes returned by the API.
type PlayedHistoryResult struct {
	Episodes []PlayedEpisode
}

// FormatText formats the played episodes as a Markdown document.
func (r *PlayedHistoryResult) FormatText() string {
	if len(r.Episodes) == 0 {
		return "(no listened episodes found)\n"
	}
	var sb strings.Builder
	fmt.Fprintf(&sb, "# Listened Episodes\n\n")
	fmt.Fprintf(&sb, "**Total:** %d\n\n---\n", len(r.Episodes))
	for i, ep := range r.Episodes {
		publishDate := time.Unix(ep.PublishTime, 0).Format("2006-01-02")
		fmt.Fprintf(&sb, "\n%d. %s\n\n", i+1, ep.Title)
		fmt.Fprintf(&sb, "- **Podcast:** %s\n", ep.PodcastName)
		fmt.Fprintf(&sb, "- **Published:** %s\n", publishDate)
		if ep.Duration != nil && *ep.Duration > 0 {
			fmt.Fprintf(&sb, "- **Duration:** %s\n", utils.FormatDuration(time.Duration(*ep.Duration)*time.Second))
		}
		if ep.Language != nil && *ep.Language != "" {
			fmt.Fprintf(&sb, "- **Language:** %s\n", *ep.Language)
		}
		fmt.Fprintf(&sb, "- **Episode URL:** %s\n", BuildEpisodeURL(int(ep.Seq)))
		sb.WriteString("\n")
	}
	return sb.String()
}

// FormatJSON serializes the played episodes as indented JSON.
func (r *PlayedHistoryResult) FormatJSON() ([]byte, error) {
	items := make([]ReadEpisodeJSON, 0, len(r.Episodes))
	for _, ep := range r.Episodes {
		var duration *string
		if ep.Duration != nil && *ep.Duration > 0 {
			dur := utils.FormatDuration(time.Duration(*ep.Duration) * time.Second)
			duration = &dur
		}
		items = append(items, ReadEpisodeJSON{
			Title:       ep.Title,
			PodcastName: ep.PodcastName,
			PublishDate: time.Unix(ep.PublishTime, 0).Format("2006-01-02"),
			EpisodeURL:  BuildEpisodeURL(int(ep.Seq)),
			Duration:    duration,
			Language:    ep.Language,
		})
	}
	return json.MarshalIndent(items, "", "  ")
}

// PlayedHistoryResponse represents the API response for played history.
type PlayedHistoryResponse struct {
	Success  bool            `json:"success"`
	Result   []PlayedEpisode `json:"result"`
	Page     int             `json:"page"`
	PageSize int             `json:"pageSize"`
}

// FetchPlayedHistory retrieves the user's played episodes history.
// It handles pagination internally to return up to limit records.
func FetchPlayedHistory(ctx context.Context, client *api.Client, limit int) (*PlayedHistoryResult, error) {
	if limit <= 0 {
		limit = 20
	}
	if limit > 100 {
		limit = 100
	}

	pageSize := 50
	var allEpisodes []PlayedEpisode
	page := 0

	for len(allEpisodes) < limit {
		query := url.Values{
			"page":     []string{fmt.Sprintf("%d", page)},
			"pageSize": []string{fmt.Sprintf("%d", pageSize)},
		}

		var resp PlayedHistoryResponse
		err := client.Get(ctx, "/open/v1/user/episodes/played", query, &resp)
		if err != nil {
			return nil, err
		}

		if len(resp.Result) == 0 {
			break
		}

		allEpisodes = append(allEpisodes, resp.Result...)

		if len(resp.Result) < pageSize {
			break
		}
		page++
	}

	if len(allEpisodes) > limit {
		allEpisodes = allEpisodes[:limit]
	}

	return &PlayedHistoryResult{Episodes: allEpisodes}, nil
}
