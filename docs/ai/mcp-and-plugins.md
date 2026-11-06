# MCP and Skills V3

Configuration state is not runtime proof. Each MCP must be evaluated separately as configured, enabled, started, initialized, authenticated, authorized and verified.

Repository MCP configuration is inherited by all configured Agents. Agent role files do not create a separate MCP installation, bypass tool approval or grant credentials and IAM. Inheritance only makes a configured server eligible for use: `configured-enabled` does not prove that the host started or initialized it, that authentication and authorization succeeded, or that a tool call was verified.

| MCP | Repository configured state |
| --- | --- |
| `gopls` | `configured-enabled` |
| `repomix` | `configured-enabled` |
| `filesystem` | `configured-enabled` |
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

## Skills

Skills are installed only from reviewed, pinned sources and selected allowlists. Installation does not authenticate MCPs, grant IAM, or prove that a skill was loaded by the host. The project policy in `AGENTS.md` has priority over third-party skill instructions.

## Cloud environment states

- `READY`: required local tools and required read-only checks passed.
- `DEGRADED`: non-required MCP or Skill is unavailable but local work can continue.
- `BLOCKED`: a required tool, credential, network path or policy prerequisite is unavailable.

No report may claim external API, OAuth, Cloud Run, IAM or production success without actual evidence.
