package lookup

import (
	"os"
	"path/filepath"
	"strings"
	"testing"

	testlog "github.com/sirupsen/logrus/hooks/test"
	"github.com/stretchr/testify/require"

	"github.com/NVIDIA/nvidia-container-toolkit/internal/ldcache"
	"github.com/NVIDIA/nvidia-container-toolkit/internal/test"
)

func TestLDCacheLookup(t *testing.T) {
	logger, _ := testlog.NewNullLogger()

	moduleRoot, err := test.GetModuleRoot()
	require.NoError(t, err)

	testCases := []struct {
		rootFs        string
		inputs        []string
		expected      string
		expectedError error
	}{
		{
			rootFs:        "rootfs-empty",
			inputs:        []string{"libcuda.so.1", "libcuda.so.*", "libcuda.so.*.*", "libcuda.so.999.88.77"},
			expectedError: ErrNotFound,
		},
		{
			rootFs: "rootfs-1",
			inputs: []string{
				"libcuda.so.1",
				"libcuda.so.*",
				"libcuda.so.*.*",
				"libcuda.so.999.88.77",
				"/lib/x86_64-linux-gnu/libcuda.so.1",
				"/lib/x86_64-linux-gnu/libcuda.so.*",
				"/lib/x86_64-linux-gnu/libcuda.so.*.*",
				"/lib/x86_64-linux-gnu/libcuda.so.999.88.77",
			},
			expected: "/lib/x86_64-linux-gnu/libcuda.so.999.88.77",
		},
		{
			rootFs: "rootfs-2",
			inputs: []string{
				"libcuda.so.1",
				"libcuda.so.*",
				"libcuda.so.*.*",
				"libcuda.so.999.88.77",
				"/var/lib/nvidia/lib64/libcuda.so.1",
				"/var/lib/nvidia/lib64/libcuda.so.*",
				"/var/lib/nvidia/lib64/libcuda.so.*.*",
				"/var/lib/nvidia/lib64/libcuda.so.999.88.77",
			},
			expected: "/var/lib/nvidia/lib64/libcuda.so.999.88.77",
		},
	}

	for _, tc := range testCases {
		for _, input := range tc.inputs {
			t.Run(tc.rootFs+" "+input, func(t *testing.T) {
				rootfs := filepath.Join(moduleRoot, "testdata", "lookup", tc.rootFs)
				l := NewFactory(
					WithLogger(logger),
					WithRoot(rootfs),
				).newLdcacheLocator()

				candidates, err := l.Locate(input)
				require.ErrorIs(t, err, tc.expectedError)
				if tc.expectedError == nil {
					require.Equal(t, []string{filepath.Join(rootfs, tc.expected)}, candidates)
				}
			})
		}
	}
}

func TestLDCacheLookupIncludes32BitLibraries(t *testing.T) {
	logger, _ := testlog.NewNullLogger()
	root := t.TempDir()

	lib64 := filepath.Join(root, "usr/lib64/libcuda.so.999.88.77")
	lib32 := filepath.Join(root, "usr/lib/libcuda.so.999.88.77")
	require.NoError(t, os.MkdirAll(filepath.Dir(lib64), 0o755))
	require.NoError(t, os.MkdirAll(filepath.Dir(lib32), 0o755))
	require.NoError(t, os.WriteFile(lib64, nil, 0o600))
	require.NoError(t, os.WriteFile(lib32, nil, 0o600))

	cache := &ldcache.LDCacheMock{
		ListFunc: func() ([]string, []string) {
			return []string{lib32}, []string{lib64}
		},
	}
	l := NewFactory(
		WithLogger(logger),
		WithRoot(root),
	).newLdcacheLocatorFrom(cache)

	candidates, err := l.Locate("libcuda.so.*")
	require.NoError(t, err)
	for i := range candidates {
		candidates[i] = strings.TrimPrefix(candidates[i], "/private")
	}
	require.Equal(t, []string{lib64, lib32}, candidates)
}
