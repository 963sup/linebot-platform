---
name: repomix-reference
description: Evaluate, generate, review and refresh narrowly scoped repository reference skills with Repomix MCP when repeated cross-file exploration would benefit from reusable context.
---

# Repomix reference skill lifecycle

Use this workflow when considering `repomix_skills.generate_skill`, or when a generated reference skill is relevant to a task. It supplies a reusable source snapshot, not new architectural authority or executable expertise. Follow root AGENTS.md; generation does not grant permissions.

## Decide before generating

Generate only when all apply:

- The same bounded cross-file subject recurs across tasks or agents, or a specific next reuse is identified.
- Existing procedural skills do not already provide the needed source context. A reference may complement `sdk-integration` or a Domain workflow; do not duplicate their instructions.
- A named maintainer role, source scope and governing contracts can be recorded, and the expected reduction in repeated exploration justifies maintaining a copy.

Good candidates: a recurring webhook delivery path, adapter/port wiring for a specific capability, or a bounded runtime lifecycle map. These are candidates, not instructions to create all of them.

Use direct reads/rg for a few known files. Use sandboxed `repomix.pack_codebase` for one-off investigation, a rapidly changing patch or a full audit. A generated skill cannot replace the full-read audit gate, current gopls evidence or tests. Do not generate the entire repository or auto-generate at startup/every commit. There is no guaranteed token saving: keep the entry point short and read only relevant reference sections.

## Check freshness before use

Read `references/provenance.md` before relying on a generated reference. It must record:

- Owner role, concrete reuse case and last validation date.
- Source repository, generation HEAD and dirty state (HEAD alone does not identify dirty inputs).
- Normalized `includePatterns` / `ignorePatterns`, Repomix version, `compress`, and effective Repomix config/ignore inputs.
- Exact selected repository-relative source paths and content SHA-256 values in `references/sources.sha256`; include governing contract files even if they are not copied into the reference.
- Output metrics and exact generation recipe, including staging and final skill names.

Compare current file hashes and the current selected path set with provenance; additions, deletions and renames matter as well as edits. `sha256sum -c <skill>/references/sources.sha256`, run from the source repository, checks recorded content but **does not detect new files**. Re-enumerate the recorded scope with the same selection rules, or use a scoped pack's file inventory when membership is uncertain. Check config/ignore files too. An unrelated HEAD change or elapsed time alone is not staleness.

| Evidence | Action |
| --- | --- |
| Same selected paths, source hashes, governing contracts and selection rules | Reuse; do not regenerate. |
| Relevant source, contract, membership, selection setting or generator change | Read live source first; refresh before the next use, normally at completion of the relevant change. |
| Scope expands or ownership/architecture changes | Reassess whether the reference is still useful; revise scope and description before generation. |
| Missing provenance, unreviewed output or uncertain freshness | Treat as unverified; use current source until reviewed/regenerated. |
| No recurring consumer, duplicate reference or repeated churn outweighs reuse | Retire the generated skill after checking consumers; do not maintain it automatically. |

A writer changing a generated skill's recorded sources checks this table during handoff. Finish source changes and tests before refreshing the reference. Do not continuously regenerate while editing. A fresh reference still requires live symbol/behavior checks before code changes.

## Choose the generation interface

The existing `repomix` remains sandboxed/read-only. The separate `repomix_skills` server exposes only `generate_skill`, with `prompt` approval. It uses the same mise-pinned Repomix; no new package or background service is needed.

Repomix 1.18.1 does **not** confine the generator's absolute `directory` argument. Its allowlist and approval policy are client controls, not an OS sandbox. For this project choose only the canonical `/workspace/linebot-platform` checkout (or a disposable fixture inside it for testing). Inspect the source paths, effective config and output parents; do not follow symlink escapes. Preserve host filesystem permissions and security scanning. Never disable scanning to make generation pass.

Before calling, verify that the selection is nonempty, task-scoped and excludes credentials, generated outputs and caches. Explicitly exclude `.claude/**` and `.agents/skills/**` to prevent recursive snapshots. Use `compress=false` when behavior/invariants matter; compressed references cannot prove implementation semantics.

Example **shape**, adapt the scope to the actual recurring task:

