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

## DDD／六邊形架構重構前的必行規劃

- **先規劃再修改**：跨 Domain、Application、Adapter、Infrastructure 的改造，必須先列出目標業務不變量、擬新增的具體檔案、刪除／合併的舊檔案、呼叫鏈、相容性界線與可驗證的完成條件。
- **以目標模型主導收斂**：Domain 是業務狀態與決策的單一來源；不能只新增 Entity／Aggregate／Value Object，卻保留 Application／Adapter 的平行規則。檢查並移除舊 Policy／Specification／Factory、無引用的接口、重複驗證。
- **命名與實作不可分離**：各 Bounded Context 的 Value Objects 目錄固定為 `value-objects/`（Go package `valueobjects`）；Entity、Aggregate、Policy、Specification、Factory、Repository、Events 僅在具體用途及測試存在時才實作，禁止空殼填滿樹。
- **遷移不可犧牲資料契約**：舊 OAuth／Firestore 技術 DTO 可在過渡期間保留，但必須明確註明為相容性資料載體、追蹤剩餘引用與退役條件，不能永久成為第二套業務模型。必須保護 OAuth State／PKCE／Nonce、Token 加密、CAS、Webhook／Cloud Tasks 冪等性。
- **驗證後再交付**：Go 格式與靜態檢查、Domain／Architecture／Contract／Race Tests、CI 必須通過；新舊持久化格式混合存活未驗證時不得合併／發布。不以文件、空目錄或「CI 綠燈」單獨宣稱 DDD 遷移完成。

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

