package config

import (
	"os"
	"path/filepath"
	"testing"
)

func TestLoad(t *testing.T) {
	tmpDir := t.TempDir()
	configPath := filepath.Join(tmpDir, "config.yaml")

	cfg, err := Load(configPath)
	if err != nil {
		t.Fatalf("failed to load config: %v", err)
	}

	if cfg.MinecraftVersion != "latest" {
		t.Errorf("expected 'latest', got %s", cfg.MinecraftVersion)
	}
	if cfg.MinRAM != 2 {
		t.Errorf("expected 2, got %d", cfg.MinRAM)
	}
}

func TestValidate(t *testing.T) {
	tests := []struct {
		name    string
		config  Config
		wantErr bool
	}{
		{
			name: "valid config",
			config: Config{
				MinecraftVersion:  "1.21.1",
				MinRAM:            2,
				MaxRAM:            4,
				AutoRAMPercentage: 85,
				BackupCount:       10,
			},
			wantErr: false,
		},
		{
			name: "empty version",
			config: Config{
				MinecraftVersion:  "",
				MinRAM:            2,
				MaxRAM:            4,
				AutoRAMPercentage: 85,
				BackupCount:       10,
			},
			wantErr: true,
		},
		{
			name: "min > max",
			config: Config{
				MinecraftVersion:  "latest",
				MinRAM:            8,
				MaxRAM:            4,
				AutoRAMPercentage: 85,
				BackupCount:       10,
			},
			wantErr: true,
		},
		{
			name: "zero min ram",
			config: Config{
				MinecraftVersion:  "latest",
				MinRAM:            0,
				MaxRAM:            4,
				AutoRAMPercentage: 85,
				BackupCount:       10,
			},
			wantErr: true,
		},
		{
			name: "max ram too high",
			config: Config{
				MinecraftVersion:  "latest",
				MinRAM:            2,
				MaxRAM:            130,
				AutoRAMPercentage: 85,
				BackupCount:       10,
			},
			wantErr: true,
		},
		{
			name: "invalid percentage low",
			config: Config{
				MinecraftVersion:  "latest",
				MinRAM:            2,
				MaxRAM:            4,
				AutoRAMPercentage: 5,
				BackupCount:       10,
			},
			wantErr: true,
		},
		{
			name: "invalid percentage high",
			config: Config{
				MinecraftVersion:  "latest",
				MinRAM:            2,
				MaxRAM:            4,
				AutoRAMPercentage: 100,
				BackupCount:       10,
			},
			wantErr: true,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			err := tt.config.Validate()
			if (err != nil) != tt.wantErr {
				t.Errorf("Validate() error = %v, wantErr %v", err, tt.wantErr)
			}
		})
	}
}

func TestEnvOverride(t *testing.T) {
	tmpDir := t.TempDir()
	configPath := filepath.Join(tmpDir, "config.yaml")

	os.Setenv("MINECRAFT_VERSION", "1.20.1")
	defer os.Unsetenv("MINECRAFT_VERSION")

	cfg, err := Load(configPath)
	if err != nil {
		t.Fatalf("failed to load config: %v", err)
	}

	if cfg.MinecraftVersion != "1.20.1" {
		t.Errorf("expected env override '1.20.1', got %s", cfg.MinecraftVersion)
	}
}
