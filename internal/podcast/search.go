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

// PodcastSearchHit is a single podcast result returned by the search API.
type PodcastSearchHit struct {
	Seq             int    `json:"seq"`
	Name            string `json:"name"`
	Owner           string `json:"owner"`
	Description     string `json:"description"`
	Genre           string `json:"genre"`
	PodcastID       string `json:"podcastId"`
	LastPublishTime int64  `json:"lastPublishTime"`
	Cover           string `json:"cover"`
}

// PodcastSearchResult holds the full podcast search response from the API.
type PodcastSearchResult struct {
	Hits               []PodcastSearchHit `json:"result"`
	EstimatedTotalHits int                `json:"estimatedTotalHits"`
	Page               int                `json:"page"`
	HitsPerPage        int                `json:"hitsPerPage"`
}

// FormatText formats the podcast search results as a Markdown document.
// Returns "(no results found)" when the result set is empty.
func (r *PodcastSearchResult) FormatText(query string) string {
	if len(r.Hits) == 0 {
		return "(no results found)"
	}
	var sb strings.Builder
	fmt.Fprintf(&sb, "# Search Podcasts: %q\n\n", query)
	fmt.Fprintf(&sb, "**Found:** %d\n\n---\n", len(r.Hits))
	for i, hit := range r.Hits {
		lastPublish := time.Unix(hit.LastPublishTime, 0).Format("2006-01-02")
		fmt.Fprintf(&sb, "\n%d. %s\n\n", i+1, hit.Name)
		fmt.Fprintf(&sb, "- **Owner:** %s\n", hit.Owner)
		fmt.Fprintf(&sb, "- **Last Published:** %s\n", lastPublish)
		fmt.Fprintf(&sb, "- **Genre:** %s\n", hit.Genre)
		fmt.Fprintf(&sb, "- **Podcast URL:** %s\n", BuildPodcastURL(hit.Seq))
		sb.WriteString("\n")
	}
	return sb.String()
}

// PodcastSearchHitJSON is the JSON-serialisable view of a single podcast search result.
type PodcastSearchHitJSON struct {
	Name            string `json:"name"`
	Owner           string `json:"owner"`
	LastPublishDate string `json:"last_publish_date"`
	Genre           string `json:"genre"`
	PodcastURL      string `json:"podcast_url"`
}

// FormatJSON serialises the podcast search results as indented JSON.
// An empty result set is represented as a JSON empty array.
func (r *PodcastSearchResult) FormatJSON() ([]byte, error) {
	hits := make([]PodcastSearchHitJSON, 0, len(r.Hits))
	for _, hit := range r.Hits {
		hits = append(hits, PodcastSearchHitJSON{
			Name:            hit.Name,
			Owner:           hit.Owner,
			LastPublishDate: time.Unix(hit.LastPublishTime, 0).Format("2006-01-02"),
			Genre:           hit.Genre,
			PodcastURL:      BuildPodcastURL(hit.Seq),
		})
	}
	return json.MarshalIndent(hits, "", "  ")
}

// SearchPodcasts queries the Podwise podcast search API and returns the first page of results.
// limit is passed as hitsPerPage; the API is always queried at page 0.
func SearchPodcasts(ctx context.Context, client *api.Client, query string, limit int) (*PodcastSearchResult, error) {
	if query == "" {
		return nil, fmt.Errorf("search query must not be empty")
	}

	q := url.Values{}
	q.Set("q", query)
	q.Set("page", "0")
	q.Set("hitsPerPage", strconv.Itoa(limit))

	var result PodcastSearchResult
	if err := client.Get(ctx, "/open/v1/podcasts/search", q, &result); err != nil {
		return nil, err
	}
	return &result, nil
}