- **三層契約：**`.codex/config.toml` 只宣告 MCP／協作功能的連線、啟用狀態與 timeout，不能放角色工作流程或憑證；`.codex/agents/*.toml` 只定義可委派角色的責任、輸入、交付與安全邊界，不能自行啟用工具或擴權；本檔負責跨角色路由、架構不變量、驗證與 fallback。三者的相同工具名稱必須一致，設定啟用不代表此工作階段已連線成功。
- 跨領域、跨 SDK、跨 package 或跨環境的複雜任務先由第 0 層 `delivery_orchestrator` 產出工作契約；小型單一邊界任務直接交對應角色，避免徒增 Agent 鏈。它不取代主 Agent 的整合與最終驗證責任。
- **層級與順序：**第 1 層 `docs_researcher`、`domain_architect` 產出外部證據與領域契約；第 2 層 `go_feature_engineer`、`line_integrator`、`google_identity_workspace_integrator`、`cloud_platform` 在契約內實作或審查能力邊界；第 3 層 `gopls_go_refactor` 只傳播已確認的跨 package 語意變更；第 4 層 `go_reviewer`、`go_architecture_reviewer`、`github_actions_reviewer` 獨立檢查；第 5 層 `technical_docs_writer`、`go_agents_md_author`、`code_cleanup_release_validation`、`cloud_run_release` 產出文件、治理、交付驗證與經明確授權的發布。同一檔案不得由多個實作角色同時改寫，最後由主 Agent 整合、驗證。
- **Git 交付閘門：**`git_release` 只在使用者明確要求 commit／push／PR／merge 時啟用；先驗證工作區、變更範圍、Go 品質閘門與 secrets，再以 fast-forward 或 GitHub MCP 的 expected head 保護更新指定分支。GitHub MCP 可寫不代表自動授權：所有寫入、PR merge、force push 與 deployment 仍需使用者明確要求；推送後必須用 GitHub MCP 驗證遠端 SHA、branch ref 與 CI 狀態，不得把 push、PR merge 或 CI 通過宣稱為 Cloud Run／staging／production 部署成功。
- **SDK 路由標準：**所有 `github.com/line/line-bot-sdk-go/v8/linebot`、`webhook`、`messaging_api`、`liff`、`channel_access_token`、`insight`、`manage_audience`、`module`、`module_attach`、`shop`、`httphandler` 變更由 `line_integrator` 負責；所有 `golang.org/x/oauth2`、`oauth2/google`、`google.golang.org/api/idtoken`、`oauth2/v2`、`option`、`calendar/v3`、`tasks/v1`、`forms/v1` 變更由 `google_identity_workspace_integrator` 負責。它們都必須先套用 `sdk-integration` Skill、查 Catalog、以 gopls 驗證符號，並維持 Domain/Application 無 SDK import、Adapter/Infrastructure 持 SDK、Bootstrap 組裝的邊界。
- **命名與交接標準：**由具體能力命名，查詢 Port 使用 `Reader`、`Finder` 或 `Inspector`，改變外部狀態使用 `Manager` 或 `Command`；禁止建立 `Provider`、萬用 `Client`、`Call` 或動態路由層。跨 LINE 與 Google 的帳號連結，先由 `domain_architect` 定義自有 DTO／Port，再分別交兩個整合 Agent；跨 package rename、DTO／Port 演進則交 `gopls_go_refactor`，最後由 `go_reviewer` 檢查邊界、錯誤、取消、效能與測試。
- 跨 package rename、interface／DTO 簽名變更、Port／Adapter 搬遷、package 拆分或需批量更新 Go references 時，優先委派 **`gopls_go_refactor`**。它必須先以 gopls MCP 查 definition、references、implementations 與 diagnostics，修改後執行 gofmt、目標測試並重新檢查 diagnostics；若 gopls MCP 不可用，才以本地 Go 工具降級並回報。
- 修改其他 Go 程式前，需要追查 definitions、references、型別或診斷時用 **gopls MCP**；若不可用則使用本地 Go 工具。
- OpenAI API、Responses API、OpenAI Go SDK、Codex、Skills、MCP 或 Plugins 的更新與實作優先使用 **openaiDeveloperDocs MCP** 查閱官方文件；不得推測不存在的 API、參數或能力。若 MCP 不可用，改用 OpenAI 官方網站並標示未驗證事項。
- **主動使用 Context7：**凡涉及任何函式庫、框架、SDK、API、CLI 工具、MCP 或版本／設定用法的問題，Codex 必須先使用 **Context7 MCP** 查詢最新文件與範例，再進行分析、修改或回答；不得只依賴記憶。若 Context7 不可用，必須明確說明並改查官方文件。
- **MCP 路由固定規則：**所有 Go 語言、Go workspace、Go symbol、Go diagnostics、重構與 Go tooling 工作必須使用 `[mcp_servers.gopls]`；所有 GitHub repository、branch、commit、PR、Issue、workflow 與 CI 操作必須使用 `[mcp_servers.github]`；所有 Codex、Codex 設定、Agents、Skills、MCP 與 Plugins 的設定或文件問題必須使用 `[mcp_servers.openaiDeveloperDocs]`。不得以其他 MCP 取代上述指定路由；指定 MCP 不可用時，必須明確回報並說明 fallback。
- 其他新版本 SDK／API 使用方式、設定或重大變更使用 **Context7 MCP**，再核對相關官方 Go／LINE／Google 文件；無法查證時明確標示。
- PR／Issue／commit／CI 狀態需要外部證據或經授權的 GitHub 寫入時，使用已啟用的 **GitHub MCP**；寫入前確認使用者授權、目標 repository、branch head 與 expected SHA。
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
- 修改任何設定時同時更新 Go contract constants、bootstrap parser/validation、正確角色的測試與 `.env.example`，使用 `go run ./cmd/configcheck --write-doc` 更新 `docs/operations/configuration.md`；`make config-check` 必須以相同 Webhook Validator 驗證 `.env.example` **實際值**，不能只核對鍵名／Secret 是否為空。Durable Queue 格式必須在 Bootstrap preflight 與 Adapter 共用相同的純函式驗證，避免前置檢查成功、SDK 初始化後才失敗。CI 必須阻止文件及可啟動設定漂移。
- `make dev` 會強制將 LINE Channel Secret / Access Token 設為空字串，避免意外連到真實 LINE；要使用測試 Channel 須在安全 shell 中明確執行 `make run-webhook`，並提供測試 Channel 的兩項憑證。
- `APP_ENV` 僅能是 local、staging、production；Cloud Run 的 `K_SERVICE` 防止降級至 local。`sync` 僅供本機 demo；公有 Webhook durable 不可持有 LINE 發送 Token，Worker 不可持有簽章 Secret。Secret 在結構上與普通 Config 分離，不得印出 config 原始結構或值。
- 程式不自動讀入 YAML／`.env`；本機 `make dev` 不會自動拿到憑證。正式環境透過短期服務身分及 Secret Manager references 注入，避免提交 Secret、URL query、user message 到 log。
- Worker 永遠使用 Durable 事件處理，若部署環境明確設定 `LINE_EVENT_MODE=sync` 或未知值必須拒絕啟動，不可默默忽略。可觀測性 logger 使用 `LOG_LEVEL` 與 `LOG_FORMAT` 的 allowlist；HTTP 只採固定路由與方法分類、status、duration，不用高基數 user/event ID 或任意 client-controlled 方法字串作為指標。正式 Staging 必須另外驗證觀測與 IAM，不把 CI 綠燈當部署成功。

## 驗證與交付

- Codex 設定使用 `go run ./cmd/codexcheck` 驗證 TOML、Agents、Cloud Setup Script 與專案不得包含 Python `.py` 檔案；不需要 Python runtime。
- Go 程式存在後，依序檢查格式、`go vet ./...`、`go test ./...`、`go test -race ./...`、`go build ./...`；若環境不支援，明確記錄原因。
- 任何與 SDK 的整合都需 unit／contract tests；實際雲端／LINE E2E 需隔離且明確授權。
- 提交 PR 必須描述變更、實際測試結果、風險、未完成事項；不得聲稱空目錄已完成實作。
- 發布前檢查 `README.md`、`docs/sources.md`、功能矩陣與相容性紀錄。
