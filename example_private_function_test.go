package goredis
package main

import (
	"testing"
	"path/to/your/dict" // Replace with actual import path
)

// Example of testing private functions in the same package
func TestComputeCapacity(t *testing.T) {
	testCases := []struct {
		input    int
		expected int
	}{
		{5, 16},     // minCapacity
		{16, 16},    // exact minCapacity
		{17, 32},    // next power of 2
		{1000, 1024}, // normal case
		{50000, 32768}, // maxCapacity
	}

	for _, tc := range testCases {
		result := computeCapacity(tc.input)
		if result != tc.expected {
			t.Errorf("computeCapacity(%d) = %d, expected %d", tc.input, result, tc.expected)
		}
	}
}

func TestFnv32(t *testing.T) {
	testCases := []struct {
		input string
		// You can add expected values if you want deterministic tests
	}{
		{"test"},
		{"hello"},
		{"world"},
		{""},
	}

	for _, tc := range testCases {
		result := fnv32(tc.input)
		// Test that it returns consistent results
		result2 := fnv32(tc.input)
		if result != result2 {
			t.Errorf("fnv32(%s) not consistent: %d != %d", tc.input, result, result2)
		}
		
		// Test that different inputs give different results (mostly)
		if tc.input != "" {
			different := fnv32(tc.input + "x")
			if result == different {
				t.Errorf("fnv32(%s) and fnv32(%s) produced same hash: %d", tc.input, tc.input+"x", result)
			}
		}
	}
}