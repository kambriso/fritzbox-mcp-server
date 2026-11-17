---
# fritzbox-mcp-server-ixcb
title: Prove three-forge release reproducibility
status: completed
type: task
priority: high
created_at: 2026-11-17T08:34:33Z
updated_at: 2026-11-17T09:00:54Z
parent: fritzbox-mcp-server-qce6
---

## Goal

Prove byte-for-byte reproducibility for one identical tagged commit built by GitHub Actions, GitLab CI, and Azure Pipelines.

## Acceptance criteria

- [x] The same test tag resolves to the same commit on all three forges.
- [x] All three systems publish the complete canonical artifact inventory and `SHA256SUMS`.
- [x] `make verify-reproducible-three-forge` succeeds for the tagged runs.
- [x] Tag, commit, run URLs, and comparison result are recorded in the summary.
- [x] Any temporary test tag is handled according to the agreed release policy.

## Summary

Validated lightweight prerelease tag `v0.8.2-rc.2` at commit `cd46364eaee8189ccbfcd023175d256e5a4ee1a8` on all three forges.

- GitHub Actions run 44050227463 succeeded and published the canonical release inventory: https://github.com/kambriso/fritzbox-mcp-server/actions/runs/44050227463
- GitLab pipeline 3077940519 succeeded; `release-build` uploaded the inventory and `publish-release` created release metadata with `glab`: https://gitlab.com/kambriso/fritzbox-mcp-server/-/pipelines/3077940519
- Azure Pipeline run 15 succeeded and published pipeline artifact `release-artifacts`: https://dev.azure.com/jochenhinrichsen/Fritzbox/_build/results?buildId=15&view=results
- `make verify-reproducible-three-forge TAG=v0.8.2-rc.2 AZURE_RUN_ID=15` reported `Three-forge reproducibility verified for v0.8.2-rc.2` after comparing all nine binary digests.

The earlier `v0.8.2-rc.1` run is retained as diagnostic prerelease evidence for the obsolete GitLab `release-cli` failure. Both RC tags/releases remain explicitly marked as prereleases and are not production releases.
