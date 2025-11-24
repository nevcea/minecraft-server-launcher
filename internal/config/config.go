package config

import (
	"fmt"
	"os"

	"gopkg.in/yaml.v3"
)

const defaultConfig = `minecraft_version: "latest"
auto_update: false
auto_backup: true
backup_count: 10
backup_worlds:
  - world
  - world_nether
  - world_the_end
auto_restart: false
min_ram: 2
max_ram: 0
use_zgc: false
auto_ram_percentage: 85
server_args:
  - nogui
`

type Config struct {
	MinecraftVersion  string   `yaml:"minecraft_version"`
	AutoUpdate        bool     `yaml:"auto_update"`
	AutoBackup        bool     `yaml:"auto_backup"`
	BackupCount       int      `yaml:"backup_count"`
	BackupWorlds      []string `yaml:"backup_worlds"`
	AutoRestart       bool     `yaml:"auto_restart"`
	MinRAM            int      `yaml:"min_ram"`
	MaxRAM            int      `yaml:"max_ram"`
	UseZGC            bool     `yaml:"use_zgc"`
	AutoRAMPercentage int      `yaml:"auto_ram_percentage"`
	ServerArgs        []string `yaml:"server_args"`
	WorkDir           string   `yaml:"work_dir"`
	JavaPath          string   `yaml:"java_path"`
	LogFile           string   `yaml:"log_file"`
}

func Load(path string) (*Config, error) {
	if _, err := os.Stat(path); os.IsNotExist(err) {
		if err := os.WriteFile(path, []byte(defaultConfig), 0644); err != nil {
			return nil, fmt.Errorf("failed to create config: %w", err)
		}
		fmt.Println("[INFO] Created config.yaml with default settings")
	}

	var cfg Config

	data, err := os.ReadFile(path)
	if err != nil {
		return nil, fmt.Errorf("failed to read config: %w", err)
	}

	if err := yaml.Unmarshal(data, &cfg); err != nil {
		return nil, fmt.Errorf("failed to parse config: %w", err)
	}

	if cfg.AutoRAMPercentage == 0 {
		cfg.AutoRAMPercentage = 85
	}
	if len(cfg.BackupWorlds) == 0 {
		cfg.BackupWorlds = []string{"world", "world_nether", "world_the_end"}
	}
	if cfg.BackupCount == 0 {
		cfg.BackupCount = 10
	}
	if cfg.LogFile == "" {
		cfg.LogFile = "launcher.log"
	}

	if v := os.Getenv("MINECRAFT_VERSION"); v != "" {
		cfg.MinecraftVersion = v
	}
	if v := os.Getenv("WORK_DIR"); v != "" {
		cfg.WorkDir = v
	}
	if v := os.Getenv("JAVA_PATH"); v != "" {
		cfg.JavaPath = v
	}
	if v := os.Getenv("LOG_FILE"); v != "" {
		cfg.LogFile = v
	}

	if v := os.Getenv("MIN_RAM"); v != "" {
		var minRAM int
		if _, err := fmt.Sscanf(v, "%d", &minRAM); err == nil && minRAM > 0 {
			cfg.MinRAM = minRAM
		} else if err != nil {
			fmt.Fprintf(os.Stderr, "[WARN] Failed to parse MIN_RAM environment variable: %v\n", err)
		}
	}
	if v := os.Getenv("MAX_RAM"); v != "" {
		var maxRAM int
		if _, err := fmt.Sscanf(v, "%d", &maxRAM); err == nil && maxRAM >= 0 {
			cfg.MaxRAM = maxRAM
		} else if err != nil {
			fmt.Fprintf(os.Stderr, "[WARN] Failed to parse MAX_RAM environment variable: %v\n", err)
		}
	}

	if err := cfg.Validate(); err != nil {
		return nil, err
	}

	return &cfg, nil
}

func (c *Config) Validate() error {
	if c.MinecraftVersion == "" {
		return fmt.Errorf("minecraft_version cannot be empty")
	}
	if c.MinRAM <= 0 {
		return fmt.Errorf("min_ram must be greater than 0")
	}
	if c.MaxRAM != 0 && c.MinRAM > c.MaxRAM {
		return fmt.Errorf("min_ram cannot be greater than max_ram")
	}
	if c.MaxRAM > 128 {
		return fmt.Errorf("max_ram exceeds safety limit (128GB)")
	}
	if c.AutoRAMPercentage < 10 || c.AutoRAMPercentage > 95 {
		return fmt.Errorf("auto_ram_percentage must be between 10 and 95")
	}
	if c.BackupCount < 1 {
		return fmt.Errorf("backup_count must be at least 1")
	}
	return nil
}
