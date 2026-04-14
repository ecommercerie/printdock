package updater

import (
	"context"
	"encoding/json"
	"fmt"
	"io"
	"net/http"
	"os"
	"strconv"
	"strings"
	"time"
)

const (
	githubAPI       = "https://api.github.com/repos/ecommercerie/printdock/releases/latest"
	checkTimeout    = 10 * time.Second
	downloadTimeout = 300 * time.Second
)

// UpdateStatus holds the result of an update check.
type UpdateStatus struct {
	CurrentVersion string `json:"currentVersion"`
	LatestVersion  string `json:"latestVersion,omitempty"`
	UpdateAvail    bool   `json:"updateAvailable"`
	DownloadURL    string `json:"downloadUrl,omitempty"`
}

type githubRelease struct {
	TagName string        `json:"tag_name"`
	Assets  []githubAsset `json:"assets"`
}

type githubAsset struct {
	Name               string `json:"name"`
	BrowserDownloadURL string `json:"browser_download_url"`
}

// Check queries GitHub for the latest release and compares with current version.
func Check(currentVersion string) UpdateStatus {
	st := UpdateStatus{CurrentVersion: currentVersion}

	ctx, cancel := context.WithTimeout(context.Background(), checkTimeout)
	defer cancel()

	req, err := http.NewRequestWithContext(ctx, http.MethodGet, githubAPI, nil)
	if err != nil {
		return st
	}
	req.Header.Set("Accept", "application/vnd.github+json")

	resp, err := http.DefaultClient.Do(req)
	if err != nil {
		return st
	}
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusOK {
		return st
	}

	var release githubRelease
	if err := json.NewDecoder(resp.Body).Decode(&release); err != nil {
		return st
	}

	st.LatestVersion = strings.TrimPrefix(release.TagName, "v")

	if currentVersion != "" && currentVersion != "dev" && st.LatestVersion != "" {
		st.UpdateAvail = isNewer(st.LatestVersion, currentVersion)
	}

	// Find the exe asset
	for _, asset := range release.Assets {
		if asset.Name == "printdock.exe" {
			st.DownloadURL = asset.BrowserDownloadURL
			break
		}
	}

	return st
}

// isNewer returns true if latest is strictly greater than current (semver comparison).
func isNewer(latest, current string) bool {
	lp := parseVersion(latest)
	cp := parseVersion(current)
	for i := 0; i < 3; i++ {
		if lp[i] > cp[i] {
			return true
		}
		if lp[i] < cp[i] {
			return false
		}
	}
	return false
}

func parseVersion(v string) [3]int {
	var parts [3]int
	for i, s := range strings.SplitN(v, ".", 3) {
		if i >= 3 {
			break
		}
		parts[i], _ = strconv.Atoi(s)
	}
	return parts
}

// DownloadTo downloads the update binary to the specified path.
func DownloadTo(url, destPath string) error {
	ctx, cancel := context.WithTimeout(context.Background(), downloadTimeout)
	defer cancel()

	req, err := http.NewRequestWithContext(ctx, http.MethodGet, url, nil)
	if err != nil {
		return fmt.Errorf("create request: %w", err)
	}

	resp, err := http.DefaultClient.Do(req)
	if err != nil {
		return fmt.Errorf("download update: %w", err)
	}
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusOK {
		return fmt.Errorf("download update: HTTP %d", resp.StatusCode)
	}

	tmpFile, err := os.CreateTemp(os.TempDir(), "printdock-update-*.exe")
	if err != nil {
		return fmt.Errorf("create temp file: %w", err)
	}
	tmpPath := tmpFile.Name()

	if _, err := io.Copy(tmpFile, resp.Body); err != nil {
		tmpFile.Close()
		os.Remove(tmpPath)
		return fmt.Errorf("write update: %w", err)
	}
	tmpFile.Close()

	if err := os.Rename(tmpPath, destPath); err != nil {
		os.Remove(tmpPath)
		return fmt.Errorf("install update: %w", err)
	}

	return nil
}
