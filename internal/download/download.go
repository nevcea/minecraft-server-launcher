package download

import (
	"encoding/json"
	"fmt"
	"io"
	"net/http"
	"os"
	"regexp"
	"strconv"
	"time"

	"github.com/nevcea-sub/minecraft-server-launcher/internal/utils"
	"github.com/schollz/progressbar/v3"
)

const (
	apiBase      = "https://api.papermc.io/v2/projects/paper"
	timeout      = 30 * time.Second
	copyBufSize  = 128 * 1024
	userAgent    = "minecraft-server-launcher/1.0"
	maxRetries   = 3
	retryDelay   = 2 * time.Second
	retryBackoff = 2.0
)

type ProjectResponse struct {
	Versions []string `json:"versions"`
}

type BuildsResponse struct {
	Builds []struct {
		Build int `json:"build"`
	} `json:"builds"`
}

type DownloadResponse struct {
	Downloads struct {
		Application struct {
			Name string `json:"name"`
		} `json:"application"`
	} `json:"downloads"`
}

var (
	jarNameRegex = regexp.MustCompile(`paper-(.+)-(\d+)\.jar`)
)

func CheckUpdate(jarName string) (bool, int, string, error) {
	matches := jarNameRegex.FindStringSubmatch(jarName)
	if len(matches) != 3 {
		return false, 0, "", fmt.Errorf("invalid jar filename format: %s", jarName)
	}

	version := matches[1]
	currentBuild, err := strconv.Atoi(matches[2])
	if err != nil {
		return false, 0, "", fmt.Errorf("invalid build number: %s", matches[2])
	}

	client := &http.Client{Timeout: timeout}
	latestBuild, err := getLatestBuild(client, apiBase, version)
	if err != nil {
		return false, 0, "", fmt.Errorf("failed to get latest build: %w", err)
	}

	if latestBuild > currentBuild {
		newJarName, err := getJarName(client, apiBase, version, latestBuild)
		if err != nil {
			return true, latestBuild, "", fmt.Errorf("failed to get new jar name: %w", err)
		}
		return true, latestBuild, newJarName, nil
	}

	return false, 0, "", nil
}

func DownloadJar(version string) (string, error) {
	client := &http.Client{Timeout: timeout}

	if version == "latest" {
		ver, err := getLatestVersion(client, apiBase)
		if err != nil {
			return "", err
		}
		version = ver
	}

	build, err := getLatestBuild(client, apiBase, version)
	if err != nil {
		return "", err
	}

	jarName, err := getJarName(client, apiBase, version, build)
	if err != nil {
		return "", err
	}

	if _, err := os.Stat(jarName); err == nil {
		fmt.Printf("[INFO] JAR file already exists: %s\n", jarName)
		checksumFile := jarName + ".sha256"
		if expectedChecksum, err := utils.LoadChecksumFile(checksumFile); err == nil && expectedChecksum != "" {
			if err := utils.ValidateChecksum(jarName, expectedChecksum); err == nil {
				fmt.Printf("[OK] Existing JAR file checksum validated\n")
				return jarName, nil
			}
		}
		fmt.Printf("[INFO] Re-downloading to ensure integrity...\n")
	}

	tempFile := jarName + ".part"
	if _, err := os.Stat(tempFile); err == nil {
		fmt.Printf("[INFO] Found incomplete download, removing...\n")
		if err := os.Remove(tempFile); err != nil {
			fmt.Fprintf(os.Stderr, "[WARN] Failed to remove incomplete download: %v\n", err)
		}
	}

	url := fmt.Sprintf("%s/versions/%s/builds/%d/downloads/%s", apiBase, version, build, jarName)
	fmt.Printf("[INFO] Downloading %s...\n", jarName)

	if err := downloadFile(client, url, jarName); err != nil {
		return "", err
	}

	checksum, err := utils.ValidateJarAndCalculateChecksum(jarName)
	if err != nil {
		return "", fmt.Errorf("JAR validation failed: %w", err)
	}

	checksumFile := jarName + ".sha256"
	if err := utils.SaveChecksumFile(checksumFile, checksum); err != nil {
		return "", fmt.Errorf("failed to save checksum file: %w", err)
	}

	fmt.Printf("[OK] Downloaded and validated JAR file (SHA-256: %s)\n", checksum[:16]+"...")
	return jarName, nil
}

func doRequest(client *http.Client, url string) (*http.Response, error) {
	var lastErr error
	delay := retryDelay

	for attempt := 0; attempt < maxRetries; attempt++ {
		if attempt > 0 {
			time.Sleep(delay)
			delay = time.Duration(float64(delay) * retryBackoff)
		}

		req, err := http.NewRequest("GET", url, nil)
		if err != nil {
			return nil, fmt.Errorf("failed to create request: %w", err)
		}
		req.Header.Set("User-Agent", userAgent)

		resp, err := client.Do(req)
		if err == nil && resp.StatusCode == 200 {
			return resp, nil
		}

		if resp != nil {
			if err := resp.Body.Close(); err != nil {
				// Ignore close error in retry loop
				_ = err
			}
		}

		if err != nil {
			lastErr = fmt.Errorf("request failed: %w", err)
		} else {
			lastErr = fmt.Errorf("API returned status %d", resp.StatusCode)
		}

		if attempt < maxRetries-1 {
			fmt.Fprintf(os.Stderr, "[WARN] Request failed (attempt %d/%d), retrying...\n", attempt+1, maxRetries)
		}
	}

	return nil, fmt.Errorf("request failed after %d attempts: %w", maxRetries, lastErr)
}

