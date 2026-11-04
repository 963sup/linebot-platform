# Roadmap

## P0 — Repository foundation（PR #8 已合併）

- [x] 建立初始目錄與治理文件；後續以真實 capability 按需建立 package，不保護空 scaffold。
- [x] 根 README、AGENTS、分層規範、技術來源與 Codex skills。
- [x] PR #8 已於 2026-11-03 合併至 `main`（Commit `901096a7`）。

## P1 — 可運行 Go 最小切片（PR #8 已合併）

- [x] 根據上游 SDK 的最低需求選擇 Go 1.27.3，建立 `go.mod`，宣告 SDK 版本。
- [x] `cmd/webhook` 提供 `GET /healthz`、PORT、graceful shutdown。
- [x] 建立 unit/contract tests、Makefile、Dockerfile、GitHub Actions CI 的實作草案。
- [x] Go AST 架構邊界檢查。
- [x] GitHub runner 取得真實的 `go.sum` 並提交。
- [x] 在 GitHub Actions 已成功執行 gofmt、go vet、race tests 與 Go build；後續 PR 仍須檢查自身 CI。
- [x] Go 最小切片隨 PR #8 併入 `main`，合併後 CI 已通過。

## SDK-Aware Codex Foundation — 開發知識與驗證

- [x] 建立直接依賴 Catalog，對齊本階段全部 12 個 `go.mod` 直接依賴（新增的直接分類為既有 gRPC 模組）。
- [x] 建立通用 `sdk-integration` Skill 與查證參考，保留原五個開發 Agent，另新增 Code Cleanup & Release Validation（共六個）。
- [x] 擴充 `cmd/codexcheck` 驗證 Catalog、架構邊界及測試，並由既有 CI 執行。
- [x] 同步 README、Skills、架構、版本與開發文件。
- [x] PR #8 已通過 CI 並合併至 `main`。
- [ ] 在新的 Codex Cloud 工作階段實測 SDK 查證流程與 MCP 可用性。

## SDK Runtime — 分階段實施

- [x] LINE Typed Client Factory 接受受管理 HTTP Client，加入 Timeout、獨立 Transport、生命週期關閉、出站安全 Telemetry、錯誤分類、Context 隔離及離線契約測試。
- [x] 依實際 SDK 特性建立原生 Typed Client Factory，區分 HTTP／gRPC、API Key／Vertex ADC／Workspace 使用者 OAuth。
- [x] Firestore、Cloud Tasks（背景工作）、Secret Manager、Storage 的 Typed Client Factory 與配置驗證。
- [x] Firestore Inbox/Outbox 交易 Adapter、Cloud Tasks OIDC 派送 Adapter、可選 Durable Webhook、私有 Worker／定期 Reconcile 程式碼。
- [ ] Google Cloud Staging IAM／Firestore TTL／Cloud Scheduler／高吞吐與實際 LINE 回覆驗證，Storage 業務 Adapter 另行實作。
- [x] OpenAI、Gemini／Vertex、Drive、Sheets、Calendar、Google Tasks（待辦）的原生 Client Factory 與離線契約測試。
- [ ] 其他 Provider（AI／Workspace／Storage／Secret Manager）業務 Adapter、使用者 OAuth 同意與 Token 隔離；**LINE 已接入預設 Webhook／CLI，Firestore 與 Cloud Tasks 已接入可選 Durable Workflow**。
- [x] 原子 Inbox/Outbox、事件 Key 去重、Worker 單次 Claim、unknown/expired 終態和 Reconcile 程式碼。
- [ ] 實際部署、定期 Cloud Scheduler 與獨立 Staging 的故障恢復驗收。
- [ ] 經授權的真實 Staging SDK／IAM／OIDC／OAuth 測試與六個全新 Codex 工作階段零先驗驗收（CI 與已清理的一次性 Emulator 測試不能取代）。

## P2 — LINE webhook 與持久工作

- [x] LINE Channel Secret 配置／使用官方 SDK 驗章（同步 MVP）。
- [x] 文字事件轉換、跨執行個體去重、Inbox/Outbox、Cloud Tasks／Worker 可選程式碼。
- [ ] Unsend、Edit、其他事件類型與高吞吐回復策略。
- [x] 基礎 Reply／Push／Flex Message adapter 及可測試 usecase。
- [x] 發送前持久化單次 Claim，結果未知標示 unknown，不進行盲目重試；保留跨執行個體事件 Fingerprint 去重。
- [ ] GCP Staging／Cloud Tasks OIDC／LINE Sandbox 的真實跨服務驗證。

## 近期實作順序與四項不可退化條件（Identity Domain 遷移進行中）

