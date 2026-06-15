---
# fritzbox-mcp-server-yybk
title: 'issue #1: fix documentation'
status: completed
type: task
priority: normal
created_at: 2026-06-15T08:04:18Z
updated_at: 2026-06-15T08:20:35Z
---

## Scope

Issue #1: the documentation currently implies Claude Desktop exists on Linux. Update the English and German READMEs to state that, as of June 15, 2026, there is no official Claude Desktop for Linux, and recommend generic MCP-compatible alternatives without presenting third-party clients as official.

## Checklist

- [x] Inspect the current English and German setup sections
- [x] Update the English README Linux guidance
- [x] Update the German README Linux guidance
- [x] Review wording for accuracy and consistency

## Summary of Changes

Updated `README.adoc` and `README-de.adoc` so they no longer imply that an official Claude Desktop app exists for Linux. Both READMEs now state explicitly that, as of June 15, 2026, Anthropic does not provide an official Linux Claude Desktop app, and they recommend using a generic MCP-compatible client/editor integration on Linux instead. Troubleshooting text was updated accordingly to refer to the user's MCP client rather than Claude Desktop specifically.
