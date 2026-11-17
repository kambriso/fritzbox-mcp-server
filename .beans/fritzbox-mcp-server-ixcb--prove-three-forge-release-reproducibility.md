---
# fritzbox-mcp-server-ixcb
title: Prove three-forge release reproducibility
status: todo
type: task
priority: high
created_at: 2026-11-17T08:34:33Z
updated_at: 2026-11-17T08:34:33Z
parent: fritzbox-mcp-server-qce6
blocked_by:
    - fritzbox-mcp-server-f4ry
---

## Goal

Prove byte-for-byte reproducibility for one identical tagged commit built by GitHub Actions, GitLab CI, and Azure Pipelines.

## Acceptance criteria

- [ ] The same test tag resolves to the same commit on all three forges.
- [ ] All three systems publish the complete canonical artifact inventory and `SHA256SUMS`.
- [ ] `make verify-reproducible-three-forge` succeeds for the tagged runs.
- [ ] Tag, commit, run URLs, and comparison result are recorded in the summary.
- [ ] Any temporary test tag is handled according to the agreed release policy.
