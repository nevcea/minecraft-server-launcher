package server

import (
	"context"
	"fmt"
	"os"
	"os/exec"
	"regexp"
	"strings"
	"time"

	"github.com/nevcea-sub/minecraft-server-launcher/internal/logger"
	"github.com/shirou/gopsutil/v3/mem"
)

var (
	javaVersionRegex = regexp.MustCompile(`"([^"]+)"|version\s+"?([0-9.]+)"?|(\d+\.\d+\.\d+)|(\d+)`)
)

const (
	minJavaVersion          = 17
	minJavaVersionZGC       = 11
	minRAMForZGC            = 4
	javaCmd                 = "java"
	gracefulShutdownTimeout = 30 * time.Second
)

var aikarFlags = []string{
	"-XX:+UseG1GC",
	"-XX:+ParallelRefProcEnabled",
	"-XX:MaxGCPauseMillis=200",
	"-XX:+UnlockExperimentalVMOptions",
	"-XX:+DisableExplicitGC",
	"-XX:+AlwaysPreTouch",
	"-XX:G1NewSizePercent=30",
	"-XX:G1MaxNewSizePercent=40",
	"-XX:G1HeapRegionSize=8M",
	"-XX:G1ReservePercent=20",
	"-XX:G1HeapWastePercent=5",
	"-XX:G1MixedGCCountTarget=4",
	"-XX:InitiatingHeapOccupancyPercent=15",
	"-XX:G1MixedGCLiveThresholdPercent=90",
	"-XX:G1RSetUpdatingPauseTimePercent=5",
	"-XX:SurvivorRatio=32",
	"-XX:+PerfDisableSharedMem",
	"-XX:MaxTenuringThreshold=1",
	"-Dusing.aikars.flags=https://mcflags.emc.gs",
	"-Daikars.new.flags=true",
	"-Dfile.encoding=UTF-8",
}

var zgcFlags = []string{
	"-XX:+UseZGC",
	"-XX:+ZGenerational",
	"-XX:+DisableExplicitGC",
	"-XX:+AlwaysPreTouch",
	"-XX:+PerfDisableSharedMem",
	"-Dfile.encoding=UTF-8",
}

func CheckJava(javaPath string) (string, int, error) {
	if javaPath == "" {
		javaPath = javaCmd
	}

	cmd := exec.Command(javaPath, "-version")
	output, err := cmd.CombinedOutput()
	if err != nil {
		return "", 0, fmt.Errorf("java is not installed or not found at: %s", javaPath)
	}

	versionStr := extractJavaVersion(string(output))
	if versionStr == "unknown" {
		return "", 0, fmt.Errorf("failed to parse Java version from output")
	}

	version, err := parseJavaVersion(versionStr)
	if err != nil {
		return "", 0, fmt.Errorf("failed to parse Java version: %w", err)
	}

	if version < minJavaVersion {
		return "", 0, fmt.Errorf("java %d or higher is required, found Java %d", minJavaVersion, version)
	}

	return versionStr, version, nil
}

func parseJavaVersion(versionStr string) (int, error) {
	versionStr = strings.TrimSpace(versionStr)

	parts := strings.Split(versionStr, ".")
	if len(parts) == 0 {
		return 0, fmt.Errorf("invalid version format: %s", versionStr)
	}

	majorStr := parts[0]
	if majorStr == "1" && len(parts) > 1 {
		majorStr = parts[1]
	}

	majorStr = strings.TrimFunc(majorStr, func(r rune) bool {
		return (r < '0' || r > '9')
	})

	var major int
	if _, err := fmt.Sscanf(majorStr, "%d", &major); err != nil {
		return 0, fmt.Errorf("invalid version number: %s (from %s)", majorStr, versionStr)
	}

	if major <= 0 {
		return 0, fmt.Errorf("invalid major version: %d (from %s)", major, versionStr)
	}

	return major, nil
}

