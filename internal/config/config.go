package config

import (
	"fmt"
	"os"
	"strconv"

	"github.com/nevcea-sub/minecraft-server-launcher/internal/logger"
	"gopkg.in/yaml.v3"
)

const defaultConfig = `# 마인크래프트 서버 설정 파일
# 이 파일은 프로그램이 처음 실행될 때 자동으로 생성됩니다.

# 마인크래프트 버전 (예: "1.20.4", "latest" 또는 "snapshot")
minecraft_version: "latest"

# 서버 시작 시 자동으로 마인크래프트 서버 버전을 확인하고 업데이트할지 여부
auto_update: true

# 런처 자체의 자동 업데이트 기능 사용 여부
auto_update_launcher: true

# GitHub Releases 업데이트 체크/다운로드에 사용할 토큰
# - 리포가 private이면 필수입니다 (토큰 없으면 GitHub API가 404를 반환합니다)
# - 권장: 환경변수 LAUNCHER_GITHUB_TOKEN 로 주입 (파일에 토큰 저장은 주의)
github_token: ""

# 서버 시작 전 월드 데이터 자동 백업 여부
auto_backup: false

# 유지할 최대 백업 파일 개수 (오래된 백업부터 삭제됨)
backup_count: 10

# 백업 파일을 저장할 디렉토리 경로
backup_dir: "backups"

# 백업할 월드 디렉토리 목록
backup_worlds:
  - world
  - world_nether
  - world_the_end

# 서버에 할당할 최소 RAM 크기 (GB 단위)
min_ram: 2

# 서버에 할당할 최대 RAM 크기 (GB 단위)
# 0으로 설정 시 시스템 메모리와 auto_ram_percentage에 따라 자동으로 계산됩니다.
max_ram: 0

# ZGC (Z Garbage Collector) 사용 여부
# 대용량 메모리 사용 시 지연 시간을 줄여주지만, Java 11 이상이 필요합니다.
use_zgc: false

# 최대 RAM 자동 설정 시 사용할 시스템 메모리 비율 (%)
# max_ram이 0일 때만 적용됩니다.
auto_ram_percentage: 50

# 서버 실행 시 추가로 전달할 Java 인수 목록
server_args:
  - nogui

# 로그 파일 생성 여부
log_file_enable: false
`

type Config struct {
	MinecraftVersion   string   `yaml:"minecraft_version"`    // 마인크래프트 버전
	AutoUpdate         bool     `yaml:"auto_update"`          // 서버 자동 업데이트 여부
	AutoUpdateLauncher bool     `yaml:"auto_update_launcher"` // 런처 자동 업데이트 여부
	GitHubToken        string   `yaml:"github_token"`         // GitHub 토큰 (private repo 업데이트용)
	AutoBackup         bool     `yaml:"auto_backup"`          // 자동 백업 사용 여부
	BackupCount        int      `yaml:"backup_count"`         // 유지할 백업 개수
	BackupDir          string   `yaml:"backup_dir"`           // 백업 디렉토리
	BackupWorlds       []string `yaml:"backup_worlds"`        // 백업할 월드 목록
	MinRAM             int      `yaml:"min_ram"`              // 최소 RAM (GB)
	MaxRAM             int      `yaml:"max_ram"`              // 최대 RAM (GB, 0=자동)
	UseZGC             bool     `yaml:"use_zgc"`              // ZGC 사용 여부
	AutoRAMPercentage  int      `yaml:"auto_ram_percentage"`  // 자동 RAM 할당 비율 (%)
	ServerArgs         []string `yaml:"server_args"`          // 서버 실행 인수
	WorkDir            string   `yaml:"work_dir"`             // 작업 디렉토리
	JavaPath           string   `yaml:"java_path"`            // Java 실행 파일 경로
	LogFileEnable      bool     `yaml:"log_file_enable"`      // 로그 파일 생성 여부
	LogFile            string   `yaml:"log_file"`             // 로그 파일 경로
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
		cfg.AutoRAMPercentage = 50
	}
	if len(cfg.BackupWorlds) == 0 {
		cfg.BackupWorlds = []string{"world", "world_nether", "world_the_end"}
	}
	if cfg.BackupCount == 0 {
		cfg.BackupCount = 10
	}
	if cfg.BackupDir == "" {
		cfg.BackupDir = "backups"
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

const maxSafeRAM = 128

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
