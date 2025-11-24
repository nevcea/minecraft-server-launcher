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

const minJarSize = 22

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
			// Ignore close error in validation
			_ = err
		}
	}()

	magic := make([]byte, 2)
	if _, err := io.ReadFull(file, magic); err != nil {
		return nil, fmt.Errorf("failed to read magic number: %w", err)
	}

	if magic[0] != 0x50 || magic[1] != 0x4B {
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
	// 공통 검증 로직 사용
	_, err := validateJarStructure(jarPath)
	if err != nil {
		return "", err
	}

	// 체크섬 계산을 위해 파일을 다시 열기
	file, err := os.Open(jarPath)
	if err != nil {
		return "", fmt.Errorf("failed to open JAR file: %w", err)
	}
	defer func() {
		if err := file.Close(); err != nil {
			// Ignore close error in validation
			_ = err
		}
	}()

	h := sha256.New()
	if _, err := io.Copy(h, file); err != nil {
		return "", fmt.Errorf("failed to calculate checksum: %w", err)
	}

	return hex.EncodeToString(h.Sum(nil)), nil
}

func ValidateChecksum(jarPath, expectedChecksum string) error {
	if expectedChecksum == "" {
		return nil
	}

	file, err := os.Open(jarPath)
	if err != nil {
		return fmt.Errorf("failed to open JAR file: %w", err)
	}
	defer func() {
		if err := file.Close(); err != nil {
			// Ignore close error in validation
			_ = err
		}
	}()

	h := sha256.New()
	if _, err := io.Copy(h, file); err != nil {
		return fmt.Errorf("failed to calculate checksum: %w", err)
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
		return "", fmt.Errorf("failed to read checksum file: %w", err)
	}

	// 공백 제거 (앞뒤 공백만 제거, 더 효율적)
	data = bytes.TrimSpace(data)
	
	// 길이 체크를 먼저 수행하여 조기 반환
	if len(data) != 64 {
		return "", fmt.Errorf("invalid checksum format: expected 64 characters, got %d", len(data))
	}

	// 정규표현식 대신 직접 바이트 검증 (더 빠름)
	// 바이트 단위 검증으로 rune 변환 오버헤드 제거
	for i := 0; i < 64; i++ {
		c := data[i]
		// 비트 연산으로 범위 체크 최적화
		isDigit := c >= '0' && c <= '9'
		isLowerHex := c >= 'a' && c <= 'f'
		isUpperHex := c >= 'A' && c <= 'F'
		if !(isDigit || isLowerHex || isUpperHex) {
			return "", fmt.Errorf("invalid checksum format: contains non-hexadecimal characters")
		}
	}

	return string(data), nil
}

func SaveChecksumFile(path, checksum string) error {
	return os.WriteFile(path, []byte(checksum), 0644)
}
