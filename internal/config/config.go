package config

import (
	"fmt"
	"os"
	"strconv"

	"github.com/nevcea-sub/minecraft-server-launcher/internal/logger"
	"gopkg.in/yaml.v3"
)

const defaultConfig = `# 마인크래프트 서버 런처 설정
# 처음 실행 시 자동으로 생성됩니다.

# 마인크래프트 버전 ("latest" 또는 "1.21.4" 같은 특정 버전)
minecraft_version: "latest"

# 서버 JAR 및 런처 자동 업데이트 여부
auto_update: true
auto_update_launcher: true

# 서버 시작 전 월드 자동 백업 여부
auto_backup: false

# 백업할 월드 폴더 목록 (커스텀 월드 이름 사용 시 수정)
backup_worlds:
  - world
  - world_nether
  - world_the_end

# 서버 RAM 설정 (GB 단위)
# max_ram을 0으로 두면 시스템 여유 메모리의 50%를 자동으로 사용합니다.
min_ram: 2
max_ram: 0

# 서버에 전달할 추가 인수
server_args:
  - nogui
`

type Config struct {
	MinecraftVersion string   `yaml:"minecraft_version"`
	AutoUpdate       bool     `yaml:"auto_update"`
	AutoBackup       bool     `yaml:"auto_backup"`
	BackupCount        int      `yaml:"backup_count"`
	BackupDir          string   `yaml:"backup_dir"`
	BackupWorlds       []string `yaml:"backup_worlds"`
	MinRAM             int      `yaml:"min_ram"`
	MaxRAM             int      `yaml:"max_ram"`
	UseZGC             bool     `yaml:"use_zgc"`
	AutoRAMPercentage  int      `yaml:"auto_ram_percentage"`
	ServerArgs         []string `yaml:"server_args"`

	// 고급 옵션 — config.yaml에 직접 추가하거나 환경변수로 설정
	GitHubToken   string `yaml:"github_token"`    // 권장: LAUNCHER_GITHUB_TOKEN 환경변수
	WorkDir       string `yaml:"work_dir"`        // 환경변수: WORK_DIR
	JavaPath      string `yaml:"java_path"`       // 환경변수: JAVA_PATH
	LogFileEnable bool   `yaml:"log_file_enable"` // 로그 파일 저장 여부
	LogFile       string `yaml:"log_file"`        // 환경변수: LOG_FILE
}

func Load(path string) (*Config, error) {
	if _, err := os.Stat(path); os.IsNotExist(err) {
		if err := os.WriteFile(path, []byte(defaultConfig), 0644); err != nil {
			return nil, fmt.Errorf("failed to create config: %w", err)
		}
		logger.Info("Created config.yaml with default settings")
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
		cfg.AutoRAMPercentage = defaultAutoRAMPercent
	}
	if len(cfg.BackupWorlds) == 0 {
		cfg.BackupWorlds = []string{"world", "world_nether", "world_the_end"}
	}
	if cfg.BackupCount == 0 {
		cfg.BackupCount = defaultBackupCount
	}
	if cfg.BackupDir == "" {
		cfg.BackupDir = defaultBackupDir
	}
	if cfg.LogFile == "" {
		cfg.LogFile = defaultLogFile
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
	if v := os.Getenv("LAUNCHER_GITHUB_TOKEN"); v != "" {
		cfg.GitHubToken = v
	} else if v := os.Getenv("GITHUB_TOKEN"); v != "" {
		cfg.GitHubToken = v
	} else if v := os.Getenv("GH_TOKEN"); v != "" {
		cfg.GitHubToken = v
	}

	if v := os.Getenv("MIN_RAM"); v != "" {
		if minRAM, err := strconv.Atoi(v); err == nil && minRAM > 0 {
			cfg.MinRAM = minRAM
		} else {
			logger.Warn("Failed to parse MIN_RAM environment variable: %v", err)
		}
	}
	if v := os.Getenv("MAX_RAM"); v != "" {
		if maxRAM, err := strconv.Atoi(v); err == nil && maxRAM >= 0 {
			cfg.MaxRAM = maxRAM
		} else {
			logger.Warn("Failed to parse MAX_RAM environment variable: %v", err)
		}
	}

	if err := cfg.Validate(); err != nil {
		return nil, err
	}

	return &cfg, nil
}

const (
	maxSafeRAM            = 128
	defaultAutoRAMPercent = 50
	defaultBackupCount    = 10
	defaultBackupDir      = "backups"
	defaultLogFile        = "launcher.log"
)

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
	if c.MaxRAM > maxSafeRAM {
		return fmt.Errorf("max_ram exceeds safety limit (%dGB)", maxSafeRAM)
	}
	if c.AutoRAMPercentage < 10 || c.AutoRAMPercentage > 95 {
		return fmt.Errorf("auto_ram_percentage must be between 10 and 95")
	}
	if c.BackupCount < 1 {
		return fmt.Errorf("backup_count must be at least 1")
	}
	return nil
}
