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
		return f.newStableRuntimeModifier(), nil
	case info.CSVRuntimeMode:
		return f.newCSVModifier()
	case info.CDIRuntimeMode, info.JitCDIRuntimeMode:
		return f.newCDIModifier(f.runtimeMode == info.JitCDIRuntimeMode)
	}
	return nil, fmt.Errorf("invalid runtime mode: %v", f.runtimeMode)
}

// supportedModifierTypes returns the modifiers supported for a specific runtime mode.
// The rlimits modifier applies in every mode: it is driven purely by the
// runtime config and is independent of how devices are injected.
func supportedModifierTypes(mode info.RuntimeMode) []string {
	switch mode {
	case info.CDIRuntimeMode, info.JitCDIRuntimeMode:
		// For CDI mode we make no additional device modifications.
		return []string{"nvidia-hook-remover", "mode", "rlimits"}
	case info.CSVRuntimeMode:
		// For CSV mode we support mode and feature-gated modification.
		return []string{"nvidia-hook-remover", "feature-gated", "mode", "rlimits"}
	default:
		return []string{"feature-gated", "graphics", "mode", "rlimits"}
	}
}
