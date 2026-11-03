# AGENTS.md — linebot-platform

本檔是 Codex 與其他程式代理人在此儲存庫的**專案規範**。優先遵循使用者當次指示與更細粒度的目錄 `AGENTS.md`；不可把規劃說成已實作。

## 先確認狀態

- 先讀 `README.md`、`docs/architecture/overview.md`、`docs/roadmap.md` 和相關子目錄規範。
- 現階段有開發用同步 LINE 回覆示範，另有明確啟用才使用的 Firestore Inbox/Outbox、Cloud Tasks OIDC Worker／Reconcile 程式碼；**尚無正式部署、IAM/Firestore TTL／Cloud Scheduler 配置與 Staging 驗收**。請依實際實作情況更新文件，不得把預留目錄當成已完成功能。
- 以最小可行、可編譯、可測試的垂直切片實作新功能。**SDK 預留空白骨架、擴充目錄及其 `.gitkeep` 必須保留**，不得因尚未接線而刪除；其他類型的占位內容，僅在確認用途、引用與授權後才可另行清理。
- 新 SDK 或 feature 在加入前確認上游官方文件、API 版本、帳號地區資格與安全修補版本。依賴已在 `go.mod` 預先宣告，但未在 runtime 接線的不算已啟用；更新後需執行 `go mod tidy` 並提交真實 `go.sum`。
- 使用已修補的 Go 1.27.3 工具鏈；CI／Docker 版本須一致，部署前再固定 image digest。

## 依賴方向

- `internal/domain` 不得 import `application`、`adapter`、`infrastructure` 或 LINE／Google／AI SDK。
- `internal/application` 可 import `domain`，但不能依賴 `adapter` 與 `infrastructure`。
- `internal/adapter` 將外部 API／HTTP 資料轉為 application 自有的 DTO、port。
- `internal/infrastructure` 負責 SDK client、認證、transport、timeout、observability。
- `internal/bootstrap` 專責依賴組裝；`cmd/` 只負責啟動程序。
- 不預先為所有構想建立空介面、空 `.go` 檔或用無用途的 abstraction 填滿架構。
- Value Objects 的**目錄**使用 `internal/domain/value-objects/`，其中 Go **package 識別字**仍是 `valueobjects`；`-` 僅存在於 import path，不得寫成 `package value-objects`，也不可將目錄改回 `valueobjects/`。新增模型前先確認其 Bounded Context。

## LINE webhook 與發送

- 必須使用 LINE raw request body 驗證簽章，驗證前不得更改內容。
- 持久模式先以 Firestore Transaction 原子保存 Inbox/Outbox，再安排固定 ID 的 Cloud Task；排程失敗回 503 並由具授權的定期 Reconcile 修復，不得在回覆 200 後靠程序內 goroutine 保證處理。
- 以 channel 與 event ID 去重；Cloud Tasks 工作須能冪等、可重試、可補償。
- Reply token 有時效，不得假設任何長時間背景任務完成後仍可 reply。
- 發送超時可能是「結果未知」；不可在未確認冪等策略時直接重送。
- 支援 unsend／edit 時要處理已刪資料、遲到事件與舊 AI 回覆。
- 各 API 需查詢 channel 權限／地區限制；未授權時明確標示 unavailable。

## 安全與隱私

- 禁止提交真實 `.env`、憑證、LINE Channel Secret、Access Token、AI API Key。
- 禁止把完整 webhook、token、使用者私訊或個資輸出到 log。
- 配置使用 ADC／工作負載身分／Secret Manager；權限最小化。私有 Worker 必須具備 Cloud Run IAM、OIDC Audience 與獲授權 Service Account Email 的雙重驗證；不能信任 `X-CloudTasks-*` Header 代替驗證。
- 外部發送、刪除、付費雲端資源與正式環境部署都須遵循授權；不自行合併 PR 或啟用 production。
- 不將未審核的外部 skills／MCP 當作可信程式自動執行。
- `skills-lock.json` 是從其他儲存庫精選的外部技能來源指紋，不代表已安裝或已驗證，禁止自動執行未審查的腳本。
- `cloudbuild.deploy.yaml` 僅允許經授權的 staging；在完善可靠性、IAM、Secret Manager 與人工核准之前，不得執行 production 部署。

