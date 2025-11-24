package update

import (
	"encoding/json"
	"fmt"
	"net/http"
	"net/http/httptest"
	"os"
	"path/filepath"
	"runtime"
	"testing"
)

func TestGetCurrentVersion(t *testing.T) {
	version := GetCurrentVersion()
	if version == "" {
		t.Error("expected non-empty version")
	}
	if version == "" {
		t.Error("version should not be empty")
	}
}

func TestNormalizeVersion(t *testing.T) {
	tests := []struct {
		input    string
		expected string
	}{
		{"v1.0.0", "1.0.0"},
		{"1.0.0", "1.0.0"},
		{" 1.0.0 ", "1.0.0"},
		{"v2.1.3", "2.1.3"},
		{"3.0", "3.0"},
	}

	for _, tt := range tests {
		t.Run(tt.input, func(t *testing.T) {
			result := normalizeVersion(tt.input)
			if result != tt.expected {
				t.Errorf("expected %s, got %s", tt.expected, result)
			}
		})
	}
}

func TestCompareVersions(t *testing.T) {
	tests := []struct {
		v1       string
		v2       string
		expected int
		desc     string
	}{
		{"1.0.0", "1.0.0", 0, "equal versions"},
		{"1.0.1", "1.0.0", 1, "v1 newer patch"},
		{"1.0.0", "1.0.1", -1, "v1 older patch"},
		{"1.1.0", "1.0.0", 1, "v1 newer minor"},
		{"1.0.0", "1.1.0", -1, "v1 older minor"},
		{"2.0.0", "1.0.0", 1, "v1 newer major"},
		{"1.0.0", "2.0.0", -1, "v1 older major"},
		{"1.2.3", "1.2.4", -1, "v1 older"},
		{"2.0.0", "1.9.9", 1, "v1 newer"},
		{"1.0", "1.0.0", 0, "different length, same"},
		{"1.1", "1.0.0", 1, "different length, v1 newer"},
	}

	for _, tt := range tests {
		t.Run(tt.desc, func(t *testing.T) {
			result := compareVersions(tt.v1, tt.v2)
			if result != tt.expected {
				t.Errorf("compareVersions(%s, %s) = %d, expected %d", tt.v1, tt.v2, result, tt.expected)
			}
		})
	}
}

func TestGetAssetForCurrentOS(t *testing.T) {
	release := &ReleaseResponse{
		Assets: []Asset{
			{Name: "paper-launcher-windows-amd64.exe"},
			{Name: "paper-launcher-linux-amd64"},
			{Name: "paper-launcher-darwin-amd64"},
			{Name: "paper-launcher-windows-arm64.exe"},
			{Name: "paper-launcher-linux-arm64"},
			{Name: "paper-launcher-darwin-arm64"},
		},
	}

	asset := getAssetForCurrentOS(release)
	if asset == nil {
		t.Skipf("no asset found for %s/%s, skipping test", runtime.GOOS, runtime.GOARCH)
	}

	expectedName := ""
	switch runtime.GOOS {
	case "windows":
		if runtime.GOARCH == "amd64" {
			expectedName = "paper-launcher-windows-amd64.exe"
		} else if runtime.GOARCH == "arm64" {
			expectedName = "paper-launcher-windows-arm64.exe"
		}
	case "linux":
		if runtime.GOARCH == "amd64" {
			expectedName = "paper-launcher-linux-amd64"
		} else if runtime.GOARCH == "arm64" {
			expectedName = "paper-launcher-linux-arm64"
		}
	case "darwin":
		if runtime.GOARCH == "amd64" {
			expectedName = "paper-launcher-darwin-amd64"
		} else if runtime.GOARCH == "arm64" {
			expectedName = "paper-launcher-darwin-arm64"
		}
	}

	if expectedName == "" {
		t.Skipf("unsupported platform %s/%s", runtime.GOOS, runtime.GOARCH)
	}

	if asset.Name != expectedName {
		t.Errorf("expected asset name %s, got %s", expectedName, asset.Name)
	}
}

func TestGetAssetForCurrentOS_NoMatch(t *testing.T) {
	release := &ReleaseResponse{
		Assets: []Asset{
			{Name: "paper-launcher-unknown-os"},
		},
	}

	asset := getAssetForCurrentOS(release)
	if asset != nil {
		t.Errorf("expected nil for unsupported OS, got %s", asset.Name)
	}
}

