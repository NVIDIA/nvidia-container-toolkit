package errutil

import (
	"bytes"
	"encoding/json"
)

// NewJSONFormatter create JSONFormatter instance
func NewJSONFormatter() *JSONFormatter {
	return &JSONFormatter{}
}

// JSONFormatter used to format error to json
type JSONFormatter struct{}

// Format error to json
func (t *JSONFormatter) Format(errin error) (errtext string, err error) {
	return t.FormatSkip(errin, 1)
}

// FormatSkip trace error line and format to json
func (t *JSONFormatter) FormatSkip(errin error, skip int) (errtext string, err error) {
	errjson, err := newJSON(skip+1, errin)
	if errjson == nil || err != nil {
		return "", err
	}

	buffer := &bytes.Buffer{}
	if err = json.NewEncoder(buffer).Encode(errjson); err != nil {
		return
	}

	return buffer.String(), nil
}

var _ ErrorFormatter = &JSONFormatter{}
var _ TraceFormatter = &JSONFormatter{}
