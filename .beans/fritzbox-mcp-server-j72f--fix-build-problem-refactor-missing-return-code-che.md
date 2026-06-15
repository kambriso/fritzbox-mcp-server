---
# fritzbox-mcp-server-j72f
title: fix build problem (refactor missing return code check)
status: completed
type: task
priority: normal
created_at: 2026-06-15T08:05:09Z
updated_at: 2026-06-15T08:30:45Z
---

## Scope

`make tidy` currently fails in CI due to unchecked `fmt.Scanln` return values in `main.go`. Update the interactive setup flow to handle input errors explicitly, then rerun the CI-aligned make targets.

## Checklist

- [x] Inspect the failing `main.go` input paths
- [x] Fix unchecked `fmt.Scanln` return values
- [x] Run `make tidy`
- [x] Run `make test`
- [x] Run `make dists`

## Summary of Changes

Updated the interactive setup flow in `main.go` to stop ignoring stdin read errors. The previous unchecked `fmt.Scanln` calls were replaced with explicit line-based reads, numeric parsing for device selection, and consistent error propagation for setup prompts. After the change, `make tidy`, `make test`, and `make dists` all completed successfully locally.
