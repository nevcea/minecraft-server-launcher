package update

import (
	"context"
	"encoding/json"
	"fmt"
	"io"
	"net/http"
	"os"
	"os/exec"
	"path/filepath"
	"runtime"
	"strings"
	"sync"

	"github.com/nevcea-sub/minecraft-server-launcher/internal/utils"
)

var (
	launcherVersion  = "dev"
	githubUserAgent  = "minecraft-server-launcher-updater"
	githubToken      string
	cachedGitVersion string
	gitVersionOnce   sync.Once
)

var (
	githubAPIBase = "https://api.github.com/repos/nevcea/minecraft-server-launcher/releases/latest"
)

type Asset struct {
	ID                 int64  `json:"id"`
	URL                string `json:"url"`
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

func SetGitHubToken(token string) {
	githubToken = strings.TrimSpace(token)
}

func getGitHubToken() string {
	if githubToken != "" {
		return githubToken
	}

	// Allow env var override (useful for private repos / CI).
	for _, key := range []string{"LAUNCHER_GITHUB_TOKEN", "GITHUB_TOKEN", "GH_TOKEN"} {
		if v := strings.TrimSpace(os.Getenv(key)); v != "" {
			return v
		}
	}

	return ""
}

func GetCurrentVersion() string {
	if launcherVersion == "" || launcherVersion == "dev" {
		return getVersionFromGit()
	}
	return launcherVersion
}

func getVersionFromGit() string {
	gitVersionOnce.Do(func() {
		cmd := exec.Command("git", "describe", "--tags", "--abbrev=0")
		output, err := cmd.Output()
		if err != nil {
			cachedGitVersion = "dev"
			return
		}

		version := strings.TrimSpace(string(output))
		version = normalizeVersion(version)
		if version == "" {
			cachedGitVersion = "dev"
		} else {
			cachedGitVersion = version
		}
	})
	return cachedGitVersion
}

func CheckForUpdate(ctx context.Context) (bool, *ReleaseResponse, error) {
	req, err := http.NewRequestWithContext(ctx, "GET", githubAPIBase, nil)
	if err != nil {
		return false, nil, fmt.Errorf("failed to create request: %w", err)
	}

	token := getGitHubToken()

	req.Header.Set("User-Agent", githubUserAgent)
	req.Header.Set("Accept", "application/vnd.github.v3+json")
	req.Header.Set("X-GitHub-Api-Version", "2022-11-28")
	if token != "" {
		req.Header.Set("Authorization", "Bearer "+token)
	}

	resp, err := utils.HTTPClient.Do(req)
	if err != nil {
		return false, nil, fmt.Errorf("failed to check for updates: %w", err)
	}
	defer resp.Body.Close()

	if resp.StatusCode != 200 {
		body, _ := io.ReadAll(io.LimitReader(resp.Body, 4096))
		bodyMsg := strings.TrimSpace(string(body))
		if bodyMsg != "" {
			bodyMsg = ": " + bodyMsg
		}

		// Private repos return 404 for unauthenticated requests.
		if resp.StatusCode == http.StatusNotFound && token == "" {
			return false, nil, fmt.Errorf("GitHub API returned status %d (repo may be private; set github_token in config.yaml or LAUNCHER_GITHUB_TOKEN env var)%s", resp.StatusCode, bodyMsg)
		}

		return false, nil, fmt.Errorf("GitHub API returned status %d%s", resp.StatusCode, bodyMsg)
	}

	var release ReleaseResponse
	if err := json.NewDecoder(resp.Body).Decode(&release); err != nil {
		return false, nil, fmt.Errorf("failed to parse release info: %w", err)
	}

	currentVersion := GetCurrentVersion()

	currentVersion = normalizeVersion(currentVersion)
	latestVersion := normalizeVersion(release.TagName)

	if currentVersion == "" || latestVersion == "" {
		return false, nil, nil
	}

	comparison := compareVersions(latestVersion, currentVersion)
	if comparison <= 0 {
		return false, nil, nil
	}

	return true, &release, nil
}

func normalizeVersion(version string) string {
	version = strings.TrimPrefix(version, "v")
	version = strings.TrimSpace(version)
	return version
}

func compareVersions(v1, v2 string) int {
	var i1, i2 int
	l1, l2 := len(v1), len(v2)

	for i1 < l1 || i2 < l2 {
		var n1, n2 int
		var valid1, valid2 = true, true

		if i1 < l1 {
			for i1 < l1 {
				c := v1[i1]
				if c == '.' {
					i1++
					break
				}
				if c >= '0' && c <= '9' {
					n1 = n1*10 + int(c-'0')
				} else {
					valid1 = false
				}
				i1++
			}
		}
		if !valid1 {
			n1 = 0
		}

		if i2 < l2 {
			for i2 < l2 {
				c := v2[i2]
				if c == '.' {
					i2++
					break
				}
				if c >= '0' && c <= '9' {
					n2 = n2*10 + int(c-'0')
				} else {
					valid2 = false
				}
				i2++
			}
		}
		if !valid2 {
			n2 = 0
		}

		if n1 > n2 {
			return 1
		}
		if n1 < n2 {
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
		switch arch {
		case "amd64":
			assetName = "paper-launcher-windows-amd64.exe"
		case "arm64":
			assetName = "paper-launcher-windows-arm64.exe"
		}
	case "linux":
		switch arch {
		case "amd64":
			assetName = "paper-launcher-linux-amd64"
		case "arm64":
			assetName = "paper-launcher-linux-arm64"
		}
	case "darwin":
		switch arch {
		case "amd64":
			assetName = "paper-launcher-darwin-amd64"
		case "arm64":
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

func DownloadUpdate(ctx context.Context, release *ReleaseResponse) (string, error) {
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

	token := getGitHubToken()
	if token != "" && asset.URL != "" {
		if err := downloadGitHubReleaseAsset(ctx, asset.URL, token, tempFile); err != nil {
			return "", fmt.Errorf("failed to download update: %w", err)
		}
	} else {
		// Use shared DownloadFile utility (public repos)
		if err := utils.DownloadFile(ctx, asset.BrowserDownloadURL, tempFile); err != nil {
			return "", fmt.Errorf("failed to download update: %w", err)
		}
	}

	if runtime.GOOS != "windows" {
		if err := os.Chmod(tempFile, 0755); err != nil {
			return "", fmt.Errorf("failed to set executable permissions: %w", err)
		}
	}

	return tempFile, nil
}

func downloadGitHubReleaseAsset(ctx context.Context, assetAPIURL, token, filename string) error {
	req, err := http.NewRequestWithContext(ctx, "GET", assetAPIURL, nil)
	if err != nil {
		return fmt.Errorf("failed to create request: %w", err)
	}
	req.Header.Set("User-Agent", githubUserAgent)
	req.Header.Set("Accept", "application/octet-stream")
	req.Header.Set("X-GitHub-Api-Version", "2022-11-28")
	req.Header.Set("Authorization", "Bearer "+token)

	resp, err := utils.HTTPClient.Do(req)
	if err != nil {
		return fmt.Errorf("request failed: %w", err)
	}
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusOK {
		body, _ := io.ReadAll(io.LimitReader(resp.Body, 4096))
		bodyMsg := strings.TrimSpace(string(body))
		if bodyMsg != "" {
			bodyMsg = ": " + bodyMsg
		}
		return fmt.Errorf("download failed with status %d%s", resp.StatusCode, bodyMsg)
	}

	tempFile := filename + ".part"
	if _, err := os.Stat(tempFile); err == nil {
		_ = os.Remove(tempFile)
	}

	out, err := os.Create(tempFile)
	if err != nil {
		return fmt.Errorf("failed to create temp file: %w", err)
	}

	closed := false
	defer func() {
		if !closed {
			_ = out.Close()
			_ = os.Remove(tempFile)
		}
	}()

	if _, err := io.Copy(out, resp.Body); err != nil {
		return fmt.Errorf("failed to write file: %w", err)
	}

	if err := out.Close(); err != nil {
		return fmt.Errorf("failed to close file: %w", err)
	}
	closed = true

	// Windows에서 Rename이 실패할 수 있어 기존 파일은 제거
	if _, err := os.Stat(filename); err == nil {
		if err := os.Remove(filename); err != nil {
			return fmt.Errorf("failed to remove existing file: %w", err)
		}
	}

	if err := os.Rename(tempFile, filename); err != nil {
		return fmt.Errorf("failed to rename temp file: %w", err)
	}

	return nil
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
