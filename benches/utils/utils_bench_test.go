package utils

import (
	"os"
	"path/filepath"
	"regexp"
	"testing"

	"github.com/nevcea-sub/minecraft-server-launcher/internal/utils"
)

func BenchmarkFindJarFile_Old(b *testing.B) {
	tmpDir := b.TempDir()
	oldDir, _ := os.Getwd()
	defer os.Chdir(oldDir)
	os.Chdir(tmpDir)

	for i := 1; i <= 10; i++ {
		jarPath := filepath.Join(tmpDir, "paper-1.21.1-100.jar")
		os.WriteFile(jarPath, []byte("test"), 0644)
	}

	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		re := regexp.MustCompile(`paper-(.+)-(\d+)\.jar`)
		_ = re.FindStringSubmatch("paper-1.21.1-100.jar")
	}
}

func BenchmarkFindJarFile_New(b *testing.B) {
	tmpDir := b.TempDir()
	oldDir, _ := os.Getwd()
	defer os.Chdir(oldDir)
	os.Chdir(tmpDir)

	for i := 1; i <= 10; i++ {
		jarPath := filepath.Join(tmpDir, "paper-1.21.1-100.jar")
		os.WriteFile(jarPath, []byte("test"), 0644)
	}

	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		utils.FindJarFile()
	}
}

