# MCP Inventory

| MCP | Transport | Default | Purpose |
| --- | --- | --- | --- |
| gopls | local stdio | enabled | Go symbols and diagnostics |
| repomix | preinstalled local stdio | enabled | Pinned repository-wide context; read-only sandbox rooted at linebot-platform |
| context7 | remote HTTP | enabled | Versioned SDK documentation |
| github | remote HTTP | enabled | Repository and CI evidence; writes require approval |
| openaiDeveloperDocs | remote HTTP | enabled | OpenAI/Codex/MCP/Plugin docs |
| google_developer_knowledge | remote HTTP | enabled | Google Cloud and Workspace docs |
| google_cloud_cli | remote HTTP | enabled, prompt | General gcloud command tool; repository policy limits intended use to read-only Observability queries |
| terraform | local Docker stdio | enabled, prompt | HashiCorp public Registry documentation only |
| openapi_line | preinstalled local stdio | enabled, prompt | Pinned LINE OpenAPI discovery and schema reads |
| line | preinstalled local stdio | enabled, prompt | Official LINE MCP with the approved read-only account tools |
| firestore_toolbox | preinstalled local stdio | disabled | Staging-only Firestore tools |

The local npm-backed servers are installed together from `tools/mcp-runtime/package-lock.json` into `/workspace/.linebot-tools/mcp-runtime`; Codex starts their fixed local binaries rather than resolving packages with `npx` at task time. The runtime receipt, lock and package versions are checked separately from MCP initialization and tool calls.

All configured Agents inherit the repository MCP configuration. This makes the servers available for host initialization; it does not grant credentials, IAM or approval bypasses. Each MCP is tracked separately as configured, enabled, started, initialized, authenticated, authorized and verified. In particular, `configured-enabled` is not evidence of initialization, authentication or a successful tool call. A remote HTTP MCP does not require a local server binary.

`google_cloud_cli` exposes `run_gcloud_command`, so the repository's read-only Observability rule is policy-only rather than a technical restriction of the tool. Prompt approval and the effective Google identity and IAM remain separate enforcement layers.

