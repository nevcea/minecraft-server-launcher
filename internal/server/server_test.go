package server

import (
	"testing"
)

func TestCalculateMaxRAM(t *testing.T) {
	tests := []struct {
		name      string
		configMax int
		totalRAM  int
		minRAM    int
		want      int
	}{
		{"no total ram", 8, 0, 2, 8},
		{"plenty of ram", 8, 32, 2, 8},
		{"limited ram", 16, 8, 2, 6},
		{"exact fit", 8, 10, 2, 8},
		{"below min", 8, 4, 2, 2},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got := CalculateMaxRAM(tt.configMax, tt.totalRAM, tt.minRAM)
			if got != tt.want {
				t.Errorf("CalculateMaxRAM() = %d, want %d", got, tt.want)
			}
		})
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

