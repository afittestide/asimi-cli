package main

import (
	"context"
	"fmt"
	"log/slog"
	"runtime"
	"strings"
	"time"

	"github.com/blang/semver"
	"github.com/rhysd/go-github-selfupdate/selfupdate"
)

const (
	githubOwner = "afittestide"
	githubRepo  = "asimi-cli"
)

// parseVersion parses a version string, handling "v" prefix
func parseVersion(v string) (semver.Version, error) {
	// Remove "v" prefix if present
	v = strings.TrimPrefix(v, "v")
	return semver.Parse(v)
}

// CheckForUpdates checks if a newer version is available on GitHub
func CheckForUpdates(currentVersion string) (*selfupdate.Release, bool, error) {
	slug := fmt.Sprintf("%s/%s", githubOwner, githubRepo)
	latest, found, err := selfupdate.DetectLatest(slug)
	if err != nil {
		return nil, false, fmt.Errorf("failed to detect latest version: %w", err)
	}

	if !found {
		return nil, false, fmt.Errorf("no release found")
	}

	current, err := parseVersion(currentVersion)
	if err != nil {
		return nil, false, fmt.Errorf("invalid current version: %w", err)
	}

	if latest.Version.LTE(current) {
		slog.Debug("current version is up to date", "current", currentVersion, "latest", latest.Version)
		return latest, false, nil
	}

	return latest, true, nil
}

// SelfUpdate performs the self-update to the latest version
func SelfUpdate(currentVersion string) error {
	current, err := parseVersion(currentVersion)
	if err != nil {
		return fmt.Errorf("invalid current version: %w", err)
	}

	slug := fmt.Sprintf("%s/%s", githubOwner, githubRepo)

	latest, err := selfupdate.UpdateSelf(current, slug)
	if err != nil {
		return fmt.Errorf("failed to update: %w", err)
	}

	if latest.Version.Equals(current) {
		slog.Info("already up to date", "version", currentVersion)
		return nil
	}

	slog.Info("successfully updated", "from", currentVersion, "to", latest.Version)
	return nil
}

// AutoCheckForUpdates checks for updates in the background (non-blocking)
// Returns true if an update is available
func AutoCheckForUpdates(currentVersion string) bool {
	// Skip if version is "dev" or empty
	if currentVersion == "" || currentVersion == "dev" {
		return false
	}

	// Check in background with timeout
	ctx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
	defer cancel()

	done := make(chan bool, 1)
	go func() {
		latest, hasUpdate, err := CheckForUpdates(currentVersion)
		if err != nil {
			slog.Debug("update check failed", "error", err)
			done <- false
			return
		}
		if hasUpdate {
			slog.Info("update available",
				"current", currentVersion,
				"latest", latest.Version,
				"url", latest.URL,
			)
		}
		done <- hasUpdate
	}()

	select {
	case hasUpdate := <-done:
		return hasUpdate
	case <-ctx.Done():
		slog.Debug("update check timed out")
		return false
	}
}

// GetUpdateCommand returns the command string to update asimi
func GetUpdateCommand() string {
	if runtime.GOOS == "darwin" || runtime.GOOS == "linux" {
		// Check if installed via Homebrew
		// This is a simple heuristic - could be improved
		return "brew upgrade asimi"
	}
	return "asimi update"
}
