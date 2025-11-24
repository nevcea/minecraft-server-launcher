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
)

const (
	minJarSize = 22
	zipMagicPK = 0x504B
)

var (
	hexChecksumRegex = regexp.MustCompile(`^[0-9a-fA-F]{64}$`)
)

func validateJarStructure(jarPath string) (*zip.Reader, error) {
	info, err := os.Stat(jarPath)
	if err != nil {
		return nil, fmt.Errorf("JAR file does not exist: %s", jarPath)
	}

	if info.Size() == 0 {
		return nil, fmt.Errorf("JAR file is empty: %s", jarPath)
	}

	if info.Size() < minJarSize {
		return nil, fmt.Errorf("JAR file is too small (%d bytes): %s", info.Size(), jarPath)
	}

	file, err := os.Open(jarPath)
	if err != nil {
		return nil, fmt.Errorf("failed to open JAR: %w", err)
	}
	defer func() {
		if err := file.Close(); err != nil {
			_ = err
		}
	}()

	magic := make([]byte, 2)
	if _, err := io.ReadFull(file, magic); err != nil {
		return nil, fmt.Errorf("failed to read magic number: %w", err)
	}

	expectedMagic := []byte{0x50, 0x4B}
	if magic[0] != expectedMagic[0] || magic[1] != expectedMagic[1] {
		return nil, fmt.Errorf("invalid JAR file: missing ZIP magic number (expected PK, found %02X%02X)", magic[0], magic[1])
	}

	if _, err := file.Seek(0, 0); err != nil {
		return nil, fmt.Errorf("failed to seek file: %w", err)
	}
	reader, err := zip.NewReader(file, info.Size())
	if err != nil {
		return nil, fmt.Errorf("failed to parse JAR as ZIP: %w", err)
	}

	if len(reader.File) == 0 {
		return nil, fmt.Errorf("JAR file contains no entries")
	}

	hasManifest := false
	for _, f := range reader.File {
		if f.Name == "META-INF/MANIFEST.MF" {
			hasManifest = true
			break
		}
	}

	if !hasManifest {
		fmt.Fprintf(os.Stderr, "[WARN] JAR file missing META-INF/MANIFEST.MF: %s\n", jarPath)
	}

	return reader, nil
}

func ValidateJarFile(jarPath string) error {
	_, err := validateJarStructure(jarPath)
	return err
}

func ValidateJarAndCalculateChecksum(jarPath string) (string, error) {
	if _, err := validateJarStructure(jarPath); err != nil {
		return "", err
	}

	file, err := os.Open(jarPath)
	if err != nil {
		return "", fmt.Errorf("failed to open JAR for checksum: %w", err)
	}
	defer func() {
		if err := file.Close(); err != nil {
			_ = err
		}
	}()

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

	for i := 0; i < len(data); i++ {
		c := data[i]
		if !((c >= '0' && c <= '9') || (c >= 'a' && c <= 'f') || (c >= 'A' && c <= 'F')) {
			return "", fmt.Errorf("invalid checksum format: contains non-hexadecimal characters")
		}
	}

	return string(data), nil
}

func SaveChecksumFile(path, checksum string) error {
	return os.WriteFile(path, []byte(checksum), 0644)
}
