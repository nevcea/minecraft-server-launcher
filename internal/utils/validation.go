package utils

import (
	"archive/zip"
	"bytes"
	"crypto/sha256"
	"encoding/hex"
	"fmt"
	"io"
	"os"
	"regexp"
	"strings"

	"github.com/nevcea-sub/minecraft-server-launcher/internal/logger"
)

const (
	minJarSize      = 22
	checksumBufSize = 32 * 1024
)

var (
	hexChecksumRegex = regexp.MustCompile(`^[0-9a-fA-F]{64}$`)
)

func validateJarStructureFromFile(jarPath string, file *os.File, size int64) error {
	if size == 0 {
		return fmt.Errorf("JAR file is empty: %s", jarPath)
	}

	if size < minJarSize {
		return fmt.Errorf("JAR file is too small (%d bytes): %s", size, jarPath)
	}

	magic := make([]byte, 2)
	if _, err := file.ReadAt(magic, 0); err != nil {
		return fmt.Errorf("failed to read magic number: %w", err)
	}

	if magic[0] != 0x50 || magic[1] != 0x4B {
		return fmt.Errorf("invalid JAR file: missing ZIP magic number (expected PK, found %02X%02X)", magic[0], magic[1])
	}

	reader, err := zip.NewReader(file, size)
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
		logger.Warn("JAR file missing META-INF/MANIFEST.MF: %s", jarPath)
	}

	return nil
}

func validateJarStructure(jarPath string) error {
	info, err := os.Stat(jarPath)
	if err != nil {
		return fmt.Errorf("failed to stat JAR file %q: %w", jarPath, err)
	}

	if info.IsDir() {
		return fmt.Errorf("JAR path is a directory: %s", jarPath)
	}

	file, err := os.Open(jarPath)
	if err != nil {
		return fmt.Errorf("failed to open JAR: %w", err)
	}
	defer file.Close()

	return validateJarStructureFromFile(jarPath, file, info.Size())
}

func ValidateJarFile(jarPath string) error {
	return validateJarStructure(jarPath)
}

func ValidateJarAndCalculateChecksum(jarPath string) (string, error) {
	info, err := os.Stat(jarPath)
	if err != nil {
		return "", fmt.Errorf("failed to stat JAR file %q: %w", jarPath, err)
	}

	if info.IsDir() {
		return "", fmt.Errorf("JAR path is a directory: %s", jarPath)
	}

	file, err := os.Open(jarPath)
	if err != nil {
		return "", fmt.Errorf("failed to open JAR: %w", err)
	}
	defer file.Close()

	if err := validateJarStructureFromFile(jarPath, file, info.Size()); err != nil {
		return "", err
	}

	if _, err := file.Seek(0, io.SeekStart); err != nil {
		return "", fmt.Errorf("failed to seek file: %w", err)
	}

	h := sha256.New()
	buf := make([]byte, checksumBufSize)
	if _, err := io.CopyBuffer(h, file, buf); err != nil {
		return "", fmt.Errorf("failed to calculate checksum: %w", err)
	}

	return hex.EncodeToString(h.Sum(nil)), nil
}

func ValidateChecksum(jarPath, expectedChecksum string) error {
	if expectedChecksum == "" {
		return nil
	}

	actual, err := ValidateJarAndCalculateChecksum(jarPath)
	if err != nil {
		return err
	}

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
		return "", fmt.Errorf("failed to read checksum file: %w", err)
	}

	data = bytes.TrimSpace(data)

	const expectedChecksumLength = 64
	if len(data) != expectedChecksumLength {
		return "", fmt.Errorf("invalid checksum format: expected %d characters, got %d", expectedChecksumLength, len(data))
	}

	if !hexChecksumRegex.Match(data) {
		return "", fmt.Errorf("invalid checksum format: contains non-hexadecimal characters")
	}

	return string(data), nil
}

func SaveChecksumFile(path, checksum string) error {
	return os.WriteFile(path, []byte(checksum), 0644)
}
