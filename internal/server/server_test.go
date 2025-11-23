package server

import (
	"testing"
)

func TestCalculateSmartRAM(t *testing.T) {
	res := CalculateSmartRAM(0, 85, 2)
	if res < 2 {
		t.Errorf("CalculateSmartRAM returned %d, expected >= 2", res)
	}
}

func TestExtractJavaVersion(t *testing.T) {
	tests := []struct {
		name   string
		output string
		want   string
	}{
		{
			name:   "java 17",
			output: `openjdk version "17.0.2" 2022-01-18`,
			want:   "17.0.2",
		},
		{
			name:   "java 21",
			output: `openjdk version "21.0.1" 2023-10-17`,
			want:   "21.0.1",
		},
		{
			name:   "unknown",
			output: "some random output",
			want:   "unknown",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got := extractJavaVersion(tt.output)
			if got != tt.want {
				t.Errorf("extractJavaVersion() = %s, want %s", got, tt.want)
			}
		})
	}
}
