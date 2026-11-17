---
# fritzbox-mcp-server-f4ry
title: Create and validate Azure-hosted pipeline
status: todo
type: task
priority: high
created_at: 2026-11-17T08:34:33Z
updated_at: 2026-11-17T08:34:33Z
parent: fritzbox-mcp-server-qce6
blocked_by:
    - fritzbox-mcp-server-k2kd
    - fritzbox-mcp-server-goj0
---

## Goal

Create the Azure Pipeline from `azure-pipelines.yml` and validate the repository-side configuration with a real Azure-hosted run.

## Acceptance criteria

- [ ] Pipeline `fritzbox-mcp-server` exists in Azure project `Fritzbox`.
- [ ] The pipeline reads `azure-pipelines.yml` from the dedicated repository.
- [ ] A run for `feature/azure-pipelines` completes successfully.
- [ ] The run uses the pinned `golang:1.25.6-bookworm` container.
- [ ] Run URL and relevant diagnostics are recorded in the summary.
