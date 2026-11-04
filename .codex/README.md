# Codex 專案設定

**正式專案配置**：[`.codex/config.toml`](config.toml)，並非早期的 `config.toml.example`。

| 內容 | 路徑 | 用途 |
|---|---|---|
| Agents | [`.codex/agents/`](agents/) | 十七個有層級的專用角色（每個一個 TOML） |
| MCP | [`config.toml`](config.toml) | gopls、Context7、OpenAI Docs、可寫 GitHub、Firestore Toolbox |
| Cloud setup | [`scripts/setup-codex-cloud.sh`](../scripts/setup-codex-cloud.sh) | 非互動式下載 Go modules、安裝固定版 gopls |
| 規範 | [`AGENTS.md`](../AGENTS.md) | 判斷什麼任務用哪個 MCP／Agent，不負責安裝 |
| Skills | [`.agents/skills/`](../.agents/skills/) | 重複性任務工作流程 |

先閱讀 [Agents 設計](../docs/ai/agents.md)、[MCP 政策](../docs/ai/mcp-and-plugins.md)、[Cloud setup 指引](../docs/ai/codex-cloud.md)。

- 專案級 `config.toml` 只在 Codex **信任此儲存庫** 時載入；不要將 token 寫到 Git。
- `gopls` 本地 MCP 透過 Cloud install script 安裝，Context7 與 OpenAI Docs 使用遠端文件 MCP。OpenAI Docs 須允許 `developers.openai.com` 的 agent-phase 網路連線。
- GitHub MCP 已啟用並可在明確授權下寫入，且僅使用安全環境提供的 `GITHUB_MCP_TOKEN`；Firestore Toolbox 維持停用。
- `google_identity_workspace_integrator` 專責 Google OAuth 與 Workspace 的每使用者整合；`line_integrator` 統一負責 LINE SDK v8 家族。`code_cleanup_release_validation` 專責驗收後的證據式清理與發行檢查；預留 SDK 直接依賴、空白骨架與必要回歸測試受保護，不得因未使用就刪除。
- Agent 檔案不能自行擴權或代替主 Session 的 sandbox、approval、MCP 授權。
- Codex 設定驗證已改用純 Go 程式 `go run ./cmd/codexcheck`，不依賴 Python 檔案或 Python 執行環境。
- 完成 PR 後，Codex Cloud 的管理者仍需在 Cloud UI 測試、Publish／Republish 才可重用此環境。
