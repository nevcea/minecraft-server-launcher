package server

import (
	"fmt"
	"os"
	"os/exec"
	"os/signal"
	"strings"
	"syscall"

	"github.com/shirou/gopsutil/v3/mem"
)

const (
	minJavaVersion    = 17
	minJavaVersionZGC = 11
	javaCmd           = "java"
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
		return "", 0, fmt.Errorf("Java is not installed or not found at: %s", javaPath)
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
		return "", 0, fmt.Errorf("Java %d or higher is required, found Java %d", minJavaVersion, version)
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
		if strings.Contains(strings.ToLower(line), "version") {
			startIdx := strings.Index(line, "\"")
			if startIdx >= 0 {
				endIdx := strings.Index(line[startIdx+1:], "\"")
				if endIdx >= 0 {
					return line[startIdx+1 : startIdx+1+endIdx]
				}
			}
			parts := strings.Fields(line)
			for _, part := range parts {
				if strings.HasPrefix(part, "1.") || (len(part) > 0 && part[0] >= '1' && part[0] <= '9') {
					part = strings.Trim(part, "\"")
					if len(part) > 0 {
						return part
					}
				}
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

func CalculateSmartRAM(configMax, percentage, minRAM int) int {
	_, available, err := GetSystemRAM()
	if err != nil {
		if configMax > 0 {
			return configMax
		}
		return minRAM + 2
	}

	if configMax > 0 {
		if configMax > available {
			fmt.Fprintf(os.Stderr, "[WARN] Configured MaxRAM (%dGB) exceeds available RAM (%dGB), adjusting to safe limit\n", configMax, available-1)
			safe := available - 1
			if safe < minRAM {
				return minRAM
			}
			return safe
		}
		return configMax
	}

	calculated := int(float64(available) * (float64(percentage) / 100.0))

	if available-calculated < 1 {
		calculated = available - 1
	}

	if calculated < minRAM {
		return minRAM
	}

	return calculated
}

func RunServer(jarFile string, minRAM, maxRAM int, useZGC bool, javaPath string, javaVersion int, serverArgs []string) error {
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
		if javaVersion < 17 && strings.Contains(strings.Join(zgcFlags, " "), "ZGenerational") {
			var filteredFlags []string
			for _, flag := range zgcFlags {
				if !strings.Contains(flag, "ZGenerational") {
					filteredFlags = append(filteredFlags, flag)
				}
			}
			args = append(args, filteredFlags...)
			fmt.Println("[INFO] Using Z Garbage Collector (ZGC) - Generational ZGC requires Java 17+")
		} else {
			args = append(args, zgcFlags...)
			fmt.Println("[INFO] Using Z Garbage Collector (ZGC)")
		}

		if maxRAM < 4 {
			fmt.Fprintf(os.Stderr, "[WARN] ZGC enabled but MaxRAM < 4GB, G1GC may perform better\n")
		}
	} else {
		fmt.Println("[INFO] Using G1 Garbage Collector (G1GC)")
		args = append(args, aikarFlags...)
	}

	args = append(args, "-jar", jarFile)
	args = append(args, serverArgs...)

	cmd := exec.Command(javaPath, args...)
	cmd.Stdin = os.Stdin
	cmd.Stdout = os.Stdout
	cmd.Stderr = os.Stderr

	sigChan := make(chan os.Signal, 1)
	signal.Notify(sigChan, os.Interrupt, syscall.SIGTERM)

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
	case sig := <-sigChan:
		fmt.Printf("\n[INFO] Received signal: %v, shutting down server...\n", sig)
		if err := cmd.Process.Signal(sig); err != nil {
			cmd.Process.Kill()
		}
		<-done
		return fmt.Errorf("server stopped by signal: %v", sig)
	}
}
