package update

import (
	"context"
	"encoding/json"
	"fmt"
	"io"
	"net/http"
	"os"
	"os/exec"
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

var githubAPIBase = "https://api.github.com/repos/nevcea/minecraft-server-launcher/releases/latest"

type ReleaseResponse struct {
	TagName string `json:"tag_name"`
	Body    string `json:"body"`
}

func SetGitHubToken(token string) {
	githubToken = strings.TrimSpace(token)
}

func getGitHubToken() string {
	if githubToken != "" {
		return githubToken
	}
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
		out, err := exec.Command("git", "describe", "--tags", "--abbrev=0").Output()
		if err != nil {
			cachedGitVersion = "dev"
			return
		}
		v := normalizeVersion(strings.TrimSpace(string(out)))
		if v == "" {
			v = "dev"
		}
		cachedGitVersion = v
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

	if resp.StatusCode != http.StatusOK {
		body, _ := io.ReadAll(io.LimitReader(resp.Body, 4096))
		msg := strings.TrimSpace(string(body))
		if msg != "" {
			msg = ": " + msg
		}
		if resp.StatusCode == http.StatusNotFound && token == "" {
			return false, nil, fmt.Errorf("GitHub API returned 404 (private repo? set LAUNCHER_GITHUB_TOKEN env var)%s", msg)
		}
		return false, nil, fmt.Errorf("GitHub API returned status %d%s", resp.StatusCode, msg)
	}

	var release ReleaseResponse
	if err := json.NewDecoder(resp.Body).Decode(&release); err != nil {
		return false, nil, fmt.Errorf("failed to parse release info: %w", err)
	}

	current := normalizeVersion(GetCurrentVersion())
	latest := normalizeVersion(release.TagName)

	if current == "" || current == "dev" || latest == "" {
		return false, nil, nil
	}

	if compareVersions(latest, current) <= 0 {
		return false, nil, nil
	}

	return true, &release, nil
}

func normalizeVersion(version string) string {
	return strings.TrimSpace(strings.TrimPrefix(version, "v"))
}

func compareVersions(v1, v2 string) int {
	var i1, i2 int
	l1, l2 := len(v1), len(v2)

	for i1 < l1 || i2 < l2 {
		var n1, n2 int
		valid1, valid2 := true, true

		for i1 < l1 {
			c := v1[i1]
			i1++
			if c == '.' {
				break
			}
			if c >= '0' && c <= '9' {
				n1 = n1*10 + int(c-'0')
			} else {
				valid1 = false
			}
		}
		for i2 < l2 {
			c := v2[i2]
			i2++
			if c == '.' {
				break
			}
			if c >= '0' && c <= '9' {
				n2 = n2*10 + int(c-'0')
			} else {
				valid2 = false
			}
		}

		if !valid1 {
			n1 = 0
		}
		if !valid2 {
			n2 = 0
		}

		if n1 != n2 {
			if n1 > n2 {
				return 1
			}
			return -1
		}
	}

	return 0
}
