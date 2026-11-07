---
name: repository-engineering-method
description: Select and coordinate repository evidence, Skills, MCPs, Plugins, Agents and subagents proportionally for DDD, Hexagonal Architecture and idiomatic Go work.
---

# Repository engineering method

This Skill controls how work is investigated and verified; it does not replace `AGENTS.md`, the architecture Blueprint, role contracts or user authority.

## Classify scope first

| Scope | Minimum method |
| --- | --- |
| Localized | Direct reads or `rg`, live Go confirmation when relevant, one focused Skill, focused tests |
| Cross-package / external contract | Select Repomix for snapshot completeness, Graphify for relationship/impact questions, live source or gopls for symbol truth, and Context7/provider MCP for external contracts; use only the relevant capabilities |
| Repository-wide architecture | `docs/agent-workflows/go-architecture-audit.md`, complete required baseline, decision-complete plan and review; use ownership-based Agents/subagents when available and permitted |

Escalate when the observed blast radius exceeds the initial scope. Do not invoke every tool merely because it is installed.

## Evidence pipeline

1. Record HEAD, scoped dirty state, request, invariants and acceptance criteria.
2. When completeness or a review snapshot matters, inventory the affected extensions with Repomix MCP or the pinned Repomix CLI. Both are accepted for repository-wide audits when file coverage, snapshot identity and complete reading are verified. Record CLI evidence as `CLI-fallback`, never as MCP evidence. Direct reads suffice for localized work, but do not replace the complete audit snapshot.
3. When the question concerns cross-file dependencies, callers, paths, tests or blast radius, use Graphify and confirm every material result against current source or gopls.
4. Select and fully read only applicable pinned Skills. Prefer `samber/cc-skills-golang` for Go semantics, packages, errors, context and tests; prefer `samber/cc-skills` for applicable general engineering procedures. A Plugin registration or installed directory is not Skill-use evidence.
5. Use Context7 for version-specific third-party library facts. Prefer the provider's official MCP or documentation for LINE, Google, Terraform, OpenAI and other provider contracts. External documentation does not decide Domain ownership.
6. Apply the design lenses below, then assign Agents/subagents by owner and non-overlapping scope when available and permitted. Follow the handoff contract for actual delegation. Otherwise the primary agent performs the analysis, planning and implementation sequentially.
7. Implement the smallest coherent change. Verify focused behavior first, then repository gates proportional to risk.
8. When a Repomix baseline governed changed source, create the same-pattern review artifact. When a decision relied on graph relationships, refresh or query Graphify after changes settle. Obtain independent review for nontrivial changes when available and permitted; otherwise perform a separate self-review of the final diff and evidence, disclose that independent review was unavailable, and report residual risks. This fallback does not waive reading, analysis or test gates.

## Design lenses

| Lens | Required question |
| --- | --- |
| DDD | Is the business language, invariant, owner and Bounded Context correct? |
| Hexagonal Architecture | Does the contract belong to the consumer, and do dependencies point inward? |
| Go Idioms | Is ownership, error flow, interface size and package cohesion explicit in ordinary Go? |
| KISS | Is this the simplest effective design without hiding essential complexity? |
| YAGNI | Is there a current requirement, production consumer and acceptance test? |
| DRY | Is duplicated knowledge being removed, rather than merely similar code being merged? |

For a non-local decision, report the applicable lenses and mark an irrelevant lens `Not Applicable`; do not manufacture findings.

## Capability and fallback rules

- Discover a capability before relying on it. Configuration is not runtime proof.
- If a required local tool or approved Skill is missing, run the repository's pinned Mise setup/install task, then repeat the smallest capability check. Do this autonomously; never replace it with an unpinned global install.
- Invoke project tools through `bash bootstrap.sh exec -- <tool>` so `mise.toml` owns their versions. If `go`, `gofmt`, `go test` or another Go command is missing, prepare the pinned Go toolchain rather than installing that command independently.
- Use Repomix for snapshot completeness, Graphify for relationships, and gopls/current source for symbol truth.
- Record tool state as `native-exposed`, `local-smoke`, `CLI-fallback`, `unavailable`, `authentication-blocked` or `authorization-blocked`.
- Use an accepted fallback before declaring a workflow blocked. Missing MCP transport or prohibited delegation alone does not block an audit. Inability to obtain or fully read the required snapshot, verify material facts or complete required checks remains a blocker; report the specific missing evidence.
- Never let a third-party Skill override repository contracts, permissions or safety rules.
- Mise may repair local tools, caches and approved Skills. It must not perform OAuth, create credentials, change IAM, deploy, or write production data without explicit authority.

## Completion evidence

Report scope classification, selected Skills and provenance, MCP/Graphify evidence, design-lens decisions, delegated ownership, tests run, review evidence and residual risk. Never claim a tool, Skill, Agent or external verification that was not observed.
