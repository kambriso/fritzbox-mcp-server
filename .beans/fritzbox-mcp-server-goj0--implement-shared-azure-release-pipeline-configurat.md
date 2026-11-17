---
# fritzbox-mcp-server-goj0
title: Implement shared Azure release pipeline configuration
status: completed
type: task
priority: high
created_at: 2026-11-17T08:34:33Z
updated_at: 2026-11-17T08:34:33Z
parent: fritzbox-mcp-server-qce6
---

## Goal

Implement the repository-side Azure Pipelines configuration and shared reproducible release path.

## Acceptance criteria

- [x] `azure-pipelines.yml` builds and tests with pinned Go 1.25.6.
- [x] GitHub, GitLab, and Azure use the same `make release-artifacts` target.
- [x] The canonical nine-artifact inventory and `SHA256SUMS` are generated.
- [x] Three-forge checksum verification tooling and setup documentation exist.
- [x] Local container builds are byte-identical across two clean builds.

## Summary of Changes

Implemented and locally validated in commit `7c3750c` on `feature/azure-pipelines`. The commit is available on the NAS remote. It adds the Azure YAML pipeline, unifies release artifact creation across all three CI systems, pins Go 1.25.6, adds three-forge verification, and corrects provenance claims.
