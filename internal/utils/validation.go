package utils

import (
	"archive/zip"
	"crypto/sha256"
	"encoding/hex"
	"fmt"
	"io"
	"os"
	"strings"
)

const minJarSize = 22

func ValidateJarFile(jarPath string) error {
	info, err := os.Stat(jarPath)
	if err != nil {
		return fmt.Errorf("JAR file does not exist: %s", jarPath)
	}

	if info.Size() == 0 {
		return fmt.Errorf("JAR file is empty: %s", jarPath)
	}

	if info.Size() < minJarSize {
		return fmt.Errorf("JAR file is too small (%d bytes): %s", info.Size(), jarPath)
	}

	file, err := os.Open(jarPath)
	if err != nil {
		return fmt.Errorf("failed to open JAR: %w", err)
	}
	defer file.Close()

	magic := make([]byte, 2)
	if _, err := io.ReadFull(file, magic); err != nil {
		return fmt.Errorf("failed to read magic number: %w", err)
	}

	if magic[0] != 0x50 || magic[1] != 0x4B {
		return fmt.Errorf("invalid JAR file: missing ZIP magic number (expected PK, found %02X%02X)", magic[0], magic[1])
	}

	file.Seek(0, 0)
	reader, err := zip.NewReader(file, info.Size())
	if err != nil {
		return fmt.Errorf("failed to parse JAR as ZIP: %w", err)
	}

	if len(reader.File) == 0 {
		return fmt.Errorf("JAR file contains no entries")
	}

	hasManifest := false
	for _, f := range reader.File {
		if f.Name == "META-INF/MANIFEST.MF" {
			hasManifest = true
			break
		}
	}

	if !hasManifest {
		fmt.Printf("Warning: JAR file missing META-INF/MANIFEST.MF: %s\n", jarPath)
	}

	return nil
}

func ValidateJarAndCalculateChecksum(jarPath string) (string, error) {
	if err := ValidateJarFile(jarPath); err != nil {
		return "", err
	}

	file, err := os.Open(jarPath)
	if err != nil {
		return "", err
	}
	defer file.Close()

	h := sha256.New()
	if _, err := io.Copy(h, file); err != nil {
		return "", err
	}

	return hex.EncodeToString(h.Sum(nil)), nil
}

func ValidateChecksum(jarPath, expectedChecksum string) error {
	if expectedChecksum == "" {
		return nil
	}

	file, err := os.Open(jarPath)
	if err != nil {
		return err
	}
	defer file.Close()

	h := sha256.New()
	if _, err := io.Copy(h, file); err != nil {
		return err
	}

	actual := hex.EncodeToString(h.Sum(nil))
	expected := strings.TrimSpace(expectedChecksum)

	if !strings.EqualFold(actual, expected) {
		return fmt.Errorf("checksum mismatch:\nExpected: %s\nActual: %s", expected, actual)
	}

	return nil
}

func LoadChecksumFile(path string) (string, error) {
	data, err := os.ReadFile(path)
	if err != nil {
		if os.IsNotExist(err) {
			return "", nil
		}
		return "", err
	}

	checksum := strings.TrimSpace(string(data))
	if len(checksum) != 64 {
		return "", nil
	}

	return checksum, nil
}

func SaveChecksumFile(path, checksum string) error {
	return os.WriteFile(path, []byte(checksum), 0644)
}

