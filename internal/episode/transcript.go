package episode

import (
	"context"
	"encoding/json"
	"fmt"
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

type transcriptResponse struct {
	Success bool      `json:"success"`
	Result  []Segment `json:"result"`
}

// FetchTranscripts returns the transcript segments for the given episode seq.
// Results are transparently cached in ~/.cache/podwise/<seq>_transcript.json;
// subsequent calls return the cached copy without hitting the network.
//
// When forceRefresh is true, the cache is bypassed only if the cached file is
// older than 10 minutes; otherwise the cached copy is still returned as-is.
func FetchTranscripts(ctx context.Context, client *api.Client, seq int, forceRefresh bool) ([]Segment, error) {
	const cacheType = "transcript"

	skipCache := false
	if forceRefresh {
		stale, err := cache.IsStale(seq, cacheType, 10*time.Minute)
		if err != nil {
			return nil, fmt.Errorf("cache: %w", err)
		}
		skipCache = stale
	}

	if !skipCache {
		var cached []Segment
		if hit, err := cache.Read(seq, cacheType, &cached); err != nil {
			return nil, fmt.Errorf("cache: %w", err)
		} else if hit {
			return cached, nil
		}
	}

	var resp transcriptResponse
	path := fmt.Sprintf("/open/v1/episodes/%d/transcripts", seq)
	if err := client.Get(ctx, path, nil, &resp); err != nil {
		return nil, formatTranscriptError(err)
	}

	if err := cache.Write(seq, cacheType, resp.Result); err != nil {
		// Non-fatal: log but don't fail the command.
		fmt.Printf("warning: could not write cache: %v\n", err)
	}

	// Mark episode as read in background
	async.Go(func() {
		_ = MarkAsRead(context.Background(), client, seq)
	})

	return resp.Result, nil
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
	return seg.Time
}

// ─── Formatters ───────────────────────────────────────────────────────────────

// FormatTranscriptText formats transcript segments as plain text.
// Each line is "[timestamp] - [speaker: ]content".
// When useSeconds is true, timestamps are shown as elapsed seconds instead of hh:mm:ss.
func FormatTranscriptText(segments []Segment, useSeconds bool) string {
	var sb strings.Builder
	for _, seg := range segments {
		t := segmentTimeLabel(seg, useSeconds)
		if seg.Speaker != "" {
			fmt.Fprintf(&sb, "[%s] - %s: %s\n", t, seg.Speaker, seg.Content)
		} else {
			fmt.Fprintf(&sb, "[%s] - %s\n", t, seg.Content)
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
	out := make([]SegmentJSON, len(segments))
	for i, seg := range segments {
		var start any
		if useSeconds {
			start = seg.Start / 1000
		} else {
			start = seg.Time
		}
		out[i] = SegmentJSON{Start: start, Speaker: seg.Speaker, Content: seg.Content}
	}
	return json.MarshalIndent(out, "", "  ")
}

// FormatTranscriptSRT formats transcript segments as an SRT subtitle file.
func FormatTranscriptSRT(segments []Segment) string {
	var sb strings.Builder
	for i, seg := range segments {
		fmt.Fprintf(&sb, "%d\n%s --> %s\n",
			i+1,
			msToTimestamp(seg.Start, ','),
			msToTimestamp(segmentEnd(seg), ','),
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

// FormatTranscriptVTT formats transcript segments as a WebVTT subtitle file.
func FormatTranscriptVTT(segments []Segment) string {
	var sb strings.Builder
	sb.WriteString("WEBVTT\n\n")
	for _, seg := range segments {
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
