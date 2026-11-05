# MCP Inventory

| MCP | Transport | Default | Purpose |
| --- | --- | --- | --- |
| gopls | local stdio | enabled | Go symbols and diagnostics |
| repomix | local npx stdio | enabled | Pinned repository-wide context; read-only sandbox rooted at linebot-platform |
| context7 | remote HTTP | enabled | Versioned SDK documentation |
| github | remote HTTP | enabled | Repository and CI evidence; writes require approval |
| openaiDeveloperDocs | remote HTTP | enabled | OpenAI/Codex/MCP/Plugin docs |
| google_developer_knowledge | remote HTTP | enabled | Google Cloud and Workspace docs |
| google_cloud_cli | remote HTTP | enabled, prompt | Google Cloud Observability read-only queries |
| terraform | local Docker stdio | enabled, prompt | HashiCorp public Registry documentation only |
| openapi_line | local npx stdio | enabled, prompt | Pinned LINE OpenAPI discovery and schema reads |
| line | local npx stdio | enabled, prompt | Official LINE MCP with the approved read-only account tools |
| firestore_toolbox | local stdio | disabled | Staging-only Firestore tools |

Each MCP is tracked as configured, enabled, started, initialized, authenticated, authorized and verified. A remote HTTP MCP does not require a local server binary.

