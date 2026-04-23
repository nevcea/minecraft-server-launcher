package utils

import (
	"os"
	"path/filepath"
	"testing"
)

func TestFindJarFile(t *testing.T) {
	tmpDir := t.TempDir()
	oldDir, err := os.Getwd()
	if err != nil {
		t.Fatal(err)
	}
	defer os.Chdir(oldDir)
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

	if err := os.WriteFile(filepath.Join(tmpDir, "paper-1.21.1-100.jar"), []byte("test"), 0644); err != nil {
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
