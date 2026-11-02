# Codex 專案 Agents 設計

本目錄 `.codex/agents/*.toml` 採用 [Codex custom agents 官方格式](https://learn.chatgpt.com/docs/agent-configuration/subagents)。各檔案必須包含 `name`、`description`、`developer_instructions`，並繼承主工作階段的 MCP、權限、供應商與工具。**Agent instructions 不是獨立的只讀沙盒或權限機制**。

## 責任分工

| Agent | 職責 | MCP 選擇 | 實作權限 |
|---|---|---|---|
| `domain_architect` | Bounded Context、Aggregate、Value Object、Port 邊界 | gopls | 預設設計分析 |
| `line_integrator` | Messaging API Webhook／Reply／Push／Flex、事件安全 | gopls、Context7、LINE 官方來源 | 被委派後可修改相應檔案 |
| `cloud_platform` | Cloud Run、Firestore、Cloud Tasks、WIF、IAM | Context7，隔離 staging 才允許 Toolbox | 預設只審查；不可自行部署 |
| `go_reviewer` | Go correctness、Race、CI、PR 安全審查 | gopls、授權 GitHub（唯讀） | 只審查 |
| `docs_researcher` | SDK 版本、Go module、API 參數、Codex 規格 | Context7、官方來源 | 只查證 |

Codex 內建 `worker`、`explorer` 可處理一般任務，不再額外建立通用 agent。主 Agent 負責拆分任務、避免併發編輯同一檔案、整合與最後驗證。只有實際互不依賴的工作才並行委派；建議最多三個子 Agent。

## 使用範例

- 修改 Conversation 聚合：先由 `domain_architect` 釐清不變量，再交給內建 `worker` 實作；`go_reviewer` 審查。
- 新增 LINE Flex／Webhook event：由 `docs_researcher` 核對版本、`line_integrator` 實作、`go_reviewer` 驗證。
- 準備 Cloud Tasks Worker：`domain_architect` 釐清 Port、`cloud_platform` 審查 IAM 與資料一致性，主 Agent 排程實作。
- 判斷 SDK 升級：`docs_researcher` 核對正式來源，主 Agent 決定升級範圍。

## 禁止誤用

- `AGENTS.md` 是每次工作的規範；`.agents/skills/` 是工作流程；`.codex/agents/` 是角色；`.codex/config.toml` 才配置 MCP。這些不是同一種設定。
- 不要把 production 的 LINE 密鑰、Google ADC 或 GitHub PAT 放到 GitHub、TOML、skill 或 Agent instructions。
- MCP 寫入能力受外部帳號權限控制；即使 Agent 文本說「唯讀」，也不是技術隔離。正式環境權限需由 IAM／MCP allowlist 控制。
- Codex CLI／IDE 官方支援專案 Agent 檔；Codex Cloud 的工作階段是否載入該角色及 MCP 仍須在**已發布的雲端環境**實測，不以檔案存在保證。
