package modifier

import (
	"fmt"

	"github.com/NVIDIA/nvidia-container-toolkit/internal/info"
	"github.com/NVIDIA/nvidia-container-toolkit/internal/oci"
)

// newModeModifier creates a modifier for the configure runtime mode.
func (f *Factory) newModeModifier() (oci.SpecModifier, error) {
	switch f.runtimeMode {
	case info.LegacyRuntimeMode:
		return f.NewStableRuntimeModifier(), nil
	case info.CSVRuntimeMode:
		return f.NewCSVModifier()
	case info.CDIRuntimeMode, info.JitCDIRuntimeMode:
		return f.NewCDIModifier(f.runtimeMode == info.JitCDIRuntimeMode)
	}
	return nil, fmt.Errorf("invalid runtime mode: %v", f.runtimeMode)
}

// supportedModifierTypes returns the modifiers supported for a specific runtime mode.
func supportedModifierTypes(mode info.RuntimeMode) []string {
	switch mode {
	case info.CDIRuntimeMode, info.JitCDIRuntimeMode:
		// For CDI mode we make no additional modifications.
		return []string{"nvidia-hook-remover", "mode"}
	case info.CSVRuntimeMode:
		// For CSV mode we support mode and feature-gated modification.
		return []string{"nvidia-hook-remover", "feature-gated", "mode"}
	default:
		return []string{"feature-gated", "graphics", "mode"}
	}
}
