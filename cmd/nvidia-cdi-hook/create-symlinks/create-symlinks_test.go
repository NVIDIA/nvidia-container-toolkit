package symlinks

import (
	"os"
	"path/filepath"
	"testing"

	testlog "github.com/sirupsen/logrus/hooks/test"

	"github.com/stretchr/testify/require"

	"github.com/NVIDIA/nvidia-container-toolkit/internal/lookup/symlinks"
)

func TestDoesLinkExist(t *testing.T) {
	tmpDir := t.TempDir()

	require.NoError(
		t,
		makeFs(tmpDir,
			dirOrLink{path: "/a/b/c", target: "d"},
			dirOrLink{path: "/a/b/e", target: "/a/b/f"},
		),
	)

	exists, err := doesLinkExist("d", filepath.Join(tmpDir, "/a/b/c"))
	require.NoError(t, err)
	require.True(t, exists)

	exists, err = doesLinkExist("/a/b/f", filepath.Join(tmpDir, "/a/b/e"))
	require.NoError(t, err)
	require.True(t, exists)

	_, err = doesLinkExist("different-target", filepath.Join(tmpDir, "/a/b/c"))
	require.Error(t, err)

	_, err = doesLinkExist("/a/b/d", filepath.Join(tmpDir, "/a/b/c"))
	require.Error(t, err)

	exists, err = doesLinkExist("foo", filepath.Join(tmpDir, "/a/b/does-not-exist"))
	require.NoError(t, err)
	require.False(t, exists)
}

func TestCreateLinkRelativePath(t *testing.T) {
	tmpDir := t.TempDir()
	hostRoot := filepath.Join(tmpDir, "/host-root/")
	containerRoot := filepath.Join(tmpDir, "/container-root")

	require.NoError(t, makeFs(hostRoot))
	require.NoError(t, makeFs(containerRoot, dirOrLink{path: "/lib/"}))

	// nvidia-cdi-hook create-symlinks --link libfoo.so.1::/lib/libfoo.so
	err := getTestCommand().createLink(containerRoot, "libfoo.so.1", "/lib/libfoo.so")
	require.NoError(t, err)

	target, err := symlinks.Resolve(filepath.Join(containerRoot, "/lib/libfoo.so"))
	require.NoError(t, err)
	require.Equal(t, "libfoo.so.1", target)
}

func TestCreateLinkAbsolutePath(t *testing.T) {
	tmpDir := t.TempDir()
	hostRoot := filepath.Join(tmpDir, "/host-root/")
	containerRoot := filepath.Join(tmpDir, "/container-root")

	require.NoError(t, makeFs(hostRoot))
	require.NoError(t, makeFs(containerRoot, dirOrLink{path: "/lib/"}))

	// nvidia-cdi-hook create-symlinks --link /lib/libfoo.so.1::/lib/libfoo.so
	err := getTestCommand().createLink(containerRoot, "/lib/libfoo.so.1", "/lib/libfoo.so")
	require.NoError(t, err)

	target, err := symlinks.Resolve(filepath.Join(containerRoot, "/lib/libfoo.so"))
	require.NoError(t, err)
	require.Equal(t, "/lib/libfoo.so.1", target)
}

func TestCreateLinkAlreadyExists(t *testing.T) {
	tmpDir := t.TempDir()
	hostRoot := filepath.Join(tmpDir, "/host-root/")
	containerRoot := filepath.Join(tmpDir, "/container-root")

	require.NoError(t, makeFs(hostRoot))
	require.NoError(t, makeFs(containerRoot, dirOrLink{path: "/lib/libfoo.so", target: "libfoo.so.1"}))

	// nvidia-cdi-hook create-symlinks --link libfoo.so.1::/lib/libfoo.so
	err := getTestCommand().createLink(containerRoot, "libfoo.so.1", "/lib/libfoo.so")
	require.NoError(t, err)
	target, err := symlinks.Resolve(filepath.Join(containerRoot, "lib/libfoo.so"))
	require.NoError(t, err)
	require.Equal(t, "libfoo.so.1", target)
}

func TestCreateLinkAlreadyExistsDifferentTarget(t *testing.T) {
	tmpDir := t.TempDir()
	hostRoot := filepath.Join(tmpDir, "/host-root/")
	containerRoot := filepath.Join(tmpDir, "/container-root")

	require.NoError(t, makeFs(hostRoot))
	require.NoError(t, makeFs(containerRoot, dirOrLink{path: "/lib/libfoo.so", target: "different-target"}))

	// nvidia-cdi-hook create-symlinks --link libfoo.so.1::/lib/libfoo.so
	err := getTestCommand().createLink(containerRoot, "libfoo.so.1", "/lib/libfoo.so")
	require.Error(t, err)
	target, err := symlinks.Resolve(filepath.Join(containerRoot, "lib/libfoo.so"))
	require.NoError(t, err)
	require.Equal(t, "different-target", target)
}

func TestCreateLinkOutOfBounds(t *testing.T) {
	tmpDir := t.TempDir()
	hostRoot := filepath.Join(tmpDir, "/host-root/")
	containerRoot := filepath.Join(tmpDir, "/container-root")

	require.NoError(t, makeFs(hostRoot))
	require.NoError(t,
		makeFs(containerRoot,
			dirOrLink{path: "/lib"},
			dirOrLink{path: "/lib/foo", target: hostRoot},
		),
	)

	path, err := symlinks.Resolve(filepath.Join(containerRoot, "/lib/foo"))
	require.NoError(t, err)
	require.Equal(t, hostRoot, path)

	// nvidia-cdi-hook create-symlinks --link ../libfoo.so.1::/lib/foo/libfoo.so
	_ = getTestCommand().createLink(containerRoot, "../libfoo.so.1", "/lib/foo/libfoo.so")
	// TODO: We need to enabled this check once we have updated the implementation.
	// require.Error(t, err)
	_, err = os.Lstat(filepath.Join(hostRoot, "libfoo.so"))
	require.ErrorIs(t, err, os.ErrNotExist)
}

type dirOrLink struct {
	path   string
	target string
}

func makeFs(tmpdir string, fs ...dirOrLink) error {
	if err := os.MkdirAll(tmpdir, 0o755); err != nil {
		return err
	}
	for _, s := range fs {
		s.path = filepath.Join(tmpdir, s.path)
		if s.target == "" {
			_ = os.MkdirAll(s.path, 0o755)
			continue
		}
		if err := os.MkdirAll(filepath.Dir(s.path), 0o755); err != nil {
			return err
		}
		if err := os.Symlink(s.target, s.path); err != nil && !os.IsExist(err) {
			return err
		}
	}
	return nil
}

// getTestCommand creates a command for running tests against.
func getTestCommand() *command {
	logger, _ := testlog.NewNullLogger()
	return &command{
		logger: logger,
	}
}
