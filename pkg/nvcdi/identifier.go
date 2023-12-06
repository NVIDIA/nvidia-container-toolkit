package nvcdi

import (
	"strconv"
	"strings"

	"github.com/google/uuid"
)

type identifier string

// isGPUIndex checks if an identifier is a full GPU index
func (i identifier) isGpuIndex() bool {
	if _, err := strconv.ParseUint(string(i), 10, 0); err != nil {
		return false
	}
	return true
}

// isMigIndex checks if an identifier is a MIG index
func (i identifier) isMigIndex() bool {
	split := strings.SplitN(string(i), ":", 2)
	if len(split) != 2 {
		return false
	}
	for _, s := range split {
		if _, err := strconv.ParseUint(s, 10, 0); err != nil {
			return false
		}
	}
	return true
}

// isUUID checks if an identifier is a UUID
func (i identifier) isUUID() bool {
	return i.isGpuUUID() || i.isMigUUID()
}

// isGpuUUID checks if an identifier is a GPU UUID
// A GPU UUID must be of the form GPU-b1028956-cfa2-0990-bf4a-5da9abb51763
func (i identifier) isGpuUUID() bool {
	if !strings.HasPrefix(string(i), "GPU-") {
		return false
	}
	_, err := uuid.Parse(strings.TrimPrefix(string(i), "GPU-"))
	return err == nil
}

// isMigUUID checks if an identifier is a MIG UUID
// A MIG UUID can be of one of two forms:
//   - MIG-b1028956-cfa2-0990-bf4a-5da9abb51763
//   - MIG-GPU-b1028956-cfa2-0990-bf4a-5da9abb51763/3/0
func (i identifier) isMigUUID() bool {
	if !strings.HasPrefix(string(i), "MIG-") {
		return false
	}
	suffix := strings.TrimPrefix(string(i), "MIG-")
	_, err := uuid.Parse(suffix)
	if err == nil {
		return true
	}
	split := strings.SplitN(suffix, "/", 3)
	if len(split) != 3 {
		return false
	}
	if !identifier(split[0]).isGpuUUID() {
		return false
	}
	for _, s := range split[1:] {
		_, err := strconv.ParseUint(s, 10, 0)
		if err != nil {
			return false
		}
	}
	return true
}
