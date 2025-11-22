package nri

import (
	"context"

	"github.com/NVIDIA/nvidia-container-toolkit/internal/logger"
)

type toNriLogger struct {
	logger.Interface
}

func (l toNriLogger) Debugf(_ context.Context, fmt string, args ...interface{}) {
	l.Interface.Debugf(fmt, args...)
}

func (l toNriLogger) Errorf(_ context.Context, fmt string, args ...interface{}) {
	l.Interface.Errorf(fmt, args...)
}

func (l toNriLogger) Infof(_ context.Context, fmt string, args ...interface{}) {
	l.Interface.Infof(fmt, args...)
}

func (l toNriLogger) Warnf(_ context.Context, fmt string, args ...interface{}) {
	l.Warningf(fmt, args...)
}
