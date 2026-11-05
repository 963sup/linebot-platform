# Codex 與儲存庫整合

- 根 `AGENTS.md` 與各層 `AGENTS.md`：固定的開發規範、工具選用準則；**不執行安裝**。
- `.agents/skills/`：已審核、可重複的開發流程；SDK 任務使用 `sdk-integration`，先讀 `docs/sdk/catalog.toml`，再按需查詢真實 API。
- `.codex/agents/*.toml`：專用 Subagents 的角色指引；由主 Agent 委派。
- `.codex/config.toml`：專案 MCP 連線與有限的協作設定，沒有憑證。
- `scripts/codex-cloud/`：集中管理 install、start、verify、repair 與唯一 PATH 契約；**不執行 MCP 登入**。
- `.agents/plugins/`、`plugins/`：預留經審核的 plugin 結構；現無自動安裝的外部 plugins。

SDK 依賴以 `go.mod` 為唯一版本來源，`go run ./cmd/codexcheck` 同時檢查 Catalog 與直接依賴是否一致。Catalog 的 `declared` 不等於服務已啟用，且不提供憑證或外部權限。

詳見 [Agents 設計](agents.md)、[Cloud setup](codex-cloud.md)、[MCP 與 Plugins](mcp-and-plugins.md)、[Skills](skills.md)。

專案級設定需被 Codex 判定為 trusted。官方 Codex CLI／IDE 支援專用 Agents；Codex Cloud 是否載入 Agents／工具，需在 Publish 後以實際工作階段驗證，不能僅以 TOML 存在推定。
