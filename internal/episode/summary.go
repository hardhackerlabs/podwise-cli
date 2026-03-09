package episode

import (
	"context"
	"fmt"

	"github.com/hardhacker/podwise-cli/internal/api"
	"github.com/hardhacker/podwise-cli/internal/cache"
)

// Keyword is a topic keyword extracted from the episode.
type Keyword struct {
	Key  string `json:"key"`
	Desc string `json:"desc"`
}

// Chapter is a time-stamped chapter in the episode.
type Chapter struct {
	Time    string `json:"time"`
	Title   string `json:"title"`
	Summary string `json:"summary"`
	HasAds  bool   `json:"has_ads"`
}

// ChapterPart is a finer-grained chapter segment.
type ChapterPart struct {
	Time  string `json:"time"`
	Title string `json:"title"`
	Start int    `json:"start"`
	End   int    `json:"end"`
}

// QA is a question-and-answer pair extracted from the episode.
type QA struct {
	Question        string `json:"question"`
	Answer          string `json:"answer"`
	QuestionSpeaker string `json:"question_speaker"`
	AnswerSpeaker   string `json:"answer_speaker"`
}

// Highlight is a notable moment in the episode.
type Highlight struct {
	Time    string `json:"time"`
	Content string `json:"content"`
}

// SummaryResult holds all AI-generated content for an episode.
type SummaryResult struct {
	Summary      string        `json:"summary"`
	Keywords     []Keyword     `json:"keywords"`
	Chapters     []Chapter     `json:"chapters"`
	Mindmap      string        `json:"mindmap"`
	ChapterParts []ChapterPart `json:"chapter_parts"`
	QAs          []QA          `json:"qas"`
	Takeaways    []string      `json:"takeaways"`
	Highlights   []Highlight   `json:"highlights"`
	Titles       []string      `json:"titles"`
	Intros       []string      `json:"intros"`
	Timestamps   []string      `json:"timestamps"`
}

type summaryResponse struct {
	Success bool          `json:"success"`
	Result  SummaryResult `json:"result"`
}

// FetchSummary returns the AI-generated summary result for the given episode seq.
// Results are transparently cached in ~/.cache/podwise/<seq>_summary.json;
// subsequent calls return the cached copy without hitting the network.
func FetchSummary(ctx context.Context, client *api.Client, seq int) (*SummaryResult, error) {
	const cacheType = "summary"

	var cached SummaryResult
	if hit, err := cache.Read(seq, cacheType, &cached); err != nil {
		return nil, fmt.Errorf("cache: %w", err)
	} else if hit {
		return &cached, nil
	}

	var resp summaryResponse
	apiPath := fmt.Sprintf("/open/v1/episodes/%d/summary", seq)
	if err := client.Get(ctx, apiPath, nil, &resp); err != nil {
		return nil, err
	}

	if err := cache.Write(seq, cacheType, resp.Result); err != nil {
		fmt.Printf("warning: could not write cache: %v\n", err)
	}

	return &resp.Result, nil
}
