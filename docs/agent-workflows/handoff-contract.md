# Agent Handoff Contract

Keep handoffs concise. The following fields may share one line; use `None` or `Not Applicable` instead of inventing work. Link existing evidence rather than copying logs or full snapshots. Every delegated result must contain:

```text
Task / Owner / Inputs / Scope / Files / Contract Changes / Evidence / Tests Run / Findings / Risks / Next Action
```

For work that needs cross-file or cross-package relationship evidence and where Repomix MCP is available, `Evidence` must also contain:

```text
Repomix Lifecycle / Repomix Status / Include Patterns / Baseline Output ID / Review Output ID or Not Applicable / Pattern Match or Not Applicable / HEAD / Scoped Fingerprint / Artifact Hash / Stale
```

The orchestrator owns the baseline identity. Writers consume it and, after scoped source changes, create or request a review pack using exactly the same normalized `includePatterns`. If scope expands, the orchestrator records a new baseline before additional edits. Reviewers validate the pair without replacing it. Read-only work with no change lifecycle records one scoped baseline and marks review output and pattern comparison `Not Applicable`; it does not create a no-op pair. Roles that can operate in either mode state which lifecycle applied. A finding still requires confirmation against current source or gopls.

Use `Not Verified` when a command, MCP, credential, deployment or external API was not actually verified. One write owner per file; reviewer roles remain independent and read-only.


Before spawning, the orchestrator records the selected role file, native or explicitly supplied role guidance, bounded scope and acceptance criteria. After spawning, record the actual child ID and verify its executor/checkout identity, HEAD and relevant dirty-state before granting write ownership or consuming artifacts; never infer role loading from a task name or shared files from a path string. Parent and child must verify artifacts exist in the consuming workspace. Final outcomes distinguish proposed, attempted, completed and independently verified work.
