package main

import (
	"encoding/json"
	"testing"

	"github.com/stretchr/testify/require"
)

func TestParseCudaVersionValid(t *testing.T) {
	var tests = []struct {
		version  string
		expected [3]uint32
	}{
		{"0", [3]uint32{0, 0, 0}},
		{"8", [3]uint32{8, 0, 0}},
		{"7.5", [3]uint32{7, 5, 0}},
		{"9.0.116", [3]uint32{9, 0, 116}},
		{"4294967295.4294967295.4294967295", [3]uint32{4294967295, 4294967295, 4294967295}},
	}
	for i, c := range tests {
		vmaj, vmin, vpatch := parseCudaVersion(c.version)

		version := [3]uint32{vmaj, vmin, vpatch}

		require.Equal(t, c.expected, version, "%d: %v", i, c)
	}
}

func TestParseCudaVersionInvalid(t *testing.T) {
	var tests = []string{
		"foo",
		"foo.5.10",
		"9.0.116.50",
		"9.0.116foo",
		"7.foo",
		"9.0.bar",
		"9.4294967296",
		"9.0.116.",
		"9..0",
		"9.",
		".5.10",
		"-9",
		"+9",
		"-9.1.116",
		"-9.-1.-116",
	}
	for _, c := range tests {
		require.Panics(t, func() {
			parseCudaVersion(c)
		}, "parseCudaVersion(%v)", c)
	}
}

func TestIsPrivileged(t *testing.T) {
	var tests = []struct {
		spec     string
		expected bool
	}{
		{
			`
			{
				"ociVersion": "1.0.0",
				"process": {
					"capabilities": {
						"bounding": [ "CAP_SYS_ADMIN" ]
					}
				}
			}
			`,
			true,
		},
		{
			`
			{
				"ociVersion": "1.0.0",
				"process": {
					"capabilities": {
						"bounding": [ "CAP_SYS_OTHER" ]
					}
				}
			}
			`,
			false,
		},
		{
			`
			{
				"ociVersion": "1.0.0",
				"process": {}
			}
			`,
			false,
		},
		{
			`
			{
				"ociVersion": "1.0.0-rc2-dev",
				"process": {
					"capabilities": [ "CAP_SYS_ADMIN" ]
				}
			}
			`,
			true,
		},
		{
			`
			{
				"ociVersion": "1.0.0-rc2-dev",
				"process": {
					"capabilities": [ "CAP_SYS_OTHER" ]
				}
			}
			`,
			false,
		},
		{
			`
			{
				"ociVersion": "1.0.0-rc2-dev",
				"process": {}
			}
			`,
			false,
		},
	}
	for i, tc := range tests {
		var spec Spec
		_ = json.Unmarshal([]byte(tc.spec), &spec)
		privileged := isPrivileged(&spec)

		require.Equal(t, tc.expected, privileged, "%d: %v", i, tc)
	}
}
