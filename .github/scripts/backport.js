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
const branches = JSON.parse(process.env.BRANCHES_JSON || '[]');

// Get PR number from event
const prNumber = context.payload.pull_request?.number || context.payload.issue.number;

// Fetch full PR data (needed when triggered via issue_comment)
const { data: pullRequest } = await github.rest.pulls.get({
  owner: context.repo.owner,
  repo: context.repo.repo,
  pull_number: prNumber
});

const prTitle = pullRequest.title;
const prAuthor = pullRequest.user.login;

// Get all commits from the PR
const { data: commits } = await github.rest.pulls.listCommits({
  owner: context.repo.owner,
  repo: context.repo.repo,
  pull_number: prNumber
});

if (commits.length === 0) {
  core.warning('No commits found in PR - skipping backport');
  return [];
}

core.info(`Backporting PR #${prNumber}: "${prTitle}"`);
core.info(`Commits to cherry-pick: ${commits.length}`);
commits.forEach((commit, index) => {
  core.info(`  ${index + 1}. ${commit.sha.substring(0, 7)} - ${commit.commit.message.split('\n')[0]}`);
});

const { execSync } = require('child_process');

const results = [];

for (const targetBranch of branches) {
  core.info(`\n========================================`);
  core.info(`Backporting to ${targetBranch}`);
  core.info(`========================================`);
  const backportBranch = `backport-${prNumber}-to-${targetBranch}`;
  try {
    // Create/reset backport branch from target release branch
    core.info(`Creating/resetting branch ${backportBranch} from ${targetBranch}`);
    execSync(`git fetch origin ${targetBranch}:${targetBranch}`, { stdio: 'inherit' });
    execSync(`git checkout -B ${backportBranch} ${targetBranch}`, { stdio: 'inherit' });
    // Cherry-pick each commit from the PR
    let hasConflicts = false;
    for (let i = 0; i < commits.length; i++) {
      const commit = commits[i];
      const commitSha = commit.sha;
      const commitMessage = commit.commit.message.split('\n')[0];
      core.info(`Cherry-picking commit ${i + 1}/${commits.length}: ${commitSha.substring(0, 7)} - ${commitMessage}`);
      try {
        execSync(`git cherry-pick -x ${commitSha}`, { 
          encoding: 'utf-8',
          stdio: 'pipe'
        });
      } catch (error) {
        // Check if it's a conflict
        const status = execSync('git status', { encoding: 'utf-8' });
        if (status.includes('Unmerged paths') || status.includes('both modified')) {
          hasConflicts = true;
          core.warning(`Cherry-pick has conflicts for commit ${commitSha.substring(0, 7)}.`);
          // Add all files (including conflicted ones) and commit
          execSync('git add .', { stdio: 'inherit' });
          try {
            execSync(`git -c core.editor=true cherry-pick --continue`, { stdio: 'inherit' });
          } catch (e) {
            // If continue fails, make a simple commit
            execSync(`git commit --no-edit --allow-empty-message || git commit -m "Cherry-pick ${commitSha} (with conflicts)"`, { stdio: 'inherit' });
          }
        } else if (error.message && error.message.includes('previous cherry-pick is now empty')) {
          // Handle empty commits (changes already exist in target branch)
          core.info(`Commit ${commitSha.substring(0, 7)} is empty (changes already in target branch), skipping`);
          execSync('git cherry-pick --skip', { stdio: 'inherit' });
        } else {
          throw error;
        }
      }
    }
    // Push the backport branch (force to handle updates)
    core.info(`Pushing ${backportBranch} to origin`);
    execSync(`git push --force-with-lease origin ${backportBranch}`, { stdio: 'inherit' });
    
    // Check if a PR already exists for this backport branch
    const { data: existingPRs } = await github.rest.pulls.list({
      owner: context.repo.owner,
      repo: context.repo.repo,
      head: `${context.repo.owner}:${backportBranch}`,
      base: targetBranch,
      state: 'open'
    });
    const existingPR = existingPRs.length > 0 ? existingPRs[0] : null;
    
    // Create pull request
    const commitList = commits.map(c => `- \`${c.sha.substring(0, 7)}\` ${c.commit.message.split('\n')[0]}`).join('\n');
    
    // Build PR body based on conflict status
    let prBody = `🤖 **Automated backport of #${prNumber} to \`${targetBranch}\`**\n\n`;
    
    if (hasConflicts) {
      prBody += `⚠️ **This PR has merge conflicts that need manual resolution.**

Original PR: #${prNumber}
Original Author: @${prAuthor}

**Cherry-picked commits (${commits.length}):**
${commitList}

**Next Steps:**
1. Review the conflicts in the "Files changed" tab
2. Check out this branch locally: \`git fetch origin ${backportBranch} && git checkout ${backportBranch}\`
3. Resolve conflicts manually
4. Push the resolution: \`git push --force-with-lease origin ${backportBranch}\`

---
<details>
<summary>Instructions for resolving conflicts</summary>

\`\`\`bash
git fetch origin ${backportBranch}
git checkout ${backportBranch}
# Resolve conflicts in your editor
git add .
git commit
git push --force-with-lease origin ${backportBranch}
\`\`\`
</details>`;
    } else {
      prBody += `✅ Cherry-pick completed successfully with no conflicts.

Original PR: #${prNumber}
Original Author: @${prAuthor}

**Cherry-picked commits (${commits.length}):**
${commitList}

This backport was automatically created by the backport bot.`;
    }

    if (existingPR) {
      // Update existing PR
      core.info(`Found existing PR #${existingPR.number}, updating it`);
      await github.rest.pulls.update({
        owner: context.repo.owner,
        repo: context.repo.repo,
        pull_number: existingPR.number,
        body: prBody,
        draft: hasConflicts
      });
      
      // Update labels
      const currentLabels = existingPR.labels.map(l => l.name);
      const desiredLabels = ['backport', hasConflicts ? 'needs-manual-resolution' : 'auto-backport'];
      
      // Remove old labels if conflict status changed
      if (hasConflicts && currentLabels.includes('auto-backport')) {
        await github.rest.issues.removeLabel({
          owner: context.repo.owner,
          repo: context.repo.repo,
          issue_number: existingPR.number,
          name: 'auto-backport'
        }).catch(() => {}); // Ignore if label doesn't exist
      } else if (!hasConflicts && currentLabels.includes('needs-manual-resolution')) {
        await github.rest.issues.removeLabel({
          owner: context.repo.owner,
          repo: context.repo.repo,
          issue_number: existingPR.number,
          name: 'needs-manual-resolution'
        }).catch(() => {}); // Ignore if label doesn't exist
      }
      
      // Add current labels
      await github.rest.issues.addLabels({
        owner: context.repo.owner,
        repo: context.repo.repo,
        issue_number: existingPR.number,
        labels: desiredLabels
      });
      
      // Comment about the update
      await github.rest.issues.createComment({
        owner: context.repo.owner,
        repo: context.repo.repo,
        issue_number: prNumber,
        body: `🤖 Updated existing backport PR for \`${targetBranch}\`: #${existingPR.number} ${hasConflicts ? '⚠️ (has conflicts)' : '✅'}`
      });
      
      results.push({
        branch: targetBranch,
        success: true,
        prNumber: existingPR.number,
        prUrl: existingPR.html_url,
        hasConflicts,
        updated: true
      });
      core.info(`✅ Successfully updated backport PR #${existingPR.number}`);
    } else {
      // Create new PR
      const newPR = await github.rest.pulls.create({
        owner: context.repo.owner,
        repo: context.repo.repo,
        title: `[${targetBranch}] ${prTitle}`,
        head: backportBranch,
        base: targetBranch,
        body: prBody,
        draft: hasConflicts
      });
      // Add labels
      await github.rest.issues.addLabels({
        owner: context.repo.owner,
        repo: context.repo.repo,
        issue_number: newPR.data.number,
        labels: ['backport', hasConflicts ? 'needs-manual-resolution' : 'auto-backport']
      });
      // Link to original PR
      await github.rest.issues.createComment({
        owner: context.repo.owner,
        repo: context.repo.repo,
        issue_number: prNumber,
        body: `🤖 Backport PR created for \`${targetBranch}\`: #${newPR.data.number} ${hasConflicts ? '⚠️ (has conflicts)' : '✅'}`
      });
      results.push({
        branch: targetBranch,
        success: true,
        prNumber: newPR.data.number,
        prUrl: newPR.data.html_url,
        hasConflicts,
        updated: false
      });
      core.info(`✅ Successfully created backport PR #${newPR.data.number}`);
    }
  } catch (error) {
    core.error(`❌ Failed to backport to ${targetBranch}: ${error.message}`);
    // Comment on original PR about the failure
    await github.rest.issues.createComment({
      owner: context.repo.owner,
      repo: context.repo.repo,
      issue_number: prNumber,
      body: `❌ Failed to create backport PR for \`${targetBranch}\`\n\nError: ${error.message}\n\nPlease backport manually.`
    });
    results.push({
      branch: targetBranch,
      success: false,
      error: error.message
    });
  } finally {
    // Clean up: go back to main branch
    try {
      execSync('git checkout main', { stdio: 'inherit' });
      execSync(`git branch -D ${backportBranch} 2>/dev/null || true`, { stdio: 'inherit' });
    } catch (e) {
      // Ignore cleanup errors
    }
  }
}

// Summary (console only)
core.info('\n========================================');
core.info('Backport Summary');
core.info('========================================');
for (const result of results) {
  if (result.success) {
    const action = result.updated ? 'Updated' : 'Created';
    core.info(`✅ ${result.branch}: ${action} PR #${result.prNumber} ${result.hasConflicts ? '(has conflicts)' : ''}`);
  } else {
    core.error(`❌ ${result.branch}: ${result.error}`);
  }
}
return results;
};
