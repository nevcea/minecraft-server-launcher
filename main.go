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
	configFile = flag.String("c", "config.yaml", "Custom config file path")
	workDir    = flag.String("w", "", "Override working directory")
	version    = flag.String("v", "", "Override Minecraft version")
	noPause    = flag.Bool("no-pause", false, "Don't pause on exit")
)

func promptYesNo(message string) bool {
	reader := bufio.NewReader(os.Stdin)
	for {
		fmt.Printf("[PROMPT] %s [Y/N]: ", message)
		response, err := reader.ReadString('\n')
		if err != nil {
			logger.Warn("Failed to read user input: %v", err)
			return false
		}

		response = strings.TrimSpace(strings.ToLower(response))

		if response == "y" || response == "yes" {
			return true
		}
		if response == "n" || response == "no" {
			return false
		}
	}
}

func main() {
	flag.Parse()

	// Initialize Logger
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

	// Setup Log File
	if cfg != nil && cfg.LogFileEnable {
		logFilePath := cfg.LogFile
		if logFilePath == "" {
			logFilePath = "launcher.log"
		}
		if err := logger.SetLogFile(logFilePath); err != nil {
			logger.Warn("Failed to open log file: %v", err)
		} else {
			defer logger.Close()
		}
	}

	// Setup Context with Cancellation
	ctx, cancel := signal.NotifyContext(context.Background(), os.Interrupt, syscall.SIGTERM)
	defer cancel()

	defer func() {
		if r := recover(); r != nil {
			logger.Error("Unexpected error (panic): %v", r)
			if !*noPause {
				utils.Pause()
			}
			os.Exit(1)
		}
	}()

	if err := run(ctx, cfg); err != nil {
		if err == context.Canceled {
			logger.Info("Operation cancelled by user")
		} else {
			logger.Error("%v", err)
		}
		if !*noPause {
			utils.Pause()
		}
		os.Exit(1)
	}

	if !*noPause {
		utils.Pause()
	}
}

