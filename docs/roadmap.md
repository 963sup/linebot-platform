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

- [x] 建立直接依賴 Catalog，並由 `codexcheck` 持續對齊 `go.mod`；不在自由格式文件硬編碼模組總數。
- [x] 建立通用 `sdk-integration` Skill 與查證參考；PR #8 當時的六角色集合屬歷史基線，現已由 `.codex/config.toml` 註冊的八個 depth-one specialist roles 取代。
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

## 近期實作順序與不可退化條件

1. **命名與架構 Gate（PR #28 已合併）**：所有 Bounded Context 的 Value Objects 目錄使用 `value-objects/`，Go package 為 `valueobjects`；架構測試禁止同義異名目錄。Identity 已建立八類具體元件並逐步接入 OAuth／Firestore；後續變更仍需按最新 commit 驗證。
2. **DDD 不退化（Identity 本批完成，其他 Context 後續）**：Identity 的 Aggregate、Entity、VO、Policy、Specification、Factory、Repository Reader、Domain Events 已實作與測試；仍待移除 OAuth 技術 DTO 的重疊欄位並設計持久事件 Outbox。Conversation/Message 在確定真實不變量後才建立 Domain Model，不把 LINE SDK DTO 當成領域模型。
3. **Hexagonal Architecture 不退化（同一功能 PR 的 Gate）**：擴充 AST import boundary 檢查，驗證 Domain 不依賴外層、Application 不 import Adapter/Infrastructure/SDK；新增 usecase 只能透過 Port 呼叫外部效果，Bootstrap 負責 DI。
4. **GPT／Codex 零先驗不退化（獨立可重現驗收）**：全新、無過往聊天上下文的 Codex session 從 `AGENTS.md` → `README.md` → `docs/architecture/overview.md` → `docs/sdk/catalog.toml`，只讀必要 skill 與 SDK symbol，完成 sandbox-only 微小 Domain 修改；能定位正確的 `value-objects/`，並以 `go.mod`、Catalog、build-tag manifest 保留核准依賴而不建立空 scaffold。將指令、測試結果、失敗與修正紀錄於 PR。
5. **Production Safety 不退化（Staging 前 Gate）**：以模擬事件驗證簽章、Inbox/Outbox 原子性、重送／亂序、unsend/edit tombstone、Reply token 過期、unknown 發送與 Cloud Tasks OIDC 拒絕未授權請求；建立受控 Staging IAM、Firestore TTL、Scheduler 與 rollback 方案，部署必須經明確批准。不可用 CI 綠燈代表真正 Cloud Run／LINE Production 安全。

**完成證據**：每個 PR 列出不變量、涉及 Port/Adapter、可重現測試命令、CI URL、未驗證的外部資源／身分與回滾步驟。不得單憑目錄存在或文件宣稱已完成功能。

### Tactical model baseline

- [x] `implemented_offline`：Audience、Experience Presentation、Promotion、Workspace Automation 與 Messaging Delivery 具備 provider-independent Aggregate、Value Object、Domain Event、Repository contract 與行為測試。
- [x] `implemented_offline`：Membership、Contact Directory、Insights Reporting 與 Content Intelligence 明確建模為 Application-owned read/process model，未虛構 Aggregate。
- [x] `implemented_offline`：架構測試禁止 Domain Context 互相匯入，Application 禁止 provider SDK type，並要求已建立的 Domain Context 同時存在行為與測試。
- [ ] `business_validated`：上述 Context（Identity 除外）仍是 `candidate/hypothesis`；需經業務訪談確認語言、owner、邊界與 Core／Supporting 分類後才能提升狀態。
- [ ] `production_integrated`：候選 Context 的 provider adapter、persistence adapter、composition wiring 與資料遷移尚未全部完成，不以 tactical code 宣稱可部署。

## Outbox 可靠性與營運驗收

- [x] `implemented_offline`：經簽章 LINE redelivery、原始 Event Timestamp、只恢復從未發送的工作、bounded Reconcile 與必要離線契約測試已存在於 `main`。
- [ ] `staging_verified`：線上索引、TTL、Backfill、Cloud Tasks OIDC、Scheduler 與故障恢復仍需經授權環境驗證。
- [ ] 只讀 Outbox 運維 CLI、Staging 驗收契約及告警策略：由下一個單獨 Draft PR 規劃；任何 Cloud Run Job、Monitoring、Scheduler 與 LINE 真實帳號驗收仍待明確核准。
- [ ] Production readiness：完成 `docs/operations/staging-acceptance.md` 每一項 Gate 的真實證據，並取得 deployment/rollout 授權。

## 設定契約整合

- [x] `implemented_offline`：Webhook／Worker 共用 `loadRuntimeConfig`，以 Role、APP_ENV 與 `K_SERVICE` 驗證合法 profile。
- [x] `implemented_offline`：獨立 `SecretConfig` 分離簽章 Secret 與發送 Token，普通設定不持有 Token 值。
- [x] `implemented_offline`：`config_contract.go` 是設定鍵唯一來源，`configcheck`、`.env.example` 與生成文件由 CI 驗證。
- [x] `implemented_offline`：`make dev` 不載入 `.env`，`--validate` 可無副作用執行啟動前檢查。
- [x] `implemented_offline`：LOG_LEVEL／LOG_FORMAT、固定路由 telemetry 與敏感資料排除已有離線測試。
- [ ] Production Gate 仍未完成：Secret Manager references、IAM、OIDC、Firestore TTL、Scheduler、LINE redelivery、正式 logs/alerts 需要經授權 Staging 測試，不能因 CI 通過宣稱 Production-ready。

## P3 — 全公開 API、AI 與雲端

- [x] `governed_offline`：依固定官方 OpenAPI revision 建立 102 項 LINE endpoint inventory；每一項都有唯一 capability/accounting 記錄，治理器驗證數量、唯一性、狀態與 ownership 目錄。
- [ ] `operation_verified`：目前 Gate 不證明每個 operation 都有專屬 symbol 與 contract test；逐項接線證據需另以可解析 symbol/test manifest 補齊，不能由 YAML 自我宣告推論。
- [ ] `staging_verified`：逐項核對真實 Channel 權限、地區資格、速率限制與帳號能力；離線 accounting 不代表外部授權成功。
- [ ] 全訊息型別、全 Webhook 業務事件等後續能力目前暫緩；本輪僅以固定 102 項 API 為目標。
- [ ] OpenAI／Gemini 業務能力與 Workspace automation 的 Domain 語言；既有 client factory 或 provider operation 不等同業務 Context。
- [ ] 批准後部署 staging，驗證後再考慮 production。
