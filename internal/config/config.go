package config

import (
	"fmt"
	"os"

	"github.com/BurntSushi/toml"
)

const defaultConfig = `# Minecraft Server Launcher Configuration
# Edit this file to customize server settings

# Minecraft version (use "latest" for the latest version)
minecraft_version = "latest"

# Minimum RAM in GB
min_ram = 2

# Maximum RAM in GB (will be auto-adjusted based on system RAM)
max_ram = 4

# Server arguments
server_args = ["nogui"]

# Working directory (optional, defaults to current directory)
# work_dir = "./server"
`

type Config struct {
	MinecraftVersion string   `toml:"minecraft_version"`
	MinRAM           int      `toml:"min_ram"`
	MaxRAM           int      `toml:"max_ram"`
	ServerArgs       []string `toml:"server_args"`
	WorkDir          string   `toml:"work_dir"`
}

func Load(path string) (*Config, error) {
	if _, err := os.Stat(path); os.IsNotExist(err) {
		if err := os.WriteFile(path, []byte(defaultConfig), 0644); err != nil {
			return nil, fmt.Errorf("failed to create config: %w", err)
		}
		fmt.Println("Created config.toml with default settings.")
	}

	var cfg Config
	if _, err := toml.DecodeFile(path, &cfg); err != nil {
		return nil, fmt.Errorf("failed to parse config: %w", err)
	}

	if v := os.Getenv("MINECRAFT_VERSION"); v != "" {
		cfg.MinecraftVersion = v
	}
	if v := os.Getenv("WORK_DIR"); v != "" {
		cfg.WorkDir = v
	}
	
	if v := os.Getenv("MIN_RAM"); v != "" {
		var minRAM int
		if _, err := fmt.Sscanf(v, "%d", &minRAM); err == nil {
			cfg.MinRAM = minRAM
		}
	}
	if v := os.Getenv("MAX_RAM"); v != "" {
		var maxRAM int
		if _, err := fmt.Sscanf(v, "%d", &maxRAM); err == nil {
			cfg.MaxRAM = maxRAM
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
	if c.MinRAM > c.MaxRAM {
		return fmt.Errorf("min_ram cannot be greater than max_ram")
	}
	if c.MaxRAM > 32 {
		return fmt.Errorf("max_ram exceeds maximum allowed (32GB)")
	}
	return nil
}
