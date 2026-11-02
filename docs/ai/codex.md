# Codex 使用方式

- 根目錄 `AGENTS.md` 與各層 `AGENTS.md`：規範與依賴邊界。
- `.agents/skills/`：可重複的開發、檢查和 release 工作流程。
- `.codex/config.toml.example`：僅是可審查的非啟用範例，不含憑證。
- `.agents/plugins/`、`plugins/`：保留正式 plugin / marketplace 空間，現無安裝 manifest。
- MCP：使用者獨立授權的外部工具；不會自動賦予 production runtime 權限。

參閱 [Skills](skills.md)、[MCP 與 Plugins](mcp-and-plugins.md)。
