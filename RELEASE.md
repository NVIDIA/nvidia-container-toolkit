# Release Process

The NVIDIA Container Toolkit consists of the following artifacts:
- The NVIDIA Container Toolkit container
- Packages for debian-based systems
- Packages for rpm-based systems

# Release Process Checklist:
- [ ] Create a release PR:
    - [ ] Run the `./hack/prepare-release.sh` script to update the version in all the needed files. This also creates a [release issue](https://github.com/NVIDIA/nvidia-container-toolkit/issues?q=is%3Aissue+is%3Aopen+label%3Arelease)
    - [ ] Run the `./hack/generate-changelog.sh` script to generate the a draft changelog and update `CHANGELOG.md` with the changes.
    - [ ] Create a PR from the created `bump-release-{{ .VERSION }}` branch.
- [ ] Merge the release PR
- [ ] Tag the release and push the tag to the `internal` mirror:
- [ ] Wait for the image release to complete.
- [ ] Push the tag to the the upstream GitHub repo.
- [ ] Wait for the [`Release`](https://github.com/NVIDIA/nvidia-container-toolkit/actions/workflows/release.yaml) GitHub Action to complete
- [ ] Publish the [draft release](https://github.com/NVIDIA/nvidia-container-toolkit/releases) created by the GitHub Action
- [ ] Publish the packages to the gh-pages branch of the libnvidia-container repo
- [ ] Create a KitPick
- [ ] For non-experimental releases, schedule the package for publication to the CUDA Downloads repositories.

## Troubleshooting

*Note*: This assumes that we have the release tag checked out locally.

- If the `Release` GitHub Action fails:
    - Check the logs for the error first.
    - Create the helm packages locally by running:
      ```bash
      ./hack/prepare-artifacts.sh {{ .VERSION }}
      ```
    - Create the draft release by running:
      ```bash
      ./hack/create-release.sh {{ .VERSION }}
      ```
