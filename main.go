package main

import (
	"bufio"
	"flag"
	"fmt"
	"log"
	"os"
	"strings"

	"github.com/nevcea-sub/minecraft-server-launcher/internal/backup"
	"github.com/nevcea-sub/minecraft-server-launcher/internal/config"
	"github.com/nevcea-sub/minecraft-server-launcher/internal/download"
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

type logLevelType int

const (
	logLevelTrace logLevelType = iota
	logLevelDebug
	logLevelInfo
	logLevelWarn
	logLevelError
)

var currentLogLevel = logLevelInfo

func parseLogLevel(level string) logLevelType {
	switch strings.ToLower(level) {
	case "trace":
		return logLevelTrace
	case "debug":
		return logLevelDebug
	case "info":
		return logLevelInfo
	case "warn", "warning":
		return logLevelWarn
	case "error":
		return logLevelError
	default:
		return logLevelInfo
	}
}

func shouldLog(level logLevelType) bool {
	return level >= currentLogLevel
}

func promptYesNo(message string) bool {
	for {
		fmt.Printf("[PROMPT] %s [Y/N]: ", message)
		reader := bufio.NewReader(os.Stdin)
		response, err := reader.ReadString('\n')
		if err != nil {
			logMessage(logLevelWarn, "Failed to read user input: %v", err)
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

func logMessage(level logLevelType, format string, args ...interface{}) {
	if shouldLog(level) {
		prefix := ""
		switch level {
		case logLevelTrace:
			prefix = "[TRACE]"
		case logLevelDebug:
			prefix = "[DEBUG]"
		case logLevelInfo:
			prefix = "[INFO]"
		case logLevelWarn:
			prefix = "[WARN]"
		case logLevelError:
			prefix = "[ERROR]"
		}
		msg := fmt.Sprintf(format, args...)
		fmt.Printf("%s %s\n", prefix, msg)
		log.Printf("%s %s", prefix, msg)
	}
}

func main() {
	flag.Parse()

	if *quiet {
		*logLevel = "error"
	} else if *verbose {
		*logLevel = "debug"
	}
	currentLogLevel = parseLogLevel(*logLevel)

	cfg, err := config.Load(*configFile)
	if err != nil {
		fmt.Fprintf(os.Stderr, "[ERROR] Failed to load config: %v\n", err)
		os.Exit(1)
	}

	logFilePath := "launcher.log"
	if cfg != nil && cfg.LogFile != "" {
		logFilePath = cfg.LogFile
	}

	logFile, err := os.OpenFile(logFilePath, os.O_CREATE|os.O_WRONLY|os.O_APPEND, 0666)
	if err == nil {
		log.SetOutput(logFile)
		defer func() {
			if err := logFile.Close(); err != nil {
				fmt.Fprintf(os.Stderr, "[WARN] Failed to close log file: %v\n", err)
			}
		}()
	} else {
		fmt.Fprintf(os.Stderr, "[WARN] Failed to open log file: %v\n", err)
	}

	defer func() {
		if r := recover(); r != nil {
			msg := fmt.Sprintf("Unexpected error (panic): %v", r)
			fmt.Fprintf(os.Stderr, "[ERROR] %s\n", msg)
			log.Printf("[ERROR] %s", msg)
			if !*noPause {
				utils.Pause()
			}
			os.Exit(1)
		}
	}()

	if err := run(cfg); err != nil {
		logMessage(logLevelError, "%v", err)
		if !*noPause {
			utils.Pause()
		}
		os.Exit(1)
	}

	if !*noPause {
		utils.Pause()
	}
}

func run(cfg *config.Config) error {
	logMessage(logLevelInfo, "Launcher started")
	currentVersion := update.GetCurrentVersion()
	logMessage(logLevelInfo, "Launcher version: %s", currentVersion)

	if cfg == nil {
		return fmt.Errorf("config is nil")
	}

	hasUpdate, release, err := update.CheckForUpdate()
	if err != nil {
		logMessage(logLevelWarn, "Failed to check for launcher updates: %v", err)
	} else if hasUpdate {
		logMessage(logLevelInfo, "New launcher version available: %s", release.TagName)
		if release.Body != "" {
			releaseNotes := strings.Split(release.Body, "\n")
			if len(releaseNotes) > 0 && releaseNotes[0] != "" {
				logMessage(logLevelInfo, "Release notes: %s", releaseNotes[0])
			}
		}

		doUpdate := false
		if cfg.AutoUpdateLauncher {
			doUpdate = true
			logMessage(logLevelInfo, "Auto-updating launcher...")
		} else {
			if promptYesNo(fmt.Sprintf("Do you want to update the launcher to %s?", release.TagName)) {
				doUpdate = true
			}
		}

		if doUpdate {
			logMessage(logLevelInfo, "Downloading launcher update...")
			tempFile, err := update.DownloadUpdate(release)
			if err != nil {
				logMessage(logLevelError, "Failed to download update: %v", err)
			} else {
				logMessage(logLevelInfo, "Update downloaded successfully")

				if err := update.ValidateUpdate(tempFile); err != nil {
					logMessage(logLevelError, "Update validation failed: %v", err)
					if err := os.Remove(tempFile); err != nil {
						logMessage(logLevelWarn, "Failed to remove invalid update file: %v", err)
					}
				} else {
					logMessage(logLevelInfo, "Installing launcher update...")
					if err := update.InstallUpdate(tempFile); err != nil {
						logMessage(logLevelError, "Failed to install update: %v", err)
						if err := os.Remove(tempFile); err != nil {
							logMessage(logLevelWarn, "Failed to remove temp file: %v", err)
						}
					} else {
						logMessage(logLevelInfo, "Launcher updated successfully! Please restart the launcher.")
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
		logMessage(logLevelInfo, "Changed working directory to: %s", cfg.WorkDir)
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
		logMessage(logLevelWarn, "Failed to get system RAM info: %v", err)
	} else {
		logMessage(logLevelInfo, "System RAM: %d GB total, %d GB available", totalRAM, availableRAM)
	}

	javaRes := <-javaChan
	if javaRes.err != nil {
		return javaRes.err
	}
	logMessage(logLevelInfo, "Java version: %s", javaRes.version)

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

		jarFile, err = download.DownloadJar(cfg.MinecraftVersion)
		if err != nil {
			return err
		}
		logMessage(logLevelInfo, "Found JAR file: %s", jarFile)
	} else {
		logMessage(logLevelInfo, "Found JAR file: %s", jarFile)

		checksumFile := jarFile + ".sha256"
		expectedChecksum, err := utils.LoadChecksumFile(checksumFile)
		if err != nil {
			logMessage(logLevelWarn, "Failed to load checksum file: %v", err)
		}

		if expectedChecksum != "" {
			if err := utils.ValidateChecksum(jarFile, expectedChecksum); err != nil {
				logMessage(logLevelWarn, "Existing JAR failed checksum validation: %v", err)
				if promptYesNo("JAR file checksum validation failed. Re-download?") {
					jarFile, err = download.DownloadJar(cfg.MinecraftVersion)
					if err != nil {
						return fmt.Errorf("failed to re-download JAR: %w", err)
					}
					logMessage(logLevelInfo, "Re-downloaded JAR file: %s", jarFile)
				} else {
					logMessage(logLevelWarn, "Continuing with invalid JAR file (not recommended)")
				}
			} else {
				logMessage(logLevelInfo, "Validated existing JAR file checksum")
			}
		} else {
			checksum, err := utils.ValidateJarAndCalculateChecksum(jarFile)
			if err != nil {
				logMessage(logLevelWarn, "Existing JAR validation failed: %v", err)
			} else {
				if err := utils.SaveChecksumFile(checksumFile, checksum); err != nil {
					logMessage(logLevelWarn, "Failed to save checksum file: %v", err)
				} else {
					logMessage(logLevelInfo, "Calculated and saved checksum for existing JAR")
				}
			}
		}

		hasUpdate, newBuild, newJarName, err := download.CheckUpdate(jarFile)
		if err != nil {
			logMessage(logLevelWarn, "Failed to check for updates: %v", err)
		} else if hasUpdate {
			logMessage(logLevelInfo, "New version available: %s (Build %d)", newJarName, newBuild)

			doUpdate := false
			if cfg.AutoUpdate {
				doUpdate = true
			} else {
				if promptYesNo("Do you want to update?") {
					doUpdate = true
				}
			}

			if doUpdate {
				logMessage(logLevelInfo, "Updating server JAR...")
				oldJarBackup := jarFile + ".old"
				if err := os.Rename(jarFile, oldJarBackup); err == nil {
					logMessage(logLevelInfo, "Backed up old JAR to %s", oldJarBackup)
				}

				downloadedJar, err := download.DownloadJar(cfg.MinecraftVersion)
				if err != nil {
					if _, renameErr := os.Stat(oldJarBackup); renameErr == nil {
						if restoreErr := os.Rename(oldJarBackup, jarFile); restoreErr == nil {
							logMessage(logLevelInfo, "Restored original JAR file")
						}
					}
					return fmt.Errorf("failed to update: %w", err)
				}

				jarFile = downloadedJar
				logMessage(logLevelInfo, "Updated to: %s", jarFile)
			}
		}
	}

	maxRAM := server.CalculateSmartRAM(cfg.MaxRAM, cfg.AutoRAMPercentage, cfg.MinRAM)

	if cfg.AutoBackup {
		worlds := cfg.BackupWorlds
		if len(worlds) == 0 {
			worlds = []string{"world", "world_nether", "world_the_end"}
		}
		if err := backup.PerformBackup(worlds, cfg.BackupCount); err != nil {
			return fmt.Errorf("backup failed: %w", err)
		}
	}

	ramMsg := fmt.Sprintf("Starting server with %dG - %dG RAM", cfg.MinRAM, maxRAM)
	if cfg.MaxRAM == 0 {
		ramMsg += fmt.Sprintf(" (auto-calculated: %d%% of available RAM)", cfg.AutoRAMPercentage)
	}
	logMessage(logLevelInfo, "%s", ramMsg)

	return server.RunServer(jarFile, cfg.MinRAM, maxRAM, cfg.UseZGC, javaPath, javaRes.versionNum, cfg.ServerArgs)
}
