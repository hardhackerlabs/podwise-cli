package utils

import (
	"fmt"
	"strconv"
	"strings"
	"time"
)

// FormatTimestampMs converts a millisecond offset into a h:mm:ss or m:ss string.
func FormatTimestampMs(ms int) string {
	total := ms / 1000
	h := total / 3600
	m := (total % 3600) / 60
	s := total % 60
	if h > 0 {
		return fmt.Sprintf("%d:%02d:%02d", h, m, s)
	}
	return fmt.Sprintf("%d:%02d", m, s)
}

// FormatDuration returns a compact human-readable duration (e.g. "2m 34s", "45s").
func FormatDuration(d time.Duration) string {
	d = d.Round(time.Second)
	h := int(d.Hours())
	m := int(d.Minutes()) % 60
	s := int(d.Seconds()) % 60
	switch {
	case h > 0:
		return fmt.Sprintf("%dh %dm %ds", h, m, s)
	case m > 0:
		return fmt.Sprintf("%dm %ds", m, s)
	default:
		return fmt.Sprintf("%ds", s)
	}
}

// NormalizeDurationString converts a numeric duration string in seconds
// into HH:MM:SS. Non-numeric values are returned unchanged.
func NormalizeDurationString(value string) string {
	trimmed := strings.TrimSpace(value)
	if trimmed == "" {
		return value
	}

	seconds, err := strconv.ParseFloat(trimmed, 64)
	if err != nil || seconds < 0 {
		return value
	}

	totalSeconds := int(seconds)
	h := totalSeconds / 3600
	m := (totalSeconds % 3600) / 60
	s := totalSeconds % 60
	return fmt.Sprintf("%02d:%02d:%02d", h, m, s)
}

// BoolToYesNo converts a boolean value to "Yes" or "No".
func BoolToYesNo(b bool) string {
	if b {
		return "Yes"
	}
	return "No"
}
