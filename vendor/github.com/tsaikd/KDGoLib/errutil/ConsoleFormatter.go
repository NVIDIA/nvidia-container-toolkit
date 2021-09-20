package errutil

import (
	"bytes"
	"time"
)

// NewConsoleFormatter create JSONErrorFormatter instance
func NewConsoleFormatter(seperator string) *ConsoleFormatter {
	return &ConsoleFormatter{
		Seperator: seperator,
	}
}

// ConsoleFormatter used to format error object in console readable
type ConsoleFormatter struct {
	// seperator between errors, e.g. "; " will output "err1; err2; err3"
	Seperator string
	// output timestamp for prefix, e.g.  "2006-01-02 15:04:05 "
	TimeFormat string
	// show error position with long filename
	LongFile bool
	// show error position with short filename
	ShortFile bool
	// show error position with line number, work with LongFile or ShortFile
	Line bool
	// replace package name for securify
	ReplacePackages map[string]string
}

// Format error object
func (t *ConsoleFormatter) Format(errin error) (errtext string, err error) {
	return t.FormatSkip(errin, 1)
}

// FormatSkip trace error line and format object
func (t *ConsoleFormatter) FormatSkip(errin error, skip int) (errtext string, err error) {
	errobj := castErrorObject(nil, skip+1, errin)
	if errobj == nil {
		return "", nil
	}

	buffer := &bytes.Buffer{}

	if t.TimeFormat != "" {
		if _, errio := buffer.WriteString(time.Now().Format(t.TimeFormat)); errio != nil {
			return buffer.String(), errio
		}
	}

	if t.LongFile || t.ShortFile {
		if _, errio := WriteCallInfo(buffer, errobj, t.LongFile, t.Line, t.ReplacePackages); errio != nil {
			return buffer.String(), errio
		}
		if _, errio := buffer.WriteString(" "); errio != nil {
			return buffer.String(), errio
		}
	}

	if t.Seperator == "" {
		if _, errio := buffer.WriteString(getErrorText(errin)); errio != nil {
			return buffer.String(), errio
		}
		return buffer.String(), nil
	}

	firstError := true
	if walkerr := WalkErrors(errobj, func(errloop ErrorObject) (stop bool, walkerr error) {
		if !firstError {
			if _, errio := buffer.WriteString(t.Seperator); errio != nil {
				return true, errio
			}
		}
		firstError = false

		if _, errio := buffer.WriteString(getErrorText(errloop)); errio != nil {
			return true, errio
		}
		return false, nil
	}); walkerr != nil {
		return buffer.String(), walkerr
	}

	return buffer.String(), nil
}

var _ ErrorFormatter = &ConsoleFormatter{}
var _ TraceFormatter = &ConsoleFormatter{}
