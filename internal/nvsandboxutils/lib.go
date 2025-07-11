/**
# Copyright 2024 NVIDIA CORPORATION
#
# Licensed under the Apache License, Version 2.0 (the "License");
# you may not use this file except in compliance with the License.
# You may obtain a copy of the License at
#
#     http://www.apache.org/licenses/LICENSE-2.0
#
# Unless required by applicable law or agreed to in writing, software
# distributed under the License is distributed on an "AS IS" BASIS,
# WITHOUT WARRANTIES OR CONDITIONS OF ANY KIND, either express or implied.
# See the License for the specific language governing permissions and
# limitations under the License.
**/

package nvsandboxutils

import (
	"errors"
	"fmt"
	"sync"

	"github.com/NVIDIA/go-nvml/pkg/dl"
)

const (
	defaultNvSandboxUtilsLibraryName      = "libnvidia-sandboxutils.so.1"
	defaultNvSandboxUtilsLibraryLoadFlags = dl.RTLD_LAZY | dl.RTLD_GLOBAL
)

var errLibraryNotLoaded = errors.New("library not loaded")
var errLibraryAlreadyLoaded = errors.New("library already loaded")

// dynamicLibrary is an interface for abstacting the underlying library.
// This also allows for mocking and testing.

//go:generate moq -rm -fmt=goimports -stub -out dynamicLibrary_mock.go . dynamicLibrary
type dynamicLibrary interface {
	Lookup(string) error
	Open() error
	Close() error
}

// library represents an nvsandboxutils library.
// This includes a reference to the underlying DynamicLibrary
type library struct {
	sync.Mutex
	path     string
	refcount refcount
	dl       dynamicLibrary
}

// libnvsandboxutils is a global instance of the nvsandboxutils library.
var libnvsandboxutils = newLibrary()

func New(opts ...LibraryOption) Interface {
	return newLibrary(opts...)
}

func newLibrary(opts ...LibraryOption) *library {
	l := &library{}
	l.init(opts...)
	return l
}

func (l *library) init(opts ...LibraryOption) {
	o := libraryOptions{}
	for _, opt := range opts {
		opt(&o)
	}

	if o.path == "" {
		o.path = defaultNvSandboxUtilsLibraryName
	}
	if o.flags == 0 {
		o.flags = defaultNvSandboxUtilsLibraryLoadFlags
	}

	l.path = o.path
	l.dl = dl.New(o.path, o.flags)
}

// LookupSymbol checks whether the specified library symbol exists in the library.
// Note that this requires that the library be loaded.
func (l *library) LookupSymbol(name string) error {
	if l == nil || l.refcount == 0 {
		return fmt.Errorf("error looking up %s: %w", name, errLibraryNotLoaded)
	}
	return l.dl.Lookup(name)
}

// load initializes the library and updates the versioned symbols.
// Multiple calls to an already loaded library will return without error.
func (l *library) load() (rerr error) {
	l.Lock()
	defer l.Unlock()

	defer func() { l.refcount.IncOnNoError(rerr) }()
	if l.refcount > 0 {
		return nil
	}

	if err := l.dl.Open(); err != nil {
		return fmt.Errorf("error opening %s: %w", l.path, err)
	}

	// Update the errorStringFunc to point to nvsandboxutils.ErrorString
	errorStringFunc = nvsanboxutilsErrorString

	// Update all versioned symbols
	l.updateVersionedSymbols()

	return nil
}

// close the underlying library and ensure that the global pointer to the
// library is set to nil to ensure that subsequent calls to open will reinitialize it.
// Multiple calls to an already closed nvsandboxutils library will return without error.
func (l *library) close() (rerr error) {
	l.Lock()
	defer l.Unlock()

	defer func() { l.refcount.DecOnNoError(rerr) }()
	if l.refcount != 1 {
		return nil
	}

	if err := l.dl.Close(); err != nil {
		return fmt.Errorf("error closing %s: %w", l.path, err)
	}

	// Update the errorStringFunc to point to defaultErrorStringFunc
	errorStringFunc = defaultErrorStringFunc

	return nil
}

// Default all versioned APIs to v1 (to infer the types)
var (
// Insert default versions for APIs here.
// Example:
// nvsandboxUtilsFunction = nvsandboxUtilsFunction_v1
)

// updateVersionedSymbols checks for versioned symbols in the loaded dynamic library.
// If newer versioned symbols exist, these replace the default `v1` symbols initialized above.
// When new versioned symbols are added, these would have to be initialized above and have
// corresponding checks and subsequent assignments added below.
func (l *library) updateVersionedSymbols() {
	// Example:
	// err := l.dl.Lookup("nvsandboxUtilsFunction_v2")
	// if err == nil {
	// 	nvsandboxUtilsFunction = nvsandboxUtilsFunction_v2
	// }
}
