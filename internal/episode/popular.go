package episode

import (
	"context"
	"encoding/json"
	"fmt"
	"net/url"
	"strconv"
	"strings"

	"github.com/hardhacker/podwise-cli/internal/api"
)

// PopularEpisode is a single entry in the trending/popular episodes list.
type PopularEpisode struct {
	EpisodeID   string  `json:"episodeId"`
	Seq         int     `json:"seq"`
	Title       string  `json:"title"`
	PodcastName string  `json:"podcastName"`
	PodcastSeq  int     `json:"podcastSeq"`
	Cover       string  `json:"cover"`
	EpCover     *string `json:"epCover"`
	LinkType    string  `json:"linkType"`
}

// PopularResult holds the list of popular episodes returned by the API.
type PopularResult struct {
	Episodes []PopularEpisode
}

// FormatText formats the popular episodes as a Markdown document.
// Returns "(no popular episodes found)" when the list is empty.
func (r *PopularResult) FormatText() string {
	if len(r.Episodes) == 0 {
		return "(no popular episodes found)"
	}
	var sb strings.Builder
	fmt.Fprintf(&sb, "# Trending Episodes\n\n**Total:** %d\n\n---\n", len(r.Episodes))
	for i, ep := range r.Episodes {
		fmt.Fprintf(&sb, "\n## %d. %s\n\n", i+1, ep.Title)
		fmt.Fprintf(&sb, "- **Podcast:** %s\n", ep.PodcastName)
		fmt.Fprintf(&sb, "- **Episode URL:** %s\n", BuildEpisodeURL(ep.Seq))
		if ep.LinkType == "youtube" {
			fmt.Fprintf(&sb, "- **Is YouTube:** %s\n", "Yes")
		} else {
			fmt.Fprintf(&sb, "- **Is YouTube:** %s\n", "No")
		}
		sb.WriteString("\n---\n")
	}
	return sb.String()
}

// PopularEpisodeJSON is the JSON-serialisable view of a single popular episode.
type PopularEpisodeJSON struct {
	Title       string `json:"title"`
	PodcastName string `json:"podcast_name"`
	EpisodeURL  string `json:"episode_url"`
	IsYouTube   string `json:"is_youtube"`
}

// FormatJSON serialises the popular episodes as indented JSON.
// An empty list is represented as a JSON empty array.
func (r *PopularResult) FormatJSON() ([]byte, error) {
	items := make([]PopularEpisodeJSON, 0, len(r.Episodes))
	for _, ep := range r.Episodes {
		isYouTube := "No"
		if ep.LinkType == "youtube" {
			isYouTube = "Yes"
		}
		items = append(items, PopularEpisodeJSON{
			Title:       ep.Title,
			PodcastName: ep.PodcastName,
			EpisodeURL:  BuildEpisodeURL(ep.Seq),
			IsYouTube:   isYouTube,
		})
	}
	return json.MarshalIndent(items, "", "  ")
}

type popularResponse struct {
	Success bool             `json:"success"`
	Result  []PopularEpisode `json:"result"`
}

// FetchPopular returns the current trending episodes from the Podwise API.
// limit controls how many results to return (max 100).
func FetchPopular(ctx context.Context, client *api.Client, limit int) (*PopularResult, error) {
	if limit > 50 {
		limit = 50
	}
	if limit < 1 {
		limit = 10
	}

	q := url.Values{}
	q.Set("limit", strconv.Itoa(limit))

	var resp popularResponse
	if err := client.Get(ctx, "/open/v1/episodes/popular", q, &resp); err != nil {
		return nil, err
	}
	return &PopularResult{Episodes: resp.Result}, nil
}
