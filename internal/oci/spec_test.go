package oci

import (
	"os"
	"path/filepath"
	"testing"

	"github.com/stretchr/testify/require"

	"github.com/NVIDIA/nvidia-container-toolkit/internal/test"
)

func TestMaintainSpec(t *testing.T) {
	moduleRoot, err := test.GetModuleRoot()
	require.NoError(t, err)

	files := []string{
		"config.clone3.json",
	}

	for _, f := range files {
		inputSpecPath := filepath.Join(moduleRoot, "tests/input", f)

		spec := NewFileSpec(inputSpecPath).(*fileSpec)

		_, err := spec.Load()
		require.NoError(t, err)

		outputSpecPath := filepath.Join(moduleRoot, "tests/output", f)
		spec.path = outputSpecPath
		spec.Flush()

		inputContents, err := os.ReadFile(inputSpecPath)
		require.NoError(t, err)

		outputContents, err := os.ReadFile(outputSpecPath)
		require.NoError(t, err)

		require.JSONEq(t, string(inputContents), string(outputContents))
	}
}
