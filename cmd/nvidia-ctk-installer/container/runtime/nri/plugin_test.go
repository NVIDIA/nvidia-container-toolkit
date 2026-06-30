/**
# Copyright (c) 2026, NVIDIA CORPORATION.  All rights reserved.
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

package nri

import (
	"testing"

	"github.com/containerd/nri/pkg/api"
	"github.com/stretchr/testify/require"
)

// nullLogger satisfies logger.Interface without any output.
type nullLogger struct{}

func (nullLogger) Debugf(string, ...any)   {}
func (nullLogger) Errorf(string, ...any)   {}
func (nullLogger) Infof(string, ...any)    {}
func (nullLogger) Warningf(string, ...any) {}
func (nullLogger) Tracef(string, ...any)   {}

func newTestPlugin(namespaces []string) *Plugin {
	return &Plugin{
		logger:     nullLogger{},
		namespaces: namespaces,
	}
}

func podWithAnnotation(namespace, annotation, value string) *api.PodSandbox {
	return &api.PodSandbox{
		Namespace:   namespace,
		Annotations: map[string]string{annotation: value},
	}
}

func TestParseCDIDevices(t *testing.T) {
	const (
		toolkitNamespace    = "gpu-operator"
		additionalNamespace = "kube-system"
		unknownNamespace    = "default"
	)

	testCases := []struct {
		description string
		namespaces  []string
		pod         *api.PodSandbox
		container   string
		expected    []string
	}{
		{
			description: "no annotations returns nil",
			namespaces:  []string{toolkitNamespace},
			pod:         &api.PodSandbox{Namespace: toolkitNamespace},
			container:   "ctr",
			expected:    nil,
		},
		{
			description: "non-management CDI device is injected in any namespace",
			namespaces:  []string{toolkitNamespace},
			pod:         podWithAnnotation(unknownNamespace, nriCDIAnnotationDomain+"/pod", "nvidia.com/gpu=0"),
			container:   "ctr",
			expected:    []string{"nvidia.com/gpu=0"},
		},
		{
			description: "management CDI device injected when pod is in the toolkit namespace",
			namespaces:  []string{toolkitNamespace},
			pod:         podWithAnnotation(toolkitNamespace, nriCDIAnnotationDomain+"/pod", "management.nvidia.com/gpu=0"),
			container:   "ctr",
			expected:    []string{"management.nvidia.com/gpu=0"},
		},
		{
			description: "management CDI device blocked when pod is outside allowed namespaces",
			namespaces:  []string{toolkitNamespace},
			pod:         podWithAnnotation(unknownNamespace, nriCDIAnnotationDomain+"/pod", "management.nvidia.com/gpu=0"),
			container:   "ctr",
			expected:    nil,
		},
		{
			description: "management CDI device injected when pod is in an additional allowed namespace",
			namespaces:  []string{toolkitNamespace, additionalNamespace},
			pod:         podWithAnnotation(additionalNamespace, nriCDIAnnotationDomain+"/pod", "management.nvidia.com/gpu=0"),
			container:   "ctr",
			expected:    []string{"management.nvidia.com/gpu=0"},
		},
		{
			description: "management CDI device blocked even with additional namespaces when pod namespace not listed",
			namespaces:  []string{toolkitNamespace, additionalNamespace},
			pod:         podWithAnnotation(unknownNamespace, nriCDIAnnotationDomain+"/pod", "management.nvidia.com/gpu=0"),
			container:   "ctr",
			expected:    nil,
		},
		{
			description: "mixed management and non-management CDI devices are injected in allowed namespace",
			namespaces:  []string{toolkitNamespace},
			pod:         podWithAnnotation(toolkitNamespace, nriCDIAnnotationDomain+"/pod", "nvidia.com/gpu=0,management.nvidia.com/gpu=0"),
			container:   "ctr",
			expected:    []string{"nvidia.com/gpu=0", "management.nvidia.com/gpu=0"},
		},
		{
			description: "mixed management and non-management CDI devices are blocked in disallowed namespace",
			namespaces:  []string{toolkitNamespace},
			pod:         podWithAnnotation(unknownNamespace, nriCDIAnnotationDomain+"/pod", "nvidia.com/gpu=0,management.nvidia.com/gpu=0"),
			container:   "ctr",
			expected:    nil,
		},
		{
			description: "container-scoped annotation takes precedence over pod-scoped",
			namespaces:  []string{toolkitNamespace},
			pod: &api.PodSandbox{
				Namespace: toolkitNamespace,
				Annotations: map[string]string{
					nriCDIAnnotationDomain + "/pod":           "nvidia.com/gpu=0",
					nriCDIAnnotationDomain + "/container.ctr": "management.nvidia.com/gpu=0",
				},
			},
			container: "ctr",
			expected:  []string{"management.nvidia.com/gpu=0"},
		},
	}

	for _, tc := range testCases {
		t.Run(tc.description, func(t *testing.T) {
			plugin := newTestPlugin(tc.namespaces)
			got := plugin.parseCDIDevices(tc.pod, nriCDIAnnotationDomain, tc.container)
			require.Equal(t, tc.expected, got)
		})
	}
}
