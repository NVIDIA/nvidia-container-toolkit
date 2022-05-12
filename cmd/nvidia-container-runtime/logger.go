/*
# Copyright (c) 2021, NVIDIA CORPORATION.  All rights reserved.
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
*/

package main

import (
	"fmt"
	"io"
	"os"
	"path/filepath"
	"runtime"
	"strconv"
	"strings"

	"github.com/sirupsen/logrus"
)

// Logger adds a way to manage output to a log file to a logrus.Logger
type Logger struct {
	*logrus.Logger
	previousLogger *logrus.Logger
	logFiles       []*os.File
}

// NewLogger creates an empty logger
func NewLogger() *Logger {
	return &Logger{
		Logger: logrus.New(),
	}
}

// UpdateLogger constructs a Logger with a preddefined formatter
func UpdateLogger(filename string, logLevel string, argv []string) (*Logger, error) {
	configFromArgs := parseArgs(argv)

	level, logLevelError := configFromArgs.getLevel(logLevel)

	var logFiles []*os.File
	var argLogFileError error

	// We don't create log files if the version argument is supplied
	if !configFromArgs.version {
		configLogFile, err := createLogFile(filename)
		if err != nil {
			return logger, fmt.Errorf("error opening debug log file: %v", err)
		}
		if configLogFile != nil {
			logFiles = append(logFiles, configLogFile)
		}

		argLogFile, err := createLogFile(configFromArgs.file)
		if argLogFile != nil {
			logFiles = append(logFiles, argLogFile)
		}
		argLogFileError = err
	}

	l := &Logger{
		Logger:         logrus.New(),
		previousLogger: logger.Logger,
		logFiles:       logFiles,
	}

	l.SetLevel(level)
	if level == logrus.DebugLevel {
		logrus.SetReportCaller(true)
		// Shorten function and file names reported by the logger, by
		// trimming common "github.com/opencontainers/runc" prefix.
		// This is only done for text formatter.
		_, file, _, _ := runtime.Caller(0)
		prefix := filepath.Dir(file) + "/"
		logrus.SetFormatter(&logrus.TextFormatter{
			CallerPrettyfier: func(f *runtime.Frame) (string, string) {
				function := strings.TrimPrefix(f.Function, prefix) + "()"
				fileLine := strings.TrimPrefix(f.File, prefix) + ":" + strconv.Itoa(f.Line)
				return function, fileLine
			},
		})
	}

	if configFromArgs.format == "json" {
		l.SetFormatter(new(logrus.JSONFormatter))
	}

	if len(logFiles) == 0 {
		l.SetOutput(io.Discard)
	} else if len(logFiles) == 1 {
		l.SetOutput(logFiles[0])
	} else if len(logFiles) > 1 {
		var writers []io.Writer
		for _, f := range logFiles {
			writers = append(writers, f)
		}
		l.SetOutput(io.MultiWriter(writers...))
	}

	if logLevelError != nil {
		l.Warn(logLevelError)
	}

	if argLogFileError != nil {
		l.Warnf("Failed to open log file: %v", argLogFileError)
	}

	return l, nil
}

// Reset closes the log file (if any) and resets the logger output to what it
// was before UpdateLogger was called.
func (l *Logger) Reset() error {
	defer func() {
		previous := l.previousLogger
		if previous == nil {
			previous = logrus.New()
		}
		logger = &Logger{Logger: previous}
	}()

	var errs []error
	for _, f := range l.logFiles {
		err := f.Close()
		if err != nil {
			errs = append(errs, err)
		}
	}

	var err error
	for _, e := range errs {
		if err == nil {
			err = e
			continue
		}
		return fmt.Errorf("%v; %w", e, err)
	}

	return err
}

func createLogFile(filename string) (*os.File, error) {
	if filename != "" && filename != os.DevNull {
		return os.OpenFile(filename, os.O_APPEND|os.O_CREATE|os.O_WRONLY, 0644)
	}

	return nil, nil
}

type loggerConfig struct {
	file    string
	format  string
	debug   bool
	version bool
}

func (c loggerConfig) getLevel(logLevel string) (logrus.Level, error) {
	if c.debug {
		return logrus.DebugLevel, nil
	}

	if logLevel, err := logrus.ParseLevel(logLevel); err == nil {
		return logLevel, nil
	}

	return logrus.InfoLevel, fmt.Errorf("invalid log-level '%v'", logLevel)
}

// Informed by Taken from https://github.com/opencontainers/runc/blob/7fd8b57001f5bfa102e89cb434d96bf71f7c1d35/main.go#L182
func parseArgs(args []string) loggerConfig {
	c := loggerConfig{}

	expected := map[string]*string{
		"log-format": &c.format,
		"log":        &c.file,
	}

	found := make(map[string]bool)

	for i := 0; i < len(args); i++ {
		if len(found) == 4 {
			break
		}

		param := args[i]

		parts := strings.SplitN(param, "=", 2)
		trimmed := strings.TrimLeft(parts[0], "-")
		// If this is not a flag we continue
		if parts[0] == trimmed {
			continue
		}

		// Check the version flag
		if trimmed == "version" {
			c.version = true
			found["version"] = true
			// For the version flag we don't process any other flags
			continue
		}

		// Check the debug flag
		if trimmed == "debug" {
			c.debug = true
			found["debug"] = true
			continue
		}

		destination, exists := expected[trimmed]
		if !exists {
			continue
		}

		var value string
		if len(parts) == 2 {
			value = parts[2]
		} else if i+1 < len(args) {
			value = args[i+1]
			i++
		} else {
			continue
		}

		*destination = value
		found[trimmed] = true
	}

	return c
}
