package utils

import (
	"archive/zip"
	"os"
	"path/filepath"
	"testing"
)

func TestValidateJarFile(t *testing.T) {
	tmpDir := t.TempDir()

	if err := ValidateJarFile(filepath.Join(tmpDir, "nonexistent.jar")); err == nil {
		t.Error("expected error for nonexistent file")
	}

	emptyJar := filepath.Join(tmpDir, "empty.jar")
	os.WriteFile(emptyJar, []byte{}, 0644)
	if err := ValidateJarFile(emptyJar); err == nil {
		t.Error("expected error for empty file")
	}

	invalidJar := filepath.Join(tmpDir, "invalid.jar")
	os.WriteFile(invalidJar, []byte{0x00, 0x00, 0x00, 0x00, 0x00}, 0644)
	if err := ValidateJarFile(invalidJar); err == nil {
		t.Error("expected error for invalid magic number")
	}

	validJar := filepath.Join(tmpDir, "valid.jar")
	createValidJar(t, validJar)
	if err := ValidateJarFile(validJar); err != nil {
		t.Errorf("unexpected error for valid JAR: %v", err)
	}
}

func TestValidateChecksum(t *testing.T) {
	tmpDir := t.TempDir()
	jarPath := filepath.Join(tmpDir, "test.jar")
	createValidJar(t, jarPath)

	checksum, err := ValidateJarAndCalculateChecksum(jarPath)
	if err != nil {
		t.Fatal(err)
	}

	if err := ValidateChecksum(jarPath, checksum); err != nil {
		t.Errorf("unexpected error for correct checksum: %v", err)
	}
	if err := ValidateChecksum(jarPath, "0000000000000000000000000000000000000000000000000000000000000000"); err == nil {
		t.Error("expected error for wrong checksum")
	}
	if err := ValidateChecksum(jarPath, ""); err != nil {
		t.Errorf("unexpected error for empty checksum: %v", err)
	}
}

func createValidJar(t *testing.T, path string) {
	t.Helper()
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