1. **命名與架構 Gate（PR #28 已合併）**：所有 Bounded Context 的 Value Objects 目錄使用 `value-objects/`，Go package 為 `valueobjects`；架構測試禁止同義異名目錄。Identity 已建立八類具體元件並逐步接入 OAuth／Firestore；後續變更仍需按最新 commit 驗證。
2. **DDD 不退化（Identity 本批完成，其他 Context 後續）**：Identity 的 Aggregate、Entity、VO、Policy、Specification、Factory、Repository Reader、Domain Events 已實作與測試；仍待移除 OAuth 技術 DTO 的重疊欄位並設計持久事件 Outbox。Conversation/Message 在確定真實不變量後才建立 Domain Model，不把 LINE SDK DTO 當成領域模型。
3. **Hexagonal Architecture 不退化（同一功能 PR 的 Gate）**：擴充 AST import boundary 檢查，驗證 Domain 不依賴外層、Application 不 import Adapter/Infrastructure/SDK；新增 usecase 只能透過 Port 呼叫外部效果，Bootstrap 負責 DI。
4. **GPT／Codex 零先驗不退化（獨立可重現驗收）**：全新、無過往聊天上下文的 Codex session 從 `AGENTS.md` → `README.md` → `docs/architecture/overview.md` → `docs/sdk/catalog.toml`，只讀必要 skill 與 SDK symbol，完成 sandbox-only 微小 Domain 修改；能定位正確的 `value-objects/`，並以 `go.mod`、Catalog、build-tag manifest 保留核准依賴而不建立空 scaffold。將指令、測試結果、失敗與修正紀錄於 PR。
5. **Production Safety 不退化（Staging 前 Gate）**：以模擬事件驗證簽章、Inbox/Outbox 原子性、重送／亂序、unsend/edit tombstone、Reply token 過期、unknown 發送與 Cloud Tasks OIDC 拒絕未授權請求；建立受控 Staging IAM、Firestore TTL、Scheduler 與 rollback 方案，部署必須經明確批准。不可用 CI 綠燈代表真正 Cloud Run／LINE Production 安全。

**完成證據**：每個 PR 列出不變量、涉及 Port/Adapter、可重現測試命令、CI URL、未驗證的外部資源／身分與回滾步驟。不得單憑目錄存在或文件宣稱已完成功能。

## Outbox 可靠性與營運驗收（堆疊 Draft PR，尚未合併）

- [ ] [PR #11](https://github.com/963sup/linebot-platform/pull/11)：經簽章 LINE redelivery、原始 Event Timestamp + 20 分鐘上限、只恢復從未發送的 Pending/Expired；公開 Webhook 憑證隔離。Go CI 已通過，但真實 LINE 驗收未完成。
- [ ] [PR #12](https://github.com/963sup/linebot-platform/pull/12)：有序且公平的 bounded Reconcile、`next_dispatch_at` 與必要 Index、常設 Firestore Emulator Transaction／Race Tests。Go/Emulator CI 已通過；線上索引和 Backfill 未完成。
- [ ] 只讀 Outbox 運維 CLI、Staging 驗收契約及告警策略：由下一個單獨 Draft PR 規劃；任何 Cloud Run Job、Monitoring、Scheduler 與 LINE 真實帳號驗收仍待明確核准。
- [ ] Production readiness：完成 `docs/operations/staging-acceptance.md` 每一項 Gate 的真實證據，並取得 deployment/rollout 授權。

## 設定契約整合（本次獨立 Draft PR，尚未合併）

- [ ] Webhook / Worker 共用 `internal/bootstrap/loadRuntimeConfig`，以 Role 與 APP_ENV 決定合法設定；local 開發預設不需憑證，staging/production 必須 durable，Cloud Run `K_SERVICE` 拒絕降級 local。
- [ ] 以獨立 `SecretConfig` 持有 LINE 簽章 Secret 與發送 Token；公開 Webhook 不持有發送 Token、私有 Worker 不持有簽章 Secret；普通設定不得包含 Token 值。
- [ ] `config_contract.go` 是設定鍵名稱、角色、Secret 分類與預設值的唯一來源；`cmd/configcheck --check` 驗證 Go 產生文件與 `.env.example` 一致，`make check` 與 CI 強制執行。
- [ ] `make dev` 提供不載入 `.env` 的安全本機啟動；`--validate` 可離線進行啟動前檢查。
- [ ] Logger LOG_LEVEL/LOG_FORMAT 安全設計；固定路由的 HTTP status/duration，不輸出敏感資料。
- [ ] Production Gate 仍未完成：Secret Manager references、IAM、OIDC、Firestore TTL、Scheduler、LINE redelivery、正式 logs/alerts 需要經授權 Staging 測試，不能因 CI 通過宣稱 Production-ready。

## P3 — 全公開 API、AI 與雲端

- [x] 依固定官方 OpenAPI revision 建立 102 項 LINE endpoint inventory（Messaging、Insights、Audience、Channel Access Token）；main 原先已有 69 項離線接線。\n- [ ] [Draft PR #24](https://github.com/963sup/linebot-platform/pull/24)：補齊 33 項 Operator-only SDK Port → Use Case → Adapter → CLI 與 offline contract tests；在 CI 通過與合併前，不視為正式完成。
- [ ] 全訊息型別、全 Webhook 業務事件等後續能力目前暫緩；本輪僅以固定 102 項 API 為目標。
- [ ] OpenAI / Gemini 可替換 adapter；Google Cloud／Workspace。
- [ ] 批准後部署 staging，驗證後再考慮 production。
