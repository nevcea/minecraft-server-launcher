package download

import (
	"context"
	"fmt"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"

	"github.com/nevcea-sub/minecraft-server-launcher/internal/utils"
)

func withMockClient(client *http.Client, fn func()) {
	oldClient := utils.HTTPClient
	utils.HTTPClient = client
	defer func() { utils.HTTPClient = oldClient }()
	fn()
}

func TestGetLatestVersion(t *testing.T) {
	ts := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if r.URL.Path == "/" {
			if _, err := fmt.Fprintln(w, `{"versions": ["1.19", "1.20", "1.21"]}`); err != nil {
				_ = err
			}
		} else {
			w.WriteHeader(http.StatusNotFound)
		}
	}))
	defer ts.Close()

	withMockClient(ts.Client(), func() {
		version, err := getLatestVersion(context.Background(), ts.URL)
		if err != nil {
			t.Fatalf("unexpected error: %v", err)
		}
		if version != "1.21" {
			t.Errorf("expected 1.21, got %s", version)
		}
	})
}

func TestGetLatestBuild(t *testing.T) {
	ts := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if strings.Contains(r.URL.Path, "/builds") {
			if _, err := fmt.Fprintln(w, `{"builds": [{"build": 10}, {"build": 20}]}`); err != nil {
				_ = err
			}
		} else {
			w.WriteHeader(http.StatusNotFound)
		}
	}))
	defer ts.Close()

	withMockClient(ts.Client(), func() {
		build, err := getLatestBuild(context.Background(), ts.URL, "1.21")
		if err != nil {
			t.Fatalf("unexpected error: %v", err)
		}
		if build != 20 {
			t.Errorf("expected 20, got %d", build)
		}
	})
}

func TestGetJarName(t *testing.T) {
	ts := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if _, err := fmt.Fprintln(w, `{"downloads": {"application": {"name": "paper-1.21-20.jar"}}}`); err != nil {
			_ = err
		}
	}))
	defer ts.Close()

	withMockClient(ts.Client(), func() {
		name, err := getJarName(context.Background(), ts.URL, "1.21", 20)
		if err != nil {
			t.Fatalf("unexpected error: %v", err)
		}
		if name != "paper-1.21-20.jar" {
			t.Errorf("expected paper-1.21-20.jar, got %s", name)
		}
	})
}
