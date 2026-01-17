package utils

import (
	"os"
	"path/filepath"
	"strings"
	"testing"
)

func TestFindJarFile(t *testing.T) {
	tmpDir := t.TempDir()
	oldDir, err := os.Getwd()
	if err != nil {
		t.Fatal(err)
	}
	defer func() {
		if err := os.Chdir(oldDir); err != nil {
			t.Errorf("failed to restore directory: %v", err)
		}
	}()
	if err := os.Chdir(tmpDir); err != nil {
		t.Fatal(err)
	}

	jar, err := FindJarFile()
	if err != nil {
		t.Fatal(err)
	}
	if jar != "" {
		t.Errorf("expected empty, got %s", jar)
	}

	jarPath := filepath.Join(tmpDir, "paper-1.21.1-100.jar")
	if err := os.WriteFile(jarPath, []byte("test"), 0644); err != nil {
		t.Fatal(err)
	}

	jar, err = FindJarFile()
	if err != nil {
		t.Fatal(err)
	}
	if jar != "paper-1.21.1-100.jar" {
		t.Errorf("expected paper-1.21.1-100.jar, got %s", jar)
	}
}

func TestHandleEULA(t *testing.T) {
	tmpDir := t.TempDir()
	oldDir, err := os.Getwd()
	if err != nil {
		t.Fatal(err)
	}
	defer func() {
		if err := os.Chdir(oldDir); err != nil {
			t.Errorf("failed to restore directory: %v", err)
		}
	}()
	if err := os.Chdir(tmpDir); err != nil {
		t.Fatal(err)
	}

	if err := HandleEULA(); err != nil {
		t.Fatal(err)
	}

	data, err := os.ReadFile("eula.txt")
	if err != nil {
		t.Fatal(err)
	}

	content := string(data)
	if !strings.Contains(content, "eula=true") {
		t.Error("eula.txt should contain eula=true")
	}
}
