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
  const commentBody = context.payload.comment.body;
  const prNumber = context.payload.issue.number;

  core.info(`Processing comment: ${commentBody}`);

  // Parse comment for /cherry-pick branches
  const cherryPickPattern = /^\/cherry-pick\s+(.+)$/m;
  const match = commentBody.match(cherryPickPattern);

  if (!match) {
    core.warning('Comment does not match /cherry-pick pattern');
    return { success: false, message: 'Invalid format' };
  }

  // Extract all release branches (space-separated)
  const branchesText = match[1].trim();
  const branchPattern = /release-\d+\.\d+(?:\.\d+)?/g;
  const branches = branchesText.match(branchPattern) || [];

  if (branches.length === 0) {
    core.warning('No valid release branches found in comment');
    await github.rest.reactions.createForIssueComment({
      owner: context.repo.owner,
      repo: context.repo.repo,
      comment_id: context.payload.comment.id,
      content: 'confused'
    });
    return { success: false, message: 'No valid branches found' };
  }

  core.info(`Found branches: ${branches.join(', ')}`);

  // Add labels to PR
  const labels = branches.map(branch => `cherry-pick/${branch}`);
  
  try {
    await github.rest.issues.addLabels({
      owner: context.repo.owner,
      repo: context.repo.repo,
      issue_number: prNumber,
      labels: labels
    });
    core.info(`Added labels: ${labels.join(', ')}`);
  } catch (error) {
    core.error(`Failed to add labels: ${error.message}`);
    await github.rest.reactions.createForIssueComment({
      owner: context.repo.owner,
      repo: context.repo.repo,
      comment_id: context.payload.comment.id,
      content: '-1'
    });
    return { success: false, message: error.message };
  }

  // React with checkmark emoji
  await github.rest.reactions.createForIssueComment({
    owner: context.repo.owner,
    repo: context.repo.repo,
    comment_id: context.payload.comment.id,
    content: '+1'
  });

  // Check if PR is already merged
  const { data: pullRequest } = await github.rest.pulls.get({
    owner: context.repo.owner,
    repo: context.repo.repo,
    pull_number: prNumber
  });

  if (pullRequest.merged) {
    core.info('PR is already merged - triggering backport immediately');
    
    // Set branches in environment and trigger backport
    process.env.BRANCHES_JSON = JSON.stringify(branches);
    
    // Run backport script
    const backportScript = require('./backport.js');
    const results = await backportScript({ github, context, core });
    
    return { 
      success: true, 
      message: `Labels added and backport triggered for: ${branches.join(', ')}`,
      backportResults: results
    };
  } else {
    core.info('PR not yet merged - labels added, backport will trigger on merge');
    return { 
      success: true, 
      message: `Labels added for: ${branches.join(', ')}. Backport will trigger on merge.`
    };
  }
};

