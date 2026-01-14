/**
# Copyright (c) NVIDIA CORPORATION.  All rights reserved.
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

package logger

import (
	"fmt"

	"github.com/bombsimon/logrusr/v4"
	"github.com/go-logr/logr"
	"github.com/sirupsen/logrus"
)

// New returns a new logger
func New() Interface {
	return Interface{
		logrusr.New(logrus.StandardLogger()),
	}
}

// NullLogger is a logger that does nothing
func NullLogger() Interface {
	return Interface{
		logr.Discard(),
	}
}

func (l Interface) Debugf(format string, a ...any) {
	l.V(4).Info(fmt.Sprintf(format, a...))
}

func (l Interface) Infof(format string, a ...any) {
	l.Info(fmt.Sprintf(format, a...))
}

func (l Interface) Warningf(format string, a ...any) {
	l.Info(fmt.Sprintf("WARNING: "+format, a...))
}

func (l Interface) Tracef(format string, a ...any) {
	l.V(6).Info(fmt.Sprintf(format, a...))
}
