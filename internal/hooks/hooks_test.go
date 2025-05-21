package hooks

import (
	"testing"
)

func TestNewHookCreator(t *testing.T) {
	tests := []struct {
		name            string
		hookPath        string
		disabledHooks   DisabledHooks
		expectedCreator HookCreator
	}{
		{
			name:     "nil disabled hooks",
			hookPath: "/usr/bin/nvidia-ctk",
			expectedCreator: &CDIHook{
				nvidiaCDIHookPath: "/usr/bin/nvidia-ctk",
				disabledHooks:     DisabledHooks{},
			},
		},
		{
			name:     "with disabled hooks",
			hookPath: "/usr/bin/nvidia-ctk",
			disabledHooks: DisabledHooks{
				EnableCudaCompat: true,
			},
			expectedCreator: &CDIHook{
				nvidiaCDIHookPath: "/usr/bin/nvidia-ctk",
				disabledHooks: DisabledHooks{
					EnableCudaCompat: true,
				},
			},
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			creator := NewHookCreator(tt.hookPath, tt.disabledHooks)
			if creator == nil {
				t.Fatal("NewHookCreator returned nil")
			}
		})
	}
}

func TestCDIHook_Create(t *testing.T) {
	tests := []struct {
		name          string
		hookPath      string
		disabledHooks DisabledHooks
		hookName      HookName
		args          []string
		expectedHook  *Hook
	}{
		{
			name:     "disabled hook returns nil",
			hookPath: "/usr/bin/nvidia-ctk",
			disabledHooks: DisabledHooks{
				EnableCudaCompat: true,
			},
			hookName:     EnableCudaCompat,
			expectedHook: nil,
		},
		{
			name:         "create symlinks with no args returns nil",
			hookPath:     "/usr/bin/nvidia-ctk",
			hookName:     CreateSymlinks,
			expectedHook: nil,
		},
		{
			name:     "create symlinks with args",
			hookPath: "/usr/bin/nvidia-ctk",
			hookName: CreateSymlinks,
			args:     []string{"/path/to/lib1", "/path/to/lib2"},
			expectedHook: &Hook{
				Lifecycle: "createContainer",
				Path:      "/usr/bin/nvidia-ctk",
				Args:      []string{"nvidia-ctk", "hook", "create-symlinks", "--link", "/path/to/lib1", "--link", "/path/to/lib2"},
			},
		},
		{
			name:     "enable cuda compat",
			hookPath: "/usr/bin/nvidia-ctk",
			hookName: EnableCudaCompat,
			expectedHook: &Hook{
				Lifecycle: "createContainer",
				Path:      "/usr/bin/nvidia-ctk",
				Args:      []string{"nvidia-ctk", "hook", "enable-cuda-compat"},
			},
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			hook := &CDIHook{
				nvidiaCDIHookPath: tt.hookPath,
				disabledHooks:     tt.disabledHooks,
			}

			result := hook.Create(tt.hookName, tt.args...)
			if tt.expectedHook == nil {
				if result != nil {
					t.Errorf("expected nil hook, got %v", result)
				}
				return
			}

			if result == nil {
				t.Fatal("expected non-nil hook, got nil")
			}

			if result.Lifecycle != tt.expectedHook.Lifecycle {
				t.Errorf("expected lifecycle %q, got %q", tt.expectedHook.Lifecycle, result.Lifecycle)
			}

			if result.Path != tt.expectedHook.Path {
				t.Errorf("expected path %q, got %q", tt.expectedHook.Path, result.Path)
			}

			if len(result.Args) != len(tt.expectedHook.Args) {
				t.Errorf("expected %d args, got %d", len(tt.expectedHook.Args), len(result.Args))
				return
			}

			for i, arg := range tt.expectedHook.Args {
				if result.Args[i] != arg {
					t.Errorf("expected arg[%d] %q, got %q", i, arg, result.Args[i])
				}
			}
		})
	}
}

func TestCDIHook_IsSupported(t *testing.T) {
	tests := []struct {
		name          string
		disabledHooks DisabledHooks
		hookName      HookName
		expected      bool
	}{
		{
			name:     "no disabled hooks",
			hookName: EnableCudaCompat,
			expected: true,
		},
		{
			name: "disabled hook",
			disabledHooks: DisabledHooks{
				EnableCudaCompat: true,
			},
			hookName: EnableCudaCompat,
			expected: false,
		},
		{
			name: "non-disabled hook",
			disabledHooks: DisabledHooks{
				EnableCudaCompat: true,
			},
			hookName: CreateSymlinks,
			expected: true,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			hook := &CDIHook{
				disabledHooks: tt.disabledHooks,
			}

			result := hook.IsSupported(tt.hookName)
			if result != tt.expected {
				t.Errorf("expected %v, got %v", tt.expected, result)
			}
		})
	}
}