func extractJavaVersion(output string) string {
	lines := strings.Split(output, "\n")
	for _, line := range lines {
		line = strings.TrimSpace(line)
		lowerLine := strings.ToLower(line)
		if !strings.Contains(lowerLine, "version") {
			continue
		}

		matches := javaVersionRegex.FindStringSubmatch(line)
		if len(matches) > 0 {
			for i := 1; i < len(matches); i++ {
				if matches[i] != "" {
					version := strings.Trim(matches[i], "\"")
					if len(version) > 0 {
						return version
					}
				}
			}
		}

		startIdx := strings.IndexByte(line, '"')
		if startIdx >= 0 {
			endIdx := strings.IndexByte(line[startIdx+1:], '"')
			if endIdx >= 0 {
				return line[startIdx+1 : startIdx+1+endIdx]
			}
		}

		parts := strings.Fields(line)
		for _, part := range parts {
			part = strings.Trim(part, "\"")
			if (strings.HasPrefix(part, "1.") || (len(part) > 0 && part[0] >= '1' && part[0] <= '9')) && len(part) > 0 {
				return part
			}
		}
	}
	return "unknown"
}

func GetSystemRAM() (totalGB int, availableGB int, err error) {
	v, err := mem.VirtualMemory()
	if err != nil {
		return 0, 0, fmt.Errorf("failed to get system memory: %w", err)
	}
	totalGB = int(v.Total / (1024 * 1024 * 1024))
	availableGB = int(v.Available / (1024 * 1024 * 1024))
	return totalGB, availableGB, nil
}

// CalculateSmartRAM determines how much RAM to allocate for the server.
// available is the system's free RAM in GB (0 if unknown).
func CalculateSmartRAM(configMax, percentage, minRAM, available int) int {
	if configMax > 0 {
		if available > 0 && configMax > available {
			logger.Warn("Configured MaxRAM (%dGB) exceeds available RAM (%dGB), adjusting", configMax, available-1)
			if safe := available - 1; safe >= minRAM {
				return safe
			}
			return minRAM
		}
		return configMax
	}

	if available <= 0 {
		return minRAM
	}

	calculated := int(float64(available) * float64(percentage) / 100.0)
	if calculated < minRAM {
		return minRAM
	}
	return calculated
}

func RunServer(ctx context.Context, jarFile string, minRAM, maxRAM int, useZGC bool, javaPath string, javaVersion int, serverArgs []string) error {
	if javaPath == "" {
		javaPath = javaCmd
	}

	args := []string{
		fmt.Sprintf("-Xms%dG", minRAM),
		fmt.Sprintf("-Xmx%dG", maxRAM),
	}

	if useZGC {
		if javaVersion < minJavaVersionZGC {
			return fmt.Errorf("ZGC requires Java %d or higher, found Java %d", minJavaVersionZGC, javaVersion)
		}

		if javaVersion < 17 {
			filteredFlags := make([]string, 0, len(zgcFlags)-1)
			for _, flag := range zgcFlags {
				if !strings.Contains(flag, "ZGenerational") {
					filteredFlags = append(filteredFlags, flag)
				}
			}
			args = append(args, filteredFlags...)
			logger.Info("Using Z Garbage Collector (ZGC) - Generational ZGC requires Java 17+")
		} else {
			args = append(args, zgcFlags...)
			logger.Info("Using Z Garbage Collector (ZGC)")
		}

		if maxRAM < minRAMForZGC {
			logger.Warn("ZGC enabled but MaxRAM < %dGB, G1GC may perform better", minRAMForZGC)
		}
	} else {
		logger.Info("Using G1 Garbage Collector (G1GC)")
		args = append(args, aikarFlags...)
	}

	args = append(args, "-jar", jarFile)
	args = append(args, serverArgs...)

	cmd := exec.Command(javaPath, args...)
	cmd.Stdin = os.Stdin
	cmd.Stdout = os.Stdout
	cmd.Stderr = os.Stderr

	if err := cmd.Start(); err != nil {
		return fmt.Errorf("failed to start server: %w", err)
	}

	done := make(chan error, 1)
	go func() {
		done <- cmd.Wait()
	}()

	select {
	case err := <-done:
		if err != nil {
			return fmt.Errorf("server stopped with error: %w", err)
		}
		return nil
	case <-ctx.Done():
		logger.Info("Stopping server...")

		if err := cmd.Process.Signal(os.Interrupt); err != nil {
			if !strings.Contains(err.Error(), "not supported by windows") {
				logger.Warn("Failed to send signal to process: %v", err)
			}
		}

		select {
		case <-done:
			return nil
		case <-time.After(gracefulShutdownTimeout):
			logger.Warn("Server did not stop in time, killing...")
			cmd.Process.Kill()
			return ctx.Err()
		}
	}
}
