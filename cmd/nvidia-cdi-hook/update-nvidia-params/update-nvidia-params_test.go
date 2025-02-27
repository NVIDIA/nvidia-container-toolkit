package nvidiaparams

import (
	"bytes"
	"testing"

	testlog "github.com/sirupsen/logrus/hooks/test"
	"github.com/stretchr/testify/require"
)

func TestGetModifiedParamsFileContentsFromReader(t *testing.T) {
	logger, _ := testlog.NewNullLogger()
	testCases := map[string]struct {
		contents         []byte
		expectedError    error
		expectedContents []byte
	}{
		"no contents": {
			contents:         nil,
			expectedError:    nil,
			expectedContents: nil,
		},
		"other contents are ignored": {
			contents: []byte(`# Some other content
			that we don't care about
			`),
			expectedError:    nil,
			expectedContents: nil,
		},
		"already zero requires no modification": {
			contents:         []byte("ModifyDeviceFiles: 0"),
			expectedError:    nil,
			expectedContents: nil,
		},
		"leading spaces require no modification": {
			contents: []byte("  ModifyDeviceFiles: 1"),
		},
		"Trailing spaces require no modification": {
			contents: []byte("ModifyDeviceFiles: 1  "),
		},
		"Not 1 require no modification": {
			contents: []byte("ModifyDeviceFiles: 11"),
		},
		"single line requires modification": {
			contents:         []byte("ModifyDeviceFiles: 1"),
			expectedError:    nil,
			expectedContents: []byte("ModifyDeviceFiles: 0\n"),
		},
		"single line with trailing newline requires modification": {
			contents:         []byte("ModifyDeviceFiles: 1\n"),
			expectedError:    nil,
			expectedContents: []byte("ModifyDeviceFiles: 0\n"),
		},
		"other content is maintained": {
			contents: []byte(`ModifyDeviceFiles: 1
			other content
			that
			is maintained`),
			expectedError: nil,
			expectedContents: []byte(`ModifyDeviceFiles: 0
			other content
			that
			is maintained
`),
		},
	}

	for description, tc := range testCases {
		t.Run(description, func(t *testing.T) {
			c := command{
				logger: logger,
			}
			contents, err := c.getModifiedParamsFileContentsFromReader(bytes.NewReader(tc.contents))
			require.EqualValues(t, tc.expectedError, err)
			require.EqualValues(t, string(tc.expectedContents), string(contents))
		})
	}

}
