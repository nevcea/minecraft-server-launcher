package utils

import (
	"fmt"
	"os"
	"path/filepath"
	"testing"
)

func BenchmarkFindJarFile(b *testing.B) {
	tmpDir := b.TempDir()
	oldDir, _ := os.Getwd()
	defer os.Chdir(oldDir)
	os.Chdir(tmpDir)

	for i := 0; i < 10; i++ {
		jarPath := filepath.Join(tmpDir, fmt.Sprintf("paper-1.21.1-%d.jar", 100+i))
		os.WriteFile(jarPath, []byte("test"), 0644)
	}

	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		FindJarFile()
	}
}
