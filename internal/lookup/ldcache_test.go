package lookup

import (
	"path/filepath"
	"testing"

	testlog "github.com/sirupsen/logrus/hooks/test"
	"github.com/stretchr/testify/require"

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
				l := NewLdcacheLocator(
					WithLogger(logger),
					WithRoot(rootfs),
				)

				candidates, err := l.Locate(input)
				require.ErrorIs(t, err, tc.expectedError)
				if tc.expectedError == nil {
					require.Equal(t, []string{filepath.Join(rootfs, tc.expected)}, candidates)
				}
			})
		}
	}
}
