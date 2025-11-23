package download

import (
	"fmt"
	"net/http"
	"net/http/httptest"
	"os"
	"path/filepath"
	"strings"
	"testing"
)

func TestGetLatestVersion(t *testing.T) {
	ts := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if r.URL.Path == "/" {
			fmt.Fprintln(w, `{"versions": ["1.19", "1.20", "1.21"]}`)
		} else {
			w.WriteHeader(http.StatusNotFound)
		}
	}))
	defer ts.Close()

	client := ts.Client()
	
	version, err := getLatestVersion(client, ts.URL)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if version != "1.21" {
		t.Errorf("expected 1.21, got %s", version)
	}
}

func TestGetLatestBuild(t *testing.T) {
	ts := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if strings.Contains(r.URL.Path, "/builds") {
			fmt.Fprintln(w, `{"builds": [{"build": 10}, {"build": 20}]}`)
		} else {
			w.WriteHeader(http.StatusNotFound)
		}
	}))
	defer ts.Close()

	client := ts.Client()
	build, err := getLatestBuild(client, ts.URL, "1.21")
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if build != 20 {
		t.Errorf("expected 20, got %d", build)
	}
}

func TestGetJarName(t *testing.T) {
	ts := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		fmt.Fprintln(w, `{"downloads": {"application": {"name": "paper-1.21-20.jar"}}}`)
	}))
	defer ts.Close()

	client := ts.Client()
	name, err := getJarName(client, ts.URL, "1.21", 20)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if name != "paper-1.21-20.jar" {
		t.Errorf("expected paper-1.21-20.jar, got %s", name)
	}
}

func TestDownloadFile(t *testing.T) {
	ts := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		fmt.Fprint(w, "file content")
	}))
	defer ts.Close()

	tmpDir := t.TempDir()
	filePath := filepath.Join(tmpDir, "test.jar")

	err := downloadFile(ts.Client(), ts.URL, filePath)
	if err != nil {
		t.Fatalf("download failed: %v", err)
	}

	content, err := os.ReadFile(filePath)
	if err != nil {
		t.Fatal(err)
	}
	if string(content) != "file content" {
		t.Errorf("content mismatch")
	}
}
