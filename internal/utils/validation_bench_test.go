package utils

import (
	"archive/zip"
	"os"
	"path/filepath"
	"strings"
	"testing"
)

const (
	checksumFileName = "test.sha256"
	validChecksum    = "6ae8a75555209fd6c44157c0aed8016e763ff435a19cf186f76863140143ff72"
)

func BenchmarkLoadChecksumFile_Old(b *testing.B) {
	tmpDir := b.TempDir()
	checksumPath := filepath.Join(tmpDir, checksumFileName)

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

func BenchmarkLoadChecksumFile_Regex(b *testing.B) {
	tmpDir := b.TempDir()
	checksumPath := filepath.Join(tmpDir, checksumFileName)

	if err := os.WriteFile(checksumPath, []byte(validChecksum), 0644); err != nil {
		b.Fatal(err)
	}

	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		data, _ := os.ReadFile(checksumPath)
		checksum := strings.TrimSpace(string(data))
		_ = hexChecksumRegex.MatchString(checksum)
	}
}

func BenchmarkLoadChecksumFile_New(b *testing.B) {
	tmpDir := b.TempDir()
	checksumPath := filepath.Join(tmpDir, checksumFileName)

	if err := os.WriteFile(checksumPath, []byte(validChecksum), 0644); err != nil {
		b.Fatal(err)
	}

	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		LoadChecksumFile(checksumPath)
	}
}

func BenchmarkValidateJarAndCalculateChecksum(b *testing.B) {
	tmpDir := b.TempDir()
	jarPath := filepath.Join(tmpDir, "test.jar")
	createValidJarForBench(b, jarPath)

	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		_, err := ValidateJarAndCalculateChecksum(jarPath)
		if err != nil {
			b.Fatal(err)
		}
	}
}

func createValidJarForBench(b *testing.B, path string) {
	file, err := os.Create(path)
	if err != nil {
		b.Fatal(err)
	}
	defer file.Close()

	w := zip.NewWriter(file)
	defer w.Close()

	f, err := w.Create("META-INF/MANIFEST.MF")
	if err != nil {
		b.Fatal(err)
	}
	if _, err := f.Write([]byte("Manifest-Version: 1.0\n")); err != nil {
		b.Fatal(err)
	}

	f, err = w.Create("test.txt")
	if err != nil {
		b.Fatal(err)
	}
	content := make([]byte, 1024*1024)
	for i := range content {
		content[i] = byte(i % 256)
	}
	if _, err := f.Write(content); err != nil {
		b.Fatal(err)
	}
}