func TestValidateUpdate(t *testing.T) {
	tmpDir := t.TempDir()
	
	validFile := filepath.Join(tmpDir, "valid.bin")
	if err := os.WriteFile(validFile, []byte("test content"), 0644); err != nil {
		t.Fatal(err)
	}

	if err := ValidateUpdate(validFile); err != nil {
		t.Errorf("expected no error for valid file, got %v", err)
	}

	emptyFile := filepath.Join(tmpDir, "empty.bin")
	if err := os.WriteFile(emptyFile, []byte{}, 0644); err != nil {
		t.Fatal(err)
	}

	if err := ValidateUpdate(emptyFile); err == nil {
		t.Error("expected error for empty file")
	}

	nonexistentFile := filepath.Join(tmpDir, "nonexistent.bin")
	if err := ValidateUpdate(nonexistentFile); err == nil {
		t.Error("expected error for nonexistent file")
	}
}

func TestCheckForUpdate_NewerVersion(t *testing.T) {
	ts := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if r.Method != "GET" {
			w.WriteHeader(http.StatusMethodNotAllowed)
			return
		}

		release := ReleaseResponse{
			TagName: "v2.0.0",
			Name:    "Release 2.0.0",
			Body:    "Test release",
			Assets: []Asset{
				{Name: "paper-launcher-windows-amd64.exe", BrowserDownloadURL: "http://example.com/download"},
			},
		}

		w.Header().Set("Content-Type", "application/json")
		if err := json.NewEncoder(w).Encode(release); err != nil {
			t.Errorf("failed to encode response: %v", err)
		}
	}))
	defer ts.Close()

	originalAPIBase := githubAPIBase
	defer func() {
		githubAPIBase = originalAPIBase
	}()

	githubAPIBase = ts.URL

	hasUpdate, release, err := CheckForUpdate()
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}

	if !hasUpdate {
		t.Error("expected update available")
	}

	if release == nil {
		t.Fatal("expected release info")
	}

	if release.TagName != "v2.0.0" {
		t.Errorf("expected tag v2.0.0, got %s", release.TagName)
	}
}

func TestCheckForUpdate_OlderVersion(t *testing.T) {
	ts := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		release := ReleaseResponse{
			TagName: "v0.2.0",
			Name:    "Release 0.2.0",
			Body:    "Older release",
			Assets:  []Asset{},
		}

		w.Header().Set("Content-Type", "application/json")
		if err := json.NewEncoder(w).Encode(release); err != nil {
			t.Errorf("failed to encode response: %v", err)
		}
	}))
	defer ts.Close()

	originalAPIBase := githubAPIBase
	defer func() {
		githubAPIBase = originalAPIBase
	}()

	githubAPIBase = ts.URL

	hasUpdate, release, err := CheckForUpdate()
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}

	if hasUpdate {
		t.Error("expected no update available")
	}

	if release != nil {
		t.Error("expected nil release for older version")
	}
}

func TestCheckForUpdate_SameVersion(t *testing.T) {
	ts := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		release := ReleaseResponse{
			TagName: "0.4.0",
			Name:    "Release 0.4.0",
			Body:    "Current release",
			Assets:  []Asset{},
		}

		w.Header().Set("Content-Type", "application/json")
		if err := json.NewEncoder(w).Encode(release); err != nil {
			t.Errorf("failed to encode response: %v", err)
		}
	}))
	defer ts.Close()

	originalAPIBase := githubAPIBase
	originalVersion := launcherVersion
	defer func() {
		githubAPIBase = originalAPIBase
		launcherVersion = originalVersion
	}()

	githubAPIBase = ts.URL
	launcherVersion = "0.4.0"

	hasUpdate, release, err := CheckForUpdate()
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}

	if hasUpdate {
		t.Error("expected no update available for same version")
	}

	if release != nil {
		t.Error("expected nil release for same version")
	}
}

func TestCheckForUpdate_APIFailure(t *testing.T) {
	ts := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.WriteHeader(http.StatusInternalServerError)
	}))
	defer ts.Close()

	originalAPIBase := githubAPIBase
	defer func() {
		githubAPIBase = originalAPIBase
	}()

	githubAPIBase = ts.URL

	hasUpdate, release, err := CheckForUpdate()
	if err == nil {
		t.Error("expected error for API failure")
	}

	if hasUpdate {
		t.Error("expected no update on error")
	}

	if release != nil {
		t.Error("expected nil release on error")
	}
}

func TestCheckForUpdate_InvalidJSON(t *testing.T) {
	ts := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.Header().Set("Content-Type", "application/json")
		fmt.Fprint(w, "invalid json")
	}))
	defer ts.Close()

	originalAPIBase := githubAPIBase
	defer func() {
		githubAPIBase = originalAPIBase
	}()

	githubAPIBase = ts.URL

	hasUpdate, release, err := CheckForUpdate()
	if err == nil {
		t.Error("expected error for invalid JSON")
	}

	if hasUpdate {
		t.Error("expected no update on error")
	}

	if release != nil {
		t.Error("expected nil release on error")
	}
}

