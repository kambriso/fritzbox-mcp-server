---
# fritzbox-mcp-server-k2kd
title: Provision Azure project and repository
status: in-progress
type: task
priority: high
created_at: 2026-11-17T08:34:33Z
updated_at: 2026-11-17T08:34:33Z
parent: fritzbox-mcp-server-qce6
---

## Goal

Provision the dedicated Azure DevOps project and repository that will host the independent Azure build of `kambriso/fritzbox-mcp-server`.

## Scope

- Azure DevOps organization: `jochenhinrichsen`
- Project: `Fritzbox`
- Repository: `fritzbox-mcp-server`
- Source branch: `feature/azure-pipelines`

## Acceptance criteria

- [ ] Azure DevOps project `Fritzbox` exists.
- [ ] Azure Repos repository `fritzbox-mcp-server` exists inside that project.
- [ ] The repository contains the canonical Git history and the Azure feature branch.
- [ ] A local `azure` remote points to the dedicated repository.
- [ ] Project and repository URLs are recorded in the summary.

## Notes

Do not reuse the unrelated `HelloWorld` repository as the Fritz!Box source repository.
