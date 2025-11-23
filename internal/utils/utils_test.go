package utils

import (
	"os"
	"path/filepath"
	"testing"
)

func TestFindJarFile(t *testing.T) {
	tmpDir := t.TempDir()
	oldDir, _ := os.Getwd()
	defer os.Chdir(oldDir)
	os.Chdir(tmpDir)

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
	oldDir, _ := os.Getwd()
	defer os.Chdir(oldDir)
	os.Chdir(tmpDir)

	if err := HandleEULA(); err != nil {
		t.Fatal(err)
	}

	data, err := os.ReadFile("eula.txt")
	if err != nil {
		t.Fatal(err)
	}

	content := string(data)
	if !contains(content, "eula=true") {
		t.Error("eula.txt should contain eula=true")
	}
}

func contains(s, substr string) bool {
	return len(s) >= len(substr) && 
		(s == substr || len(s) > len(substr) && anySubstring(s, substr))
}

func anySubstring(s, substr string) bool {
	for i := 0; i <= len(s)-len(substr); i++ {
		if s[i:i+len(substr)] == substr {
			return true
		}
	}
	return false
}
