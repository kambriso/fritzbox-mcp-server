---
# fritzbox-mcp-server-qce6
title: Azure Pipelines cross-forge reproducible builds
status: completed
type: epic
priority: normal
created_at: 2026-11-16T11:15:47Z
updated_at: 2026-11-17T09:01:55Z
---

## Goal

Add Azure DevOps/Azure Pipelines as a third independent CI and release platform for `kambriso/fritzbox-mcp-server`, producing byte-identical release binaries for the same tagged commit as GitHub Actions and GitLab CI.

## Acceptance criteria

- [x] The Azure DevOps repository and pipeline use the `kambriso/fritzbox-mcp-server` sources, not the unrelated HelloWorld POC sources.
- [x] Azure Pipelines builds tagged commits through the existing reproducible Makefile targets with a pinned toolchain and clean agent environment.
- [x] Azure publishes the complete expected release binary set and `SHA256SUMS`.
- [x] For one test tag, every Azure-built release binary has the same SHA-256 digest as its GitHub- and GitLab-built counterpart.
- [x] A repeatable three-forge verification command reports actionable differences when any artifact is missing or differs.
- [x] Azure pipeline configuration and verification instructions are committed to the canonical project.
- [x] Security and release claims distinguish verified reproducibility from SLSA provenance levels and accurately name GitHub Actions, GitLab CI, and Azure Pipelines.
- [x] CI configuration is validated and the successful Azure run plus three-way digest comparison are recorded.

## Summary

Created the private Azure DevOps project `Fritzbox`, repository `fritzbox-mcp-server`, and YAML pipeline `fritzbox-mcp-server` (definition 2). The repository remains a mirror of canonical GitHub source.

The implementation centralizes nine platform binaries and `SHA256SUMS` in `make release-artifacts`, pins Go 1.25.6 across all three forges, publishes tagged Azure artifacts, and adds `make verify-reproducible-three-forge`.

Validated prerelease tag `v0.8.2-rc.2` at commit `cd46364eaee8189ccbfcd023175d256e5a4ee1a8`:

- GitHub Actions run 44050227463: https://github.com/kambriso/fritzbox-mcp-server/actions/runs/44050227463
- GitLab pipeline 3077940519: https://gitlab.com/kambriso/fritzbox-mcp-server/-/pipelines/3077940519
- Azure Pipeline run 15: https://dev.azure.com/jochenhinrichsen/Fritzbox/_build/results?buildId=15&view=results
- Comparison result: `Three-forge reproducibility verified for v0.8.2-rc.2` for all nine binary digests.

The validated feature was integrated by fast-forward and synchronized to `main` on NAS, GitHub, GitLab, and Azure Repos. Azure artifacts currently provide checksums but do not claim project-generated SLSA provenance or Cosign signatures.
