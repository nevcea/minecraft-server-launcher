package backup

import (
	"os"
	"path/filepath"
	"testing"
	"time"

	"github.com/nevcea-sub/minecraft-server-launcher/internal/backup"
)

func BenchmarkRotateBackups_ManyFiles(b *testing.B) {
	tmpDir := b.TempDir()
	backupDir := filepath.Join(tmpDir, "backups")
	os.MkdirAll(backupDir, 0755)

	for i := 0; i < 20; i++ {
		backupFile := filepath.Join(backupDir, "backup-2024-01-01_00-00-00.zip")
		os.WriteFile(backupFile, []byte("test"), 0644)
		os.Chtimes(backupFile, time.Now().Add(time.Duration(i)*time.Hour), time.Now().Add(time.Duration(i)*time.Hour))
	}

	oldDir, _ := os.Getwd()
	defer os.Chdir(oldDir)
	os.Chdir(tmpDir)

	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		backup.PerformBackup([]string{}, 10)
	}
}

func BenchmarkRotateBackups_FewFiles(b *testing.B) {
	tmpDir := b.TempDir()
	backupDir := filepath.Join(tmpDir, "backups")
	os.MkdirAll(backupDir, 0755)

	for i := 0; i < 5; i++ {
		backupFile := filepath.Join(backupDir, "backup-2024-01-01_00-00-00.zip")
		os.WriteFile(backupFile, []byte("test"), 0644)
		os.Chtimes(backupFile, time.Now().Add(time.Duration(i)*time.Hour), time.Now().Add(time.Duration(i)*time.Hour))
	}

	oldDir, _ := os.Getwd()
	defer os.Chdir(oldDir)
	os.Chdir(tmpDir)

	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		backup.PerformBackup([]string{}, 10)
	}
}

