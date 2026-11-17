---
# fritzbox-mcp-server-ixcb
title: Prove three-forge release reproducibility
status: in-progress
type: task
priority: high
created_at: 2026-11-17T08:34:33Z
updated_at: 2026-11-17T08:50:03Z
parent: fritzbox-mcp-server-qce6
---

## Goal

Prove byte-for-byte reproducibility for one identical tagged commit built by GitHub Actions, GitLab CI, and Azure Pipelines.

## Acceptance criteria

- [x] The same test tag resolves to the same commit on all three forges.
- [ ] All three systems publish the complete canonical artifact inventory and `SHA256SUMS`.
- [ ] `make verify-reproducible-three-forge` succeeds for the tagged runs.
- [ ] Tag, commit, run URLs, and comparison result are recorded in the summary.
- [ ] Any temporary test tag is handled according to the agreed release policy.

## Progress

- Created lightweight pre-release tag `v0.8.2-rc.1` on commit `84bdcb2a52e174bbfe8a9da84a11f34f81c411c8` on GitHub, GitLab, and Azure Repos; all three refs were verified identical.
- GitHub Actions run `44048988262` succeeded and published the release plus SLSA provenance.
- Azure Pipeline run `14` succeeded, built the canonical release inventory, and published pipeline artifact `release-artifacts`.
- GitLab pipeline `3077908656` built, signed, and uploaded the artifacts, then failed only in the generated release step because the Go build image contained neither `glab >= 1.58` nor the deprecated `release-cli`.
- Refactored `.gitlab-ci.yml` to keep the pinned Go build in `release-build` and create release metadata in a separate `publish-release` job using `registry.gitlab.com/gitlab-org/cli:v1.130.0`, as required by current GitLab documentation.
- GitLab CI lint accepts the corrected configuration. A second RC tag is required to validate the fix and perform the final three-forge comparison.
