package update

import (
	"encoding/json"
	"errors"
	"fmt"
	"net/http"
	"os"
	"path/filepath"
	"strings"
	"time"
)

const (
	checkInterval    = 24 * time.Hour
	githubReleaseURL = "https://api.github.com/repos/hardhackerlabs/podwise-cli/releases/latest"
	releasesPageURL  = "https://github.com/hardhackerlabs/podwise-cli/releases"
	httpTimeout      = 5 * time.Second
)

// Result holds the outcome of an update check.
type Result struct {
	LatestVersion string
	HasUpdate     bool
}

type checkState struct {
	LastCheck     time.Time `json:"last_check"`
	LatestVersion string    `json:"latest_version"`
}

// Check returns whether a newer release exists compared to currentVersion.
// Results are cached for 24 hours. All errors are silently swallowed —
// update checks are best-effort and must never break the CLI.
func Check(currentVersion string) Result {
	state := loadState()

	var latestVersion string
	if time.Since(state.LastCheck) < checkInterval && state.LatestVersion != "" {
		latestVersion = state.LatestVersion
	} else {
		var err error
		latestVersion, err = fetchLatestVersion()
		if err != nil {
			return Result{}
		}
		_ = saveState(&checkState{
			LastCheck:     time.Now(),
			LatestVersion: latestVersion,
		})
	}

	return Result{
		LatestVersion: latestVersion,
		HasUpdate:     isNewer(latestVersion, currentVersion),
	}
}

// UpgradeHint returns a human-readable upgrade instruction based on how the
// binary was installed (Homebrew vs. manual/script install).
func UpgradeHint() string {
	if execPath, err := os.Executable(); err == nil {
		lower := strings.ToLower(execPath)
		if strings.Contains(lower, "/homebrew/") || strings.Contains(lower, "/linuxbrew/") {
			return "brew update && brew upgrade podwise"
		}
	}
	return fmt.Sprintf(
		"curl -sL https://raw.githubusercontent.com/hardhackerlabs/podwise-cli/main/install.sh | sh\n  or download from: %s",
		releasesPageURL,
	)
}

func fetchLatestVersion() (string, error) {
	client := &http.Client{Timeout: httpTimeout}
	resp, err := client.Get(githubReleaseURL)
	if err != nil {
		return "", err
	}
	defer resp.Body.Close()

	var release struct {
		TagName string `json:"tag_name"`
	}
	if err := json.NewDecoder(resp.Body).Decode(&release); err != nil {
		return "", err
	}

	v := strings.TrimPrefix(release.TagName, "v")
	if v == "" {
		return "", errors.New("empty tag_name in GitHub response")
	}
	return v, nil
}

// isNewer reports whether latest is strictly greater than current (semver).
func isNewer(latest, current string) bool {
	if current == "dev" || current == "" {
		return false
	}
	latest = strings.TrimPrefix(latest, "v")
	current = strings.TrimPrefix(current, "v")
	if latest == current {
		return false
	}
	return semverGT(latest, current)
}

func semverGT(a, b string) bool {
	var aMaj, aMin, aPatch int
	var bMaj, bMin, bPatch int
	fmt.Sscanf(a, "%d.%d.%d", &aMaj, &aMin, &aPatch)
	fmt.Sscanf(b, "%d.%d.%d", &bMaj, &bMin, &bPatch)
	if aMaj != bMaj {
		return aMaj > bMaj
	}
	if aMin != bMin {
		return aMin > bMin
	}
	return aPatch > bPatch
}

func stateFilePath() (string, error) {
	home, err := os.UserHomeDir()
	if err != nil {
		return "", err
	}
	return filepath.Join(home, ".cache", "podwise", "update_check.json"), nil
}

func loadState() checkState {
	path, err := stateFilePath()
	if err != nil {
		return checkState{}
	}
	data, err := os.ReadFile(path)
	if err != nil {
		return checkState{}
	}
	var s checkState
	if err := json.Unmarshal(data, &s); err != nil {
		return checkState{}
	}
	return s
}

func saveState(s *checkState) error {
	path, err := stateFilePath()
	if err != nil {
		return err
	}
	if err := os.MkdirAll(filepath.Dir(path), 0o700); err != nil {
		return err
	}
	data, err := json.MarshalIndent(s, "", "  ")
	if err != nil {
		return err
	}
	return os.WriteFile(path, data, 0o600)
}
