package controlplane

import (
	"testing"
)

func TestCompareVersions(t *testing.T) {
	// Test cases for CompareVersions

	tests := []struct {
		version1 string
		version2 string
		expected int
	}{
		{"v1.0.0+1.abc1234", "v1.0.0+1.abc1234", 0}, // versions are equal
		{"v1.0.0+1", "v1.0.0+2", -1},                // version1 is less than version2
		{"v1.0.0+2", "v1.0.0+1", 1},                 // version1 is greater than version2
		{"v1.0.1+1", "v1.0.0+1", 1},                 // major and minor versions are equal, but patch is different
		{"v1.0.0+1", "v1.0.0+2", -1},                // major, minor, and patch are equal, but the numeric part is different
	}

	for _, test := range tests {
		result := CompareVersions(test.version1, test.version2)
		if result != test.expected {
			t.Errorf("CompareVersions(%s, %s) = %d; want %d", test.version1, test.version2, result, test.expected)
		}
	}
}

func TestParseVersionPart(t *testing.T) {
	// Test cases for ParseVersionPart

	tests := []struct {
		part     string
		expected int
	}{
		{"5", 5},      // Valid integer
		{"abc", 0},    // Invalid integer
		{"", 0},       // Empty string
		{"10abc", 10}, // Partial integer at the beginning
		{"abc10", 0},  // Partial integer at the end
	}

	for _, test := range tests {
		result := parseVersionPart(test.part)
		if result != test.expected {
			t.Errorf("parseVersionPart(%s) = %d; want %d", test.part, result, test.expected)
		}
	}
}
