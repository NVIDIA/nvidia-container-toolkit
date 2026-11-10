/*
 * Copyright (c) 2026, NVIDIA CORPORATION.  All rights reserved.
 *
 * Licensed under the Apache License, Version 2.0 (the "License");
 * you may not use this file except in compliance with the License.
 * You may obtain a copy of the License at
 *
 *     http://www.apache.org/licenses/LICENSE-2.0
 *
 * Unless required by applicable law or agreed to in writing, software
 * distributed under the License is distributed on an "AS IS" BASIS,
 * WITHOUT WARRANTIES OR CONDITIONS OF ANY KIND, either express or implied.
 * See the License for the specific language governing permissions and
 * limitations under the License.
 */

package e2e

import (
	"os"
	"os/exec"
	"path/filepath"
	"strings"
	"testing"
)

func TestNvidiaCdiRefreshEffectiveRestartLimits(t *testing.T) {
	const expectedProperties = "StartLimitBurst=5\nStartLimitIntervalUSec=10min\nTimeoutStartUSec=1min 30s\n"
	const matchingUnitText = "[Service]\nStartLimitBurst=5\nStartLimitInterval=10min\n"
	for _, tc := range []struct {
		name       string
		properties string
		unitText   string
		showExit   string
		wantError  string
	}{
		{
			name:       "effective limits match",
			properties: expectedProperties,
			unitText:   matchingUnitText,
		},
		{
			name:       "equivalent modern unit spelling",
			properties: expectedProperties,
			unitText:   "[Unit]\nStartLimitBurst=5\nStartLimitIntervalSec=600s\n",
		},
		{
			name:       "drop-in disables the burst limit",
			properties: strings.ReplaceAll(expectedProperties, "StartLimitBurst=5", "StartLimitBurst=0"),
			unitText:   matchingUnitText,
			wantError:  "does not have the expected restart burst",
		},
		{
			name:       "drop-in shortens the restart window",
			properties: strings.ReplaceAll(expectedProperties, "StartLimitIntervalUSec=10min", "StartLimitIntervalUSec=10s"),
			unitText:   matchingUnitText,
			wantError:  "does not have the expected restart interval",
		},
		{
			name:       "unsupported interval property",
			properties: strings.ReplaceAll(expectedProperties, "StartLimitIntervalUSec=10min\n", ""),
			unitText:   matchingUnitText,
			wantError:  "does not have the expected restart interval",
		},
		{
			name:       "timeout only has matching prefix",
			properties: strings.ReplaceAll(expectedProperties, "TimeoutStartUSec=1min 30s", "TimeoutStartUSec=1min 30s 500ms"),
			unitText:   matchingUnitText,
			wantError:  "does not have the expected 90s start timeout",
		},
		{
			name:       "systemctl fails despite matching output",
			properties: expectedProperties,
			unitText:   matchingUnitText,
			showExit:   "1",
			wantError:  "Could not read nvidia-cdi-refresh.service restart settings",
		},
		{
			name:       "missing burst property",
			properties: strings.ReplaceAll(expectedProperties, "StartLimitBurst=5\n", ""),
			unitText:   matchingUnitText,
			wantError:  "does not have the expected restart burst",
		},
		{
			name:       "missing timeout property",
			properties: strings.ReplaceAll(expectedProperties, "TimeoutStartUSec=1min 30s\n", ""),
			unitText:   matchingUnitText,
			wantError:  "does not have the expected 90s start timeout",
		},
	} {
		t.Run(tc.name, func(t *testing.T) {
			dir := t.TempDir()
			dropIn := filepath.Join(dir, "10-container-engines.conf")
			if err := os.WriteFile(dropIn, nil, 0600); err != nil {
				t.Fatal(err)
			}
			// Never call the host's systemctl. Model raw unit text independently
			// from the effective properties, as with a later overriding drop-in.
			stub := `#!/bin/sh
command=$1
shift
[ "$1" = "nvidia-cdi-refresh.service" ] || exit 99
shift
case "$command" in
cat)
    [ "$#" -eq 0 ] || exit 99
    printf '%s' "$UNIT_TEXT"
    ;;
show)
    [ "$#" -gt 0 ] || exit 99
    check_settings=0
    while [ "$#" -gt 0 ]; do
        [ "$1" = "-p" ] && [ "$#" -ge 2 ] || exit 99
        case "$2" in
        Before) printf '%s\n' 'Before=docker.service containerd.service crio.service' ;;
        TimeoutStartUSec|StartLimitBurst|StartLimitIntervalUSec)
            printf '%s' "$EFFECTIVE_PROPERTIES" | grep "^$2=" || true
            check_settings=1
            ;;
        *) exit 99 ;;
        esac
        shift 2
    done
    [ "$check_settings" -eq 0 ] || exit "${SHOW_EXIT:-0}"
    ;;
*) exit 99 ;;
esac
`
			if err := os.WriteFile(filepath.Join(dir, "systemctl"), []byte(stub), 0700); err != nil {
				t.Fatal(err)
			}
			script := strings.ReplaceAll(nvidiaCdiRefreshOrderingDropInInstalledTemplate,
				"/lib/systemd/system/nvidia-cdi-refresh.service.d/10-container-engines.conf", dropIn)
			cmd := exec.Command("bash", "-c", script)
			cmd.Env = append(os.Environ(),
				"PATH="+dir+string(os.PathListSeparator)+os.Getenv("PATH"),
				"UNIT_TEXT="+tc.unitText,
				"EFFECTIVE_PROPERTIES="+tc.properties,
				"SHOW_EXIT="+tc.showExit)
			output, err := cmd.CombinedOutput()
			if (err != nil) != (tc.wantError != "") {
				t.Fatalf("template error = %v, want error = %q; output:\n%s", err, tc.wantError, output)
			}
			if tc.wantError != "" && !strings.Contains(string(output), tc.wantError) {
				t.Fatalf("template output = %q, want diagnostic containing %q", output, tc.wantError)
			}
		})
	}
}
