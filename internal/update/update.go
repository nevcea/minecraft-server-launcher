package update

import (
	"encoding/json"
	"fmt"
	"io"
	"net/http"
	"os"
	"path/filepath"
	"runtime"
	"strings"
	"time"
)

const (
	timeout          = 30 * time.Second
	launcherVersion = "0.3.0"
	githubUserAgent  = "minecraft-server-launcher-updater/0.3"
)

var (
	githubAPIBase = "https://api.github.com/repos/nevcea-sub/minecraft-server-launcher/releases/latest"
)

var updateHTTPClient = &http.Client{
	Timeout: timeout,
	Transport: &http.Transport{
		MaxIdleConns:        10,
		MaxIdleConnsPerHost: 5,
		IdleConnTimeout:     30 * time.Second,
	},
}

type Asset struct {
	Name               string `json:"name"`
	BrowserDownloadURL string `json:"browser_download_url"`
	Size               int64  `json:"size"`
}

type ReleaseResponse struct {
	TagName     string  `json:"tag_name"`
	Name        string  `json:"name"`
	Body        string  `json:"body"`
	PublishedAt string  `json:"published_at"`
	Assets      []Asset `json:"assets"`
}

func GetCurrentVersion() string {
	return launcherVersion
}

func CheckForUpdate() (bool, *ReleaseResponse, error) {
	req, err := http.NewRequest("GET", githubAPIBase, nil)
	if err != nil {
		return false, nil, fmt.Errorf("failed to create request: %w", err)
	}
	req.Header.Set("User-Agent", githubUserAgent)
	req.Header.Set("Accept", "application/vnd.github.v3+json")

	resp, err := updateHTTPClient.Do(req)
	if err != nil {
		return false, nil, fmt.Errorf("failed to check for updates: %w", err)
	}
	defer func() {
		if err := resp.Body.Close(); err != nil {
			_ = err
		}
	}()

	if resp.StatusCode != 200 {
		return false, nil, fmt.Errorf("GitHub API returned status %d", resp.StatusCode)
	}

	var release ReleaseResponse
	if err := json.NewDecoder(resp.Body).Decode(&release); err != nil {
		return false, nil, fmt.Errorf("failed to parse release info: %w", err)
	}

	currentVersion := normalizeVersion(launcherVersion)
	latestVersion := normalizeVersion(release.TagName)

	if compareVersions(latestVersion, currentVersion) > 0 {
		return true, &release, nil
	}

	return false, nil, nil
}

func normalizeVersion(version string) string {
	version = strings.TrimPrefix(version, "v")
	version = strings.TrimSpace(version)
	return version
}

func compareVersions(v1, v2 string) int {
	parts1 := strings.Split(v1, ".")
	parts2 := strings.Split(v2, ".")

	maxLen := len(parts1)
	if len(parts2) > maxLen {
		maxLen = len(parts2)
	}

	for i := 0; i < maxLen; i++ {
		var num1, num2 int
		if i < len(parts1) {
			fmt.Sscanf(parts1[i], "%d", &num1)
		}
		if i < len(parts2) {
			fmt.Sscanf(parts2[i], "%d", &num2)
		}

		if num1 > num2 {
			return 1
		}
		if num1 < num2 {
			return -1
		}
	}

	return 0
}

func getAssetForCurrentOS(release *ReleaseResponse) *Asset {
	osName := runtime.GOOS
	arch := runtime.GOARCH

	var assetName string
	switch osName {
	case "windows":
		if arch == "amd64" {
			assetName = "paper-launcher-windows-amd64.exe"
		} else if arch == "arm64" {
			assetName = "paper-launcher-windows-arm64.exe"
		}
	case "linux":
		if arch == "amd64" {
			assetName = "paper-launcher-linux-amd64"
		} else if arch == "arm64" {
			assetName = "paper-launcher-linux-arm64"
		}
	case "darwin":
		if arch == "amd64" {
			assetName = "paper-launcher-darwin-amd64"
		} else if arch == "arm64" {
			assetName = "paper-launcher-darwin-arm64"
		}
	}

	if assetName == "" {
		return nil
	}

	for i := range release.Assets {
		if release.Assets[i].Name == assetName {
			return &release.Assets[i]
		}
	}

	return nil
}

