package oci

import (
	"testing"

	"github.com/stretchr/testify/require"
)

func TestGetBundleDir(t *testing.T) {
	type expected struct {
		bundle  string
		isError bool
	}
	testCases := []struct {
		argv     []string
		expected expected
	}{
		{
			argv: []string{},
			expected: expected{
				bundle: "",
			},
		},
		{
			argv: []string{"create"},
			expected: expected{
				bundle: "",
			},
		},
		{
			argv: []string{"--bundle"},
			expected: expected{
				isError: true,
			},
		},
		{
			argv: []string{"-b"},
			expected: expected{
				isError: true,
			},
		},
		{
			argv: []string{"--bundle", "/foo/bar"},
			expected: expected{
				bundle: "/foo/bar",
			},
		},
		{
			argv: []string{"--not-bundle", "/foo/bar"},
			expected: expected{
				bundle: "",
			},
		},
		{
			argv: []string{"--"},
			expected: expected{
				bundle: "",
			},
		},
		{
			argv: []string{"-bundle", "/foo/bar"},
			expected: expected{
				bundle: "/foo/bar",
			},
		},
		{
			argv: []string{"--bundle=/foo/bar"},
			expected: expected{
				bundle: "/foo/bar",
			},
		},
		{
			argv: []string{"-b=/foo/bar"},
			expected: expected{
				bundle: "/foo/bar",
			},
		},
		{
			argv: []string{"-b=/foo/=bar"},
			expected: expected{
				bundle: "/foo/=bar",
			},
		},
		{
			argv: []string{"-b", "/foo/bar"},
			expected: expected{
				bundle: "/foo/bar",
			},
		},
		{
			argv: []string{"create", "-b", "/foo/bar"},
			expected: expected{
				bundle: "/foo/bar",
			},
		},
		{
			argv: []string{"-b", "create", "create"},
			expected: expected{
				bundle: "create",
			},
		},
		{
			argv: []string{"-b=create", "create"},
			expected: expected{
				bundle: "create",
			},
		},
		{
			argv: []string{"-b", "create"},
			expected: expected{
				bundle: "create",
			},
		},
	}

	for i, tc := range testCases {
		bundle, err := GetBundleDir(tc.argv)

		if tc.expected.isError {
			require.Errorf(t, err, "%d: %v", i, tc)
		} else {
			require.NoErrorf(t, err, "%d: %v", i, tc)
		}

		require.Equalf(t, tc.expected.bundle, bundle, "%d: %v", i, tc)
	}
}

func TestGetSpecFilePathAppendsFilename(t *testing.T) {
	testCases := []struct {
		bundleDir string
		expected  string
	}{
		{
			bundleDir: "",
			expected:  "config.json",
		},
		{
			bundleDir: "/not/empty/",
			expected:  "/not/empty/config.json",
		},
		{
			bundleDir: "not/absolute",
			expected:  "not/absolute/config.json",
		},
	}

	for i, tc := range testCases {
		specPath := GetSpecFilePath(tc.bundleDir)

		require.Equalf(t, tc.expected, specPath, "%d: %v", i, tc)
	}
}

func TestHasCreateSubcommand(t *testing.T) {
	testCases := []struct {
		args         []string
		shouldModify bool
	}{
		{
			shouldModify: false,
		},
		{
			args:         []string{"create"},
			shouldModify: true,
		},
		{
			args:         []string{"--bundle=create"},
			shouldModify: false,
		},
		{
			args:         []string{"--bundle", "create"},
			shouldModify: false,
		},
		{
			args:         []string{"create"},
			shouldModify: true,
		},
	}

	for i, tc := range testCases {
		require.Equal(t, tc.shouldModify, HasCreateSubcommand(tc.args), "%d: %v", i, tc)
	}
}
