package podcast

import "time"

// Episode represents a podcast episode on podwise.ai.
type Episode struct {
	ID          string
	Title       string
	PodcastName string
	AudioURL    string
	PublishedAt time.Time
}

// InsightType is the kind of AI-processed content to fetch.
type InsightType string

const (
	InsightSummary    InsightType = "summary"
	InsightOutline    InsightType = "outline"
	InsightTranscript InsightType = "transcript"
	InsightQA         InsightType = "qa"
	InsightMindmap    InsightType = "mindmap"
)

// Insight is the AI-processed result for a given episode.
type Insight struct {
	EpisodeID string
	Type      InsightType
	Language  string
	Content   string // raw content (markdown, srt, etc.)
}
