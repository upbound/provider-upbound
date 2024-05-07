package controlplane

import (
	"testing"
)

func TestCompareVersionsWithBuildMetadata(t *testing.T) {
	// Test cases
	testCases := []struct {
		version1      string
		version2      string
		expectedValue int
	}{
		{"v0.0.0+13.7104e10", "v0.0.0+13.7104e10", 0},
		{"v0.0.0+14.6b092f3", "v0.0.0+13.7104e10", 1},
		{"v0.0.0+13.7104e10", "v0.0.0+14.6b092f3", -1},
		{"v0.0.0+10.7104e10", "v0.0.0+9.6b092f3", 1},
	}

	for _, tc := range testCases {
		t.Run(tc.version1+"-"+tc.version2, func(t *testing.T) {
			result := CompareVersions(tc.version1, tc.version2)
			if result != tc.expectedValue {
				t.Errorf("Expected %d, but got %d for %s vs %s", tc.expectedValue, result, tc.version1, tc.version2)
			}
		})
	}
}
