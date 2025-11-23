package server

import (
	"fmt"
	"os"
	"os/exec"
	"runtime"
	"strings"

	"github.com/shirou/gopsutil/v3/mem"
)

const (
	minJavaVersion = 17
	javaCmd        = "java"
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

func CheckJava() (string, error) {
	cmd := exec.Command(javaCmd, "-version")
	output, err := cmd.CombinedOutput()
	if err != nil {
		return "", fmt.Errorf("Java is not installed or not in PATH")
	}

	version := extractJavaVersion(string(output))
	return version, nil
}

func extractJavaVersion(output string) string {
	lines := strings.Split(output, "\n")
	for _, line := range lines {
		if strings.Contains(line, "version") {
			parts := strings.Split(line, "\"")
			if len(parts) >= 2 {
				return parts[1]
			}
		}
	}
	return "unknown"
}

func GetTotalRAMGB() int {
	if runtime.GOOS == "windows" || runtime.GOOS == "linux" || runtime.GOOS == "darwin" {
		v, err := mem.VirtualMemory()
		if err == nil {
			return int(v.Total / (1024 * 1024 * 1024))
		}
	}
	return 0
}

func CalculateMaxRAM(configMax, totalRAM, minRAM int) int {
	if totalRAM <= 0 {
		return configMax
	}

	reserved := 2
	if totalRAM > 16 {
		reserved = 4
	}

	available := totalRAM - reserved
	if available < minRAM {
		return minRAM
	}

	if configMax > available {
		return available
	}

	return configMax
}

func RunServer(jarFile string, minRAM, maxRAM int, serverArgs []string) error {
	args := []string{
		fmt.Sprintf("-Xms%dG", minRAM),
		fmt.Sprintf("-Xmx%dG", maxRAM),
	}

	args = append(args, aikarFlags...)
	args = append(args, "-jar", jarFile)
	args = append(args, serverArgs...)

	cmd := exec.Command(javaCmd, args...)
	cmd.Stdin = os.Stdin
	cmd.Stdout = os.Stdout
	cmd.Stderr = os.Stderr

	if err := cmd.Run(); err != nil {
		return fmt.Errorf("server stopped with error: %w", err)
	}

	return nil
}

