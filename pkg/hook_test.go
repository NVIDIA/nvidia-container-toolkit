package main

import (
	"encoding/json"
	"testing"
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
	for _, c := range tests {
		vmaj, vmin, vpatch := parseCudaVersion(c.version)
		if vmaj != c.expected[0] || vmin != c.expected[1] || vpatch != c.expected[2] {
			t.Errorf("parseCudaVersion(%s): %d.%d.%d (expected: %v)", c.version, vmaj, vmin, vpatch, c.expected)
		}
	}
}

func mustPanic(t *testing.T, f func()) {
	defer func() {
		if err := recover(); err == nil {
			t.Error("Test didn't panic!")
		}
	}()

	f()
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
		mustPanic(t, func() {
			t.Logf("parseCudaVersion(%s)", c)
			parseCudaVersion(c)
		})
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
	for _, tc := range tests {
		var spec Spec
		_ = json.Unmarshal([]byte(tc.spec), &spec)
		privileged := isPrivileged(&spec)
		if privileged != tc.expected {
			t.Errorf("isPrivileged() returned unexpectred value (privileged: %v, tc.expected: %v)", privileged, tc.expected)
		}
	}
}
