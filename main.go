package main

import (
	"flag"
	"fmt"
	"os"

	"github.com/nevcea-sub/minecraft-server-launcher/internal/config"
	"github.com/nevcea-sub/minecraft-server-launcher/internal/download"
	"github.com/nevcea-sub/minecraft-server-launcher/internal/server"
	"github.com/nevcea-sub/minecraft-server-launcher/internal/utils"
)

var (
	logLevel  = flag.String("log-level", "info", "Log level (trace, debug, info, warn, error)")
	verbose   = flag.Bool("verbose", false, "Enable verbose logging")
	quiet     = flag.Bool("q", false, "Suppress all output except errors")
	configFile = flag.String("c", "config.toml", "Custom config file path")
	workDir   = flag.String("w", "", "Override working directory")
	version   = flag.String("v", "", "Override Minecraft version")
	noPause   = flag.Bool("no-pause", false, "Don't pause on exit")
)

func main() {
	flag.Parse()

	if *quiet {
		*logLevel = "error"
	} else if *verbose {
		*logLevel = "debug"
	}

	if err := run(); err != nil {
		if !*noPause {
			utils.Pause()
		}
		fmt.Fprintf(os.Stderr, "\n[ERROR] %v\n", err)
		os.Exit(1)
	}

	if !*noPause {
		utils.Pause()
	}
}

func run() error {
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
	}

	type javaCheckResult struct {
		version string
		err     error
	}
	type ramCheckResult struct {
		total int
	}

	javaChan := make(chan javaCheckResult, 1)
	ramChan := make(chan ramCheckResult, 1)

	go func() {
		ver, err := server.CheckJava()
		javaChan <- javaCheckResult{version: ver, err: err}
	}()

	go func() {
		ramChan <- ramCheckResult{total: server.GetTotalRAMGB()}
	}()

	javaRes := <-javaChan
	if javaRes.err != nil {
		return javaRes.err
	}
	fmt.Printf("Java version: %s\n", javaRes.version)

	ramRes := <-ramChan
	totalRAM := ramRes.total
	if totalRAM > 0 {
		fmt.Printf("Total system RAM: %d GB\n", totalRAM)
	}

	jarFile, err := utils.FindJarFile()
	if err != nil {
		return err
	}

	if jarFile != "" {
		checksumFile := jarFile + ".sha256"
		expectedChecksum, _ := utils.LoadChecksumFile(checksumFile)
		
		if expectedChecksum != "" {
			if err := utils.ValidateChecksum(jarFile, expectedChecksum); err != nil {
				fmt.Printf("Warning: Existing JAR failed checksum validation: %v\n", err)
			} else {
				fmt.Printf("Validated existing JAR file checksum: %s\n", jarFile)
			}
		} else {
			checksum, err := utils.ValidateJarAndCalculateChecksum(jarFile)
			if err != nil {
				fmt.Printf("Warning: Existing JAR validation failed: %v\n", err)
			} else {
				utils.SaveChecksumFile(checksumFile, checksum)
				fmt.Printf("Calculated and saved checksum for existing JAR: %s\n", jarFile)
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

	if err := utils.HandleEULA(); err != nil {
		return err
	}

	maxRAM := server.CalculateMaxRAM(cfg.MaxRAM, totalRAM, cfg.MinRAM)
	fmt.Printf("Starting server with %dG - %dG RAM\n", cfg.MinRAM, maxRAM)

	return server.RunServer(jarFile, cfg.MinRAM, maxRAM, cfg.ServerArgs)
}
