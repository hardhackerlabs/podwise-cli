package podcast

import (
	"context"
	"encoding/json"
	"fmt"
	"net/url"
	"strconv"
	"strings"
	"time"

	"github.com/hardhacker/podwise-cli/internal/api"
	"github.com/hardhacker/podwise-cli/internal/episode"
)

// PodcastEpisode is a single episode returned when listing a podcast's episodes.
type PodcastEpisode struct {
	EpisodeID   string  `json:"episodeId"`
	Seq         int     `json:"seq"`
	Title       string  `json:"title"`
	PodcastName string  `json:"podcastName"`
	Cover       string  `json:"cover"`
	EpCover     *string `json:"epCover"`
	Description *string `json:"description"`
	PublishTime int64   `json:"publishTime"`
	Link        string  `json:"link"`
	LinkType    string  `json:"linkType"`
	Duration    *string `json:"duration"`
	Transcribed bool    `json:"transcribed"`
	Language    *string `json:"language"`
}

// PodcastEpisodesResult holds the list of episodes for a specific podcast.
type PodcastEpisodesResult struct {
	PodcastSeq int
	Episodes   []PodcastEpisode
}

// FormatText formats the podcast episodes as a Markdown document.
func (r *PodcastEpisodesResult) FormatText(date string, days int) string {
	if len(r.Episodes) == 0 {
		return fmt.Sprintf("(no episodes found for podcast %s in the %d day(s) up to %s)\n",
			BuildPodcastURL(r.PodcastSeq), days, date)
	}
	var sb strings.Builder
	podcastName := r.Episodes[0].PodcastName
	fmt.Fprintf(&sb, "# %s\n\n", podcastName)
	fmt.Fprintf(&sb, "**Podcast URL:** %s\n", BuildPodcastURL(r.PodcastSeq))
	fmt.Fprintf(&sb, "**Date:** up to %s  **Days:** %d  **Total:** %d\n\n---\n", date, days, len(r.Episodes))
	for i, ep := range r.Episodes {
		publishDate := time.Unix(ep.PublishTime, 0).Format("2006-01-02")
		fmt.Fprintf(&sb, "\n%d. %s\n\n", i+1, ep.Title)
		fmt.Fprintf(&sb, "- **Published:** %s\n", publishDate)
		if ep.Duration != nil && *ep.Duration != "" {
			fmt.Fprintf(&sb, "- **Duration:** %s\n", *ep.Duration)
		}
		if ep.Language != nil && *ep.Language != "" {
			fmt.Fprintf(&sb, "- **Language:** %s\n", *ep.Language)
		}
		processedLabel := "No"
		if ep.Transcribed {
			processedLabel = "Yes"
		}
		fmt.Fprintf(&sb, "- **Processed:** %s\n", processedLabel)
		fmt.Fprintf(&sb, "- **Episode URL:** %s\n", episode.BuildEpisodeURL(ep.Seq))
		sb.WriteString("\n")
	}
	return sb.String()
}

// PodcastEpisodeJSON is the JSON-serialisable view of a single podcast episode.
type PodcastEpisodeJSON struct {
	Title       string  `json:"title"`
	PublishDate string  `json:"publish_date"`
	EpisodeURL  string  `json:"episode_url"`
	Duration    *string `json:"duration,omitempty"`
	Language    *string `json:"language,omitempty"`
	Processed   bool    `json:"processed"`
}

// FormatJSON serialises the podcast episodes as indented JSON.
func (r *PodcastEpisodesResult) FormatJSON() ([]byte, error) {
	items := make([]PodcastEpisodeJSON, 0, len(r.Episodes))
	for _, ep := range r.Episodes {
		items = append(items, PodcastEpisodeJSON{
			Title:       ep.Title,
			PublishDate: time.Unix(ep.PublishTime, 0).Format("2006-01-02"),
			EpisodeURL:  fmt.Sprintf("https://podwise.ai/dashboard/episodes/%d", ep.Seq),
			Duration:    ep.Duration,
			Language:    ep.Language,
			Processed:   ep.Transcribed,
		})
	}
	return json.MarshalIndent(items, "", "  ")
}

type podcastEpisodesResponse struct {
	Success bool             `json:"success"`
	Result  []PodcastEpisode `json:"result"`
}

// FetchPodcastEpisodes returns episodes for the given podcast seq within
// [date - days + 1, date] (inclusive). date must be in "YYYY-MM-DD" format.
func FetchPodcastEpisodes(ctx context.Context, client *api.Client, podcastSeq int, date string, days int) (*PodcastEpisodesResult, error) {
	q := url.Values{}
	q.Set("date", date)
	q.Set("days", strconv.Itoa(days))

	path := fmt.Sprintf("/open/v1/podcasts/%d/episodes", podcastSeq)
	var resp podcastEpisodesResponse
	if err := client.Get(ctx, path, q, &resp); err != nil {
		return nil, err
	}
	return &PodcastEpisodesResult{PodcastSeq: podcastSeq, Episodes: resp.Result}, nil
}
