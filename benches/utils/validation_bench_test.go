package utils

import (
	"os"
	"path/filepath"
	"testing"

	"github.com/nevcea-sub/minecraft-server-launcher/internal/utils"
)

func BenchmarkLoadChecksumFile_Old(b *testing.B) {
	tmpDir := b.TempDir()
	checksumPath := filepath.Join(tmpDir, "test.sha256")
	validChecksum := "6ae8a75555209fd6c44157c0aed8016e763ff435a19cf186f76863140143ff72"
	
	if err := os.WriteFile(checksumPath, []byte(validChecksum), 0644); err != nil {
		b.Fatal(err)
	}

	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		data, _ := os.ReadFile(checksumPath)
		checksum := string(data)
		for _, c := range checksum {
			_ = (c >= '0' && c <= '9') || (c >= 'a' && c <= 'f') || (c >= 'A' && c <= 'F')
		}
	}
}

func BenchmarkLoadChecksumFile_New(b *testing.B) {
	tmpDir := b.TempDir()
	checksumPath := filepath.Join(tmpDir, "test.sha256")
	validChecksum := "6ae8a75555209fd6c44157c0aed8016e763ff435a19cf186f76863140143ff72"
	
	if err := os.WriteFile(checksumPath, []byte(validChecksum), 0644); err != nil {
		b.Fatal(err)
	}

	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		utils.LoadChecksumFile(checksumPath)
	}
}

