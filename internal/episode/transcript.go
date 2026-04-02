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
	"github.com/hardhacker/podwise-cli/internal/async"
	"github.com/hardhacker/podwise-cli/internal/cache"
)

// Segment is one transcript chunk returned by the API.
type Segment struct {
	Time     string  `json:"time"`
	Start    float64 `json:"start,omitempty"`
	End      float64 `json:"end,omitempty"`
	Content  string  `json:"content"`
	Speaker  string  `json:"speaker,omitempty"`
	Language string  `json:"language,omitempty"`
}

// TranscriptResult holds the transcript segments and the episode metadata.
type TranscriptResult struct {
	Segments []Segment    `json:"segments"`
	Episode  *EpisodeInfo `json:"episode,omitempty"`
}

type transcriptResponse struct {
	Success bool        `json:"success"`
	Result  []Segment   `json:"result"`
	Episode EpisodeInfo `json:"episode"`
}

// FetchTranscripts returns the transcript segments for the given episode seq.
// Results are transparently cached in ~/.cache/podwise/<seq>_transcript[_<language>].json;
// subsequent calls return the cached copy without hitting the network.
//
// When language is non-empty (e.g. "Chinese", "English"), the API returns
// translated segments and the result is cached under a separate key so it does
// not overwrite the original-language cache.
//
// When forceRefresh is true, the cache is bypassed only if the cached file is
// older than 10 minutes; otherwise the cached copy is still returned as-is.
func FetchTranscripts(ctx context.Context, client *api.Client, seq int, forceRefresh bool, language string) (*TranscriptResult, error) {
	cacheType := "transcript"
	if language != "" {
		cacheType = "transcript_" + language
	}

	skipCache := false
	if forceRefresh {
		stale, _ := cache.IsStale(seq, cacheType, 10*time.Minute)
		skipCache = stale
	}

	if !skipCache {
		var cached TranscriptResult
		if hit, err := cache.Read(seq, cacheType, &cached); err == nil && hit {
			return &cached, nil
		}
	}

	var query url.Values
	if language != "" {
		query = url.Values{"translation": {strings.ReplaceAll(language, "-", " ")}}
	}

	var resp transcriptResponse
	path := fmt.Sprintf("/open/v1/episodes/%d/transcripts", seq)
	if err := client.Get(ctx, path, query, &resp); err != nil {
		return nil, formatTranscriptError(err)
	}

	result := &TranscriptResult{
		Segments: resp.Result,
		Episode:  &resp.Episode,
	}
	if err := cache.Write(seq, cacheType, result); err != nil {
		// Non-fatal: log but don't fail the command.
		fmt.Printf("warning: could not write cache: %v\n", err)
	}

	// Mark episode as read in background
	async.Go(func() {
		_ = MarkAsRead(context.Background(), client, seq)
	})

	return result, nil
}

// formatTranscriptError translates API errors into user-friendly messages.
func formatTranscriptError(err error) error {
	apiErr, ok := err.(*api.APIError)
	if !ok {
		return err
	}

	switch apiErr.ErrCode {
	case "not_found":
		return fmt.Errorf("episode does not exist")
	case "not_transcribed":
		return fmt.Errorf("episode has not been processed yet")
	case "not_translated":
		return fmt.Errorf("episode has not been translated yet")
	default:
		return err
	}
}

// ─── Helpers ──────────────────────────────────────────────────────────────────

// segmentEnd returns the end timestamp (ms) for a segment, falling back to start+2s.
func segmentEnd(seg Segment) float64 {
	if seg.End > seg.Start {
		return seg.End
	}
	return seg.Start + 2000
}

// msToTimestamp converts milliseconds to "HH:MM:SS<sep>mmm".
func msToTimestamp(ms float64, sep byte) string {
	total := int(ms)
	millis := total % 1000
	total /= 1000
	secs := total % 60
	total /= 60
	mins := total % 60
	hours := total / 60
	return fmt.Sprintf("%02d:%02d:%02d%c%03d", hours, mins, secs, sep, millis)
}

// segmentTimeLabel returns the timestamp string for a segment.
// When useSeconds is true, it returns seconds as a decimal string; otherwise hh:mm:ss.
func segmentTimeLabel(seg Segment, useSeconds bool) string {
	if useSeconds {
		return strconv.FormatFloat(seg.Start/1000, 'f', -1, 64)
	}
	return trimTime(seg.Time)
}

// endsWithSentenceTerminator reports whether s ends with a sentence-terminal
// punctuation mark (supports both ASCII and full-width CJK variants).
func endsWithSentenceTerminator(s string) bool {
	s = strings.TrimSpace(s)
	if s == "" {
		return false
	}
	runes := []rune(s)
	switch runes[len(runes)-1] {
	case '.', '?', '!', '…', ';',
		'。', '？', '！', '；':
		return true
	}
	return false
}

