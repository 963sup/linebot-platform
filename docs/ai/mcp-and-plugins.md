# MCP and Skills V3

Configuration state is not runtime proof. Each MCP must be evaluated separately as configured, enabled, started, initialized, authenticated, authorized and verified.

Repository MCP configuration is inherited by all configured Agents. Agent role files do not create a separate MCP installation, bypass tool approval or grant credentials and IAM. Inheritance only makes a configured server eligible for use: `configured-enabled` does not prove that the host started or initialized it, that authentication and authorization succeeded, or that a tool call was verified.

| MCP | Repository configured state |
| --- | --- |
| `graphify` | `configured-enabled` |
| `gopls` | `configured-enabled` |
| `repomix` | `configured-enabled` |
| `repomix_skills` | `configured-enabled` |
| `filesystem` | `configured-enabled` |
| `chrome_devtools` | `configured-enabled` |
| `playwright` | `configured-enabled` |
| `context7` | `configured-enabled` |
| `github` | `configured-enabled` |
| `openaiDeveloperDocs` | `configured-enabled` |
| `google_developer_knowledge` | `configured-disabled` |
| `google_cloud_cli` | `configured-enabled` |
| `terraform` | `configured-disabled` |
| `line` | `configured-disabled` |
| `openapi_line` | `configured-disabled` |
| `firestore_toolbox` | `configured-disabled` |

## Routing

Narrow questions use direct search. Cross-package structure uses the preinstalled Repomix 1.18.1 runtime in read-only sandbox mode rooted at `/workspace/linebot-platform`; Go symbols and diagnostics then use `gopls`. Versioned third-party SDK documentation uses Context7; OpenAI/Codex documentation uses `openaiDeveloperDocs`; Google Cloud and Workspace documentation uses `google_developer_knowledge`; GitHub state uses GitHub MCP.

With valid runtime credentials, `google_cloud_cli` exposes `run_gcloud_command` and prompt approval. Repository policy limits its intended use to read-only Logging, Monitoring and Trace queries, but that policy text is not a technical read-only boundary on the general command tool. Every invocation still requires approval, and the actual identity and IAM determine what Google Cloud accepts. Terraform exposes public Registry documentation only. The official LINE MCP exposes the approved read-only account tools, while OpenAPI exposes discovery and schema reads without endpoint invocation. Cloud write operations require explicit authorization and a separately verified identity.

`repomix_skills` uses the same pinned executable without Repomix's `--sandbox`, because that mode does not register skill generation. Only `generate_skill` is client-allowlisted; the generator itself accepts arbitrary accessible absolute paths. This is not workspace confinement. Follow the [reference-skill workflow](../../.agents/skills/repomix-reference/SKILL.md), inspect scope/output before approval, and keep the existing read-only `repomix` for analysis.

Browser inspection uses `chrome_devtools` for DOM/CSS, console, network and performance debugging, and `playwright` for UI interaction and accessibility-based flows. Choose the relevant one per task rather than loading both tool schemas or duplicating a browser session. Both launch isolated headless profiles with the existing `/usr/bin/chromium`, preserve TLS validation and use the environment proxy. Host permissions must allow Chromium to start; neither MCP registration nor `tools/list` proves page automation.

## Skills

Skills are installed only from reviewed, pinned sources and selected allowlists. Installation does not authenticate MCPs, grant IAM, or prove that a skill was loaded by the host. The project policy in `AGENTS.md` has priority over third-party skill instructions.

## Cloud environment states

- `READY`: required local tools and required read-only checks passed.
- `DEGRADED`: non-required MCP or Skill is unavailable but local work can continue.
- `BLOCKED`: a required tool, credential, network path or policy prerequisite is unavailable.

No report may claim external API, OAuth, Cloud Run, IAM or production success without actual evidence.

## Graphify local code graph

`mise.toml` owns Python 3.12.14, uv 0.13.7 and `graphifyy[mcp]` 0.9.103 from [Graphify-Labs/graphify](https://github.com/Graphify-Labs/graphify). `bash bootstrap.sh setup` installs these tools and builds a local AST code graph. No separate database or LLM API is required: `GRAPHIFY_BACKEND=none` and `--code-only` skip semantic document/media extraction. Dependency versions below the pinned tools are resolved by the package installer.

After checkout, pull, or relevant source changes, run `bash bootstrap.sh run graphify-update` before querying. The rebuild reads the current checkout (including local edits) and writes `/workspace/.cache/linebot-graphify/graphify-out/`; generated graphs are not tracked. MCP startup only launches the installed `graphify-mcp` against that graph; it never installs packages or rebuilds automatically. A missing graph needs setup/update, not a server configuration workaround.

Use Graphify for relationship/neighborhood/path exploration, with a bounded `query_graph` token budget (for example 1500). Use Repomix for exact scoped source context and gopls for current Go semantics. Graph results can become stale and are not architecture contracts. Seven graph-query tools are client-allowlisted; PR/network/LLM triage tools are excluded. The upstream server accepts a `project_path` override, so this setting is not a filesystem security boundary. Host sandbox permissions remain authoritative. Query logging is disabled.

Verify the actual stdio connection with `python3 scripts/codex-cloud/mcp_smoke.py graphify`; this checks initialization, advertised tools, graph statistics and a code query. A successful local protocol test does not prove that an already-running Codex session has reloaded its MCP registry.

The [project Graphify skill](../../.agents/skills/graphify/SKILL.md) supplies the CLI workflow when native tools are absent. It adapts the reviewed 0.9.103 instructions to the pinned mise executable and external cache, without upstream auto-upgrade steps or unconditional graph-first hooks. Explicit CLI queries work under the current shell sandbox. The Graphify MCP probe first checks local socket IPC: the observed sandbox rejects socket-pair sends with `EPERM`, preventing Python asyncio worker wakeups. Approved execution outside that sandbox passed the same IPC and actual MCP query tests. This does not alter sandbox policy or prove native host registration; the current conversation still has no exposed Graphify tool. Host configuration/loading and process permissions must be resolved by the host, not by installing the skill.
