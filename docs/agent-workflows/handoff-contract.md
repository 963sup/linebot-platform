# Agent Handoff Contract

Every delegated result must contain:

```text
Task / Owner / Inputs / Scope / Files / Contract Changes / Evidence / Tests Run / Findings / Risks / Next Action
```

For cross-file work where Repomix MCP is available, `Evidence` must also contain:

```text
Repomix Status / Include Patterns / Initial Output ID / Review Output ID / Pattern Match / HEAD / Scoped Fingerprint / Artifact Hash / Stale
```

Initial and review packs use the same normalized `includePatterns`. If scope expands, record a new initial output before additional edits. Children consume the orchestrator-owned immutable artifact identity; they do not overwrite it. A finding still requires confirmation against current source or gopls.

Use `Not Verified` when a command, MCP, credential, deployment or external API was not actually verified. One write owner per file; reviewer roles remain independent and read-only.

