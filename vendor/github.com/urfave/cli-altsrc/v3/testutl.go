package altsrc

import (
	"context"
	"os/exec"
	"path/filepath"
	"strings"
)

func MustTestdataDir(ctx context.Context) string {
	testdataDir, err := TestdataDir(ctx)
	if err != nil {
		panic(err)
	}

	return testdataDir
}

func TestdataDir(ctx context.Context) (string, error) {
	topBytes, err := exec.CommandContext(ctx, "git", "rev-parse", "--show-toplevel").Output()
	if err != nil {
		return "", err
	}

	return filepath.Join(strings.TrimSpace(string(topBytes)), ".testdata"), nil
}
