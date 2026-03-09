package cache

import (
	"encoding/json"
	"errors"
	"fmt"
	"os"
	"path/filepath"
	"time"
)

// Dir returns the podwise cache directory (~/.cache/podwise).
// The directory can be overridden by setting PODWISE_CACHE_DIR (useful in tests).
func Dir() (string, error) {
	if override := os.Getenv("PODWISE_CACHE_DIR"); override != "" {
		return override, nil
	}
	home, err := os.UserHomeDir()
	if err != nil {
		return "", err
	}
	return filepath.Join(home, ".cache", "podwise"), nil
}

// filePath builds the cache file path for a given episode seq and content type.
// Layout: ~/.cache/podwise/<seq>_<contentType>.json
// This makes it trivial to:
//   - list all cached data for one episode: <seq>_*.json
//   - purge by type across episodes:        *_<contentType>.json
//   - purge a single episode entirely:      <seq>_*.json
func filePath(seq int, contentType string) (string, error) {
	dir, err := Dir()
	if err != nil {
		return "", err
	}
	return filepath.Join(dir, fmt.Sprintf("%d_%s.json", seq, contentType)), nil
}

// Read loads a cached value for (seq, contentType) into out.
// Returns (false, nil) when no cache entry exists (miss).
// Returns (false, err) when the file exists but cannot be read/decoded.
func Read(seq int, contentType string, out any) (hit bool, err error) {
	path, err := filePath(seq, contentType)
	if err != nil {
		return false, err
	}

	data, err := os.ReadFile(path)
	if errors.Is(err, os.ErrNotExist) {
		return false, nil
	}
	if err != nil {
		return false, fmt.Errorf("cache read: %w", err)
	}

	if err := json.Unmarshal(data, out); err != nil {
		return false, fmt.Errorf("cache decode: %w", err)
	}
	return true, nil
}

// Write serialises val as JSON and writes it to the cache file for (seq, contentType).
func Write(seq int, contentType string, val any) error {
	path, err := filePath(seq, contentType)
	if err != nil {
		return err
	}

	if err := os.MkdirAll(filepath.Dir(path), 0o700); err != nil {
		return fmt.Errorf("cache mkdir: %w", err)
	}

	data, err := json.MarshalIndent(val, "", "  ")
	if err != nil {
		return fmt.Errorf("cache encode: %w", err)
	}

	if err := os.WriteFile(path, data, 0o600); err != nil {
		return fmt.Errorf("cache write: %w", err)
	}
	return nil
}

// IsStale reports whether the cache file for (seq, contentType) is older than
// maxAge, or does not exist. A missing file is treated as stale.
func IsStale(seq int, contentType string, maxAge time.Duration) (bool, error) {
	modTime, exists, err := Stat(seq, contentType)
	if err != nil {
		return false, err
	}
	if !exists {
		return true, nil
	}
	return time.Since(modTime) > maxAge, nil
}

// Stat returns the modification time of the cache file, and whether it exists.
func Stat(seq int, contentType string) (modTime time.Time, exists bool, err error) {
	path, err := filePath(seq, contentType)
	if err != nil {
		return time.Time{}, false, err
	}
	info, err := os.Stat(path)
	if errors.Is(err, os.ErrNotExist) {
		return time.Time{}, false, nil
	}
	if err != nil {
		return time.Time{}, false, err
	}
	return info.ModTime(), true, nil
}
