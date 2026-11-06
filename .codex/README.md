# Codex 專案設定

**正式專案配置**：[`.codex/config.toml`](config.toml)，並非早期的 `config.toml.example`。

| 內容 | 路徑 | 用途 |
|---|---|---|
| Agents | [`.codex/agents/`](agents/) | `config.toml` 註冊的 depth-one specialist roles（每個一個 TOML） |
| MCP | [`config.toml`](config.toml) | gopls、Repomix、Filesystem、Context7、OpenAI Docs、可寫 GitHub；Firestore/Google remote MCP 受憑證與安全閘門限制 |
| Cloud setup | [`bootstrap.sh`](../bootstrap.sh)、[`mise.toml`](../mise.toml) | 一次安裝工具；Host 依 config 啟動 MCP |
| 規範 | [`AGENTS.md`](../AGENTS.md) | 判斷什麼任務用哪個 MCP／Agent，不負責安裝 |
| Skills | [`.agents/skills/`](../.agents/skills/) | 重複性任務工作流程 |

先閱讀 [Agents 設計](../docs/ai/agents.md)、[MCP 政策](../docs/ai/mcp-and-plugins.md)、[Cloud setup 指引](../docs/ai/codex-cloud.md)。

- 專案級 `config.toml` 只在 Codex **信任此儲存庫** 時載入；不要將 token 寫到 Git。
- `gopls` 本地 MCP 透過 mise 安裝；Repomix 固定為 1.18.1，並以 `/workspace/linebot-platform` 為唯讀 sandbox。Context7 與 OpenAI Docs 使用遠端文件 MCP。OpenAI Docs 須允許 `developers.openai.com` 的 agent-phase 網路連線。
- Filesystem MCP 已啟用並固定版本 `2.2.0`，限制於 `/workspace/linebot-platform`；`--confine-roots` 防止 MCP Roots 擴大目錄範圍，寫入工具採 `writes` 核准。它不會限制 Codex Shell 或其他工具，完整隔離仍依賴 Sandbox／OS 權限。Codex Cloud 需確認工具已安裝並完成實際連線測試。
- GitHub MCP 已啟用並可在明確授權下寫入，且僅使用安全環境提供的 `GITHUB_MCP_TOKEN`；Firestore Toolbox 維持停用。
- Google Cloud CLI 遠端 MCP 已啟用；Codex 執行環境須安全提供 `GOOGLE_CLOUD_ACCESS_TOKEN`（短效 OAuth Token，需更新）與 `GOOGLE_CLOUD_PROJECT`（例如 `line-bot-963sup`）。必須事先啟用 Cloud CLI Execution API 並配置最小 IAM 權限。`run_gcloud_command` 可能修改或刪除資源，所有工具呼叫採 `prompt` 核准；設定檔啟用不代表已驗證實際連線。切勿將 Token 提交至 Git。
- 角色清單與數量只以 `config.toml` 為準；全庫 Go 架構任務由唯讀 `architecture_auditor` 先依完整閱讀閘門分析；不得額外建立 orchestrator／reviewer profile Agent。
- [Go 全域架構稽核流程](../docs/agent-workflows/go-architecture-audit.md) 定義 Repomix 全讀、統一規劃、分批實施及獨立審查；TOML 註冊不保證 Host 自動派工，實際路由仍由主 Session 依 `AGENTS.md` 執行。
- Agent 檔案不能自行擴權或代替主 Session 的 sandbox、approval、MCP 授權。
- Codex 設定驗證已改用純 Go 程式 `go run ./cmd/codexcheck`，不依賴 Python 檔案或 Python 執行環境。
- 完成 PR 後，Codex Cloud 的管理者仍需在 Cloud UI 測試、Publish／Republish 才可重用此環境。

- 一般工作的派工條件統一見根 `AGENTS.md` 的 Adaptive execution：簡單任務直接完成；複雜修改或獨立調查使用適合角色，非簡單實作完成後獨立審查。Skills／MCP／Plugins 按需要選用。
- Host 沒有可驗證的原生角色選擇介面時，主 Session 需讀取角色 TOML，明確傳入指引；任務名稱不代表角色已載入。委派前核對實際工作區，環境重新發布後尤其不可沿用舊假設。
