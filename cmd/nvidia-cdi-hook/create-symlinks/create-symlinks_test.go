package symlinks

import (
	"os"
	"path/filepath"
	"strings"
	"testing"

	testlog "github.com/sirupsen/logrus/hooks/test"
	"github.com/stretchr/testify/require"

	"github.com/NVIDIA/nvidia-container-toolkit/internal/lookup/symlinks"
)

func TestLinkExist(t *testing.T) {
	tmpDir := t.TempDir()
	require.NoError(
		t,
		makeFs(tmpDir,
			dirOrLink{path: "/a/b/c", target: "d"},
			dirOrLink{path: "/a/b/e", target: "/a/b/f"},
		),
	)

	exists, err := linkExists("d", filepath.Join(tmpDir, "/a/b/c"))
	require.NoError(t, err)
	require.True(t, exists)

	exists, err = linkExists("/a/b/f", filepath.Join(tmpDir, "/a/b/e"))
	require.NoError(t, err)
	require.True(t, exists)

	exists, err = linkExists("different-target", filepath.Join(tmpDir, "/a/b/c"))
	require.NoError(t, err)
	require.False(t, exists)

	exists, err = linkExists("/a/b/d", filepath.Join(tmpDir, "/a/b/c"))
	require.NoError(t, err)
	require.False(t, exists)

	exists, err = linkExists("foo", filepath.Join(tmpDir, "/a/b/does-not-exist"))
	require.NoError(t, err)
	require.False(t, exists)
}

func TestCreateLink(t *testing.T) {
	type link struct {
		path   string
		target string
	}
	type expectedLink struct {
		link
		err error
	}

	testCases := []struct {
		description         string
		containerContents   []dirOrLink
		link                link
		expectedCreateError error
		expectedLinks       []expectedLink
	}{
		{
			description: "link to / resolves to container root",
			containerContents: []dirOrLink{
				{path: "/lib/foo", target: "/"},
			},
			link: link{
				path:   "/lib/foo/libfoo.so",
				target: "libfoo.so.1",
			},
			expectedLinks: []expectedLink{
				{
					link: link{
						path:   "{{ .containerRoot }}/libfoo.so",
						target: "libfoo.so.1",
					},
				},
			},
		},
		{
			description: "link to / resolves to container root; parent relative link",
			containerContents: []dirOrLink{
				{path: "/lib/foo", target: "/"},
			},
			link: link{
				path:   "/lib/foo/libfoo.so",
				target: "../libfoo.so.1",
			},
			expectedLinks: []expectedLink{
				{
					link: link{
						path:   "{{ .containerRoot }}/libfoo.so",
						target: "../libfoo.so.1",
					},
				},
			},
		},
		{
			description: "link to / resolves to container root; absolute link",
			containerContents: []dirOrLink{
				{path: "/lib/foo", target: "/"},
			},
			link: link{
				path:   "/lib/foo/libfoo.so",
				target: "/a-path-in-container/foo/libfoo.so.1",
			},
			expectedLinks: []expectedLink{
				{
					link: link{
						path:   "{{ .containerRoot }}/libfoo.so",
						target: "/a-path-in-container/foo/libfoo.so.1",
					},
				},
				{
					// We also check that the target is NOT created.
					link: link{
						path: "{{ .containerRoot }}/a-path-in-container/foo/libfoo.so.1",
					},
					err: os.ErrNotExist,
				},
			},
		},
	}

	for _, tc := range testCases {
		t.Run(tc.description, func(t *testing.T) {
			tmpDir := t.TempDir()
			hostRoot := filepath.Join(tmpDir, "/host-root/")
			containerRoot := filepath.Join(tmpDir, "/container-root")

			require.NoError(t, makeFs(hostRoot))
			require.NoError(t, makeFs(containerRoot, tc.containerContents...))

			// nvidia-cdi-hook create-symlinks --link linkSpec
			err := getTestCommand().createLink(containerRoot, tc.link.target, tc.link.path)
			// TODO: We may be able to replace this with require.ErrorIs.
			if tc.expectedCreateError != nil {
				require.Error(t, err)
			} else {
				require.NoError(t, err)
			}

			for _, expectedLink := range tc.expectedLinks {
				path := strings.ReplaceAll(expectedLink.path, "{{ .containerRoot }}", containerRoot)
				path = strings.ReplaceAll(path, "{{ .hostRoot }}", hostRoot)
				if expectedLink.target != "" {
					target, err := symlinks.Resolve(path)
					require.ErrorIs(t, err, expectedLink.err)
					require.Equal(t, expectedLink.target, target)
				} else {
					_, err := os.Stat(path)
					require.ErrorIs(t, err, expectedLink.err)
				}
			}
		})
	}
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
	testCases := []struct {
		description       string
		containerContents []dirOrLink
		shouldExist       []string
	}{
		{
			description:       "link already exists with correct target",
			containerContents: []dirOrLink{{path: "/lib/libfoo.so", target: "libfoo.so.1"}},
			shouldExist:       []string{},
		},
		{
			description:       "link already exists with different target",
			containerContents: []dirOrLink{{path: "/lib/libfoo.so", target: "different-target"}, {path: "different-target"}},
			shouldExist:       []string{"{{ .containerRoot }}/different-target"},
		},
	}

	for _, tc := range testCases {
		t.Run(tc.description, func(t *testing.T) {
			tmpDir := t.TempDir()
			hostRoot := filepath.Join(tmpDir, "/host-root/")
			containerRoot := filepath.Join(tmpDir, "/container-root")
			require.NoError(t, makeFs(hostRoot))
			require.NoError(t, makeFs(containerRoot, tc.containerContents...))

			// nvidia-cdi-hook create-symlinks --link libfoo.so.1::/lib/libfoo.so
			err := getTestCommand().createLink(containerRoot, "libfoo.so.1", "/lib/libfoo.so")
			require.NoError(t, err)
			target, err := symlinks.Resolve(filepath.Join(containerRoot, "lib/libfoo.so"))
			require.NoError(t, err)
			require.Equal(t, "libfoo.so.1", target)

			for _, p := range tc.shouldExist {
				require.DirExists(t, strings.ReplaceAll(p, "{{ .containerRoot }}", containerRoot))
			}
		})
	}
}

func TestCreateLinkOutOfBounds(t *testing.T) {
	tmpDir := t.TempDir()
	hostRoot := filepath.Join(tmpDir, "/host-root")
	containerRoot := filepath.Join(tmpDir, "/container-root")

	require.NoError(t,
		makeFs(hostRoot,
			dirOrLink{path: "libfoo.so"},
		),
	)
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
	require.NoError(t, err)

	target, err := symlinks.Resolve(filepath.Join(containerRoot, hostRoot, "libfoo.so"))
	require.NoError(t, err)
	require.Equal(t, "../libfoo.so.1", target)

	require.DirExists(t, filepath.Join(hostRoot, "libfoo.so"))
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
