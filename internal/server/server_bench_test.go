package server

import (
	"fmt"
	"testing"
)

const (
	javaVersionMajor = 17
	javaVersionMinor = 0
	javaVersionPatch = 2
	javaBuildNumber  = 8
)

var javaVersionOutputs = []string{
	fmt.Sprintf(`openjdk version "%d.%d.%d" 2022-01-18
OpenJDK Runtime Environment (build %d.%d.%d+%d-Ubuntu-120.04)
OpenJDK 64-Bit Server VM (build %d.%d.%d+%d-Ubuntu-120.04, mixed mode, sharing)`,
		javaVersionMajor, javaVersionMinor, javaVersionPatch,
		javaVersionMajor, javaVersionMinor, javaVersionPatch, javaBuildNumber,
		javaVersionMajor, javaVersionMinor, javaVersionPatch, javaBuildNumber),
	`java version "1.8.0_291"
Java(TM) SE Runtime Environment (build 1.8.0_291-b10)
Java HotSpot(TM) 64-Bit Server VM (build 25.291-b10, mixed mode)`,
	`openjdk version "11.0.12" 2021-07-20
OpenJDK Runtime Environment (build 11.0.12+7-Ubuntu-0ubuntu1)
OpenJDK 64-Bit Server VM (build 11.0.12+7-Ubuntu-0ubuntu1, mixed mode)`,
}

func BenchmarkExtractJavaVersion_Old(b *testing.B) {
	output := javaVersionOutputs[0]

	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		lines := []string{output}
		for _, line := range lines {
			if len(line) > 0 {
				for j := 0; j < len(line); j++ {
					if line[j] == '"' {
						break
					}
				}
			}
		}
	}
}

func BenchmarkExtractJavaVersion_New(b *testing.B) {
	output := javaVersionOutputs[0]

	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		extractJavaVersion(output)
	}
}

