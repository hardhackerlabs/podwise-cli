package episode

import (
	"context"
	"encoding/json"
	"fmt"
	"net/url"
	"strconv"
	"strings"
	"time"

	"github.com/hardhacker/podwise-cli/internal/api"
)

// FollowedEpisode is a single episode from a podcast the user has followed.
type FollowedEpisode struct {
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

// FollowedResult holds the list of followed episodes returned by the API.
type FollowedResult struct {
	Episodes []FollowedEpisode
}

// FormatText formats the followed episodes as a Markdown document.
// Returns "(no episodes found)" when the list is empty.
func (r *FollowedResult) FormatText(date string, days int) string {
	if len(r.Episodes) == 0 {
		return fmt.Sprintf("(no episodes found for the %d day(s) up to %s)\n", days, date)
	}
	var sb strings.Builder
	fmt.Fprintf(&sb, "# Followed Episodes\n\n")
	fmt.Fprintf(&sb, "**Date:** up to %s  **Days:** %d  **Total:** %d\n\n---\n", date, days, len(r.Episodes))
	for i, ep := range r.Episodes {
		publishDate := time.Unix(ep.PublishTime, 0).Format("2006-01-02")
		fmt.Fprintf(&sb, "\n## %d. %s\n\n", i+1, ep.Title)
		fmt.Fprintf(&sb, "- **Podcast:** %s\n", ep.PodcastName)
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
		fmt.Fprintf(&sb, "- **Episode URL:** %s\n", BuildEpisodeURL(ep.Seq))
		sb.WriteString("\n---\n")
	}
	return sb.String()
}

// FollowedEpisodeJSON is the JSON-serialisable view of a single followed episode.
type FollowedEpisodeJSON struct {
	Title       string  `json:"title"`
	PodcastName string  `json:"podcast_name"`
	PublishDate string  `json:"publish_date"`
	EpisodeURL  string  `json:"episode_url"`
	Duration    *string `json:"duration,omitempty"`
	Language    *string `json:"language,omitempty"`
	Processed   bool    `json:"processed"`
}

// FormatJSON serialises the followed episodes as indented JSON.
// An empty list is represented as a JSON empty array.
func (r *FollowedResult) FormatJSON() ([]byte, error) {
	items := make([]FollowedEpisodeJSON, 0, len(r.Episodes))
	for _, ep := range r.Episodes {
		items = append(items, FollowedEpisodeJSON{
			Title:       ep.Title,
			PodcastName: ep.PodcastName,
			PublishDate: time.Unix(ep.PublishTime, 0).Format("2006-01-02"),
			EpisodeURL:  BuildEpisodeURL(ep.Seq),
			Duration:    ep.Duration,
			Language:    ep.Language,
			Processed:   ep.Transcribed,
		})
	}
	return json.MarshalIndent(items, "", "  ")
}

type followedResponse struct {
	Success bool              `json:"success"`
	Result  []FollowedEpisode `json:"result"`
}

// FetchFollowedEpisodes returns episodes from podcasts the authenticated user follows
// within [date - days + 1, date] (inclusive). date must be in "YYYY-MM-DD" format.
func FetchFollowedEpisodes(ctx context.Context, client *api.Client, date string, days int) (*FollowedResult, error) {
	q := url.Values{}
	q.Set("date", date)
	q.Set("days", strconv.Itoa(days))

	var resp followedResponse
	if err := client.Get(ctx, "/open/v1/user/episodes/followed", q, &resp); err != nil {
		return nil, err
	}
	return &FollowedResult{Episodes: resp.Result}, nil
}

// Today returns today's date in "YYYY-MM-DD" format.
func Today() string {
	return time.Now().Format("2006-01-02")
}

// ParseDate converts a human-friendly date string into "YYYY-MM-DD".
// Accepted values:
//
//	"today"      → current local date
//	"yesterday"  → yesterday's local date
//	"YYYY-MM-DD" → returned as-is after validation
func ParseDate(s string) (string, error) {
	now := time.Now()
	switch strings.ToLower(strings.TrimSpace(s)) {
	case "today":
		return now.Format("2006-01-02"), nil
	case "yesterday":
		return now.AddDate(0, 0, -1).Format("2006-01-02"), nil
	default:
		t, err := time.Parse("2006-01-02", s)
		if err != nil {
			return "", fmt.Errorf("invalid date %q: use today, yesterday, or YYYY-MM-DD", s)
		}
		return t.Format("2006-01-02"), nil
	}
}
