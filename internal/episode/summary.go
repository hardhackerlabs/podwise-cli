package episode

import (
	"context"
	"fmt"
	"strings"
	"time"

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

// FormatSummary returns the summary text followed by a numbered takeaway list.
// Returns an empty string when both summary and takeaways are absent.
func (r *SummaryResult) FormatSummary() string {
	var sb strings.Builder
	if r.Summary != "" {
		sb.WriteString(r.Summary)
	}
	if len(r.Takeaways) > 0 {
		sb.WriteString("\n\nTakeaways:\n")
		for i, t := range r.Takeaways {
			fmt.Fprintf(&sb, "%d. %s\n", i+1, t)
		}
	}
	return sb.String()
}

// FormatQA returns Q&A pairs as formatted text, or a placeholder when none exist.
func (r *SummaryResult) FormatQA() string {
	if len(r.QAs) == 0 {
		return "(no Q&A available)"
	}
	var sb strings.Builder
	for i, qa := range r.QAs {
		if qa.QuestionSpeaker != "" {
			fmt.Fprintf(&sb, "Q%d [%s]: %s\n", i+1, qa.QuestionSpeaker, qa.Question)
		} else {
			fmt.Fprintf(&sb, "Q%d: %s\n", i+1, qa.Question)
		}
		if qa.AnswerSpeaker != "" {
			fmt.Fprintf(&sb, "A%d [%s]: %s\n\n", i+1, qa.AnswerSpeaker, qa.Answer)
		} else {
			fmt.Fprintf(&sb, "A%d: %s\n\n", i+1, qa.Answer)
		}
	}
	return sb.String()
}

// FormatChapters returns chapters as a Markdown-style list, or a placeholder when none exist.
// Ad chapters are labelled [ad].
func (r *SummaryResult) FormatChapters() string {
	if len(r.Chapters) == 0 {
		return "(no chapters available)"
	}
	var sb strings.Builder
	for i, ch := range r.Chapters {
		adLabel := ""
		if ch.HasAds {
			adLabel = " [ad]"
		}
		fmt.Fprintf(&sb, "### [%s] Chapter %d: %s%s\n\n", ch.Time, i+1, ch.Title, adLabel)
		if ch.Summary != "" {
			fmt.Fprintf(&sb, "%s\n", ch.Summary)
		}
		sb.WriteByte('\n')
	}
	return sb.String()
}

// FormatMindmap returns the mind map markdown, or a placeholder when unavailable.
func (r *SummaryResult) FormatMindmap() string {
	if r.Mindmap == "" {
		return "(no mind map available)"
	}
	return r.Mindmap
}

// FormatHighlights returns numbered highlights with timestamps, or a placeholder when none exist.
func (r *SummaryResult) FormatHighlights() string {
	if len(r.Highlights) == 0 {
		return "(no highlights available)"
	}
	var sb strings.Builder
	for i, h := range r.Highlights {
		fmt.Fprintf(&sb, "%d. [%s] %s\n", i+1, h.Time, h.Content)
	}
	return sb.String()
}

// FormatKeywords returns numbered keywords with optional descriptions, or a placeholder when none exist.
func (r *SummaryResult) FormatKeywords() string {
	if len(r.Keywords) == 0 {
		return "(no keywords available)"
	}
	var sb strings.Builder
	for i, kw := range r.Keywords {
		if kw.Desc != "" {
			fmt.Fprintf(&sb, "%d. **%s**: %s\n", i+1, kw.Key, kw.Desc)
		} else {
			fmt.Fprintf(&sb, "%d. **%s**\n", i+1, kw.Key)
		}
	}
	return sb.String()
}

type summaryResponse struct {
	Success bool          `json:"success"`
	Result  SummaryResult `json:"result"`
}

// FetchSummary returns the AI-generated summary result for the given episode seq.
// Results are transparently cached in ~/.cache/podwise/<seq>_summary.json;
// subsequent calls return the cached copy without hitting the network.
//
// When forceRefresh is true, the cache is bypassed only if the cached file is
// older than 10 minutes; otherwise the cached copy is still returned as-is.
func FetchSummary(ctx context.Context, client *api.Client, seq int, forceRefresh bool) (*SummaryResult, error) {
	const cacheType = "summary"

	skipCache := false
	if forceRefresh {
		stale, err := cache.IsStale(seq, cacheType, 10*time.Minute)
		if err != nil {
			return nil, fmt.Errorf("cache: %w", err)
		}
		skipCache = stale
	}

	if !skipCache {
		var cached SummaryResult
		if hit, err := cache.Read(seq, cacheType, &cached); err != nil {
			return nil, fmt.Errorf("cache: %w", err)
		} else if hit {
			return &cached, nil
		}
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
