package oci

import (
	"os"
	"path/filepath"
	"testing"

	"github.com/NVIDIA/nvidia-container-toolkit/internal/test"
	"github.com/stretchr/testify/require"
)

func TestMaintainSpec(t *testing.T) {
	moduleRoot, err := test.GetModuleRoot()
	require.NoError(t, err)

	files := []string{
		"config.clone3.json",
	}

	for _, f := range files {
		inputSpecPath := filepath.Join(moduleRoot, "test/input", f)

		spec := NewFileSpec(inputSpecPath).(*fileSpec)

		spec.Load()

		outputSpecPath := filepath.Join(moduleRoot, "test/output", f)
		spec.path = outputSpecPath
		spec.Flush()

		inputContents, err := os.ReadFile(inputSpecPath)
		require.NoError(t, err)

		outputContents, err := os.ReadFile(outputSpecPath)
		require.NoError(t, err)

		require.JSONEq(t, string(inputContents), string(outputContents))
	}
}
