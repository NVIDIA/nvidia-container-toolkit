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

  // Get PR number
  const prNumber = context.payload.pull_request?.number || context.payload.issue?.number;

  if (!prNumber) {
    core.warning('Could not determine PR number from event - skipping backport');
    return [];
  }

  // Check PR body
  if (context.payload.pull_request?.body) {
    const prBody = context.payload.pull_request.body;
    // Strict ASCII, anchored; allow X.Y or X.Y.Z
    // Support multiple space-separated branches on one line
    const lineMatches = prBody.matchAll(/^\/cherry-pick\s+(.+)$/gmi);
    for (const match of lineMatches) {
      const branchMatches = match[1].matchAll(/release-\d+\.\d+(?:\.\d+)?/g);
      branches.push(...Array.from(branchMatches, m => m[0]));
    }
  }

  // Check all comments
  const comments = await github.rest.issues.listComments({
    owner: context.repo.owner,
    repo: context.repo.repo,
    issue_number: prNumber
  });

  for (const comment of comments.data) {
    const lineMatches = comment.body.matchAll(/^\/cherry-pick\s+(.+)$/gmi);
    for (const match of lineMatches) {
      const branchMatches = match[1].matchAll(/release-\d+\.\d+(?:\.\d+)?/g);
      branches.push(...Array.from(branchMatches, m => m[0]));
    }
  }

  // Deduplicate
  branches = [...new Set(branches)];

  if (branches.length === 0) {
    core.info('No cherry-pick requests found - skipping backport');
    return [];
  }

  core.info(`Target branches: ${branches.join(', ')}`);
  return branches;
};
