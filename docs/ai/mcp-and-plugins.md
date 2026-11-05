# MCP and Skills V3

Configuration state is not runtime proof. Each MCP must be evaluated separately as configured, enabled, started, initialized, authenticated, authorized and verified.

| MCP | 狀態 |
| --- | --- |
| `gopls` | 啟用 |
| `context7` | 啟用 |
| `github` | 啟用 |
| `openaiDeveloperDocs` | 啟用 |
| `google_developer_knowledge` | 啟用 |
| `google_cloud_cli` | 啟用 |
| `terraform` | 啟用 |
| `line` | 啟用 |
| `openapi_line` | 啟用 |
| `firestore_toolbox` | 停用 |

## Routing

Go symbols use `gopls`; versioned third-party SDK documentation uses Context7; OpenAI/Codex documentation uses `openaiDeveloperDocs`; Google Cloud and Workspace documentation uses `google_developer_knowledge`; GitHub state uses GitHub MCP. `google_cloud_cli` is restricted by policy to read-only Logging, Monitoring, and Trace queries; every command still requires approval. Terraform exposes public Registry documentation only. The official LINE MCP exposes the approved read-only account tools, while OpenAPI exposes discovery and schema reads without endpoint invocation. Cloud write operations require explicit authorization and a separately verified identity.

## Skills

Skills are installed only from reviewed, pinned sources and selected allowlists. Installation does not authenticate MCPs, grant IAM, or prove that a skill was loaded by the host. The project policy in `AGENTS.md` has priority over third-party skill instructions.

## Cloud environment states

- `READY`: required local tools and required read-only checks passed.
- `DEGRADED`: non-required MCP or Skill is unavailable but local work can continue.
- `BLOCKED`: a required tool, credential, network path or policy prerequisite is unavailable.

No report may claim external API, OAuth, Cloud Run, IAM or production success without actual evidence.
