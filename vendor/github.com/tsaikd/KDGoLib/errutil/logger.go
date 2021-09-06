package errutil

import (
	"bytes"
	"fmt"
	"io"
	"os"
	"strings"
)

// LoggerType declare general log types
type LoggerType interface {
	Debug(v ...interface{})
	Print(v ...interface{})
	Error(v ...interface{})
	Trace(errin error)
	TraceSkip(errin error, skip int)
}

// Logger return default LoggerType instance
func Logger() LoggerType {
	return defaultLogger
}

// NewLogger create LoggerType instance
func NewLogger(opt LoggerOptions) LoggerType {
	opt.check()
	return loggerImpl{
		opt: opt,
	}
}

// LoggerOptions for Logger
type LoggerOptions struct {
	DefaultOutput   io.Writer
	ErrorOutput     io.Writer
	HideFile        bool
	ShortFile       bool
	HideLine        bool
	ReplacePackages map[string]string
	TraceFormatter  TraceFormatter
}

func (t *LoggerOptions) check() {
	if t.DefaultOutput == nil {
		t.DefaultOutput = os.Stdout
	}
	if t.ErrorOutput == nil {
		t.ErrorOutput = os.Stderr
	}
	if t.TraceFormatter == nil {
		t.TraceFormatter = &ConsoleFormatter{
			Seperator:  "; ",
			TimeFormat: "2006-01-02 15:04:05 ",
			LongFile:   true,
			Line:       true,
		}
	}
}

var defaultLogger = NewLogger(LoggerOptions{})

// SetDefaultLogger set default LoggerType
func SetDefaultLogger(logger LoggerType) {
	defaultLogger = logger
}

type loggerImpl struct {
	opt LoggerOptions
}

func (t loggerImpl) Debug(v ...interface{}) {
	t.log(t.opt.DefaultOutput, 1, v...)
}

func (t loggerImpl) Print(v ...interface{}) {
	t.log(t.opt.DefaultOutput, 1, v...)
}

func (t loggerImpl) Error(v ...interface{}) {
	t.log(t.opt.ErrorOutput, 1, v...)
}

func (t loggerImpl) log(output io.Writer, skip int, v ...interface{}) {
	errtext := fmt.Sprint(v...)
	if errtext == "" {
		return
	}

	opt := t.opt
	if !opt.HideFile {
		buffer := &bytes.Buffer{}
		callinfo, _ := RuntimeCaller(skip + 1)
		if _, err := WriteCallInfo(buffer, callinfo, !opt.ShortFile, !opt.HideLine, opt.ReplacePackages); err != nil {
			panic(err)
		}
		errtext = buffer.String() + " " + errtext
	}

	if !strings.HasSuffix(errtext, "\n") {
		errtext += "\n"
	}
	if _, err := output.Write([]byte(errtext)); err != nil {
		panic(err)
	}
}

func (t loggerImpl) Trace(errin error) {
	TraceSkip(errin, 1)
}

func (t loggerImpl) TraceSkip(errin error, skip int) {
	var errtext string
	var errfmt error
	if errtext, errfmt = t.opt.TraceFormatter.FormatSkip(errin, skip+1); errfmt != nil {
		panic(errfmt)
	}
	if errtext == "" {
		return
	}
	if !strings.HasSuffix(errtext, "\n") {
		errtext += "\n"
	}
	if _, errfmt = t.opt.ErrorOutput.Write([]byte(errtext)); errfmt != nil {
		panic(errfmt)
	}
}
