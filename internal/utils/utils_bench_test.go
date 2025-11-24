package utils

import (
	"fmt"
	"os"
	"path/filepath"
	"regexp"
	"testing"
)

const (
	jarFileCount = 10
	jarVersion   = "1.21.1"
	jarBuildBase = 100
)

func BenchmarkFindJarFile_Old(b *testing.B) {
	tmpDir := b.TempDir()
	oldDir, _ := os.Getwd()
	defer os.Chdir(oldDir)
	os.Chdir(tmpDir)

	for i := 0; i < jarFileCount; i++ {
		jarName := fmt.Sprintf("paper-%s-%d.jar", jarVersion, jarBuildBase+i)
		jarPath := filepath.Join(tmpDir, jarName)
		os.WriteFile(jarPath, []byte("test"), 0644)
	}

	testJarName := fmt.Sprintf("paper-%s-%d.jar", jarVersion, jarBuildBase)
	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		re := regexp.MustCompile(`paper-(.+)-(\d+)\.jar`)
		_ = re.FindStringSubmatch(testJarName)
	}
}

func BenchmarkFindJarFile_New(b *testing.B) {
	tmpDir := b.TempDir()
	oldDir, _ := os.Getwd()
	defer os.Chdir(oldDir)
	os.Chdir(tmpDir)

	for i := 0; i < jarFileCount; i++ {
		jarName := fmt.Sprintf("paper-%s-%d.jar", jarVersion, jarBuildBase+i)
		jarPath := filepath.Join(tmpDir, jarName)
		os.WriteFile(jarPath, []byte("test"), 0644)
	}

	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		FindJarFile()
	}
}

