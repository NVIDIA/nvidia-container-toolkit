package logrusutil

import (
	"os"

	"github.com/sirupsen/logrus"
)

// NewConsoleLogger create new Logger with ConsoleLogFormatter
func NewConsoleLogger() *logrus.Logger {
	return &logrus.Logger{
		Out:       os.Stdout,
		Formatter: &ConsoleLogFormatter{},
		Hooks:     make(logrus.LevelHooks),
		Level:     logrus.InfoLevel,
	}
}
