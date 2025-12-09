package system

import (
	"context"
	"encoding/json"
	"fmt"
	"net/http"
	"time"
)

// VersionChecker checks for new versions
type VersionChecker struct {
	repoOwner string
	repoName  string
	client    *http.Client
	cache     *versionCache
}

type versionCache struct {
	latest    string
	checkedAt time.Time
	ttl       time.Duration
}

// NewVersionChecker creates a new version checker
func NewVersionChecker(repoOwner, repoName string) *VersionChecker {
	return &VersionChecker{
		repoOwner: repoOwner,
		repoName:  repoName,
		client: &http.Client{
			Timeout: 10 * time.Second,
		},
		cache: &versionCache{
			ttl: 1 * time.Hour,
		},
	}
}

// GitHubRelease represents a GitHub release
type GitHubRelease struct {
	TagName     string    `json:"tag_name"`
	Name        string    `json:"name"`
	PublishedAt time.Time `json:"published_at"`
	HTMLURL     string    `json:"html_url"`
	Prerelease  bool      `json:"prerelease"`
	Draft       bool      `json:"draft"`
}

// UpdateInfo contains information about available updates
type UpdateInfo struct {
	CurrentVersion  string `json:"current_version"`
	LatestVersion   string `json:"latest_version"`
	UpdateAvailable bool   `json:"update_available"`
	ReleaseURL      string `json:"release_url,omitempty"`
}

// CheckForUpdates checks for available updates
func (c *VersionChecker) CheckForUpdates(ctx context.Context) (*UpdateInfo, error) {
	// Check cache first
	if c.cache.latest != "" && time.Since(c.cache.checkedAt) < c.cache.ttl {
		return &UpdateInfo{
			CurrentVersion:  Version,
			LatestVersion:   c.cache.latest,
			UpdateAvailable: isNewerVersion(c.cache.latest, Version),
		}, nil
	}

	// Fetch latest release from GitHub
	url := fmt.Sprintf("https://api.github.com/repos/%s/%s/releases/latest", c.repoOwner, c.repoName)

	req, err := http.NewRequestWithContext(ctx, "GET", url, nil)
	if err != nil {
		return nil, err
	}
	req.Header.Set("Accept", "application/vnd.github.v3+json")

	resp, err := c.client.Do(req)
	if err != nil {
		return nil, err
	}
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusOK {
		return &UpdateInfo{
			CurrentVersion:  Version,
			LatestVersion:   Version,
			UpdateAvailable: false,
		}, nil
	}

	var release GitHubRelease
	if err := json.NewDecoder(resp.Body).Decode(&release); err != nil {
		return nil, err
	}

	// Update cache
	c.cache.latest = release.TagName
	c.cache.checkedAt = time.Now()

	return &UpdateInfo{
		CurrentVersion:  Version,
		LatestVersion:   release.TagName,
		UpdateAvailable: isNewerVersion(release.TagName, Version),
		ReleaseURL:      release.HTMLURL,
	}, nil
}

// isNewerVersion compares semantic versions (simplified)
func isNewerVersion(latest, current string) bool {
	if current == "dev" || current == "unknown" {
		return false // Development version
	}

	// Remove 'v' prefix if present
	if len(latest) > 0 && latest[0] == 'v' {
		latest = latest[1:]
	}
	if len(current) > 0 && current[0] == 'v' {
		current = current[1:]
	}

	// Simple string comparison for semver
	// In production, use proper semver library
	return latest > current
}
