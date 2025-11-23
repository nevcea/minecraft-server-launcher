package utils

import (
	"archive/zip"
	"os"
	"path/filepath"
	"testing"
)

func TestValidateJarFile(t *testing.T) {
	tmpDir := t.TempDir()

	err := ValidateJarFile(filepath.Join(tmpDir, "nonexistent.jar"))
	if err == nil {
		t.Error("expected error for nonexistent file")
	}

	emptyJar := filepath.Join(tmpDir, "empty.jar")
	os.WriteFile(emptyJar, []byte{}, 0644)
	err = ValidateJarFile(emptyJar)
	if err == nil {
		t.Error("expected error for empty file")
	}

	invalidJar := filepath.Join(tmpDir, "invalid.jar")
	os.WriteFile(invalidJar, []byte{0x00, 0x00, 0x00, 0x00, 0x00}, 0644)
	err = ValidateJarFile(invalidJar)
	if err == nil {
		t.Error("expected error for invalid magic number")
	}

	validJar := filepath.Join(tmpDir, "valid.jar")
	createValidJar(t, validJar)
	err = ValidateJarFile(validJar)
	if err != nil {
		t.Errorf("unexpected error for valid JAR: %v", err)
	}
}

func TestValidateJarAndCalculateChecksum(t *testing.T) {
	tmpDir := t.TempDir()
	jarPath := filepath.Join(tmpDir, "test.jar")
	createValidJar(t, jarPath)

	checksum, err := ValidateJarAndCalculateChecksum(jarPath)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}

	if len(checksum) != 64 {
		t.Errorf("expected 64-char checksum, got %d", len(checksum))
	}
}

func TestValidateChecksum(t *testing.T) {
	tmpDir := t.TempDir()
	testFile := filepath.Join(tmpDir, "test.txt")
	os.WriteFile(testFile, []byte("test content"), 0644)

	checksum, _ := ValidateJarAndCalculateChecksum(testFile)

	err := ValidateChecksum(testFile, checksum)
	if err != nil {
		t.Errorf("unexpected error: %v", err)
	}

	err = ValidateChecksum(testFile, "0000000000000000000000000000000000000000000000000000000000000000")
	if err == nil {
		t.Error("expected error for invalid checksum")
	}

	err = ValidateChecksum(testFile, "")
	if err != nil {
		t.Errorf("unexpected error for empty checksum: %v", err)
	}
}

func TestLoadAndSaveChecksumFile(t *testing.T) {
	tmpDir := t.TempDir()
	checksumPath := filepath.Join(tmpDir, "test.sha256")
	expectedChecksum := "6ae8a75555209fd6c44157c0aed8016e763ff435a19cf186f76863140143ff72"

	err := SaveChecksumFile(checksumPath, expectedChecksum)
	if err != nil {
		t.Fatalf("failed to save: %v", err)
	}

	loaded, err := LoadChecksumFile(checksumPath)
	if err != nil {
		t.Fatalf("failed to load: %v", err)
	}

	if loaded != expectedChecksum {
		t.Errorf("expected %s, got %s", expectedChecksum, loaded)
	}
}

func createValidJar(t *testing.T, path string) {
	file, err := os.Create(path)
	if err != nil {
		t.Fatal(err)
	}
	defer file.Close()

	w := zip.NewWriter(file)
	defer w.Close()

	f, err := w.Create("META-INF/MANIFEST.MF")
	if err != nil {
		t.Fatal(err)
	}
	f.Write([]byte("Manifest-Version: 1.0\n"))

	f, err = w.Create("test.txt")
	if err != nil {
		t.Fatal(err)
	}
	f.Write([]byte("test content"))
}
