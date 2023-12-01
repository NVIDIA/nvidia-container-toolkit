package main

import (
	"bytes"
	"encoding/json"
	"io"
	"log"
	"os"
	"os/exec"
	"path/filepath"
	"strings"
	"testing"

	"github.com/opencontainers/runtime-spec/specs-go"
	testlog "github.com/sirupsen/logrus/hooks/test"
	"github.com/stretchr/testify/require"

	"github.com/NVIDIA/nvidia-container-toolkit/internal/modifier"
	"github.com/NVIDIA/nvidia-container-toolkit/internal/test"
)

const (
	nvidiaRuntime            = "nvidia-container-runtime"
	nvidiaHook               = "nvidia-container-runtime-hook"
	bundlePathSuffix         = "test/output/bundle/"
	specFile                 = "config.json"
	unmodifiedSpecFileSuffix = "test/input/test_spec.json"
)

const (
	runcExecutableName = "runc"
)

type testConfig struct {
	root    string
	binPath string
}

var cfg *testConfig

func TestMain(m *testing.M) {
	// TEST SETUP
	// Determine the module root and the test binary path
	var err error
	moduleRoot, err := test.GetModuleRoot()
	if err != nil {
		log.Fatalf("error in test setup: could not get module root: %v", err)
	}
	testBinPath := filepath.Join(moduleRoot, "test", "bin")
	testInputPath := filepath.Join(moduleRoot, "test", "input")

	// Set the environment variables for the test
	os.Setenv("PATH", test.PrependToPath(testBinPath, moduleRoot))
	os.Setenv("XDG_CONFIG_HOME", testInputPath)

	// Confirm that the environment is configured correctly
	runcPath, err := exec.LookPath(runcExecutableName)
	if err != nil || filepath.Join(testBinPath, runcExecutableName) != runcPath {
		log.Fatalf("error in test setup: mock runc path set incorrectly in TestMain(): %v", err)
	}
	hookPath, err := exec.LookPath(nvidiaHook)
	if err != nil || filepath.Join(testBinPath, nvidiaHook) != hookPath {
		log.Fatalf("error in test setup: mock hook path set incorrectly in TestMain(): %v", err)
	}

	// Store the root and binary paths in the test Config
	cfg = &testConfig{
		root:    moduleRoot,
		binPath: testBinPath,
	}

	// RUN TESTS
	exitCode := m.Run()

	// TEST CLEANUP
	os.Remove(specFile)

	os.Exit(exitCode)
}

// case 1) nvidia-container-runtime run --bundle
// case 2) nvidia-container-runtime create --bundle
//   - Confirm the runtime handles bad input correctly
func TestBadInput(t *testing.T) {
	err := cfg.generateNewRuntimeSpec()
	if err != nil {
		t.Fatal(err)
	}

	//nolint:gosec // TODO: Can we harden this so that there is less risk of command injection
	cmdCreate := exec.Command(nvidiaRuntime, "create", "--bundle")
	t.Logf("executing: %s\n", strings.Join(cmdCreate.Args, " "))
	err = cmdCreate.Run()
	require.Error(t, err, "runtime should return an error")
}

// case 1) nvidia-container-runtime run --bundle <bundle-name> <ctr-name>
//   - Confirm the runtime runs with no errors
//
// case 2) nvidia-container-runtime create --bundle <bundle-name> <ctr-name>
//   - Confirm the runtime inserts the NVIDIA prestart hook correctly
func TestGoodInput(t *testing.T) {
	err := cfg.generateNewRuntimeSpec()
	if err != nil {
		t.Fatalf("error generating runtime spec: %v", err)
	}

	//nolint:gosec // TODO: Can we harden this so that there is less risk of command injection
	cmdRun := exec.Command(nvidiaRuntime, "run", "--bundle", cfg.bundlePath(), "testcontainer")
	t.Logf("executing: %s\n", strings.Join(cmdRun.Args, " "))
	output, err := cmdRun.CombinedOutput()
	require.NoErrorf(t, err, "runtime should not return an error", "output=%v", string(output))

	// Check config.json and confirm there are no hooks
	spec, err := cfg.getRuntimeSpec()
	require.NoError(t, err, "should be no errors when reading and parsing spec from config.json")
	require.Empty(t, spec.Hooks, "there should be no hooks in config.json")

	//nolint:gosec // TODO: Can we harden this so that there is less risk of command injection
	cmdCreate := exec.Command(nvidiaRuntime, "create", "--bundle", cfg.bundlePath(), "testcontainer")
	t.Logf("executing: %s\n", strings.Join(cmdCreate.Args, " "))
	err = cmdCreate.Run()
	require.NoError(t, err, "runtime should not return an error")

	// Check config.json for NVIDIA prestart hook
	spec, err = cfg.getRuntimeSpec()
	require.NoError(t, err, "should be no errors when reading and parsing spec from config.json")
	require.NotEmpty(t, spec.Hooks, "there should be hooks in config.json")
	require.Equal(t, 1, nvidiaHookCount(spec.Hooks), "exactly one nvidia prestart hook should be inserted correctly into config.json")
}

