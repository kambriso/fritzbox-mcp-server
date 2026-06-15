---
# fritzbox-mcp-server-gtty
title: upgrade GitHub Actions to Node 24-compatible versions
status: completed
type: task
priority: normal
created_at: 2026-06-15T08:40:02Z
updated_at: 2026-06-15T08:40:36Z
---

## Scope

GitHub Actions emits a deprecation warning because `actions/checkout@v4` and `actions/setup-go@v5` still run on Node.js 20. Update workflow references to the current major versions that use Node.js 24.

## Checklist

- [x] Inspect workflow action versions
- [x] Update checkout and setup-go versions
- [x] Review workflow diff

## Summary of Changes

Updated all workflow references from `actions/checkout@v4` to `actions/checkout@v5` and from `actions/setup-go@v5` to `actions/setup-go@v6`. These current major versions use the Node.js 24 runtime and remove the Node.js 20 deprecation warning on GitHub-hosted runners.
