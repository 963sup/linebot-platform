# Codex 專案 Agents 設計

本目錄 `.codex/agents/*.toml` 採用 [Codex custom agents 官方格式](https://learn.chatgpt.com/docs/agent-configuration/subagents)。各檔案必須包含 `name`、`description`、`developer_instructions`，並繼承主工作階段的 MCP、權限、供應商與工具。**Agent instructions 不是獨立的只讀沙盒或權限機制**。

## 責任分工

| Agent | 職責 | MCP 選擇 | 實作權限 |
|---|---|---|---|
| `domain_architect` | Bounded Context、Aggregate、Value Object、Port 邊界 | gopls | 預設設計分析 |
| `line_integrator` | Messaging API Webhook／Reply／Push／Flex、事件安全 | gopls、Context7、LINE 官方來源 | 被委派後可修改相應檔案 |
| `google_identity_workspace_integrator` | Google OAuth／ID token、使用者 token、Calendar／Tasks／Forms 的每使用者存取 | gopls、Context7、Google 官方來源 | 被委派後可修改相應檔案；不可自行開 OAuth、部署或改 IAM |
| `cloud_platform` | Cloud Run、Firestore、Cloud Tasks、WIF、IAM | Context7，隔離 staging 才允許 Toolbox | 預設只審查；不可自行部署 |
| `go_reviewer` | Go correctness、Race、CI、PR 安全審查 | gopls、授權 GitHub（唯讀） | 只審查 |
| `gopls_go_refactor` | 跨 package rename、interface／DTO 演進、Port／Adapter 搬遷與語意批量修改 | gopls | 被委派後可修改 Go 檔案 |
| `docs_researcher` | SDK 版本、Go module、API 參數、Codex 規格；LINE Developers 文件來源 | SDK Catalog、gopls、Context7、OpenAI Docs、LINE 文件來源、官方來源 | 只查證 |
| `code_cleanup_release_validation` | 驗收後範圍檢查、引用追蹤、清理、受保護 SDK 及正式測試保留、發行驗證 | gopls、Go 工具鏈、CI、SDK Catalog | 經授權才可修改；不可自行合併或部署 |

Codex 內建 `worker`、`explorer` 可處理一般任務，不再額外建立通用 agent。主 Agent 負責拆分任務、避免併發編輯同一檔案、整合與最後驗證。只有實際互不依賴的工作才並行委派；建議最多三個子 Agent。

SDK 功能開發仍以 [sdk-integration Skill](../../.agents/skills/sdk-integration/SKILL.md) 按需查證。LINE 的十一個 v8 package 統一交 `line_integrator`，Google OAuth／Workspace 的八個 package 統一交 `google_identity_workspace_integrator`；不可依 import path 建立互相重疊的角色。兩者都必須將 SDK 型別留在 Adapter／Infrastructure、由 Bootstrap 組裝，並以自有 DTO／小型 Port 隔開 Domain 與 Application。`gopls_go_refactor` 專責 Go 語意批量修改，`code_cleanup_release_validation` 專責交付前清理與驗證。`go.mod` 預先核准的 SDK 直接依賴及其預留空白骨架不得因尚未使用被刪除，Catalog 的能力標籤只是搜尋提示。

## 使用範例

- 修改 Conversation 聚合：先由 `domain_architect` 釐清不變量，再交給內建 `worker` 實作；`go_reviewer` 審查。
- 新增 LINE Flex／Webhook event：由 `docs_researcher` 核對版本、`line_integrator` 實作、`go_reviewer` 驗證。
- 查 LINE 平台規格或文件差異：由 `docs_researcher` 讀取 [line-developers-docs-source](https://github.com/line/line-developers-docs-source) 與已發布的 LINE Developers 頁面，再交 `line_integrator` 依鎖定 SDK 實作；不另建重疊的文件 Agent。
- 新增每使用者 Google Calendar／Tasks／Forms：由 `domain_architect` 先定義自有 DTO／Port，`google_identity_workspace_integrator` 實作 BFF + PKCE、加密 token repository 與 SDK Adapter，`cloud_platform` 審查 public/private IAM 邊界，最後由 `go_reviewer` 驗證。
- LINE Account Linking 串接 Google 授權：`line_integrator` 處理 LINE webhook／link token，`google_identity_workspace_integrator` 處理 Google consent／connection；共用 Port 或 DTO 的演進先交 `gopls_go_refactor`，不可平行改同一檔案。
- 準備 Cloud Tasks Worker：`domain_architect` 釐清 Port、`cloud_platform` 審查 IAM 與資料一致性，主 Agent 排程實作。
- 判斷 SDK 升級：`docs_researcher` 核對正式來源，主 Agent 決定升級範圍。
- 將 application port 的 method 改名並更新所有 adapter、fake 與 call site：委派 `gopls_go_refactor`，先取得 references／implementations，再做語意批量修改與 diagnostics 驗證。
- 準備合併／交付：委派 `code_cleanup_release_validation`，先列出保留與候選清理項、檢查引用，再以 `codexcheck`、`make check`、Actions CI 驗證；不得刪除 SDK Reserved Modules／Scaffolds、必要 Regression Tests 或跳過真實 Staging 安全門檻。

## 禁止誤用

- `AGENTS.md` 是每次工作的規範；`.agents/skills/` 是工作流程；`.codex/agents/` 是角色；`.codex/config.toml` 才配置 MCP。這些不是同一種設定。
- 不要把 production 的 LINE 密鑰、Google ADC 或 GitHub PAT 放到 GitHub、TOML、skill 或 Agent instructions。
- MCP 寫入能力受外部帳號權限控制；即使 Agent 文本說「唯讀」，也不是技術隔離。正式環境權限需由 IAM／MCP allowlist 控制。
- Codex CLI／IDE 官方支援專案 Agent 檔；Codex Cloud 的工作階段是否載入該角色及 MCP 仍須在**已發布的雲端環境**實測，不以檔案存在保證。
