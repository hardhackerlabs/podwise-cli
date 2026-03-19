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
)

// FollowedPodcast is a single podcast from the user's followed list.
type FollowedPodcast struct {
	PodcastID       string `json:"podcastId"`
	Seq             int    `json:"seq"`
	Name            string `json:"name"`
	Owner           string `json:"owner"`
	Cover           string `json:"cover"`
	Description     string `json:"description"`
	LastPublishTime int64  `json:"lastPublishTime"`
	IsStarred       bool   `json:"isStarred"`
	IsPrivate       bool   `json:"isPrivate"`
	FollowedAt      int64  `json:"followedAt"`
}

// FollowedPodcastsResult holds the list of followed podcasts returned by the API.
type FollowedPodcastsResult struct {
	Podcasts []FollowedPodcast
}

// FormatText formats the followed podcasts as a Markdown document.
func (r *FollowedPodcastsResult) FormatText(date string, days int) string {
	if len(r.Podcasts) == 0 {
		return fmt.Sprintf("(no followed podcasts with new episodes found for the %d day(s) up to %s)\n", days, date)
	}
	var sb strings.Builder
	fmt.Fprintf(&sb, "# Followed Podcasts\n\n")
	fmt.Fprintf(&sb, "**Date:** up to %s  **Days:** %d  **Total:** %d\n\n---\n", date, days, len(r.Podcasts))
	for i, p := range r.Podcasts {
		lastPublish := time.Unix(p.LastPublishTime, 0).Format("2006-01-02")
		fmt.Fprintf(&sb, "\n%d. %s\n\n", i+1, p.Name)
		fmt.Fprintf(&sb, "- **Owner:** %s\n", p.Owner)
		fmt.Fprintf(&sb, "- **Last Published:** %s\n", lastPublish)
		if p.IsStarred {
			fmt.Fprintf(&sb, "- **Starred:** Yes\n")
		}
		fmt.Fprintf(&sb, "- **Podcast URL:** %s\n", BuildPodcastURL(p.Seq))
		sb.WriteString("\n")
	}
	return sb.String()
}

// FollowedPodcastJSON is the JSON-serialisable view of a single followed podcast.
type FollowedPodcastJSON struct {
	Name            string `json:"name"`
	Owner           string `json:"owner"`
	LastPublishDate string `json:"last_publish_date"`
	IsStarred       bool   `json:"is_starred"`
	PodcastURL      string `json:"podcast_url"`
}

// FormatJSON serialises the followed podcasts as indented JSON.
func (r *FollowedPodcastsResult) FormatJSON() ([]byte, error) {
	items := make([]FollowedPodcastJSON, 0, len(r.Podcasts))
	for _, p := range r.Podcasts {
		items = append(items, FollowedPodcastJSON{
			Name:            p.Name,
			Owner:           p.Owner,
			LastPublishDate: time.Unix(p.LastPublishTime, 0).Format("2006-01-02"),
			IsStarred:       p.IsStarred,
			PodcastURL:      BuildPodcastURL(p.Seq),
		})
	}
	return json.MarshalIndent(items, "", "  ")
}

type followedPodcastsResponse struct {
	Success bool              `json:"success"`
	Result  []FollowedPodcast `json:"result"`
}

// FetchFollowedPodcasts returns podcasts the authenticated user follows that
// have new episodes within [date - days + 1, date] (inclusive).
// date must be in "YYYY-MM-DD" format.
func FetchFollowedPodcasts(ctx context.Context, client *api.Client, date string, days int) (*FollowedPodcastsResult, error) {
	q := url.Values{}
	q.Set("date", date)
	q.Set("days", strconv.Itoa(days))

	var resp followedPodcastsResponse
	if err := client.Get(ctx, "/open/v1/user/podcasts/followed", q, &resp); err != nil {
		return nil, err
	}
	return &FollowedPodcastsResult{Podcasts: resp.Result}, nil
}