```json
{
  "directory": "/workspace/linebot-platform",
  "skillName": "repomix-reference-runtime-candidate-UNIQUE",
  "includePatterns": "internal/bootstrap/runtime.go,internal/bootstrap/shutdown.go,docs/architecture/blueprint.md",
  "ignorePatterns": ".claude/**,.agents/skills/**,**/.env,**/.env.*,**/*_test.go",
  "compress": false
}
```

Use a fresh valid kebab-case name (replace `UNIQUE` with a lowercase suffix). Native MCP writes `.claude/skills/<name>/` and rejects an existing destination. Do not delete the installed skill to make regeneration succeed; generate a fresh candidate. A shell/CLI fallback is not evidence that this MCP was called.

### CLI can write directly into `.agents/skills`

The **CLI** has `--skill-output`; the 1.18.1 **MCP input schema does not**. Do not add unsupported arguments to `generate_skill` or claim a CLI invocation as MCP evidence. When the task is direct Codex skill creation and an MCP call is not required, use the existing pinned CLI without a wrapper or intermediate `.claude` directory:

```sh
bash bootstrap.sh exec -- repomix \
  --skill-generate repomix-reference-runtime \
  --skill-output .agents/skills/repomix-reference-runtime \
  --include 'internal/bootstrap/runtime.go,internal/bootstrap/shutdown.go,docs/architecture/blueprint.md' \
  --ignore '.claude/**,.agents/skills/**,**/.env,**/.env.*,**/*_test.go'
```

`--skill-output` names the **skill directory itself**, containing `SKILL.md` and `references/`; do not point it at the shared `.agents/skills` parent. Review and add provenance before relying on the new skill. For an update, generate into a fresh temporary output directory and compare before replacement. `--force` only skips overwrite confirmation; use it only for an intentionally replaceable generated directory after checking ownership and preserving the previous reviewed version. Never use it on hand-authored skills or the shared parent.

Remote CLI generation is a separate option; use a reviewed pinned source revision, not a moving branch, and record it in provenance. It is not needed for this repository's local lifecycle.

## Review and promote for Codex

1. Verify the returned path and read generated `SKILL.md`, summary, structure and all selected reference content. Reject leaked credentials, missing required source, invented instructions or excessive scope. Inspect security exclusions: a successful call can omit detected sensitive files.
2. Add provenance and source hashes. Keep copied source clearly subordinate to the Blueprint, manifests, accepted ADRs and current source. Keep `SKILL.md` focused on when to use it, the freshness check and navigation; avoid repeating all reference contents in the entry point.
3. For MCP/staged output, promote the reviewed candidate into `.agents/skills/<stable-name>/`. For direct CLI output, the file is already in the final directory. Normalize the frontmatter `name`, description and any staging-name references/links. Check every relative link. Never overwrite hand-authored skills or foreign content. For an update, compare with the previous reviewed generated reference and replace only its owned files; keep recovery through Git/the previous candidate until validation succeeds.
4. Verify final files, provenance and relevant repository checks. Commit reviewed references only when justified by the recurring consumer; temporary packed output stays out of Git. Remove only this operation's staging artifacts after successful promotion. If promotion fails, leave the installed skill intact and report the candidate path.
5. Report source scope, reuse rationale, MCP output metrics, freshness evidence and actual validation. Files on disk are not proof that this host discovered the skill; verify discovery in a new host session when available, otherwise say `Not Verified`.

## Use with agents

Apply the shared [architecture-context workflow](../../../docs/agent-workflows/architecture-context.md) when a recurring cross-component task needs overview, interactions, data flow, decision evidence or constraints. The orchestrator selects this reference plus the appropriate procedural skill, verifies freshness for the shared snapshot, and passes only the relevant sections and provenance identity in `Inputs`.

A child verifies the snapshot/checkout identity and reads its selected instructions; it does not regenerate a private copy. Writers report whether their edits invalidate the reference. Assign one refresh owner after dependent changes and tests settle; the reviewer checks evidence without editing. Use `Not Applicable` when no reusable reference is warranted. These rules apply through the common handoff contract to all nine roles, without duplicating the full workflow in every role TOML.
