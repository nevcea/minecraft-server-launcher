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

	existingWorlds := filterExistingWorlds(worlds)
	if len(existingWorlds) == 0 {
		fmt.Println("No worlds found to backup. Skipping backup.")
		return nil
	}

	timestamp := time.Now().Format("2006-01-02_15-04-05")
	backupFile := filepath.Join(BackupDir, fmt.Sprintf("backup-%s.zip", timestamp))

	fmt.Printf("Creating backup: %s\n", backupFile)

	if err := createZip(backupFile, existingWorlds); err != nil {
		return err
	}

	fmt.Println("Backup created successfully.")

	if err := rotateBackups(retentionCount); err != nil {
		fmt.Printf("Warning: Failed to rotate backups: %v\n", err)
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
	defer zipFile.Close()

	archive := zip.NewWriter(zipFile)
	defer archive.Close()

	for _, world := range worlds {
		if _, err := os.Stat(world); os.IsNotExist(err) {
			continue
		}

		err := filepath.Walk(world, func(path string, info os.FileInfo, err error) error {
			if err != nil {
				return err
			}

			if info.Name() == "session.lock" {
				return nil
			}

			header, err := zip.FileInfoHeader(info)
			if err != nil {
				return err
			}

			header.Name = filepath.ToSlash(path)

			if info.IsDir() {
				header.Name += "/"
			} else {
				header.Method = zip.Deflate
			}

			writer, err := archive.CreateHeader(header)
			if err != nil {
				return err
			}

			if info.IsDir() {
				return nil
			}

			file, err := os.Open(path)
			if err != nil {
				return err
			}
			defer file.Close()

			_, err = io.Copy(writer, file)
			return err
		})

		if err != nil {
			return fmt.Errorf("failed to backup %s: %w", world, err)
		}
	}

	return nil
}

func rotateBackups(limit int) error {
	files, err := os.ReadDir(BackupDir)
	if err != nil {
		return err
	}

	var backups []os.DirEntry
	for _, file := range files {
		if !file.IsDir() && strings.HasPrefix(file.Name(), "backup-") && strings.HasSuffix(file.Name(), ".zip") {
			backups = append(backups, file)
		}
	}

	if len(backups) <= limit {
		return nil
	}

	// Sort by name (timestamp) ascending
	sort.Slice(backups, func(i, j int) bool {
		return backups[i].Name() < backups[j].Name()
	})

	// Delete oldest
	toDelete := len(backups) - limit
	for i := 0; i < toDelete; i++ {
		path := filepath.Join(BackupDir, backups[i].Name())
		fmt.Printf("Deleting old backup: %s\n", backups[i].Name())
		if err := os.Remove(path); err != nil {
			return err
		}
	}

	return nil
}