// MergeSegments merges consecutive same-speaker segments into longer passages.
//
// The primary goal is to accumulate as much content as possible up to
// maxDurationMs.  Sentence-terminating punctuation acts only as a soft hint:
// when a duration limit is hit, the cut is made at the last punctuation
// boundary within the current accumulation so the output ends at a natural
// sentence break; if no such boundary exists the cut falls at the last
// accumulated segment.
//
// Hard flush conditions (always start a new segment):
//   - The speaker changes.
//   - Adding the next segment would exceed maxDurationMs.
//
// The merged segment inherits the Time/Start/Speaker of the first segment in
// the group and the End of the last segment.  Content pieces are joined with a
// single space.
func MergeSegments(segments []Segment, maxDurationMs float64) []Segment {
	if len(segments) == 0 {
		return nil
	}

	var result []Segment
	var acc []Segment

	flushAcc := func(a []Segment) {
		if len(a) == 0 {
			return
		}
		merged := a[0]
		var sb strings.Builder
		for i, s := range a {
			c := strings.TrimSpace(s.Content)
			if c == "" {
				continue
			}
			if i > 0 && sb.Len() > 0 {
				sb.WriteByte(' ')
			}
			sb.WriteString(c)
		}
		merged.Content = sb.String()
		merged.End = segmentEnd(a[len(a)-1])
		if merged.Content != "" {
			result = append(result, merged)
		}
	}

	for _, seg := range segments {
		if seg.Content == "" {
			continue
		}

		if len(acc) == 0 {
			acc = append(acc, seg)
			continue
		}

		speakerChanged := seg.Speaker != acc[0].Speaker
		durationExceeded := segmentEnd(seg)-acc[0].Start > maxDurationMs

		if speakerChanged || durationExceeded {
			if durationExceeded && !speakerChanged {
				// Prefer to cut at the last sentence-terminating punctuation
				// already in acc so the merged segment ends naturally.
				lastTerm := -1
				for i, s := range acc {
					if endsWithSentenceTerminator(s.Content) {
						lastTerm = i
					}
				}
				if lastTerm >= 0 {
					flushAcc(acc[:lastTerm+1])
					acc = append(acc[lastTerm+1:], seg)
					continue
				}
			}
			flushAcc(acc)
			acc = acc[:0]
		}

		acc = append(acc, seg)
	}

	flushAcc(acc)
	return result
}

// ─── Formatters ───────────────────────────────────────────────────────────────

// FormatTranscriptText formats transcript segments as plain text.
// Each line is "[timestamp] - [speaker: ]content".
// When useSeconds is true, timestamps are shown as elapsed seconds instead of hh:mm:ss.
func FormatTranscriptText(segments []Segment, useSeconds bool) string {
	var sb strings.Builder
	for _, seg := range segments {
		if seg.Content == "" {
			continue
		}
		t := segmentTimeLabel(seg, useSeconds)
		if seg.Speaker != "" {
			fmt.Fprintf(&sb, "[%s] - %s: %s\n", t, seg.Speaker, seg.Content)
		} else {
			fmt.Fprintf(&sb, "[%s] - %s\n", t, seg.Content)
		}
	}
	return sb.String()
}

// FormatMergedTranscript formats merged transcript segments as plain text.
// Each segment is rendered as "[timestamp] - [speaker: ]content" with a blank
// line between entries, which improves readability in Markdown previews.
func FormatMergedTranscript(segments []Segment) string {
	var sb strings.Builder
	for i, seg := range segments {
		if seg.Content == "" {
			continue
		}
		if i > 0 {
			sb.WriteByte('\n')
		}
		t := trimTime(seg.Time)
		if seg.Speaker != "" {
			fmt.Fprintf(&sb, "[%s] **%s**\n%s\n", t, seg.Speaker, seg.Content)
		} else {
			fmt.Fprintf(&sb, "[%s]\n%s\n", t, seg.Content)
		}
	}
	return sb.String()
}

// SegmentJSON is the JSON-serialisable view of a single transcript segment.
type SegmentJSON struct {
	Start   any    `json:"start"`
	Speaker string `json:"speaker,omitempty"`
	Content string `json:"content"`
}

// FormatTranscriptJSON serialises transcript segments as indented JSON.
// When useSeconds is true, the start field is a float (seconds); otherwise a string timestamp.
func FormatTranscriptJSON(segments []Segment, useSeconds bool) ([]byte, error) {
	out := make([]SegmentJSON, 0, len(segments))
	for _, seg := range segments {
		if seg.Content == "" {
			continue
		}
		var start any
		if useSeconds {
			start = seg.Start / 1000
		} else {
			start = seg.Time
		}
		out = append(out, SegmentJSON{Start: start, Speaker: seg.Speaker, Content: seg.Content})
	}
	return json.MarshalIndent(out, "", "  ")
}

// FormatTranscriptSRT formats transcript segments as an SRT subtitle file.
func FormatTranscriptSRT(segments []Segment) string {
	var sb strings.Builder
	idx := 1
	for _, seg := range segments {
		if seg.Content == "" {
			continue
		}
		fmt.Fprintf(&sb, "%d\n%s --> %s\n",
			idx,
			msToTimestamp(seg.Start, ','),
			msToTimestamp(segmentEnd(seg), ','),
		)
		idx++
		if seg.Speaker != "" {
			fmt.Fprintf(&sb, "%s: %s\n", seg.Speaker, seg.Content)
		} else {
			sb.WriteString(seg.Content)
			sb.WriteByte('\n')
		}
		sb.WriteByte('\n')
	}
	return sb.String()
}

// FormatTranscriptVTT formats transcript segments as a WebVTT subtitle file.
func FormatTranscriptVTT(segments []Segment) string {
	var sb strings.Builder
	sb.WriteString("WEBVTT\n\n")
	for _, seg := range segments {
		if seg.Content == "" {
			continue
		}
		fmt.Fprintf(&sb, "%s --> %s\n",
			msToTimestamp(seg.Start, '.'),
			msToTimestamp(segmentEnd(seg), '.'),
		)
		if seg.Speaker != "" {
			fmt.Fprintf(&sb, "%s: %s\n", seg.Speaker, seg.Content)
		} else {
			sb.WriteString(seg.Content)
			sb.WriteByte('\n')
		}
		sb.WriteByte('\n')
	}
	return sb.String()
}
