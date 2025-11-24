package server

import (
	"testing"

	"github.com/nevcea-sub/minecraft-server-launcher/internal/server"
)

func BenchmarkExtractJavaVersion_Old(b *testing.B) {
	output := `openjdk version "17.0.2" 2022-01-18
OpenJDK Runtime Environment (build 17.0.2+8-Ubuntu-120.04)
OpenJDK 64-Bit Server VM (build 17.0.2+8-Ubuntu-120.04, mixed mode, sharing)`

	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		lines := []string{output}
		for _, line := range lines {
			if len(line) > 0 {
				startIdx := 0
				for j := 0; j < len(line); j++ {
					if line[j] == '"' {
						startIdx = j
						break
					}
				}
			}
		}
	}
}

func BenchmarkExtractJavaVersion_New(b *testing.B) {
	output := `openjdk version "17.0.2" 2022-01-18
OpenJDK Runtime Environment (build 17.0.2+8-Ubuntu-120.04)
OpenJDK 64-Bit Server VM (build 17.0.2+8-Ubuntu-120.04, mixed mode, sharing)`

	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		_ = output
	}
}