## SDK 自動辨識與開發路由

- 新增或修改任何 `go.mod` 直接依賴 SDK 時，先讀 [SDK Catalog](docs/sdk/catalog.toml) 和 [SDK 整合架構](docs/architecture/sdk-integration.md)，再按需套用 [sdk-integration Skill](.agents/skills/sdk-integration/SKILL.md)。只讀相關模組，勿預載全部 SDK。
- `go.mod` 是版本唯一來源；Catalog 用來定位 Provider／Package／Factory／Adapter 與整合狀態，不能當成完整 API 清單或實際授權證明。
- **保留規則：所有已核准直接 SDK／工具依賴（包含尚未使用的 `garr`）、`go.sum`、`internal/infrastructure/dependencies/modules.go`、SDK Catalog 與預留 SDK 空白骨架不得因未使用而刪除；`cmd/codexcheck` 在 CI 中檢查受保護項目。**
- 優先使用已鎖定模組的 Go 原始碼、`go list`、`go doc` 和 gopls 驗證真實 Symbol，再對照官方文件查核認證、配額、冪等性及限制；不可猜測 API。
- 修改依賴、Factory 或 Adapter 時，連同 Catalog、架構文件、相關能力矩陣和測試一併同步；執行 `go run ./cmd/codexcheck` 與 `make check`。
- Application 與 Domain 使用 Go 標準庫及專案自有 Port，不直接 import 外部 SDK；不建立萬用動態 SDK 呼叫器。已宣告、已整合、已經 staging 驗證是不同狀態。
- SDK Runtime 使用明確的 Factory 與 Bootstrap 所擁有的 HTTP Transport；不要在共享 LINE Client 上直接呼叫會修改 Receiver 的 `WithContext`。錯誤需去敏且保留可供程式辨識的型別，不要預設自動重試。

## Codex Agents 與 MCP 選用（不負責安裝）

- 複雜且互不依賴的任務才委派給 `domain_architect`、`line_integrator`、`cloud_platform`、`go_reviewer` 或 `docs_researcher`；合併前清理與驗證則使用 `code_cleanup_release_validation`，且不得刪 SDK 預留模組／骨架或必要的回歸測試。同一檔案不得由多個 Agent 同時改寫，最後由主 Agent 整合、驗證。
- 修改 Go 程式前，需要追查 definitions、references、型別或診斷時用 **gopls MCP**；若不可用則使用本地 Go 工具。
- OpenAI API、Responses API、OpenAI Go SDK、Codex、Skills、MCP 或 Plugins 的更新與實作優先使用 **openaiDeveloperDocs MCP** 查閱官方文件；不得推測不存在的 API、參數或能力。若 MCP 不可用，改用 OpenAI 官方網站並標示未驗證事項。
- 其他新版本 SDK／API 使用方式、設定或重大變更使用 **Context7 MCP**，再核對相關官方 Go／LINE／Google 文件；無法查證時明確標示。
- PR／Issue／commit／CI 狀態需要外部證據時，優先選擇既有已授權 GitHub 工具，必要時才使用 **GitHub MCP（唯讀）**。
- **Firestore Toolbox** 只在使用者授權的隔離 staging 環境、具最小權限 ADC 時使用，不得存取或修改 production。
- `AGENTS.md` 只負責路由規則；**不可因指示而自行安裝 MCP、產生 token、執行 OAuth 或修改個人 Codex 設定**。MCP 的啟用與安裝由 `.codex/config.toml`、Codex Cloud Install script 與使用者環境設定處理。

請閱讀 [Agents 職責](docs/ai/agents.md)、[MCP 設定與權限](docs/ai/mcp-and-plugins.md)、[Codex Cloud](docs/ai/codex-cloud.md)。

