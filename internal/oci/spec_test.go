package oci

import (
	"fmt"
	"os"
	"path/filepath"
	"runtime"
	"testing"

	"github.com/stretchr/testify/require"
)

func TestMaintainSpec(t *testing.T) {
	moduleRoot, err := getModuleRoot()
	require.NoError(t, err)

	files := []string{
		"config.clone3.json",
	}

	for _, f := range files {
		inputSpecPath := filepath.Join(moduleRoot, "test/input", f)

		spec := NewSpecFromFile(inputSpecPath).(*fileSpec)

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

func getModuleRoot() (string, error) {
	_, filename, _, _ := runtime.Caller(0)

	return hasGoMod(filename)
}

func hasGoMod(dir string) (string, error) {
	if dir == "" || dir == "/" {
		return "", fmt.Errorf("module root not found")
	}

	_, err := os.Stat(filepath.Join(dir, "go.mod"))
	if err != nil {
		return hasGoMod(filepath.Dir(dir))
	}
	return dir, nil
}
