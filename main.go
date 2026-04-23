package main

import (
	"bufio"
	"context"
	"flag"
	"fmt"
	"os"
	"os/signal"
	"strings"
	"syscall"

	"github.com/nevcea-sub/minecraft-server-launcher/internal/backup"
	"github.com/nevcea-sub/minecraft-server-launcher/internal/config"
	"github.com/nevcea-sub/minecraft-server-launcher/internal/download"
	"github.com/nevcea-sub/minecraft-server-launcher/internal/logger"
	"github.com/nevcea-sub/minecraft-server-launcher/internal/server"
	"github.com/nevcea-sub/minecraft-server-launcher/internal/update"
	"github.com/nevcea-sub/minecraft-server-launcher/internal/utils"
)

var (
	logLevel   = flag.String("log-level", "info", "Log level (trace, debug, info, warn, error)")
	verbose    = flag.Bool("verbose", false, "Enable verbose logging")
	quiet      = flag.Bool("q", false, "Suppress all output except errors")
	configFile = flag.String("c", "config.yaml", "Config file path")
	workDir    = flag.String("w", "", "Override working directory")
	version    = flag.String("v", "", "Override Minecraft version")
	noPause    = flag.Bool("no-pause", false, "Don't pause on exit")
)

func main() {
	flag.Parse()

	level := logger.LevelInfo
	if *quiet {
		level = logger.LevelError
	} else if *verbose {
		level = logger.LevelDebug
	} else {
		level = logger.ParseLevel(*logLevel)
	}
	logger.SetLevel(level)

	cfg, err := config.Load(*configFile)
	if err != nil {
		logger.Fatal("Failed to load config: %v", err)
	}

	if cfg.LogFileEnable {
		logPath := cfg.LogFile
		if logPath == "" {
			logPath = "launcher.log"
		}
		if err := logger.SetLogFile(logPath); err != nil {
			logger.Warn("Failed to open log file: %v", err)
		} else {
			defer logger.Close()
		}
	}

	ctx, cancel := signal.NotifyContext(context.Background(), os.Interrupt, syscall.SIGTERM)
	defer cancel()

	defer func() {
		if r := recover(); r != nil {
			logger.Error("Unexpected error (panic): %v", r)
			pauseAndExit(1)
		}
	}()

	if err := run(ctx, cfg); err != nil {
		if err == context.Canceled {
			logger.Info("Operation cancelled by user")
		} else {
			logger.Error("%v", err)
		}
		pauseAndExit(1)
	}

	if !*noPause {
		utils.Pause()
	}
}

func pauseAndExit(code int) {
	if !*noPause {
		utils.Pause()
	}
	os.Exit(code)
}

type javaCheckResult struct {
	version    string
	versionNum int
	err        error
}

func run(ctx context.Context, cfg *config.Config) error {
	logger.Info("Launcher started (version: %s)", update.GetCurrentVersion())

	update.SetGitHubToken(cfg.GitHubToken)
	checkLauncherUpdate(ctx)

	if *version != "" {
		cfg.MinecraftVersion = *version
	}
	if *workDir != "" {
		cfg.WorkDir = *workDir
	}
	if cfg.WorkDir != "" && cfg.WorkDir != "." {
		if err := os.Chdir(cfg.WorkDir); err != nil {
			return fmt.Errorf("failed to change directory: %w", err)
		}
		logger.Info("Changed working directory to: %s", cfg.WorkDir)
	}

	javaPath := cfg.JavaPath
	if javaPath == "" {
		javaPath = "java"
	}
	javaChan := make(chan javaCheckResult, 1)
	go func() {
		ver, num, err := server.CheckJava(javaPath)
		javaChan <- javaCheckResult{ver, num, err}
	}()

	var avail int
	if total, a, err := server.GetSystemRAM(); err != nil {
		logger.Warn("Failed to get system RAM info: %v", err)
	} else {
		avail = a
		logger.Info("System RAM: %d GB total, %d GB available", total, avail)
	}

	select {
	case javaRes := <-javaChan:
		if javaRes.err != nil {
			return javaRes.err
		}
		logger.Info("Java version: %s", javaRes.version)

		jarFile, err := prepareServerJar(ctx, cfg)
		if err != nil {
			return err
		}

		if cfg.AutoBackup {
			if err := backup.PerformBackup(cfg.BackupWorlds, cfg.BackupDir, cfg.BackupCount); err != nil {
				return fmt.Errorf("backup failed: %w", err)
			}
		}

		maxRAM := server.CalculateSmartRAM(cfg.MaxRAM, cfg.AutoRAMPercentage, cfg.MinRAM, avail)
		if cfg.MaxRAM == 0 {
			logger.Info("Starting server with %dG - %dG RAM (auto: %d%% of available)", cfg.MinRAM, maxRAM, cfg.AutoRAMPercentage)
		} else {
			logger.Info("Starting server with %dG - %dG RAM", cfg.MinRAM, maxRAM)
		}

		return server.RunServer(ctx, jarFile, cfg.MinRAM, maxRAM, cfg.UseZGC, javaPath, javaRes.versionNum, cfg.ServerArgs)

	case <-ctx.Done():
		return ctx.Err()
	}
}

