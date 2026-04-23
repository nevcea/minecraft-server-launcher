package config

import (
	"path/filepath"
	"testing"
)

func TestLoad(t *testing.T) {
	cfg, err := Load(filepath.Join(t.TempDir(), "config.yaml"))
	if err != nil {
		t.Fatalf("failed to load config: %v", err)
	}
	if cfg.MinecraftVersion != "latest" {
		t.Errorf("expected 'latest', got %s", cfg.MinecraftVersion)
	}
	if cfg.MinRAM != 2 {
		t.Errorf("expected MinRAM 2, got %d", cfg.MinRAM)
	}
}

func TestValidate(t *testing.T) {
	tests := []struct {
		name    string
		config  Config
		wantErr bool
	}{
		{
			"valid",
			Config{MinecraftVersion: "1.21.1", MinRAM: 2, MaxRAM: 4, AutoRAMPercentage: 85, BackupCount: 10},
			false,
		},
		{
			"empty version",
			Config{MinecraftVersion: "", MinRAM: 2, MaxRAM: 4, AutoRAMPercentage: 85, BackupCount: 10},
			true,
		},
		{
			"min > max",
			Config{MinecraftVersion: "latest", MinRAM: 8, MaxRAM: 4, AutoRAMPercentage: 85, BackupCount: 10},
			true,
		},
		{
			"zero min ram",
			Config{MinecraftVersion: "latest", MinRAM: 0, MaxRAM: 4, AutoRAMPercentage: 85, BackupCount: 10},
			true,
		},
		{
			"max ram too high",
			Config{MinecraftVersion: "latest", MinRAM: 2, MaxRAM: 130, AutoRAMPercentage: 85, BackupCount: 10},
			true,
		},
		{
			"percentage too low",
			Config{MinecraftVersion: "latest", MinRAM: 2, MaxRAM: 4, AutoRAMPercentage: 5, BackupCount: 10},
			true,
		},
		{
			"percentage too high",
			Config{MinecraftVersion: "latest", MinRAM: 2, MaxRAM: 4, AutoRAMPercentage: 100, BackupCount: 10},
			true,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			if err := tt.config.Validate(); (err != nil) != tt.wantErr {
				t.Errorf("Validate() error = %v, wantErr %v", err, tt.wantErr)
			}
		})
	}
}
