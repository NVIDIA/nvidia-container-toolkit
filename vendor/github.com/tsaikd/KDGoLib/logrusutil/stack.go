package logrusutil

import "github.com/sirupsen/logrus"

// StackLogLevel temporary change log level and return recover function
func StackLogLevel(logger *logrus.Logger, level logrus.Level) (recover func()) {
	if logger.Level == level {
		return func() {}
	}

	originLevel := logger.Level
	logger.Level = level

	return func() {
		logger.Level = originLevel
	}
}
