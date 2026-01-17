package download

import (
	"context"
	"encoding/json"
	"fmt"
	"os"
	"regexp"
	"strconv"

	"github.com/nevcea-sub/minecraft-server-launcher/internal/logger"
	"github.com/nevcea-sub/minecraft-server-launcher/internal/utils"
)

const (
	apiBase = "https://api.papermc.io/v2/projects/paper"
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

func CheckUpdate(ctx context.Context, jarName string) (bool, int, string, error) {
	matches := jarNameRegex.FindStringSubmatch(jarName)
	if len(matches) != 3 {
		return false, 0, "", fmt.Errorf("invalid jar filename format: %s", jarName)
	}

	version := matches[1]
	currentBuild, err := strconv.Atoi(matches[2])
	if err != nil {
		return false, 0, "", fmt.Errorf("invalid build number: %s", matches[2])
	}

	latestBuild, err := getLatestBuild(ctx, apiBase, version)
	if err != nil {
		return false, 0, "", fmt.Errorf("failed to get latest build: %w", err)
	}

	if latestBuild > currentBuild {
		newJarName, err := getJarName(ctx, apiBase, version, latestBuild)
		if err != nil {
			return true, latestBuild, "", fmt.Errorf("failed to get new jar name: %w", err)
		}
		return true, latestBuild, newJarName, nil
	}

	return false, 0, "", nil
}

func DownloadJar(ctx context.Context, version string) (string, error) {
	if version == "latest" {
		ver, err := getLatestVersion(ctx, apiBase)
		if err != nil {
			return "", err
		}
		version = ver
	}

	build, err := getLatestBuild(ctx, apiBase, version)
	if err != nil {
		return "", err
	}

	jarName, err := getJarName(ctx, apiBase, version, build)
	if err != nil {
		return "", err
	}

	if _, err := os.Stat(jarName); err == nil {
		logger.Info("JAR file already exists: %s", jarName)
		checksumFile := jarName + ".sha256"
		if expectedChecksum, err := utils.LoadChecksumFile(checksumFile); err == nil && expectedChecksum != "" {
			if err := utils.ValidateChecksum(jarName, expectedChecksum); err == nil {
				logger.Info("Existing JAR file checksum validated")
				return jarName, nil
			}
			logger.Info("Checksum validation failed, re-downloading...")
		} else {
			logger.Info("No checksum file found, re-downloading to ensure integrity...")
		}
	}

	url := fmt.Sprintf("%s/versions/%s/builds/%d/downloads/%s", apiBase, version, build, jarName)
	logger.Info("Downloading %s...", jarName)

	if err := utils.DownloadFile(ctx, url, jarName); err != nil {
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

	logger.Info("Downloaded and validated JAR file (SHA-256: %s)", checksum[:16]+"...")
	return jarName, nil
}

func getLatestVersion(ctx context.Context, baseURL string) (string, error) {
	resp, err := utils.DoRequest(ctx, nil, baseURL)
	if err != nil {
		return "", fmt.Errorf("failed to fetch versions: %w", err)
	}
	defer resp.Body.Close()

	var proj ProjectResponse
	if err := json.NewDecoder(resp.Body).Decode(&proj); err != nil {
		return "", fmt.Errorf("failed to parse response: %w", err)
	}

	if len(proj.Versions) == 0 {
		return "", fmt.Errorf("no versions found")
	}

	return proj.Versions[len(proj.Versions)-1], nil
}

func getLatestBuild(ctx context.Context, baseURL, version string) (int, error) {
	url := fmt.Sprintf("%s/versions/%s/builds", baseURL, version)
	resp, err := utils.DoRequest(ctx, nil, url)
	if err != nil {
		return 0, fmt.Errorf("failed to fetch builds: %w", err)
	}
	defer resp.Body.Close()

	var builds BuildsResponse
	if err := json.NewDecoder(resp.Body).Decode(&builds); err != nil {
		return 0, fmt.Errorf("failed to parse response: %w", err)
	}

	if len(builds.Builds) == 0 {
		return 0, fmt.Errorf("no builds found")
	}

	return builds.Builds[len(builds.Builds)-1].Build, nil
}

func getJarName(ctx context.Context, baseURL, version string, build int) (string, error) {
	url := fmt.Sprintf("%s/versions/%s/builds/%d", baseURL, version, build)
	resp, err := utils.DoRequest(ctx, nil, url)
	if err != nil {
		return "", fmt.Errorf("failed to fetch download info: %w", err)
	}
	defer resp.Body.Close()

	var download DownloadResponse
	if err := json.NewDecoder(resp.Body).Decode(&download); err != nil {
		return "", fmt.Errorf("failed to parse response: %w", err)
	}

	return download.Downloads.Application.Name, nil
}
