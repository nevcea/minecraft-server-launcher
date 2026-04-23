package backup

import (
	"archive/zip"
	"fmt"
	"io"
	"os"
	"path/filepath"
	"sort"
	"strings"
	"time"

	"github.com/nevcea-sub/minecraft-server-launcher/internal/logger"
)

const (
	backupBufSize    = 32 * 1024
	backupTimeLayout = "2006-01-02_15-04-05"
)

func PerformBackup(worlds []string, backupDir string, retentionCount int) error {
	if backupDir == "" {
		backupDir = "backups"
	}

	if err := os.MkdirAll(backupDir, 0755); err != nil {
		return fmt.Errorf("failed to create backup directory: %w", err)
	}

	testFile := filepath.Join(backupDir, ".write-test")
	if err := os.WriteFile(testFile, []byte("test"), 0644); err != nil {
		return fmt.Errorf("backup directory is not writable: %w", err)
	}
	if err := os.Remove(testFile); err != nil {
		return fmt.Errorf("failed to clean up test file: %w", err)
	}

	existingWorlds := filterExistingWorlds(worlds)
	if len(existingWorlds) == 0 {
		logger.Info("No worlds found to backup, skipping")
		return nil
	}

	timestamp := time.Now().Format(backupTimeLayout)
	backupFile := filepath.Join(backupDir, fmt.Sprintf("backup-%s.zip", timestamp))

	logger.Info("Creating backup: %s", backupFile)

	if err := createZip(backupFile, existingWorlds); err != nil {
		return err
	}

	logger.Info("Backup created successfully")

	if err := rotateBackups(backupDir, retentionCount); err != nil {
		logger.Warn("Failed to rotate backups: %v", err)
	}

	return nil
}

func filterExistingWorlds(worlds []string) []string {
	result := make([]string, 0, len(worlds))
	for _, w := range worlds {
		info, err := os.Stat(w)
		if err == nil && info.IsDir() {
			result = append(result, w)
		}
	}
	return result
}

func createZip(targetFile string, worlds []string) error {
	zipFile, err := os.Create(targetFile)
	if err != nil {
		return fmt.Errorf("failed to create zip file: %w", err)
	}
	defer func() {
		if err := zipFile.Close(); err != nil {
			logger.Warn("Failed to close zip file: %v", err)
		}
	}()

	archive := zip.NewWriter(zipFile)
	defer func() {
		if err := archive.Close(); err != nil {
			logger.Warn("Failed to close zip archive: %v", err)
		}
	}()

	for _, world := range worlds {
		if _, err := os.Stat(world); os.IsNotExist(err) {
			continue
		}

		err := filepath.Walk(world, func(path string, info os.FileInfo, err error) error {
			if err != nil {
				return fmt.Errorf("failed to walk directory: %w", err)
			}

			if info.Name() == "session.lock" || strings.HasSuffix(info.Name(), ".tmp") {
				return nil
			}

			header, err := zip.FileInfoHeader(info)
			if err != nil {
				return fmt.Errorf("failed to create zip header: %w", err)
			}

			header.Name = filepath.ToSlash(path)

			if info.IsDir() {
				header.Name += "/"
			} else {
				header.Method = zip.Deflate
			}

			writer, err := archive.CreateHeader(header)
			if err != nil {
				return fmt.Errorf("failed to create zip entry: %w", err)
			}

			if info.IsDir() {
				return nil
			}

			file, err := os.Open(path)
			if err != nil {
				return fmt.Errorf("failed to open file: %w", err)
			}

			buf := make([]byte, backupBufSize)
			_, err = io.CopyBuffer(writer, file, buf)
			if closeErr := file.Close(); closeErr != nil {
				if err == nil {
					err = fmt.Errorf("failed to close file: %w", closeErr)
				}
			}
			return err
		})

		if err != nil {
			return fmt.Errorf("failed to backup %s: %w", world, err)
		}
	}

	return nil
}

func rotateBackups(backupDir string, limit int) error {
	if limit <= 0 {
		return nil
	}

	files, err := os.ReadDir(backupDir)
	if err != nil {
		return fmt.Errorf("failed to read backup directory: %w", err)
	}

	type backupInfo struct {
		entry   os.DirEntry
		modTime time.Time
	}

	backups := make([]backupInfo, 0, len(files))
	for _, file := range files {
		if !file.IsDir() && strings.HasPrefix(file.Name(), "backup-") && strings.HasSuffix(file.Name(), ".zip") {
			info, err := file.Info()
			if err != nil {
				continue
			}
			backups = append(backups, backupInfo{
				entry:   file,
				modTime: info.ModTime(),
			})
		}
	}

	if len(backups) <= limit {
		return nil
	}

	sort.Slice(backups, func(i, j int) bool {
		return backups[i].modTime.Before(backups[j].modTime)
	})

	toDelete := len(backups) - limit
	for i := 0; i < toDelete; i++ {
		path := filepath.Join(backupDir, backups[i].entry.Name())
		logger.Info("Deleting old backup: %s", backups[i].entry.Name())
		if err := os.Remove(path); err != nil {
			return fmt.Errorf("failed to remove backup file: %w", err)
		}
	}

	return nil
}
