package update

import (
	"context"
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"testing"

	"github.com/nevcea-sub/minecraft-server-launcher/internal/utils"
)

func withMockAPI(handler http.HandlerFunc, testVersion string, fn func()) {
	ts := httptest.NewServer(handler)
	defer ts.Close()

	originalAPIBase := githubAPIBase
	originalVersion := launcherVersion
	oldClient := utils.HTTPClient

	defer func() {
		githubAPIBase = originalAPIBase
		launcherVersion = originalVersion
		utils.HTTPClient = oldClient
	}()

	githubAPIBase = ts.URL
	utils.HTTPClient = ts.Client()

	if testVersion != "" {
		launcherVersion = testVersion
	}

	fn()
}

func TestCompareVersions(t *testing.T) {
	tests := []struct {
		v1, v2   string
		expected int
		desc     string
	}{
		{"1.0.0", "1.0.0", 0, "equal"},
		{"1.0.1", "1.0.0", 1, "newer patch"},
		{"1.0.0", "1.0.1", -1, "older patch"},
		{"2.0.0", "1.0.0", 1, "newer major"},
		{"1.0.0", "2.0.0", -1, "older major"},
		{"2.0.0", "1.9.9", 1, "major beats minor"},
		{"1.0", "1.0.0", 0, "different length same"},
		{"1.1", "1.0.0", 1, "different length newer"},
	}

	for _, tt := range tests {
		t.Run(tt.desc, func(t *testing.T) {
			if got := compareVersions(tt.v1, tt.v2); got != tt.expected {
				t.Errorf("compareVersions(%s, %s) = %d, want %d", tt.v1, tt.v2, got, tt.expected)
			}
		})
	}
}

func TestCheckForUpdate_NewerVersion(t *testing.T) {
	withMockAPI(func(w http.ResponseWriter, r *http.Request) {
		release := ReleaseResponse{TagName: "v2.0.0"}
		w.Header().Set("Content-Type", "application/json")
		json.NewEncoder(w).Encode(release)
	}, "1.0.0", func() {
		hasUpdate, release, err := CheckForUpdate(context.Background())
		if err != nil {
			t.Fatalf("unexpected error: %v", err)
		}
		if !hasUpdate {
			t.Error("expected update available")
		}
		if release == nil || release.TagName != "v2.0.0" {
			t.Errorf("unexpected release: %v", release)
		}
	})
}

func TestCheckForUpdate_OlderVersion(t *testing.T) {
	withMockAPI(func(w http.ResponseWriter, r *http.Request) {
		release := ReleaseResponse{TagName: "v0.2.0"}
		w.Header().Set("Content-Type", "application/json")
		json.NewEncoder(w).Encode(release)
	}, "1.0.0", func() {
		hasUpdate, release, err := CheckForUpdate(context.Background())
		if err != nil {
			t.Fatalf("unexpected error: %v", err)
		}
		if hasUpdate || release != nil {
			t.Error("expected no update for older version")
		}
	})
}
