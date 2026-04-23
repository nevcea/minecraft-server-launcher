package utils

import (
	"archive/zip"
	"os"
	"path/filepath"
	"testing"
)

func BenchmarkLoadChecksumFile(b *testing.B) {
	tmpDir := b.TempDir()
	checksumPath := filepath.Join(tmpDir, "test.sha256")
	validChecksum := "6ae8a75555209fd6c44157c0aed8016e763ff435a19cf186f76863140143ff72"

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
	createBenchJar(b, jarPath)

	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		if _, err := ValidateJarAndCalculateChecksum(jarPath); err != nil {
			b.Fatal(err)
		}
	}
}

func createBenchJar(b *testing.B, path string) {
	b.Helper()
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
