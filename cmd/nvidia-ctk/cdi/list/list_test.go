package list

import (
	"os"
	"path/filepath"
	"testing"

	testlog "github.com/sirupsen/logrus/hooks/test"
	"github.com/stretchr/testify/require"
	"github.com/urfave/cli/v2"
)

func TestValidateFlags(t *testing.T) {
	logger, _ := testlog.NewNullLogger()
	// Create a temporary directory for config
	tmpDir, err := os.MkdirTemp("", "nvidia-container-toolkit-test-*")
	require.NoError(t, err)
	defer os.RemoveAll(tmpDir)

	// Create a temporary config file
	configContent := `
[nvidia-container-runtime]
mode = "cdi"
[[nvidia-container-runtime.modes.cdi]]
spec-dirs = ["/etc/cdi", "/usr/local/cdi"]
`
	configPath := filepath.Join(tmpDir, "config.toml")
	err = os.WriteFile(configPath, []byte(configContent), 0600)
	require.NoError(t, err)

	// Set XDG_CONFIG_HOME to point to our temporary directory
	oldXDGConfigHome := os.Getenv("XDG_CONFIG_HOME")
	os.Setenv("XDG_CONFIG_HOME", tmpDir)
	defer os.Setenv("XDG_CONFIG_HOME", oldXDGConfigHome)

	tests := []struct {
		name          string
		cliArgs       []string
		envVars       map[string]string
		expectedDirs  []string
		expectError   bool
		errorContains string
	}{
		{
			name:         "command line takes precedence",
			cliArgs:      []string{"--spec-dir=/custom/dir1", "--spec-dir=/custom/dir2"},
			expectedDirs: []string{"/custom/dir1", "/custom/dir2"},
		},
		{
			name:         "environment variable takes precedence over config",
			envVars:      map[string]string{"NVIDIA_CTK_CDI_SPEC_DIRS": "/env/dir1:/env/dir2"},
			expectedDirs: []string{"/env/dir1", "/env/dir2"},
		},
		{
			name:         "config file used as fallback",
			expectedDirs: []string{"/etc/cdi", "/usr/local/cdi"},
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			// Set up environment variables
			for k, v := range tt.envVars {
				old := os.Getenv(k)
				os.Setenv(k, v)
				defer os.Setenv(k, old)
			}

			// Create command
			cmd := NewCommand(logger)

			// Create a new context with the command
			app := &cli.App{
				Commands: []*cli.Command{
					{
						Name:        "cdi",
						Subcommands: []*cli.Command{cmd},
					},
				},
			}

			// Run command
			args := append([]string{"nvidia-ctk", "cdi", "list"}, tt.cliArgs...)
			err := app.Run(args)

			if tt.expectError {
				require.Error(t, err)
				require.Contains(t, err.Error(), tt.errorContains)
				return
			}

			require.NoError(t, err)
		})
	}
}
