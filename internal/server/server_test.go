package server

import (
	"testing"
)

func TestExtractJavaVersion(t *testing.T) {
	tests := []struct {
		name   string
		output string
		want   string
	}{
		{"java 17", `openjdk version "17.0.2" 2022-01-18`, "17.0.2"},
		{"java 21", `openjdk version "21.0.1" 2023-10-17`, "21.0.1"},
		{"unknown", "some random output", "unknown"},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			if got := extractJavaVersion(tt.output); got != tt.want {
				t.Errorf("extractJavaVersion() = %s, want %s", got, tt.want)
			}
		})
	}
}

func TestParseJavaVersion(t *testing.T) {
	tests := []struct {
		version   string
		want      int
		wantError bool
	}{
		{"17.0.2", 17, false},
		{"21.0.1", 21, false},
		{"1.8.0_352", 8, false},
		{"11.0.19", 11, false},
		{"invalid", 0, true},
		{"", 0, true},
	}

	for _, tt := range tests {
		t.Run(tt.version, func(t *testing.T) {
			got, err := parseJavaVersion(tt.version)
			if (err != nil) != tt.wantError {
				t.Errorf("parseJavaVersion(%q) error = %v, wantError %v", tt.version, err, tt.wantError)
			}
			if got != tt.want {
				t.Errorf("parseJavaVersion(%q) = %d, want %d", tt.version, got, tt.want)
			}
		})
	}
}
