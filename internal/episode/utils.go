package episode

import (
	"fmt"
	"net/url"
	"os"
	"strconv"
	"strings"
)

// ParseSeq extracts the integer episode seq from a podwise episode URL.
// Expected format: https://podwise.ai/dashboard/episodes/<seq>
func ParseSeq(input string) (int, error) {
	const hint = "(expected https://podwise.ai/dashboard/episodes/<id>)"

	u, err := url.Parse(input)
	if err != nil || u.Scheme != "https" || (u.Host != "podwise.ai" && u.Host != "beta.podwise.ai") {
		return 0, fmt.Errorf("%q is not a valid podwise episode URL %s", input, hint)
	}

	parts := strings.Split(strings.Trim(u.Path, "/"), "/")
	if len(parts) != 3 || parts[0] != "dashboard" || parts[1] != "episodes" || parts[2] == "" {
		return 0, fmt.Errorf("%q is not a valid podwise episode URL %s", input, hint)
	}

	seq, err := strconv.Atoi(parts[2])
	if err != nil || seq <= 0 {
		return 0, fmt.Errorf("episode ID %q is not a positive integer %s", parts[2], hint)
	}
	return seq, nil
}

// BuildEpisodeURL builds a podwise episode URL from a sequence number.
func BuildEpisodeURL(seq int) string {
	return fmt.Sprintf("https://podwise.ai/dashboard/episodes/%d", seq)
}

// IsYouTubeURL reports whether rawURL points to a YouTube video.
// Recognised hosts: youtube.com (and www.), youtu.be.
func IsYouTubeURL(rawURL string) bool {
	u, err := url.Parse(rawURL)
	if err != nil || u.Scheme != "https" {
		return false
	}
	switch u.Hostname() {
	case "youtube.com", "www.youtube.com":
		return u.Query().Get("v") != ""
	case "youtu.be":
		return len(u.Path) > 1
	}
	return false
}

// IsLocalMediaFile reports whether path refers to an existing regular file
// (not a directory or URL). Extension validation is left to episode.Upload
// so that the error message lists all supported formats.
func IsLocalMediaFile(path string) bool {
	info, err := os.Stat(path)
	return err == nil && !info.IsDir()
}

// IsXiaoyuzhouURL reports whether rawURL points to a Xiaoyuzhou episode.
func IsXiaoyuzhouURL(rawURL string) bool {
	u, err := url.Parse(rawURL)
	if err != nil || u.Scheme != "https" {
		return false
	}
	return u.Hostname() == "www.xiaoyuzhoufm.com" && strings.HasPrefix(u.Path, "/episode/")
}
