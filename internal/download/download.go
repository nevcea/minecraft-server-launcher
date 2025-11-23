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
	apiBase     = "https://api.papermc.io/v2/projects/paper"
	timeout     = 30 * time.Second
	copyBufSize = 128 * 1024
	userAgent   = "minecraft-server-launcher/1.0"
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

func CheckUpdate(jarName string) (bool, int, string, error) {
	re := regexp.MustCompile(`paper-(.+)-(\d+)\.jar`)
	matches := re.FindStringSubmatch(jarName)
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

	tempFile := jarName + ".part"
	if _, err := os.Stat(tempFile); err == nil {
		fmt.Printf("Found incomplete download %s, removing...\n", tempFile)
		os.Remove(tempFile)
	}

	if _, err := os.Stat(jarName); err == nil {
		fmt.Printf("JAR file already exists: %s\n", jarName)
		return jarName, nil
	}

	url := fmt.Sprintf("%s/versions/%s/builds/%d/downloads/%s", apiBase, version, build, jarName)
	fmt.Printf("Downloading from: %s\n", url)

	if err := downloadFile(client, url, jarName); err != nil {
		return "", err
	}

	checksum, err := utils.ValidateJarAndCalculateChecksum(jarName)
	if err != nil {
		return "", fmt.Errorf("JAR validation failed: %w", err)
	}

	checksumFile := jarName + ".sha256"
	if err := utils.SaveChecksumFile(checksumFile, checksum); err != nil {
		return "", fmt.Errorf("failed to save checksum: %w", err)
	}

	fmt.Printf("Downloaded JAR file checksum (SHA-256): %s\n", checksum)
	fmt.Printf("Validated downloaded JAR file: %s\n", jarName)
	return jarName, nil
}

func doRequest(client *http.Client, url string) (*http.Response, error) {
	req, err := http.NewRequest("GET", url, nil)
	if err != nil {
		return nil, err
	}
	req.Header.Set("User-Agent", userAgent)
	return client.Do(req)
}

func getLatestVersion(client *http.Client, baseURL string) (string, error) {
	resp, err := doRequest(client, baseURL)
	if err != nil {
		return "", fmt.Errorf("failed to fetch versions: %w", err)
	}
	defer resp.Body.Close()

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
	defer resp.Body.Close()

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
	defer resp.Body.Close()

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
	defer resp.Body.Close()

	if resp.StatusCode != 200 {
		return fmt.Errorf("download failed with status %d", resp.StatusCode)
	}

	tempFile := filename + ".part"
	
	if _, err := os.Stat(tempFile); err == nil {
		os.Remove(tempFile)
	}

	out, err := os.Create(tempFile)
	if err != nil {
		return fmt.Errorf("failed to create temp file: %w", err)
	}

	var closed bool
	defer func() {
		if !closed {
			out.Close()
		}
	}()

	success := false
	defer func() {
		if !success {
			if !closed {
				out.Close()
				closed = true
			}
			os.Remove(tempFile)
		}
	}()

	bar := progressbar.DefaultBytes(
		resp.ContentLength,
		"Downloading",
	)

	buf := make([]byte, copyBufSize)
	_, err = io.CopyBuffer(io.MultiWriter(out, bar), resp.Body, buf)
	if err != nil {
		return fmt.Errorf("failed to write file: %w", err)
	}

	out.Close()
	closed = true

	success = true
	
	if _, err := os.Stat(filename); err == nil {
		os.Remove(filename)
	}

	if err := os.Rename(tempFile, filename); err != nil {
		return fmt.Errorf("failed to rename temp file: %w", err)
	}

	fmt.Println("\nDownload complete!")
	return nil
}
