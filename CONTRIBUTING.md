# Contribute to the NVIDIA Container Toolkit

To file feature requests or bugs, submit an issue at https://github.com/NVIDIA/nvidia-container-toolkit/issues. For questions, start a discussion at https://github.com/NVIDIA/nvidia-container-toolkit/discussions or ask in the `#gpu-operator` channel in our community Slack at https://inviter.co/dsx-ai-factory.

All external contributions start with an issue. External contributors must have an accepted issue that is assigned to them before they open a pull request. See [Contribution workflow](#contribution-workflow). Contributions do not require explicit contributor license agreements (CLA), but we expect contributors to [sign](#sign-your-work) their work.

Reviewers are automatically assigned to Pull Requests from the [OWNERS](CODEOWNERS) file.
PR approval and decision-making for major changes are documented in [GOVERNANCE.md](GOVERNANCE.md).

All contributions must adhere to the [Code of Conduct](CODE_OF_CONDUCT.md).

### AI Usage
AI-assisted contributions are welcome, however, **you must understand your code**. If you can't explain what your changes do and how they interact with the greater system without the aid of AI tools, your contribution may not be accepted. AI should assist your effort, not replace it or shift the burden of understanding, testing, and reviewing onto maintainers.

The human contributor is accountable for all code, regardless of how much was AI-generated. Each contribution must name the human behind the change via a [DCO sign-off](#sign-your-work).

Maintainers may close issues and PRs that are unproductive or low-quality, without explanation, regardless of whether AI tools were used or not. If a contributor repeatedly opens unproductive issues or PRs, they may be blocked from contributing to the project because it is disruptive and disrespectful of the maintainers' time.

Below are some unacceptable usages of AI that will likely result in your contribution being closed:

- Submitting AI-generated issues or PRs without thorough review and verification
- Submitting AI-generated code that you cannot explain or answer questions about during review
- Using AI to generate replies to review comments or issue discussions
- Submitting AI-generated code that contains unnecessary refactoring or unrelated changes not scoped to the problem the PR is solving

### Contribution workflow

**To propose a new issue:** start at step 1.\
**To work on an existing issue:** skip to step 3 and [claim an `accepted` issue](https://github.com/NVIDIA/nvidia-container-toolkit/issues?q=is%3Aopen+label%3Aaccepted+no%3Aassignee) with no assignees.

1. **Open an issue.** Use the bug report template to report a defect, or the feature request template to propose new behavior or a [major change](GOVERNANCE.md#core-decisions). Indicate if you want to work on the issue. Each new issue gets the `needs-triage` label.
2. **Wait for triage.** A maintainer reviews the issue against the [triage criteria](#triage-criteria). The maintainer then labels the issue `accepted`, `needs-more-info`, or closes it.
3. **Claim the issue.** When an issue is `accepted`, comment on it expressing interest and summarizing your proposed solution. Issue authors have priority if they indicate they would like to work on it. Other claims are first-come, first-served. A maintainer discusses your plan with you and assigns the issue to you when the plan is ready.
4. **Open a pull request.** File a pull request at https://github.com/NVIDIA/nvidia-container-toolkit/pulls and link the issue in the description (for example, `Fixes #1234`). Maintainers close pull requests that do not link an issue assigned to the author, and pull requests that are duplicates of existing ones.

### Issue labels

Each open issue has one state label and one category label. For example, `needs-triage` and `bug` together identify a bug report that no maintainer has reviewed yet.

| Label | Type | Meaning |
| ----- | ---- | ------- |
| `needs-triage` | State | No maintainer has reviewed the issue yet. |
| `needs-more-info` | State | Maintainers are waiting for more information from the author. If the issue has no activity for 30 days, it is closed. |
| `accepted` | State | The issue is approved for work. Contributors can claim it. |
| `bug` | Category | A defect in existing behavior. |
| `feature` | Category | A request for new behavior. |

### Triage criteria

Maintainers use these questions to triage an issue:

- Is the problem clear and reproducible?
- Is the issue a duplicate, or does other work already address it?
- Does the issue fit the supported use cases and the project priorities?
- Is the proposed approach appropriate?
- Does the issue align with the product roadmap?

## Sign your work

The sign-off is a simple line at the end of the explanation for the patch. Your
signature certifies that you wrote the patch or otherwise have the right to pass
it on as an open-source patch. The rules are pretty simple: if you can certify
the below (from [developercertificate.org](http://developercertificate.org/)):

```
Developer Certificate of Origin
Version 1.1

Copyright (C) 2004, 2006 The Linux Foundation and its contributors.
1 Letterman Drive
Suite D4700
San Francisco, CA, 94129

Everyone is permitted to copy and distribute verbatim copies of this
license document, but changing it is not allowed.

Developer's Certificate of Origin 1.1

By making a contribution to this project, I certify that:

(a) The contribution was created in whole or in part by me and I
    have the right to submit it under the open source license
    indicated in the file; or

(b) The contribution is based upon previous work that, to the best
    of my knowledge, is covered under an appropriate open source
    license and I have the right under that license to submit that
    work with modifications, whether created in whole or in part
    by me, under the same open source license (unless I am
    permitted to submit under a different license), as indicated
    in the file; or

(c) The contribution was provided directly to me by some other
    person who certified (a), (b) or (c) and I have not modified
    it.

(d) I understand and agree that this project and the contribution
    are public and that a record of the contribution (including all
    personal information I submit with it, including my sign-off) is
    maintained indefinitely and may be redistributed consistent with
    this project or the open source license(s) involved.
```

Then you just add a line to every git commit message:

    Signed-off-by: Joe Smith <joe.smith@email.com>

Use your real name (sorry, no pseudonyms or anonymous contributions.)

If you set your `user.name` and `user.email` git configs, you can sign your
commit automatically with `git commit -s`.

