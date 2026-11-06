# Codex 專案設定

**正式專案配置**：[`.codex/config.toml`](config.toml)，並非早期的 `config.toml.example`。

| 內容 | 路徑 | 用途 |
|---|---|---|
| Agents | [`.codex/agents/`](agents/) | `config.toml` 註冊的 depth-one specialist roles（每個一個 TOML） |
| MCP | [`config.toml`](config.toml) | gopls、Repomix、Context7、OpenAI Docs、可寫 GitHub；Firestore/Google remote MCP 受憑證與安全閘門限制 |
| Cloud lifecycle | [`scripts/codex-cloud/`](../scripts/codex-cloud/) | 集中的 install、start、verify、repair 與 PATH 契約 |
| 規範 | [`AGENTS.md`](../AGENTS.md) | 判斷什麼任務用哪個 MCP／Agent，不負責安裝 |
| Skills | [`.agents/skills/`](../.agents/skills/) | 重複性任務工作流程 |

先閱讀 [Agents 設計](../docs/ai/agents.md)、[MCP 政策](../docs/ai/mcp-and-plugins.md)、[Cloud setup 指引](../docs/ai/codex-cloud.md)。

- 專案級 `config.toml` 只在 Codex **信任此儲存庫** 時載入；不要將 token 寫到 Git。
- `gopls` 本地 MCP 透過 Cloud install script 安裝；Repomix 固定為 1.18.1，並以 `/workspace/linebot-platform` 為唯讀 sandbox。Context7 與 OpenAI Docs 使用遠端文件 MCP。OpenAI Docs 須允許 `developers.openai.com` 的 agent-phase 網路連線。
- GitHub MCP 已啟用並可在明確授權下寫入，且僅使用安全環境提供的 `GITHUB_MCP_TOKEN`；Firestore Toolbox 維持停用。
- 角色清單與數量只以 `config.toml` 為準；全庫 Go 架構任務由唯讀 `architecture_auditor` 先依完整閱讀閘門分析；不得額外建立 orchestrator／reviewer profile Agent。
- [Go 全域架構稽核流程](../docs/agent-workflows/go-architecture-audit.md) 定義 Repomix 全讀、統一規劃、分批實施及獨立審查；TOML 註冊不保證 Host 自動派工，實際路由仍由主 Session 依 `AGENTS.md` 執行。
- Agent 檔案不能自行擴權或代替主 Session 的 sandbox、approval、MCP 授權。
- Codex 設定驗證已改用純 Go 程式 `go run ./cmd/codexcheck`，不依賴 Python 檔案或 Python 執行環境。
- 完成 PR 後，Codex Cloud 的管理者仍需在 Cloud UI 測試、Publish／Republish 才可重用此環境。
