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

// SearchHit is a single episode result returned by the search API.
type SearchHit struct {
	Seq         int    `json:"seq"`
	Title       string `json:"title"`
	PodcastName string `json:"podcastName"`
	Content     string `json:"content"`
	EpisodeID   string `json:"episodeId"`
	PodcastID   string `json:"podcastId"`
	PublishTime int64  `json:"publishTime"`
	Cover       string `json:"cover"`
}

// SearchResult holds the full search response from the API.
type SearchResult struct {
	Hits               []SearchHit `json:"result"`
	EstimatedTotalHits int         `json:"estimatedTotalHits"`
	Page               int         `json:"page"`
	HitsPerPage        int         `json:"hitsPerPage"`
}

// FormatText formats the search results as a Markdown document for the given query.
// Returns "(no results found)" when the result set is empty.
func (r *SearchResult) FormatText(query string) string {
	if len(r.Hits) == 0 {
		return "(no results found)"
	}
	var sb strings.Builder
	fmt.Fprintf(&sb, "# Search: %q\n\n", query)
	fmt.Fprintf(&sb, "**Found:** %d\n\n---\n", len(r.Hits))
	for i, hit := range r.Hits {
		publishDate := time.Unix(hit.PublishTime, 0).Format("2006-01-02")
		fmt.Fprintf(&sb, "\n## %d. %s\n\n", i+1, hit.Title)
		fmt.Fprintf(&sb, "- **Podcast:** %s\n", hit.PodcastName)
		fmt.Fprintf(&sb, "- **Published:** %s\n", publishDate)
		fmt.Fprintf(&sb, "- **Episode URL:** %s\n", BuildEpisodeURL(hit.Seq))
		if hit.Content != "" {
			fmt.Fprintf(&sb, "\n> %s\n", hit.Content)
		}
		sb.WriteString("\n---\n")
	}
	return sb.String()
}

// SearchHitJSON is the JSON-serialisable view of a single search result,
// with field names and date formatting suited for external consumers.
type SearchHitJSON struct {
	Title       string `json:"title"`
	PodcastName string `json:"podcast_name"`
	PublishDate string `json:"publish_date"`
	EpisodeURL  string `json:"episode_url"`
	Description string `json:"description,omitempty"`
}

// FormatJSON serialises the search results as indented JSON.
// An empty result set is represented as a JSON empty array.
func (r *SearchResult) FormatJSON() ([]byte, error) {
	hits := make([]SearchHitJSON, 0, len(r.Hits))
	for _, hit := range r.Hits {
		hits = append(hits, SearchHitJSON{
			Title:       hit.Title,
			PodcastName: hit.PodcastName,
			PublishDate: time.Unix(hit.PublishTime, 0).Format("2006-01-02"),
			EpisodeURL:  BuildEpisodeURL(hit.Seq),
			Description: hit.Content,
		})
	}
	return json.MarshalIndent(hits, "", "  ")
}

// Search queries the Podwise episode search API and returns the first page of results.
// limit is passed as hitsPerPage; the API is always queried at page 0.
func Search(ctx context.Context, client *api.Client, query string, limit int) (*SearchResult, error) {
	if query == "" {
		return nil, fmt.Errorf("search query must not be empty")
	}

	q := url.Values{}
	q.Set("q", query)
	q.Set("page", "0")
	q.Set("hitsPerPage", strconv.Itoa(limit))

	var result SearchResult
	if err := client.Get(ctx, "/open/v1/episodes/search", q, &result); err != nil {
		return nil, err
	}
	return &result, nil
}
