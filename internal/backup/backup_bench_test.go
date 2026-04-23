package backup

import (
	"fmt"
	"os"
	"path/filepath"
	"testing"
	"time"
)

const (
	manyFilesCount = 20
	fewFilesCount  = 5
	retentionLimit = 10
)

func createBackupFiles(backupDir string, count int) error {
	baseTime := time.Now()
	for i := 0; i < count; i++ {
		timestamp := baseTime.Add(time.Duration(i) * time.Hour).Format("2006-01-02_15-04-05")
		backupName := fmt.Sprintf("backup-%s.zip", timestamp)
		backupFile := filepath.Join(backupDir, backupName)
		if err := os.WriteFile(backupFile, []byte("test"), 0644); err != nil {
			return err
		}
		modTime := baseTime.Add(time.Duration(i) * time.Hour)
		if err := os.Chtimes(backupFile, modTime, modTime); err != nil {
			return err
		}
	}
	return nil
}

func BenchmarkRotateBackups_ManyFiles(b *testing.B) {
	tmpDir := b.TempDir()
	backupDir := filepath.Join(tmpDir, "backups")
	if err := os.MkdirAll(backupDir, 0755); err != nil {
		b.Fatal(err)
	}

	if err := createBackupFiles(backupDir, manyFilesCount); err != nil {
		b.Fatal(err)
	}

	oldDir, _ := os.Getwd()
	defer os.Chdir(oldDir)
	if err := os.Chdir(tmpDir); err != nil {
		b.Fatal(err)
	}

	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		PerformBackup([]string{}, "backups", retentionLimit)
	}
}

func BenchmarkRotateBackups_FewFiles(b *testing.B) {
	tmpDir := b.TempDir()
	backupDir := filepath.Join(tmpDir, "backups")
	if err := os.MkdirAll(backupDir, 0755); err != nil {
		b.Fatal(err)
	}

	if err := createBackupFiles(backupDir, fewFilesCount); err != nil {
		b.Fatal(err)
	}

	oldDir, _ := os.Getwd()
	defer os.Chdir(oldDir)
	if err := os.Chdir(tmpDir); err != nil {
		b.Fatal(err)
	}

	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		PerformBackup([]string{}, "backups", retentionLimit)
	}
}
