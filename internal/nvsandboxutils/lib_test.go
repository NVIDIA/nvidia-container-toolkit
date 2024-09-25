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
	"testing"

	"github.com/stretchr/testify/require"
)

func newTestLibrary(dl dynamicLibrary) *library {
	return &library{dl: dl}
}

func TestLookupFromDefault(t *testing.T) {
	errClose := errors.New("close error")
	errOpen := errors.New("open error")
	errLookup := errors.New("lookup error")

	testCases := []struct {
		description          string
		dl                   dynamicLibrary
		skipLoadLibrary      bool
		expectedLoadError    error
		expectedLookupErrror error
		expectedCloseError   error
	}{
		{
			description:          "library not loaded yields error",
			dl:                   &dynamicLibraryMock{},
			skipLoadLibrary:      true,
			expectedLookupErrror: errLibraryNotLoaded,
		},
		{
			description: "open error is returned",
			dl: &dynamicLibraryMock{
				OpenFunc: func() error {
					return errOpen
				},
			},

			expectedLoadError:    errOpen,
			expectedLookupErrror: errLibraryNotLoaded,
		},
		{
			description: "lookup error is returned",
			dl: &dynamicLibraryMock{
				OpenFunc: func() error {
					return nil
				},
				LookupFunc: func(s string) error {
					return fmt.Errorf("%w: %s", errLookup, s)
				},
				CloseFunc: func() error {
					return nil
				},
			},

			expectedLookupErrror: errLookup,
		},
		{
			description: "lookup succeeds",
			dl: &dynamicLibraryMock{
				OpenFunc: func() error {
					return nil
				},
				LookupFunc: func(s string) error {
					return nil
				},
				CloseFunc: func() error {
					return nil
				},
			},
		},
		{
			description: "lookup succeeds",
			dl: &dynamicLibraryMock{
				OpenFunc: func() error {
					return nil
				},
				LookupFunc: func(s string) error {
					return nil
				},
				CloseFunc: func() error {
					return nil
				},
			},
		},
		{
			description: "close error is returned",
			dl: &dynamicLibraryMock{
				OpenFunc: func() error {
					return nil
				},
				LookupFunc: func(s string) error {
					return nil
				},
				CloseFunc: func() error {
					return errClose
				},
			},
			expectedCloseError: errClose,
		},
	}

	for _, tc := range testCases {
		t.Run(tc.description, func(t *testing.T) {
			l := newTestLibrary(tc.dl)
			if !tc.skipLoadLibrary {
				require.ErrorIs(t, l.load(), tc.expectedLoadError)
			}
			require.ErrorIs(t, l.LookupSymbol("symbol"), tc.expectedLookupErrror)
			require.ErrorIs(t, l.close(), tc.expectedCloseError)
			if tc.expectedCloseError == nil {
				require.Equal(t, 0, int(l.refcount))
			} else {
				require.Equal(t, 1, int(l.refcount))
			}
		})
	}
}

func TestLoadAndCloseNesting(t *testing.T) {
	dl := &dynamicLibraryMock{
		OpenFunc: func() error {
			return nil
		},
		CloseFunc: func() error {
			return nil
		},
	}

	l := newTestLibrary(dl)

	// When calling close before opening the library nothing happens.
	require.Equal(t, 0, len(dl.calls.Close))
	require.Nil(t, l.close())
	require.Equal(t, 0, len(dl.calls.Close))

	// When calling load twice, the library was only opened once
	require.Equal(t, 0, len(dl.calls.Open))
	require.Nil(t, l.load())
	require.Equal(t, 1, len(dl.calls.Open))
	require.Nil(t, l.load())
	require.Equal(t, 1, len(dl.calls.Open))

	// Only after calling close twice, was the library closed
	require.Equal(t, 0, len(dl.calls.Close))
	require.Nil(t, l.close())
	require.Equal(t, 0, len(dl.calls.Close))
	require.Nil(t, l.close())
	require.Equal(t, 1, len(dl.calls.Close))

	// Calling close again doesn't attempt to close the library again
	require.Nil(t, l.close())
	require.Equal(t, 1, len(dl.calls.Close))
}

func TestLoadAndCloseWithErrors(t *testing.T) {
	testCases := []struct {
		description           string
		dl                    dynamicLibrary
		expectedLoadRefcount  refcount
		expectedCloseRefcount refcount
	}{
		{
			description: "regular flow",
			dl: &dynamicLibraryMock{
				OpenFunc: func() error {
					return nil
				},
				CloseFunc: func() error {
					return nil
				},
			},
			expectedLoadRefcount:  1,
			expectedCloseRefcount: 0,
		},
		{
			description: "open error",
			dl: &dynamicLibraryMock{
				OpenFunc: func() error {
					return errors.New("")
				},
				CloseFunc: func() error {
					return nil
				},
			},
			expectedLoadRefcount:  0,
			expectedCloseRefcount: 0,
		},
		{
			description: "close error",
			dl: &dynamicLibraryMock{
				OpenFunc: func() error {
					return nil
				},
				CloseFunc: func() error {
					return errors.New("")
				},
			},
			expectedLoadRefcount:  1,
			expectedCloseRefcount: 1,
		},
		{
			description: "open and close error",
			dl: &dynamicLibraryMock{
				OpenFunc: func() error {
					return errors.New("")
				},
				CloseFunc: func() error {
					return errors.New("")
				},
			},
			expectedLoadRefcount:  0,
			expectedCloseRefcount: 0,
		},
	}

	for _, tc := range testCases {
		t.Run(tc.description, func(t *testing.T) {
			l := newTestLibrary(tc.dl)
			_ = l.load()
			require.Equal(t, tc.expectedLoadRefcount, l.refcount)
			_ = l.close()
			require.Equal(t, tc.expectedCloseRefcount, l.refcount)
		})
	}
}