// NVIDIA prestart hook already present in config file
func TestDuplicateHook(t *testing.T) {
	err := cfg.generateNewRuntimeSpec()
	if err != nil {
		t.Fatal(err)
	}

	var spec specs.Spec
	spec, err = cfg.getRuntimeSpec()
	if err != nil {
		t.Fatal(err)
	}

	t.Logf("inserting nvidia prestart hook to config.json")
	if err = addNVIDIAHook(&spec); err != nil {
		t.Fatal(err)
	}

	jsonOutput, err := json.MarshalIndent(spec, "", "\t")
	if err != nil {
		t.Fatal(err)
	}

	jsonFile, err := os.OpenFile(cfg.specFilePath(), os.O_RDWR, 0644)
	if err != nil {
		t.Fatal(err)
	}
	_, err = jsonFile.WriteAt(jsonOutput, 0)
	if err != nil {
		t.Fatal(err)
	}

	// Test how runtime handles already existing prestart hook in config.json
	//nolint:gosec // TODO: Can we harden this so that there is less risk of command injection
	cmdCreate := exec.Command(nvidiaRuntime, "create", "--bundle", cfg.bundlePath(), "testcontainer")
	t.Logf("executing: %s\n", strings.Join(cmdCreate.Args, " "))
	output, err := cmdCreate.CombinedOutput()
	require.NoErrorf(t, err, "runtime should not return an error", "output=%v", string(output))

	// Check config.json for NVIDIA prestart hook
	spec, err = cfg.getRuntimeSpec()
	require.NoError(t, err, "should be no errors when reading and parsing spec from config.json")
	require.NotEmpty(t, spec.Hooks, "there should be hooks in config.json")
	require.Equal(t, 1, nvidiaHookCount(spec.Hooks), "exactly one nvidia prestart hook should be inserted correctly into config.json")
}

// addNVIDIAHook is a basic wrapper for an addHookModifier that is used for
// testing.
func addNVIDIAHook(spec *specs.Spec) error {
	logger, _ := testlog.NewNullLogger()
	m := modifier.NewStableRuntimeModifier(logger, nvidiaHook)
	return m.Modify(spec)
}

func (c testConfig) getRuntimeSpec() (specs.Spec, error) {
	filePath := c.specFilePath()

	var spec specs.Spec
	jsonFile, err := os.OpenFile(filePath, os.O_RDWR, 0644)
	if err != nil {
		return spec, err
	}
	defer jsonFile.Close()

	jsonContent, err := io.ReadAll(jsonFile)
	switch {
	case err != nil:
		return spec, err
	case json.Valid(jsonContent):
		err = json.Unmarshal(jsonContent, &spec)
		if err != nil {
			return spec, err
		}
	default:
		err = json.NewDecoder(bytes.NewReader(jsonContent)).Decode(&spec)
		if err != nil {
			return spec, err
		}
	}

	return spec, err
}

func (c testConfig) bundlePath() string {
	return filepath.Join(c.root, bundlePathSuffix)
}

func (c testConfig) specFilePath() string {
	return filepath.Join(c.bundlePath(), specFile)
}

func (c testConfig) unmodifiedSpecFile() string {
	return filepath.Join(c.root, unmodifiedSpecFileSuffix)
}

func (c testConfig) generateNewRuntimeSpec() error {
	var err error

	err = os.MkdirAll(c.bundlePath(), 0755)
	if err != nil {
		return err
	}

	//nolint:gosec // TODO: Can we harden this so that there is less risk of command injection
	cmd := exec.Command("cp", c.unmodifiedSpecFile(), c.specFilePath())
	err = cmd.Run()
	if err != nil {
		return err
	}
	return nil
}

// Return number of valid NVIDIA prestart hooks in runtime spec
func nvidiaHookCount(hooks *specs.Hooks) int {
	if hooks == nil {
		return 0
	}

	count := 0
	for _, hook := range hooks.Prestart {
		if strings.Contains(hook.Path, nvidiaHook) {
			count++
		}
	}
	return count
}
