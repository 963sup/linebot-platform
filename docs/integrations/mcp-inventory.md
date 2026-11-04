# MCP Inventory

| MCP | Transport | Default | Purpose |
| --- | --- | --- | --- |
| gopls | local stdio | enabled | Go symbols and diagnostics |
| context7 | remote HTTP | enabled | Versioned SDK documentation |
| github | remote HTTP | enabled | Repository and CI evidence; writes require approval |
| openaiDeveloperDocs | remote HTTP | enabled | OpenAI/Codex/MCP/Plugin docs |
| google_developer_knowledge | remote HTTP | enabled | Google Cloud and Workspace docs |
| google_cloud_cli | remote HTTP | disabled | Authorized cloud operations only |
| firestore_toolbox | local stdio | disabled | Staging-only Firestore tools |

Each MCP is tracked as configured, enabled, started, initialized, authenticated, authorized and verified. A remote HTTP MCP does not require a local server binary.