func run(ctx context.Context, cfg *config.Config) error {
	logger.Info("Launcher started")
	currentVersion := update.GetCurrentVersion()
	logger.Info("Launcher version: %s", currentVersion)

	if cfg == nil {
		return fmt.Errorf("config is nil")
	}

	// Private repo 업데이트 체크/다운로드를 위해 토큰을 주입할 수 있음
	update.SetGitHubToken(cfg.GitHubToken)

	hasUpdate, release, err := update.CheckForUpdate(ctx)
	if err != nil {
		logger.Warn("Failed to check for launcher updates: %v", err)
	} else if hasUpdate {
		logger.Info("New launcher version available: %s", release.TagName)
		if release.Body != "" {
			releaseNotes := strings.Split(release.Body, "\n")
			if len(releaseNotes) > 0 && releaseNotes[0] != "" {
				logger.Info("Release notes: %s", releaseNotes[0])
			}
		}

		doUpdate := false
		if cfg.AutoUpdateLauncher {
			doUpdate = true
			logger.Info("Auto-updating launcher...")
		} else {
			if promptYesNo(fmt.Sprintf("Do you want to update the launcher to %s?", release.TagName)) {
				doUpdate = true
			}
		}

		if doUpdate {
			logger.Info("Downloading launcher update...")
			tempFile, err := update.DownloadUpdate(ctx, release)
			if err != nil {
				logger.Error("Failed to download update: %v", err)
			} else {
				logger.Info("Update downloaded successfully")

				if err := update.ValidateUpdate(tempFile); err != nil {
					logger.Error("Update validation failed: %v", err)
					if err := os.Remove(tempFile); err != nil {
						logger.Warn("Failed to remove invalid update file: %v", err)
					}
				} else {
					logger.Info("Installing launcher update...")
					if err := update.InstallUpdate(tempFile); err != nil {
						logger.Error("Failed to install update: %v", err)
						if err := os.Remove(tempFile); err != nil {
							logger.Warn("Failed to remove temp file: %v", err)
						}
					} else {
						logger.Info("Launcher updated successfully! Please restart the launcher.")
						fmt.Println("[INFO] Launcher updated successfully! Please restart the launcher.")
						return nil
					}
				}
			}
		}
	}

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

	type javaCheckResult struct {
		version    string
		versionNum int
		err        error
	}
	javaChan := make(chan javaCheckResult, 1)

	javaPath := cfg.JavaPath
	if javaPath == "" {
		javaPath = "java"
	}

	go func() {
		ver, verNum, err := server.CheckJava(javaPath)
		javaChan <- javaCheckResult{version: ver, versionNum: verNum, err: err}
	}()

	totalRAM, availableRAM, err := server.GetSystemRAM()
	if err != nil {
		logger.Warn("Failed to get system RAM info: %v", err)
	} else {
		logger.Info("System RAM: %d GB total, %d GB available", totalRAM, availableRAM)
	}

	select {
	case javaRes := <-javaChan:
		if javaRes.err != nil {
			return javaRes.err
		}
		logger.Info("Java version: %s", javaRes.version)

		if err := utils.HandleEULA(); err != nil {
			return err
		}

		jarFile, err := utils.FindJarFile()
		if err != nil {
			return err
		}

		if jarFile == "" {
			if !promptYesNo("No Paper JAR file found. Download automatically?") {
				return fmt.Errorf("cannot start server without JAR file")
			}

			jarFile, err = download.DownloadJar(ctx, cfg.MinecraftVersion)
			if err != nil {
				return err
			}
			logger.Info("Found JAR file: %s", jarFile)
		} else {
			logger.Info("Found JAR file: %s", jarFile)

			checksumFile := jarFile + ".sha256"
			expectedChecksum, err := utils.LoadChecksumFile(checksumFile)
			if err != nil {
				logger.Warn("Failed to load checksum file: %v", err)
			}

			if expectedChecksum != "" {
				if err := utils.ValidateChecksum(jarFile, expectedChecksum); err != nil {
					logger.Warn("Existing JAR failed checksum validation: %v", err)
					if promptYesNo("JAR file checksum validation failed. Re-download?") {
						jarFile, err = download.DownloadJar(ctx, cfg.MinecraftVersion)
						if err != nil {
							return fmt.Errorf("failed to re-download JAR: %w", err)
						}
						logger.Info("Re-downloaded JAR file: %s", jarFile)
					} else {
						logger.Warn("Continuing with invalid JAR file (not recommended)")
					}
				} else {
					logger.Info("Validated existing JAR file checksum")
				}
			} else {
				checksum, err := utils.ValidateJarAndCalculateChecksum(jarFile)
				if err != nil {
					logger.Warn("Existing JAR validation failed: %v", err)
				} else {
					if err := utils.SaveChecksumFile(checksumFile, checksum); err != nil {
						logger.Warn("Failed to save checksum file: %v", err)
					} else {
						logger.Info("Calculated and saved checksum for existing JAR")
					}
				}
			}

			hasUpdate, newBuild, newJarName, err := download.CheckUpdate(ctx, jarFile)
			if err != nil {
				logger.Warn("Failed to check for updates: %v", err)
			} else if hasUpdate {
				logger.Info("New version available: %s (Build %d)", newJarName, newBuild)

				doUpdate := false
				if cfg.AutoUpdate {
					doUpdate = true
				} else {
					if promptYesNo("Do you want to update?") {
						doUpdate = true
					}
				}

				if doUpdate {
					logger.Info("Updating server JAR...")
					oldJarBackup := jarFile + ".old"
					if err := os.Rename(jarFile, oldJarBackup); err == nil {
						logger.Info("Backed up old JAR to %s", oldJarBackup)
					}

					downloadedJar, err := download.DownloadJar(ctx, cfg.MinecraftVersion)
					if err != nil {
						if _, renameErr := os.Stat(oldJarBackup); renameErr == nil {
							if restoreErr := os.Rename(oldJarBackup, jarFile); restoreErr == nil {
								logger.Info("Restored original JAR file")
							}
						}
						return fmt.Errorf("failed to update: %w", err)
					}

					jarFile = downloadedJar
					logger.Info("Updated to: %s", jarFile)
				}
			}
		}

		maxRAM := server.CalculateSmartRAM(cfg.MaxRAM, cfg.AutoRAMPercentage, cfg.MinRAM)

		if cfg.AutoBackup {
			worlds := cfg.BackupWorlds
			if len(worlds) == 0 {
				worlds = []string{"world", "world_nether", "world_the_end"}
			}
			if err := backup.PerformBackup(worlds, cfg.BackupDir, cfg.BackupCount); err != nil {
				return fmt.Errorf("backup failed: %w", err)
			}
		}

		ramMsg := fmt.Sprintf("Starting server with %dG - %dG RAM", cfg.MinRAM, maxRAM)
		if cfg.MaxRAM == 0 {
			ramMsg += fmt.Sprintf(" (auto-calculated: %d%% of available RAM)", cfg.AutoRAMPercentage)
		}
		logger.Info("%s", ramMsg)

		return server.RunServer(ctx, jarFile, cfg.MinRAM, maxRAM, cfg.UseZGC, javaPath, javaRes.versionNum, cfg.ServerArgs)

	case <-ctx.Done():
		return ctx.Err()
	}
}
