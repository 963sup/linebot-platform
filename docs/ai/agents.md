# Codex 專案 Agents 設計

角色註冊以 [config.toml](../../.codex/config.toml) 與 `.codex/agents/*.toml` 為準；目前保留原有 27 個角色，新增 `go_code_refactor`，共 28 個。`codexcheck` 動態核對註冊、檔案、名稱、未知鍵、路徑及受保護依賴，不把角色數量寫死在解析器。

角色指令不是權限隔離。所有角色繼承 Host 的可用工具與安全控制；既有角色的模型設定保留，新重構角色繼承 Host。省略 reasoning effort 不代表新增了名為 `auto` 的設定值，也不能證明某個模型 ID 在使用者帳號可用。模型與角色載入需在目標 Host 驗證。

## 分工與寫入邊界

| 層級 | Agent | 責任與邊界 |
| --- | --- | --- |
| L0 | `delivery_orchestrator` | 唯讀編排複雜工作、驗收條件、依賴與檔案 owner；不取代主 Agent |
| L1 | `docs_researcher` | 通用 SDK／API 證據查證；不改程式或安裝工具 |
| L1 | `openai_docs_researcher` | OpenAI／Codex 官方規格查證，避免通用研究角色重複查同一問題 |
| L1 | `domain_architect` | Bounded Context、不變量、Aggregate、自有 DTO／Port 契約 |
| L2/L3 | `architecture_alignment_engineer` | 以唯一標靶 DESIGN.md 規劃增量遷移，不把抽象目錄樹當現況 |
| L2 | `go_feature_engineer` | 受委派 Go Domain／Application 垂直切片 |
| L2 | `go_code_refactor` | 行為保持的局部簡化、去重、降低無效抽象；不改外部契約 |
| L2 | `line_integrator` | LINE SDK／Webhook／Messaging／LIFF／Account Linking |
| L2 | `google_identity_workspace_integrator` | Google OAuth／使用者 Token 與 Workspace Adapter |
| L2 | `firestore_integrator` | Firestore 存取、Transaction、一致性與 Adapter |
| L2 | `cloud_tasks_integrator` | 非同步執行、重試、冪等與補償 |
| L2 | `data_migration_engineer` | 持久化相容性、資料遷移與回復；不自行操作 production |
| L2 | `observability_engineer` | 去敏 logging、metrics、tracing 實作 |
| L2 | `cloud_platform` | IAM、WIF、Cloud Run 平台／環境；不取代服務實作者或自行部署 |
| L3 | `gopls_go_refactor` | 在既定契約內傳播跨 package rename／interface／DTO／Port 變更 |
| L4 | `go_reviewer` | 獨立審查 Go 正確性、錯誤、取消與併發 |
| L4 | `go_architecture_reviewer` | 獨立審查依賴方向、責任、consumer、無效抽象 |
| L4 | `go_test_engineer` | 行為／Contract／Regression／Architecture／Race 測試；測試檔需明確 owner |
| L4 | `security_reviewer` | 信任邊界、OAuth、密鑰、個資、權限審查 |
| L4 | `performance_reviewer` | 唯讀量測與瓶頸分析，不以結構推論效能 |
| L4 | `api_contract_reviewer` | HTTP／Webhook／外部資料契約與相容性 |
| L4 | `github_actions_reviewer` | Workflow、OIDC、permissions、secrets、cache／artifact 品質閘門 |
| L4 | `incident_responder` | 正式事故唯讀調查與復原建議，不自行改 production |
| L5 | `technical_docs_writer` | 技術文件與操作手冊，區分實作、驗證與部署 |
| L5 | `go_agents_md_author` | 根／巢狀 AGENTS、角色註冊、Codex 規則一致性 |
| L5 | `code_cleanup_release_validation` | 驗收後最小清理、受保護項目及交付驗證 |
| L5 | `git_release` | 明確授權的 commit／push／PR 等 Git 交付與遠端 HEAD 驗證 |
| L5 | `cloud_run_release` | 明確授權環境的發行與發布後證據，不由 CI 推論部署 |

L0–L5 是工作順序，不是權限繼承或每次必跑的流程。小任務直接交適當角色。最多五個子 thread；只有互不依賴的工作平行，同一檔案同時只有一位寫入 owner。跨角色修改共用 DTO／Port 必須串行交接。

## 重構路由

局部控制流程／重複邏輯由 `go_code_refactor` 處理。跨 package 語意變更交 `gopls_go_refactor`；架構決策不明先交 `domain_architect` 或 `architecture_alignment_engineer`。驗收後清理不得反向改寫已確認架構；任何程式清理都使相關舊驗收失效，需重跑測試與審查。

所有角色依根 [AGENTS.md](../../AGENTS.md) 與 [agent-tool-preflight](../../.agents/skills/agent-tool-preflight/SKILL.md) 自主選用 MCP；共用規則不在 28 份角色中複製。Google 文件優先 Developer Knowledge，OpenAI 規格優先 OpenAI Docs，SDK 版本用 Context7，Go 語意用 gopls，GitHub 現況用 GitHub MCP。沒有掛載或認證就明示降級，不假裝使用。

受控安裝由主 Agent 執行核准 setup，唯讀 reviewer／researcher／orchestrator 不因工具缺失變成安裝者。原生 Plugins、skills-only 檔案、MCP 連線是不同狀態。

## 可重複的 Host 驗收案例

| 任務 | 預期路由 | 必須保存的證據 |
| --- | --- | --- |
| 簡化同 package 重複錯誤處理 | go_code_refactor → go_reviewer | 行為基線、最小 diff、目標測試 |
| Rename Application Port method | gopls_go_refactor → go_test_engineer | references／implementations、呼叫點與診斷 |
| Domain 反向依賴 SDK | domain_architect → 實作者 → go_architecture_reviewer | 違反邊界、consumer、小型切片與架構測試 |
| LINE Webhook 事件演進 | line_integrator → 必要契約／安全審查 | SDK 版本、raw body／簽章與重送測試 |
| Google OAuth／Calendar 整合 | Google 整合角色 → 測試／安全審查 | 官方來源、本地 Symbol、每使用者憑證隔離 |

這些是 Host 驗收規格，不是假造的已執行紀錄。`codexcheck` 通過只能證明儲存庫靜態契約，不能證明角色真的被啟動、模型可用或五個 thread 的排程行為。
