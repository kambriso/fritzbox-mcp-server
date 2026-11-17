---
# fritzbox-mcp-server-k2kd
title: Provision Azure project and repository
status: completed
type: task
priority: high
created_at: 2026-11-17T08:34:33Z
updated_at: 2026-11-17T08:39:02Z
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

- [x] Azure DevOps project `Fritzbox` exists.
- [x] Azure Repos repository `fritzbox-mcp-server` exists inside that project.
- [x] The repository contains the canonical Git history and the Azure feature branch.
- [x] A local `azure` remote points to the dedicated repository.
- [x] Project and repository URLs are recorded in the summary.

## Notes

Do not reuse the unrelated `HelloWorld` repository as the Fritz!Box source repository.

## Summary of Changes

- Created private Azure DevOps project `Fritzbox`; the organization policy does not permit public projects.
- Renamed the automatically created repository to `fritzbox-mcp-server`.
- Imported `https://github.com/kambriso/fritzbox-mcp-server.git` server-side so GitHub remains the unambiguous canonical source.
- Set `refs/heads/main` as the default branch.
- Verified `feature/azure-pipelines` resolves to `84bdcb2a52e174bbfe8a9da84a11f34f81c411c8`, identical to the local and GitHub feature branch.
- Configured local remote `azure` for `https://dev.azure.com/jochenhinrichsen/Fritzbox/_git/fritzbox-mcp-server`.
- Repository URL: `https://dev.azure.com/jochenhinrichsen/Fritzbox/_git/fritzbox-mcp-server`.
