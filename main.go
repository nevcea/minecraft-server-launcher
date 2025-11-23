package main

import (
	"flag"
	"fmt"
	"log"
	"os"

	"github.com/nevcea-sub/minecraft-server-launcher/internal/backup"
	"github.com/nevcea-sub/minecraft-server-launcher/internal/config"
	"github.com/nevcea-sub/minecraft-server-launcher/internal/download"
	"github.com/nevcea-sub/minecraft-server-launcher/internal/server"
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

func main() {
	flag.Parse()

	logFile, err := os.OpenFile("launcher.log", os.O_CREATE|os.O_WRONLY|os.O_APPEND, 0666)
	if err == nil {
		log.SetOutput(logFile)
	} else {
		fmt.Printf("Failed to open log file: %v\n", err)
	}
	defer logFile.Close()

	defer func() {
		if r := recover(); r != nil {
			msg := fmt.Sprintf("Unexpected error (panic): %v", r)
			fmt.Println(msg)
			log.Println(msg)
			if !*noPause {
				utils.Pause()
			}
			os.Exit(1)
		}
	}()

	if *quiet {
		*logLevel = "error"
	} else if *verbose {
		*logLevel = "debug"
	}

	if err := run(); err != nil {
		msg := fmt.Sprintf("[ERROR] %v", err)
		fmt.Fprintf(os.Stderr, "\n%s\n", msg)
		log.Println(msg)
		
		if !*noPause {
			utils.Pause()
		}
		os.Exit(1)
	}

	if !*noPause {
		utils.Pause()
	}
}

func run() error {
	log.Println("Launcher started")
	
	cfg, err := config.Load(*configFile)
	if err != nil {
		return fmt.Errorf("failed to load config: %w", err)
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
		fmt.Printf("Changed working directory to: %s\n", cfg.WorkDir)
		log.Printf("Changed working directory to: %s", cfg.WorkDir)
	}

	type javaCheckResult struct {
		version string
		err     error
	}
	javaChan := make(chan javaCheckResult, 1)

	go func() {
		ver, err := server.CheckJava()
		javaChan <- javaCheckResult{version: ver, err: err}
	}()

	totalRAM, availableRAM, err := server.GetSystemRAM()
	if err != nil {
		fmt.Printf("Warning: Failed to get system RAM info: %v\n", err)
		log.Printf("Warning: Failed to get system RAM info: %v", err)
	} else {
		fmt.Printf("Total system RAM: %d GB\n", totalRAM)
		fmt.Printf("Available system RAM: %d GB\n", availableRAM)
		log.Printf("Total system RAM: %d GB, Available: %d GB", totalRAM, availableRAM)
	}

	javaRes := <-javaChan
	if javaRes.err != nil {
		return javaRes.err
	}
	fmt.Printf("Java version: %s\n", javaRes.version)
	log.Printf("Java version: %s", javaRes.version)

	jarFile, err := utils.FindJarFile()
	if err != nil {
		return err
	}

	if jarFile != "" {
		hasUpdate, newBuild, newJarName, err := download.CheckUpdate(jarFile)
		if err != nil {
			log.Printf("Warning: Failed to check for updates: %v", err)
		} else if hasUpdate {
			msg := fmt.Sprintf("New version available: %s (Build %d)", newJarName, newBuild)
			fmt.Println(msg)
			log.Println(msg)

			doUpdate := false
			if cfg.AutoUpdate {
				doUpdate = true
			} else {
				fmt.Print("Do you want to update? [Y/N]: ")
				var response string
				fmt.Scanln(&response)
				if response == "Y" || response == "y" {
					doUpdate = true
				}
			}

			if doUpdate {
				fmt.Println("Updating server JAR...")
				downloadedJar, err := download.DownloadJar(cfg.MinecraftVersion)
				if err != nil {
					return fmt.Errorf("failed to update: %w", err)
				}
				
				oldJarBackup := jarFile + ".old"
				if err := os.Rename(jarFile, oldJarBackup); err == nil {
					fmt.Printf("Backed up old JAR to %s\n", oldJarBackup)
				}
				
				jarFile = downloadedJar
			}
		}
	}

	if jarFile != "" {
		checksumFile := jarFile + ".sha256"
		expectedChecksum, _ := utils.LoadChecksumFile(checksumFile)
		
		if expectedChecksum != "" {
			if err := utils.ValidateChecksum(jarFile, expectedChecksum); err != nil {
				fmt.Printf("Warning: Existing JAR failed checksum validation: %v\n", err)
				log.Printf("Warning: Existing JAR failed checksum validation: %v", err)
			} else {
				fmt.Printf("Validated existing JAR file checksum: %s\n", jarFile)
				log.Printf("Validated existing JAR file checksum: %s", jarFile)
			}
		} else {
			checksum, err := utils.ValidateJarAndCalculateChecksum(jarFile)
			if err != nil {
				fmt.Printf("Warning: Existing JAR validation failed: %v\n", err)
				log.Printf("Warning: Existing JAR validation failed: %v", err)
			} else {
				utils.SaveChecksumFile(checksumFile, checksum)
				fmt.Printf("Calculated and saved checksum for existing JAR: %s\n", jarFile)
				log.Printf("Calculated and saved checksum for existing JAR: %s", jarFile)
			}
		}
	}

	if jarFile == "" {
		fmt.Print("No Paper JAR file found. Download automatically? [Y/N]: ")
		var response string
		fmt.Scanln(&response)
		
		if response != "Y" && response != "y" {
			return fmt.Errorf("cannot start server without JAR file")
		}

		jarFile, err = download.DownloadJar(cfg.MinecraftVersion)
		if err != nil {
			return err
		}
	}

	fmt.Printf("Found JAR file: %s\n", jarFile)
	log.Printf("Found JAR file: %s", jarFile)

	if err := utils.HandleEULA(); err != nil {
		return err
	}

	if cfg.AutoBackup {
		worlds := cfg.BackupWorlds
		if len(worlds) == 0 {
			worlds = []string{"world", "world_nether", "world_the_end"}
		}
		if err := backup.PerformBackup(worlds, cfg.BackupCount); err != nil {
			return fmt.Errorf("backup failed: %w", err)
		}
	}

	maxRAM := server.CalculateSmartRAM(cfg.MaxRAM, cfg.AutoRAMPercentage, cfg.MinRAM)
	
	ramMsg := fmt.Sprintf("Starting server with %dG - %dG RAM", cfg.MinRAM, maxRAM)
	if cfg.MaxRAM == 0 {
		ramMsg += fmt.Sprintf(" (Auto-calculated: %d%% of available RAM)", cfg.AutoRAMPercentage)
	}
	fmt.Println(ramMsg)
	log.Println(ramMsg)

	return server.RunServer(jarFile, cfg.MinRAM, maxRAM, cfg.UseZGC, cfg.ServerArgs)
}
