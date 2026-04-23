package update

import (
	"testing"
)

func BenchmarkNormalizeVersion(b *testing.B) {
	versions := []string{"v1.0.0", "1.0.0", " 1.0.0 ", "v2.1.3", "3.0", "v10.20.30"}
	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		for _, v := range versions {
			_ = normalizeVersion(v)
		}
	}
}

func BenchmarkCompareVersions(b *testing.B) {
	pairs := [][2]string{
		{"1.0.0", "1.0.0"}, {"1.0.1", "1.0.0"}, {"1.1.0", "1.0.0"},
		{"2.0.0", "1.0.0"}, {"1.2.3", "1.2.4"}, {"2.0.0", "1.9.9"},
	}
	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		for _, p := range pairs {
			_ = compareVersions(p[0], p[1])
		}
	}
}
