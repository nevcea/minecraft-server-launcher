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
)

const BackupDir = "backups"

func PerformBackup(worlds []string, retentionCount int) error {
	if err := os.MkdirAll(BackupDir, 0755); err != nil {
		return fmt.Errorf("failed to create backup directory: %w", err)
	}

	testFile := filepath.Join(BackupDir, ".write-test")
	if err := os.WriteFile(testFile, []byte("test"), 0644); err != nil {
		return fmt.Errorf("backup directory is not writable: %w", err)
	}
	if err := os.Remove(testFile); err != nil {
		// Ignore remove error for test file
		_ = err
	}

	existingWorlds := filterExistingWorlds(worlds)
	if len(existingWorlds) == 0 {
		fmt.Println("[INFO] No worlds found to backup, skipping")
		return nil
	}

	timestamp := time.Now().Format("2006-01-02_15-04-05")
	backupFile := filepath.Join(BackupDir, fmt.Sprintf("backup-%s.zip", timestamp))

	fmt.Printf("[INFO] Creating backup: %s\n", backupFile)

	if err := createZip(backupFile, existingWorlds); err != nil {
		return err
	}

	fmt.Println("[OK] Backup created successfully")

	if err := rotateBackups(retentionCount); err != nil {
		fmt.Fprintf(os.Stderr, "[WARN] Failed to rotate backups: %v\n", err)
	}

	return nil
}

func filterExistingWorlds(worlds []string) []string {
	var result []string
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
			// Ignore close error in defer
			_ = err
		}
	}()

	archive := zip.NewWriter(zipFile)
	defer func() {
		if err := archive.Close(); err != nil {
			// Ignore close error in defer
			_ = err
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

			_, err = io.Copy(writer, file)
			if closeErr := file.Close(); closeErr != nil {
				// Ignore close error after copy
				_ = closeErr
			}
			return err
		})

		if err != nil {
			return fmt.Errorf("failed to backup %s: %w", world, err)
		}
	}

	return nil
}

func rotateBackups(limit int) error {
	if limit <= 0 {
		return nil
	}

	files, err := os.ReadDir(BackupDir)
	if err != nil {
		return fmt.Errorf("failed to read backup directory: %w", err)
	}

	type backupInfo struct {
		entry   os.DirEntry
		modTime time.Time
	}

	var backups []backupInfo
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
		path := filepath.Join(BackupDir, backups[i].entry.Name())
		fmt.Printf("[INFO] Deleting old backup: %s\n", backups[i].entry.Name())
		if err := os.Remove(path); err != nil {
			return fmt.Errorf("failed to remove backup file: %w", err)
		}
	}

	return nil
}
