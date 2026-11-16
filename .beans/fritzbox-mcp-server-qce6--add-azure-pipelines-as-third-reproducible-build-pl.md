---
# fritzbox-mcp-server-qce6
title: add Azure Pipelines as third reproducible-build platform
status: in-progress
type: task
priority: normal
created_at: 2026-11-16T11:15:47Z
updated_at: 2026-11-16T14:47:21Z
---

## Goal

Add Azure DevOps/Azure Pipelines as a third independent CI and release platform for `kambriso/fritzbox-mcp-server`. For the same tagged commit, Azure Pipelines must produce byte-identical release binaries to the existing GitHub Actions and GitLab CI builds.

## Context

- Source project: `/home/jot/repos/fritzbox-mcp-server`
- Canonical GitHub scope: `kambriso/fritzbox-mcp-server` (not `jhinrichsen`)
- Existing implementation: reproducible cross-platform Go builds in `Makefile`, GitHub release workflow in `.github/workflows/release.yml`, GitLab release pipeline in `.gitlab-ci.yml`, and GitHub/GitLab verification targets in `Makefile`
- A previous Azure DevOps proof of concept may exist as `jochenhinrichsen/HelloWorld/HelloWorld`; inspect it for reusable organization, project, service-connection, agent-pool, and pipeline setup, but do not treat that repository as the Fritz!Box source project.
- The claim demonstrated by this work is Azure DevOps/Azure Pipelines competence and independent cross-forge reproducibility. It does not by itself demonstrate broad Azure infrastructure expertise.

## Scope

- Mirror or import the `kambriso/fritzbox-mcp-server` sources into the appropriate Azure DevOps project while retaining an unambiguous link to the canonical source and tagged commit.
- Add an Azure Pipelines YAML definition that uses the same pinned Go toolchain and the existing reproducible build path rather than reimplementing build logic.
- Build the same release binaries currently produced by GitHub and GitLab for Linux, macOS, Windows, and WASI.
- Publish the binaries and `SHA256SUMS` as Azure Pipeline artifacts for tagged releases.
- Add a documented or automated three-way verification that compares artifact hashes from GitHub, GitLab, and Azure DevOps for the same tag.
- Record Azure-specific provenance or signing capabilities separately; do not claim a SLSA level unless the generated provenance and verification procedure justify it.
- Update project documentation with setup, release, and verification instructions.

## Acceptance criteria

- [ ] The Azure DevOps repository and pipeline use the `kambriso/fritzbox-mcp-server` sources, not the unrelated `HelloWorld` POC sources.
- [x] Azure Pipelines builds tagged commits through the existing reproducible `Makefile` targets with a pinned toolchain and clean agent environment.
- [ ] Azure publishes the complete expected release binary set and `SHA256SUMS`.
- [ ] For one test tag, every Azure-built release binary has the same SHA-256 digest as its GitHub- and GitLab-built counterpart.
- [x] A repeatable three-forge verification command or documented procedure reports actionable differences when any artifact is missing or differs.
- [ ] Azure pipeline configuration and verification instructions are committed to the canonical project.
- [x] README security/release claims distinguish verified reproducibility from SLSA provenance levels and accurately name GitHub Actions, GitLab CI, and Azure Pipelines.
- [ ] CI configuration is validated and the successful Azure run plus three-way digest comparison are linked or recorded in the task summary.

## Suggested implementation sequence

1. Locate and inspect the existing Azure DevOps `HelloWorld` POC and confirm the intended Azure organization/project.
2. Establish the Azure repository mirror/import for the canonical `kambriso` project.
3. Implement and validate `azure-pipelines.yml` using the existing reproducible build targets.
4. Publish tagged artifacts and checksums.
5. Extend verification tooling from two forges to three and prove equality on a test tag.
6. Document the result and only then update CV wording in the separate CV repository.

## Progress

- Added `azure-pipelines.yml` on `feature/azure-pipelines` with `main` and `v*` triggers, full-history checkout, Go 1.25.6 in `golang:1.25.6-bookworm`, tests, tagged release builds, and `PublishPipelineArtifact@1`.
- Added the shared `make release-artifacts` path and changed GitHub Actions and GitLab CI to use the same canonical nine-file artifact set and checksum manifest.
- Added `scripts/verify-three-forge.sh` plus `make verify-reproducible-three-forge`; the verifier validates the complete artifact inventory and emits digest diffs.
- Corrected README claims: GitHub releases have SLSA provenance, GitLab releases have Cosign signatures, and Azure currently has checksums without a claimed SLSA level.
- Validated YAML parsing, shell syntax, Make targets, unit tests, all cross-platform builds, checksum verification, deliberate mismatch rejection, and two byte-identical clean builds in the pinned container.
- Azure CLI and an authenticated browser are unavailable in the current environment. The remaining work is to confirm the organization/project, create or import the dedicated Azure repository, create the pipeline, mirror a test tag, and record the successful run and three-forge digest comparison.
