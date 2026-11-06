---
name: graphify
description: Query this repository's local Graphify code graph for cross-file relationships, callers, paths and architecture exploration; also use when the user invokes $graphify.
---

# Graphify for this repository

Project adaptation of [Graphify's Codex workflow](https://github.com/Graphify-Labs/graphify/blob/v8/README.md#cli-skills-or-mcp-instructions), reviewed against installed `graphifyy` 0.9.103. Root AGENTS.md and architecture contracts remain authoritative. This skill uses the CLI; it does not register MCP tools.

## Choose the query path

Use a graph for cross-file relationships and repeated exploration. A narrow known-file lookup needs only rg/direct reads. If the host actually exposes Graphify MCP tools, use them against the configured graph. Otherwise use the sandbox-compatible local CLI below; identify that evidence as CLI, not MCP. Do not duplicate both paths for the same question.

Run from `/workspace/linebot-platform`. All commands use the pinned mise environment:

```bash
bash bootstrap.sh exec -- graphify query "validateConfig" --graph /workspace/.cache/linebot-graphify/graphify-out/graph.json --budget 1500 --depth 1
```

Replace the quoted question with the task's symbol or relationship. Quote arguments safely. Start shallow and narrow to relevant symbols. The budget is advisory: Graphify may preserve a complete answer above it. Do not read the entire graph JSON or report into context. Follow the returned source locations with direct reads or gopls before changing code. Graph edges can be inferred and are not proof of architecture compliance.

## Freshness and installation

If the graph is missing, or after checkout/pull/relevant source edits, refresh once before relying on it:

```bash
bash bootstrap.sh run graphify-update
```

One owner refreshes shared graphs; delegated agents reuse that result. Graphs outside the checkout may not report commit freshness correctly, so do not rely only on their metadata. Include checkout HEAD, scoped dirty state and refresh status in handoffs. Use existing graph context during editing and refresh after changes settle.

Missing executables require `bash bootstrap.sh setup` under the normal installation permissions. Do not use unpinned uvx/pip installs, upgrade packages during queries, overwrite user Codex configuration, or install unconditional hooks. `GRAPHIFY_BACKEND=none` and code-only extraction keep this workflow local; document/media semantics and optional Terraform parsing are not included.

## MCP diagnosis

`python3 scripts/codex-cloud/mcp_smoke.py graphify` checks local IPC, initializes stdio, lists tools, reads statistics and queries a source symbol. A local IPC denial is a host sandbox prerequisite failure; do not bypass sandbox policy or increase timeouts to hide it. An approved unsandboxed test proves only that execution context. Native availability requires the host to load project configuration and expose callable Graphify tools in the session. Skill discovery, CLI success and an MCP configuration entry do not establish native availability.
