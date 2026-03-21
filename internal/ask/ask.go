package ask

import (
	"context"
	"fmt"
	"regexp"
	"strconv"
	"strings"

	"github.com/hardhacker/podwise-cli/internal/api"
	"github.com/hardhacker/podwise-cli/internal/episode"
	"github.com/hardhacker/podwise-cli/internal/utils"
)

// citationRe matches [citation:N] markers in answer text, tolerating optional
// whitespace around "citation", the colon, and the number.
var citationRe = regexp.MustCompile(`\[\s*citation\s*:\s*(\d+)\s*\]`)

// Source is a single cited podcast excerpt backing the AI answer.
type Source struct {
	Title       string `json:"title"`
	PodName     string `json:"podName"`
	Text        string `json:"text"`
	StartTime   int    `json:"startTime"`
	EndTime     int    `json:"endTime"`
	Speaker     string `json:"speaker"`
	EpID        string `json:"epId"`
	EpSeq       int    `json:"epSeq"`
	AudioLink   string `json:"audioLink"`
	LinkType    string `json:"linkType"`
	Transcribed bool   `json:"transcribed"`
}

// Result holds the full AI answer response.
type Result struct {
	Hash             string   `json:"hash"`
	Answer           string   `json:"answer"`
	RelatedQuestions []string `json:"relatedQuestions"`
	Sources          []Source `json:"sources"`
}

// linkCitations replaces every [citation:N] in text with a Markdown link
// [citation:N](episode-url) using the episode URL from sources[N-1].
// Citations with an out-of-range index are left unchanged.
func (r *Result) linkCitations(text string) string {
	return citationRe.ReplaceAllStringFunc(text, func(match string) string {
		sub := citationRe.FindStringSubmatch(match)
		if len(sub) < 2 {
			return match
		}
		n, err := strconv.Atoi(sub[1])
		if err != nil || n < 1 || n > len(r.Sources) {
			return match
		}
		url := episode.BuildEpisodeURL(r.Sources[n-1].EpSeq)
		return fmt.Sprintf("[citation:%d](%s)", n, url)
	})
}

// FormatText formats the ask result as a Markdown document.
// When showSources is true, cited excerpts and episode links are appended below the answer.
func (r *Result) FormatText(question string, showSources bool) string {
	var sb strings.Builder
	fmt.Fprintf(&sb, "# Q: %s\n\n", question)
	fmt.Fprintf(&sb, "%s\n", r.linkCitations(r.Answer))

	if showSources && len(r.Sources) > 0 {
		sb.WriteString("\n---\n\n## Sources\n\n")
		for i, src := range r.Sources {
			fmt.Fprintf(&sb, "%d. %s\n\n", i+1, src.Title)
			fmt.Fprintf(&sb, "- **Timestamp:** %s\n", utils.FormatTimestampMs(src.StartTime))
			fmt.Fprintf(&sb, "- **Episode URL:** %s\n", episode.BuildEpisodeURL(src.EpSeq))
			fmt.Fprintf(&sb, "\n%s\n", src.Text)
			sb.WriteString("\n")
		}
	}

	return sb.String()
}

type askResponse struct {
	Success bool   `json:"success"`
	Result  Result `json:"result"`
}

// Ask sends a question to the Podwise AI and returns the answer with sources.
func Ask(ctx context.Context, client *api.Client, question string) (*Result, error) {
	if question == "" {
		return nil, fmt.Errorf("question must not be empty")
	}

	body := map[string]string{"question": question}
	var resp askResponse
	if err := client.Post(ctx, "/open/v1/ask", body, &resp); err != nil {
		return nil, formatAskError(err)
	}
	return &resp.Result, nil
}

// formatAskError translates API errors into user-friendly messages.
func formatAskError(err error) error {
	apiErr, ok := err.(*api.APIError)
	if !ok {
		return err
	}

	switch apiErr.ErrCode {
	case "out_of_limit":
		return fmt.Errorf("daily ask limit exceeded for your plan")
	default:
		return err
	}
}
