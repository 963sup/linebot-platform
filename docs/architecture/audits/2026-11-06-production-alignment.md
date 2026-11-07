# Production architecture alignment audit — 2026-11-06

## Scope and decision gate

Baseline: `9692e5fe2868a8fb963f0489fba596959dbbf424`. The preceding agent-routing change was pushed before this audit. This report describes the subsequent working-tree implementation, not a deployment.

Actual configured Repomix **MCP** calls packed `**/*.go`, excluding `**/*_test.go`. Baseline output `b181154ccb0c4f3a` contains 234 production files. Four disjoint readers covered all 28,754 flattened lines including file headers; the orchestrator verified an exact union without gaps and checked packed contents against Git. The architect separately read the Blueprint, domain and capability manifests, dependency contract, four prior ADRs and every architecture asset. No source or asset edits preceded that read gate and unified plan approval.

The role instructions were explicitly supplied to agents; a role name alone was not treated as native role loading. Repomix and gopls were exercised through their configured stdio MCP protocol, not inferred from configuration or a CLI listing. This does not certify native tool exposure in every future host session.

The complete baseline inventory classified 2,872 declarations (including package entries, interface methods and assertions): **Keep 2,676; Delete 180; Exception 14; Rename 1; Merge 1**. One unused private helper was reclassified after final gopls evidence and architect approval. All 87 baseline packages were assessed against the six Domain/Application/Adapter/Infrastructure/Strategic/Global matrices. Missing patterns were not treated as scaffolding requirements.

## Unified implementation

Keep the existing layer-first structure, consumer-owned capability ports, live domain transitions, explicit errors and lifecycle ownership. Remove abstractions without production consumers. Preserve candidate strategic status and Identity's real event/outbox path. Correct provider projections additively rather than silently changing existing CLI fields. [ADR 0005](../decisions/0005-production-boundaries-and-compatibility.md) records compatibility and catalog-factory exceptions, owners, tests and exit conditions.

| Finding | Disposition and implementation |
| --- | --- |
| AUD-01 | Fix: route `channel-tokens` from public CLI root; verify private output, parent deadline and cancellation. |
| AUD-02 | Delete duplicate CLI interfaces and impossible dynamic assertions; use existing static runtime contracts. |
| INT-1 / INT-1b | Fix typed audience envelopes and UNIX-second timestamps; retain flat list/create shapes; reject absent nested groups. |
| INT-2 | Delete test-only `codec.Command`; construct local Google route fixtures, preserving absent typed-nil bodies. |
| INT-3 | Rename/specialize workplace upsert to its concrete document; preserve transactional creation time. |
| INT-4 | Delete duplicate OAuthStore compile assertions. |
| INT-5 | Delete private `boolValue`; actual gopls found only its declaration. |
| A1 | Delete test-only content-intelligence orchestration; retain the documented candidate hypothesis. |
| A2 | Delete unused generic insights service; merge live context validation into reports. |
| A3 / A4 | Delete unused presentation/coupon manager families; retain consumed readers, commands and document boundaries. |
| A5 / A6 / A7 | Delete echo-only recipient resolution, unused delivery-count DTO, and old delivery aliases. |
| A8 | Fix demographics, nested report metrics, dates and area/index identity; add accurate follower fields while retaining legacy projections. |
| A9 | Preserve all membership subscriptions, nullable plan/user children, seconds-based join time and provider calendar date; do not invent status/expiry. |
| A10 | Correct candidate Application package comments. |
| D1 | Delete five unused candidate event packages/queues and discard calls. Preserve state/revision/restore invariants, Promotion delivery intent and Identity events. |
| D2 | Stop requiring a repository merely because an aggregate is implemented. Continue validating every declared repository and its ownership. |
| D4 | Delete test-only constructors that lose lifecycle ownership; use managed runtime/client construction and cleanup. |
| V1 / V5 | Correct global-port implications and Application capability versus Domain persistence ownership in assets. |
| V3 | Correct conceptual retry and hypothetical publication semantics. |
| V4 | Label every conceptual Mermaid view as a hypothesis, not an implemented model. |
| V7 | Clarify prospective DeliveryFact publication in the manifest and regenerate the context map. |

## Measured change

Final same-pattern MCP output: `6635827b2b74a357`. Its 226 paths and contents were checked against the live tree, allowing only Repomix's trailing-newline normalization. Production fingerprint: `6d8ba4743fc28da23d2a65ba001bc58b04d089130dcf6647dae08ef41630b8e5`.

| Measurement | Baseline | Final |
| --- | ---: | ---: |
| Production Go files | 234 | 226 |
| Production packages | 87 | 81 |
| Named interfaces | 115 | 101 |
| Physical source lines, including comments/blanks | 28,520 | 27,726 |

AST branch scores were compared for changed functions using the same counter (one plus `if`, loops, non-default cases, and boolean branches). The workplace helper drops from 9 to 4. Typed audience mapping increases from 2 to 13 and statistics mapping from 5 to 20 because they now distinguish real provider shapes and preserve previously discarded data. These are structural measurements, not runtime performance claims; no benchmark improvement is claimed.

## Verification and limits

The final `make check` exited **0**: configuration/domain checks, formatting, `go vet ./...`, `go test -race -count=1 ./...`, integration-tagged compilation, and `go build ./...` all passed. `go run ./cmd/codexcheck`, renderer reproducibility and `git diff --check` also passed. Actual gopls MCP reported zero error diagnostics. The independent reviewer validated the final diff, baseline/review pack pair and evidence, with no remaining actionable findings. The acceptance suite includes realistic nonempty pinned-SDK HTTP responses, all seven insight endpoints, audience null/missing envelopes, flat create shapes, zero/one/multiple subscriptions, provider errors, retained domain transitions and Identity event behavior.

The overview PNG was regenerated and reproduced with the repository's pinned Docker renderer; SVG/PNG hashes and architecture tests were updated together. Configuration/domain generation, Codex declaration checks and actual gopls diagnostics were also exercised. Remaining gopls messages are modernization hints and deliberate nil-context rejection tests; build-tagged integration files are checked through their explicit compilation target.

Firestore emulator execution is **Not Verified**: this workspace has no emulator binary, configured endpoint or cached image. A workplace update/creation-time integration fixture was added; compilation is distinct from executing its transaction. Existing CI provides the pinned emulator job. Live provider credentials, IAM, staging/production behavior and deployment are outside this local verification.

Session evidence is retained outside the repository at `/workspace/scratch/architecture-audit/`: `unified-plan.md`, complete `declarations.json`, `package-matrices.json`, `coverage-union.json`, `findings-register.json`, baseline/review MCP artifacts and identities, gopls references/diagnostics, metrics and test logs. Packed source is deliberately not committed. This report is an audit record, not a replacement for the Blueprint, domain manifest or accepted ADRs.
