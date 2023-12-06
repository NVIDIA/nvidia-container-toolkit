package nvcdi

import (
	"fmt"
	"testing"

	"github.com/stretchr/testify/require"
)

func TestIsGpuIndex(t *testing.T) {
	testCases := []struct {
		id       string
		expected bool
	}{
		{"", false},
		{"0", true},
		{"1", true},
		{"not an integer", false},
	}
	for i, tc := range testCases {
		t.Run(fmt.Sprintf("test case %d", i), func(t *testing.T) {
			actual := identifier(tc.id).isGpuIndex()
			require.Equal(t, tc.expected, actual)
		})
	}
}

func TestIsMigIndex(t *testing.T) {
	testCases := []struct {
		id       string
		expected bool
	}{
		{"", false},
		{"0", false},
		{"not an integer", false},
		{"0:0", true},
		{"0:0:0", false},
		{"0:0.0", false},
		{"0:foo", false},
		{"foo:0", false},
	}
	for i, tc := range testCases {
		t.Run(fmt.Sprintf("test case %d", i), func(t *testing.T) {
			actual := identifier(tc.id).isMigIndex()
			require.Equal(t, tc.expected, actual)
		})
	}
}

func TestIsGpuUUID(t *testing.T) {
	testCases := []struct {
		id       string
		expected bool
	}{
		{"", false},
		{"0", false},
		{"not an integer", false},
		{"GPU-foo", false},
		{"GPU-ebd34bdf-1083-eaac-2aff-4b71a022f9bd", true},
		{"MIG-ebd34bdf-1083-eaac-2aff-4b71a022f9bd", false},
		{"ebd34bdf-1083-eaac-2aff-4b71a022f9bd", false},
	}
	for i, tc := range testCases {
		t.Run(fmt.Sprintf("test case %d", i), func(t *testing.T) {
			actual := identifier(tc.id).isGpuUUID()
			require.Equal(t, tc.expected, actual)
		})
	}
}

func TestIsMigUUID(t *testing.T) {
	testCases := []struct {
		id       string
		expected bool
	}{
		{"", false},
		{"0", false},
		{"not an integer", false},
		{"MIG-foo", false},
		{"MIG-ebd34bdf-1083-eaac-2aff-4b71a022f9bd", true},
		{"GPU-ebd34bdf-1083-eaac-2aff-4b71a022f9bd", false},
		{"ebd34bdf-1083-eaac-2aff-4b71a022f9bd", false},
	}
	for i, tc := range testCases {
		t.Run(fmt.Sprintf("test case %d", i), func(t *testing.T) {
			actual := identifier(tc.id).isMigUUID()
			require.Equal(t, tc.expected, actual)
		})
	}
}