func checkLauncherUpdate(ctx context.Context) {
	hasUpdate, release, err := update.CheckForUpdate(ctx)
	if err != nil {
		logger.Warn("Failed to check for launcher updates: %v", err)
		return
	}
	if !hasUpdate {
		return
	}
	logger.Info("New launcher version available: %s", release.TagName)
	if first := strings.SplitN(release.Body, "\n", 2)[0]; first != "" {
		logger.Info("Release notes: %s", first)
	}
	logger.Info("Download: https://github.com/nevcea/minecraft-server-launcher/releases/latest")
}

func prepareServerJar(ctx context.Context, cfg *config.Config) (string, error) {
	if err := utils.HandleEULA(); err != nil {
		return "", err
	}

	jarFile, err := utils.FindJarFile()
	if err != nil {
		return "", err
	}

	if jarFile == "" {
		if !promptYesNo("No Paper JAR found. Download automatically?") {
			return "", fmt.Errorf("cannot start server without JAR file")
		}
		jarFile, err = download.DownloadJar(ctx, cfg.MinecraftVersion)
		if err != nil {
			return "", err
		}
		logger.Info("Downloaded JAR: %s", jarFile)
		return jarFile, nil
	}

	logger.Info("Found JAR: %s", jarFile)
	return validateAndUpdateJar(ctx, jarFile, cfg)
}

func validateAndUpdateJar(ctx context.Context, jarFile string, cfg *config.Config) (string, error) {
	checksumFile := jarFile + ".sha256"
	expected, err := utils.LoadChecksumFile(checksumFile)
	if err != nil {
		logger.Warn("Failed to load checksum file: %v", err)
	}

	if expected != "" {
		if err := utils.ValidateChecksum(jarFile, expected); err != nil {
			logger.Warn("JAR checksum mismatch: %v", err)
			if promptYesNo("Checksum validation failed. Re-download?") {
				jarFile, err = download.DownloadJar(ctx, cfg.MinecraftVersion)
				if err != nil {
					return "", fmt.Errorf("failed to re-download JAR: %w", err)
				}
				logger.Info("Re-downloaded JAR: %s", jarFile)
			} else {
				logger.Warn("Continuing with unverified JAR (not recommended)")
			}
		} else {
			logger.Info("JAR checksum OK")
		}
	} else {
		if checksum, err := utils.ValidateJarAndCalculateChecksum(jarFile); err != nil {
			logger.Warn("JAR validation failed: %v", err)
		} else if err := utils.SaveChecksumFile(checksumFile, checksum); err != nil {
			logger.Warn("Failed to save checksum: %v", err)
		} else {
			logger.Info("Checksum saved for existing JAR")
		}
	}

	hasUpdate, newBuild, newJarName, err := download.CheckUpdate(ctx, jarFile)
	if err != nil {
		logger.Warn("Failed to check for server updates: %v", err)
		return jarFile, nil
	}
	if !hasUpdate {
		return jarFile, nil
	}

	logger.Info("Update available: %s (build %d)", newJarName, newBuild)
	if !cfg.AutoUpdate && !promptYesNo("Update server JAR?") {
		return jarFile, nil
	}

	oldBackup := jarFile + ".old"
	if err := os.Rename(jarFile, oldBackup); err == nil {
		logger.Info("Backed up old JAR to %s", oldBackup)
	}

	newJar, err := download.DownloadJar(ctx, cfg.MinecraftVersion)
	if err != nil {
		if _, statErr := os.Stat(oldBackup); statErr == nil {
			if os.Rename(oldBackup, jarFile) == nil {
				logger.Info("Restored original JAR")
			}
		}
		return "", fmt.Errorf("failed to update server JAR: %w", err)
	}

	logger.Info("Updated to: %s", newJar)
	return newJar, nil
}

func promptYesNo(message string) bool {
	reader := bufio.NewReader(os.Stdin)
	for {
		fmt.Printf("[PROMPT] %s [Y/N]: ", message)
		response, err := reader.ReadString('\n')
		if err != nil {
			return false
		}
		switch strings.TrimSpace(strings.ToLower(response)) {
		case "y", "yes":
			return true
		case "n", "no":
			return false
		}
	}
}
