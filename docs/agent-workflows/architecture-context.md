# Architecture context for development agents

Adapted from Repomix's [Architecture Documentation prompt](https://repomix.com/guide/prompt-examples#architecture-documentation) ([official source](https://github.com/yamadashy/repomix/blob/main/website/client/src/en/guide/prompt-examples.md)). Its five questions—overview, interactions, data flow, rationale, constraints—are useful inputs to implementation, not only documentation outputs. Repomix packages source; the assigned agent analyzes it and verifies the result.

## When this helps

Use when a task crosses package/role boundaries, onboarding requires a runtime path, or an agent cannot confidently identify the owner and affected contracts. Skip it for a known local edit. For repository-wide architecture work, follow the existing [full-read gate](go-architecture-audit.md); this narrower context cannot replace it.

The orchestrator chooses the minimum scope, includes the relevant source and governing contracts, and supplies one shared snapshot/fingerprint. Read only applicable sections of the [Blueprint](../architecture/blueprint.md), [Domain manifest](../domain/model.yaml), accepted ADRs and API/persistence contracts. Use current files/gopls to confirm symbols. Include tests when explaining executable behavior. Never expand to the full repository merely to fill a template.

`architect` is the usual owner when design choices need resolution. A specialist can build a small local map as part of its existing assignment; do not create an extra planning agent solely to fill these fields.

## Task prompt

Replace placeholders with the actual task, scope and evidence. Send only the applicable questions and relevant reference sections to each child.

```text
Role: <registered role and explicitly supplied/native-verified role guidance>
Task: <concrete requested behavior and acceptance criteria>
Inputs: <current scoped snapshot ID, patterns, HEAD + dirty fingerprint>
Optional reference: <skill path, provenance identity, freshness, relevant sections>
Authority: <applicable Blueprint / manifest / ADR / API and persistence contracts>

Explain the architecture of this task's path:
1. Overview: entry point, user/business purpose, owner and invariants.
2. Interactions: consuming use case, capability port, implementation and composition
   root; distinguish runtime call flow from allowed Go import direction.
3. Data flow: inputs, validation, state changes, persistence and external effects.
   Include errors, cancellation, retries/idempotency and resource ownership where relevant.
   Use a small Mermaid sequence/flow only if it makes the path clearer.
4. Decisions: explain verified choices with source/ADR evidence. Label inferred
   rationale and proposed alternatives explicitly; do not invent historical intent.
5. Constraints: dependency boundaries, security/transaction/public contract limits,
   missing evidence and unverified external behavior.

Conclude with only what implementation needs: affected files/contracts, one write
owner per scope, the smallest ordered change, regression checks, and open questions.
Attach file/symbol evidence for claims. Distinguish observed, inferred and proposed.
Report missing source instead of filling gaps from a stale snapshot.
```

## Apply the result

| Consumer | Use of shared context | Handoff responsibility |
| --- | --- | --- |
| `architect` | Resolve ownership and contract choices; avoid speculative components. | Decisions, evidence and acceptance criteria. |
| `architecture_auditor` / `researcher` | Navigate to relevant evidence; perform their required independent source checks. | Confirmed findings/unknowns; remain read-only. |
| `domain` | Identify invariants, transitions and persistence semantics. | Tests and model impact, including reference freshness impact. |
| `application` | Identify orchestration, consumer-owned ports, errors and cancellation. | Changed contract/behavior and affected callers. |
| `integration` | Identify SDK/persistence mapping, provider failure and lifecycle boundaries. | Contract fixtures, compatibility and affected adapters. |
| `platform` / `environment` | Identify prerequisites and composition/tooling ownership. | Actual readiness evidence; context is not deployment/IAM permission. |
| `reviewer` | Check claims, changed source and regression evidence independently. | Findings and stale/unknown inputs; never refresh or approve its own authored output. |

Start with the shared orientation, then read only the live files needed for the assigned scope. Send differences and evidence links back, not another copy of the entire map. If a discovery changes shared contracts, notify the orchestrator before dependent edits.

## Keep only useful knowledge

A one-off map belongs in task evidence. A maintained architecture explanation belongs in the existing relevant documentation and remains subordinate to normative contracts. A recurring source-reference need may justify a generated skill under the [reference lifecycle](../../.agents/skills/repomix-reference/SKILL.md); do not create one merely because multiple agents exist.

For generated references, share one reviewed version across roles, check source/path-set/config/contract freshness once per immutable task snapshot, and invalidate it when any writer changes those inputs. Each child verifies it received that same snapshot and checks its live symbols. Assign one refresh owner after source changes and tests settle; reviewers remain read-only. No independent per-agent generations, automatic per-commit refreshes or wholesale copying into every prompt.
