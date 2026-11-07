# Agent Handoff Contract

Use this protocol whenever work crosses an agent boundary. It is a four-state handshake: `PREPARE -> READY -> START -> RESULT`. A spawn is not permission to edit; write ownership begins only at `START`.

Keep handoffs concise. Use `None`, `Not Applicable`, or `Not Verified` instead of inventing evidence, and link existing artifacts rather than copying logs or snapshots. The orchestrator assigns a stable `Handoff ID` and retains it through every state.

## PREPARE — orchestrator to child

Before spawning, record:

```text
Handoff ID / State=PREPARE / Task / Requested Role / Role Guidance Mode / Lifecycle / Acceptance Criteria / Inputs / Scope / File Ownership / Authority / Expected HEAD / Scoped Dirty State or Fingerprint / Snapshot Identity / Selected Skills and Fingerprints / Required Tool Capabilities / Output Contract
```

- `Role Guidance Mode` is `native-verified` only when the host role selector was observed; otherwise it is `explicit` and the selected `.codex/agents/<role>.toml` instructions are supplied in the task.
- `Lifecycle` is `read-only`, `writer`, `reviewer`, or a bounded `read-only-or-writer` choice. `Authority` lists permitted mutations and external effects; it never exceeds user or host authorization.
- `File Ownership` names the sole writer for every writable file. Shared files are sequenced, not concurrently owned.
- `Snapshot Identity` records normalized include patterns, output ID, HEAD, scoped fingerprint and artifact hash when Repomix or another generated reference is used.
- `Selected Skills and Fingerprints` records only relevant skills, their path/provenance, and content or source identity. Naming a skill is not proof it was read.
- `Required Tool Capabilities` distinguishes `native-exposed`, `local-smoke`, `CLI-fallback`, `unavailable`, `authentication-blocked`, and `authorization-blocked`.

## READY — child to orchestrator

Before analysis, edits, installation, or artifact consumption, the child replies with observed state:

```text
Handoff ID / State=READY / Actual Role Guidance / Lifecycle / pwd / Resolved Checkout / HEAD / Scoped Dirty State or Fingerprint / Loaded Contracts and Skills / Tool Capability Matrix / Snapshot or Artifact Visibility / Ownership Accepted / Blockers
```

The child must read the supplied role guidance, this contract, and every selected skill before claiming them as loaded. `Loaded Contracts and Skills` includes paths and fingerprints or source identities. Configuration such as `enabled=true` is not capability evidence. The child reports mismatched HEAD, missing artifacts, overlapping ownership, unavailable required tools, or scope ambiguity as a blocker instead of silently adapting.

The orchestrator compares `READY` with `PREPARE`. If identity or scope differs, it refreshes the baseline or issues a corrected `PREPARE`; it does not approve stale work.

## START — orchestrator authorization

After a matching `READY`, the orchestrator records:

```text
Handoff ID / State=START / Approved Scope / Approved File Ownership / Accepted Identity / Accepted Tool Fallbacks / Dependencies / Start Conditions
```

Only `START` grants writer ownership. Read-only investigation may proceed only after identity and inputs are accepted. Scope expansion, ownership changes, a new checkout/HEAD, or a changed reference source returns the handoff to `PREPARE` and `READY`. A degraded optional tool may use an accepted fallback; a missing required capability blocks the task.

## RESULT — child to orchestrator

Every delegated result contains:

```text
Handoff ID / State=RESULT / Outcome / Task / Owner / Inputs / Scope / Files / Contract Changes / Workspace Identity / Tool Evidence / Skill Evidence / Ownership / Evidence / Tests Run / Findings / Risks / Next Action
```

`Outcome` is `COMPLETED`, `PARTIAL`, or `BLOCKED`. Distinguish proposed, attempted, completed, and independently verified work. `Workspace Identity` reports final pwd, checkout, HEAD and scoped dirty state/fingerprint. `Ownership` confirms that only approved files were written. Parent and child verify material artifacts exist in the consuming workspace before relying on them.

Use `Not Verified` when a command, MCP, credential, deployment, host integration, or external API was not actually verified. Reviewers remain independent and read-only.

## Repomix and generated references

For cross-file or cross-package relationship evidence where Repomix MCP is available, `Evidence` also contains:

```text
Repomix Lifecycle / Repomix Status / Include Patterns / Baseline Output ID / Review Output ID or Not Applicable / Pattern Match or Not Applicable / HEAD / Scoped Fingerprint / Artifact Hash / Stale
```

The orchestrator owns baseline identity. Writers consume it and, after scoped source changes, create or request a review pack using exactly the same normalized `includePatterns`. If scope expands, return to `PREPARE` with a new baseline before further edits. Reviewers validate the pair without replacing it. Read-only work records one scoped baseline and marks review output and pattern comparison `Not Applicable`; it does not manufacture a no-op pair. Findings still require confirmation against current source or gopls.

For cross-boundary orientation, use [architecture-context](architecture-context.md) inside the existing role assignment. A known local edit needs no extra architecture document or agent.

When a reference skill is relevant, `Inputs` records its path, provenance/source fingerprint, freshness result and only the needed sections. The orchestrator evaluates the [reference lifecycle](../../.agents/skills/repomix-reference/SKILL.md) once for the shared immutable snapshot; children verify that identity, explicitly read the selected skill and confirm live symbols before edits. Do not paste the whole pack or make every child regenerate it.

Writers report whether changes leave reference sources and governing contracts `unchanged`, `refreshed` with new identity, or `stale/deferred` with reason. A stale reference cannot be handed onward as current. The orchestrator assigns one refresh owner after relevant changes and tests settle. When no reference is used, record `Not Applicable`; no separate form or extra agent is required.