## 外部環境驗收邊界

- 真實 Cloud IAM／OIDC、Cloud Scheduler、Firestore TTL、LINE Channel 及使用者 OAuth 僅能在隔離且經明確授權的環境中驗證，不得以 CI 通過當作已部署或取得權限。
- 不可自行部署付費服務、執行正式 API 發送或合併 PR；正式驗收結果應於 PR 審查紀錄中明確註明。

## Pragmatic DDD + Lightweight Hexagonal + Idiomatic Go

- 優先使用小型、由呼叫端需求決定的 Port，明確 `context.Context` 與錯誤回傳；只有真實業務不變量才建立 Aggregate／Value Object。不要因資料表、LINE API Endpoint、報表而預先增加空抽象。
- 營運 Outbox 狀態不是 Domain Aggregate：只讀查詢放在 Application Outbound Port，由 Firestore Adapter 實作；CLI 只經 Bootstrap 組裝，不得繞過核心安全規範直接發送 LINE。
- `ops events` 僅限經明確 allowlist 的隔離 Staging，區分 `--inspect` 與 `--check`；不得讀取真實用戶訊息、Secret 或 Reply Token。Unknown 不得自動重送，Expired 不得無授權轉成 Push。
- 所有運行中的告警、Job、Scheduler、IAM、TTL 都必須有真實隔離 Staging 驗收證據；文件或 Firestore Emulator 的成功不能取代實際雲端測試。詳見 `docs/operations/outbox-operations.md` 與 `docs/operations/staging-acceptance.md`。

## 統一設定契約（必讀）

- 執行期環境變數的唯一名稱、角色、預設、Secret 分類來源為 `internal/bootstrap/config_contract.go`；Webhook 與 Worker 使用同一條 `loadRuntimeConfig` 入口，角色驗證不可散落在 `cmd/` 或 Adapter。
- 修改任何設定時同時更新 Go contract constants、bootstrap parser/validation、正確角色的測試與 `.env.example`，使用 `go run ./cmd/configcheck --write-doc` 更新 `docs/operations/configuration.md`；`make config-check` 及 CI 必須阻止文件漂移。
- `make dev` 會強制將 LINE Channel Secret / Access Token 設為空字串，避免意外連到真實 LINE；要使用測試 Channel 須在安全 shell 中明確執行 `make run-webhook`，並提供測試 Channel 的兩項憑證。
- `APP_ENV` 僅能是 local、staging、production；Cloud Run 的 `K_SERVICE` 防止降級至 local。`sync` 僅供本機 demo；公有 Webhook durable 不可持有 LINE 發送 Token，Worker 不可持有簽章 Secret。Secret 在結構上與普通 Config 分離，不得印出 config 原始結構或值。
- 程式不自動讀入 YAML／`.env`；本機 `make dev` 不會自動拿到憑證。正式環境透過短期服務身分及 Secret Manager references 注入，避免提交 Secret、URL query、user message 到 log。
- 可觀測性 logger 使用 `LOG_LEVEL` 與 `LOG_FORMAT` 的 allowlist；HTTP 只採固定路由分類、status、duration，不用高基數 user/event ID 作為指標。正式 Staging 必須另外驗證觀測與 IAM，不把 CI 綠燈當部署成功。

## 驗證與交付

- Codex 設定使用 `go run ./cmd/codexcheck` 驗證 TOML、Agents、Cloud Setup Script 與專案不得包含 Python `.py` 檔案；不需要 Python runtime。
- Go 程式存在後，依序檢查格式、`go vet ./...`、`go test ./...`、`go test -race ./...`、`go build ./...`；若環境不支援，明確記錄原因。
- 任何與 SDK 的整合都需 unit／contract tests；實際雲端／LINE E2E 需隔離且明確授權。
- 提交 PR 必須描述變更、實際測試結果、風險、未完成事項；不得聲稱空目錄已完成實作。
- 發布前檢查 `README.md`、`docs/sources.md`、功能矩陣與相容性紀錄。
