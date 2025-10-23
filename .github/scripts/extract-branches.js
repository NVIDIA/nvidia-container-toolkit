/**
 * Copyright 2025 NVIDIA CORPORATION
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

module.exports = async ({ github, context, core }) => {
  let branches = [];

  // Get PR labels
  const labels = context.payload.pull_request?.labels || [];

  if (labels.length === 0) {
    core.info('No labels found on PR - skipping backport');
    return [];
  }

  // Extract branches from cherry-pick/* labels
  const cherryPickPattern = /^cherry-pick\/(release-\d+\.\d+(?:\.\d+)?)$/;
  
  for (const label of labels) {
    const match = label.name.match(cherryPickPattern);
    if (match) {
      branches.push(match[1]);
      core.info(`Found cherry-pick label: ${label.name} -> ${match[1]}`);
    }
  }

  if (branches.length === 0) {
    core.info('No cherry-pick labels found - skipping backport');
    return [];
  }

  core.info(`Target branches: ${branches.join(', ')}`);
  return branches;
};