func DownloadUpdate(release *ReleaseResponse) (string, error) {
	asset := getAssetForCurrentOS(release)
	if asset == nil {
		return "", fmt.Errorf("no compatible binary found for %s/%s", runtime.GOOS, runtime.GOARCH)
	}

	exePath, err := os.Executable()
	if err != nil {
		return "", fmt.Errorf("failed to get executable path: %w", err)
	}

	exeDir := filepath.Dir(exePath)
	exeName := filepath.Base(exePath)
	
	tempFile := filepath.Join(exeDir, exeName+".new")

	req, err := http.NewRequest("GET", asset.BrowserDownloadURL, nil)
	if err != nil {
		return "", fmt.Errorf("failed to create request: %w", err)
	}
	req.Header.Set("User-Agent", githubUserAgent)

	resp, err := updateHTTPClient.Do(req)
	if err != nil {
		return "", fmt.Errorf("failed to download update: %w", err)
	}
	defer func() {
		if err := resp.Body.Close(); err != nil {
			_ = err
		}
	}()

	if resp.StatusCode != 200 {
		return "", fmt.Errorf("download failed with status %d", resp.StatusCode)
	}

	out, err := os.Create(tempFile)
	if err != nil {
		return "", fmt.Errorf("failed to create temp file: %w", err)
	}

	var closed bool
	defer func() {
		if !closed {
			if err := out.Close(); err != nil {
				_ = err
			}
		}
	}()

	success := false
	defer func() {
		if !success {
			if !closed {
				if err := out.Close(); err != nil {
					_ = err
				}
				closed = true
			}
			if err := os.Remove(tempFile); err != nil {
				_ = err
			}
		}
	}()

	buf := make([]byte, 128*1024)
	_, err = io.CopyBuffer(out, resp.Body, buf)
	if err != nil {
		return "", fmt.Errorf("failed to write file: %w", err)
	}

	if err := out.Close(); err != nil {
		return "", fmt.Errorf("failed to close file: %w", err)
	}
	closed = true

	if runtime.GOOS != "windows" {
		if err := os.Chmod(tempFile, 0755); err != nil {
			return "", fmt.Errorf("failed to set executable permissions: %w", err)
		}
	}

	success = true
	return tempFile, nil
}

func InstallUpdate(tempFile string) error {
	exePath, err := os.Executable()
	if err != nil {
		return fmt.Errorf("failed to get executable path: %w", err)
	}

	exeDir := filepath.Dir(exePath)
	exeName := filepath.Base(exePath)
	backupFile := filepath.Join(exeDir, exeName+".old")

	if runtime.GOOS == "windows" {
		if _, err := os.Stat(backupFile); err == nil {
			if err := os.Remove(backupFile); err != nil {
				return fmt.Errorf("failed to remove old backup: %w", err)
			}
		}

		if err := os.Rename(exePath, backupFile); err != nil {
			return fmt.Errorf("failed to backup current executable: %w", err)
		}

		if err := os.Rename(tempFile, exePath); err != nil {
			if restoreErr := os.Rename(backupFile, exePath); restoreErr != nil {
				return fmt.Errorf("failed to install update and restore backup: %w (restore error: %v)", err, restoreErr)
			}
			return fmt.Errorf("failed to install update (backup restored): %w", err)
		}
	} else {
		if _, err := os.Stat(backupFile); err == nil {
			if err := os.Remove(backupFile); err != nil {
				return fmt.Errorf("failed to remove old backup: %w", err)
			}
		}

		if err := os.Rename(exePath, backupFile); err != nil {
			return fmt.Errorf("failed to backup current executable: %w", err)
		}

		if err := os.Rename(tempFile, exePath); err != nil {
			if restoreErr := os.Rename(backupFile, exePath); restoreErr != nil {
				return fmt.Errorf("failed to install update and restore backup: %w (restore error: %v)", err, restoreErr)
			}
			return fmt.Errorf("failed to install update (backup restored): %w", err)
		}

		if err := os.Chmod(exePath, 0755); err != nil {
			return fmt.Errorf("failed to set executable permissions: %w", err)
		}
	}

	return nil
}

func ValidateUpdate(tempFile string) error {
	info, err := os.Stat(tempFile)
	if err != nil {
		return fmt.Errorf("failed to stat update file: %w", err)
	}
	if info.Size() == 0 {
		return fmt.Errorf("update file is empty")
	}
	return nil
}