func getLatestVersion(client *http.Client, baseURL string) (string, error) {
	resp, err := doRequest(client, baseURL)
	if err != nil {
		return "", fmt.Errorf("failed to fetch versions: %w", err)
	}
	defer func() {
		if err := resp.Body.Close(); err != nil {
			// Ignore close error in defer
			_ = err
		}
	}()

	if resp.StatusCode != 200 {
		return "", fmt.Errorf("API returned status %d", resp.StatusCode)
	}

	var proj ProjectResponse
	if err := json.NewDecoder(resp.Body).Decode(&proj); err != nil {
		return "", fmt.Errorf("failed to parse response: %w", err)
	}

	if len(proj.Versions) == 0 {
		return "", fmt.Errorf("no versions found")
	}

	return proj.Versions[len(proj.Versions)-1], nil
}

func getLatestBuild(client *http.Client, baseURL, version string) (int, error) {
	url := fmt.Sprintf("%s/versions/%s/builds", baseURL, version)
	resp, err := doRequest(client, url)
	if err != nil {
		return 0, fmt.Errorf("failed to fetch builds: %w", err)
	}
	defer func() {
		if err := resp.Body.Close(); err != nil {
			// Ignore close error in defer
			_ = err
		}
	}()

	if resp.StatusCode != 200 {
		return 0, fmt.Errorf("API returned status %d", resp.StatusCode)
	}

	var builds BuildsResponse
	if err := json.NewDecoder(resp.Body).Decode(&builds); err != nil {
		return 0, fmt.Errorf("failed to parse response: %w", err)
	}

	if len(builds.Builds) == 0 {
		return 0, fmt.Errorf("no builds found")
	}

	return builds.Builds[len(builds.Builds)-1].Build, nil
}

func getJarName(client *http.Client, baseURL, version string, build int) (string, error) {
	url := fmt.Sprintf("%s/versions/%s/builds/%d", baseURL, version, build)
	resp, err := doRequest(client, url)
	if err != nil {
		return "", fmt.Errorf("failed to fetch download info: %w", err)
	}
	defer func() {
		if err := resp.Body.Close(); err != nil {
			// Ignore close error in defer
			_ = err
		}
	}()

	if resp.StatusCode != 200 {
		return "", fmt.Errorf("API returned status %d", resp.StatusCode)
	}

	var download DownloadResponse
	if err := json.NewDecoder(resp.Body).Decode(&download); err != nil {
		return "", fmt.Errorf("failed to parse response: %w", err)
	}

	return download.Downloads.Application.Name, nil
}

func downloadFile(client *http.Client, url, filename string) error {
	req, err := http.NewRequest("GET", url, nil)
	if err != nil {
		return fmt.Errorf("failed to create request: %w", err)
	}
	req.Header.Set("User-Agent", userAgent)

	resp, err := client.Do(req)
	if err != nil {
		return fmt.Errorf("failed to download: %w", err)
	}
	defer func() {
		if err := resp.Body.Close(); err != nil {
			// Ignore close error in defer
			_ = err
		}
	}()

	if resp.StatusCode != 200 {
		return fmt.Errorf("download failed with status %d", resp.StatusCode)
	}

	tempFile := filename + ".part"

	if _, err := os.Stat(tempFile); err == nil {
		if err := os.Remove(tempFile); err != nil {
			return fmt.Errorf("failed to remove existing temp file: %w", err)
		}
	}

	out, err := os.Create(tempFile)
	if err != nil {
		return fmt.Errorf("failed to create temp file: %w", err)
	}

	var closed bool
	defer func() {
		if !closed {
			if err := out.Close(); err != nil {
				// Ignore close error in defer
				_ = err
			}
		}
	}()

	success := false
	defer func() {
		if !success {
			if !closed {
				if err := out.Close(); err != nil {
					// Ignore close error in cleanup
					_ = err
				}
				closed = true
			}
			if err := os.Remove(tempFile); err != nil {
				// Ignore remove error in cleanup
				_ = err
			}
		}
	}()

	var bar *progressbar.ProgressBar
	if resp.ContentLength > 0 {
		bar = progressbar.DefaultBytes(
			resp.ContentLength,
			"Downloading",
		)
	} else {
		bar = progressbar.DefaultBytes(-1, "Downloading")
	}

	buf := make([]byte, copyBufSize)
	var writer io.Writer = out
	if bar != nil {
		writer = io.MultiWriter(out, bar)
	}
	_, err = io.CopyBuffer(writer, resp.Body, buf)
	if err != nil {
		return fmt.Errorf("failed to write file: %w", err)
	}

	if err := out.Close(); err != nil {
		return fmt.Errorf("failed to close file: %w", err)
	}
	closed = true

	success = true

	if _, err := os.Stat(filename); err == nil {
		if err := os.Remove(filename); err != nil {
			return fmt.Errorf("failed to remove existing file: %w", err)
		}
	}

	if err := os.Rename(tempFile, filename); err != nil {
		return fmt.Errorf("failed to rename temp file: %w", err)
	}

	fmt.Println("[OK] Download complete!")
	return nil
}
