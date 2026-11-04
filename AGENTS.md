# AGENTS.md — linebot-platform

本檔是 Codex 與其他程式代理人在此儲存庫的**專案規範**。優先遵循使用者當次指示與更細粒度的目錄 `AGENTS.md`；不可把規劃說成已實作。

## 先確認狀態

- 先讀 `README.md`、`DESIGN.md`、`docs/architecture/overview.md`、`docs/roadmap.md` 和相關子目錄規範。
- 現階段有開發用同步 LINE 回覆示範，另有明確啟用才使用的 Firestore Inbox/Outbox、Cloud Tasks OIDC Worker／Reconcile 程式碼；**尚無正式部署、IAM/Firestore TTL／Cloud Scheduler 配置與 Staging 驗收**。請依實際實作情況更新文件，不得把預留目錄當成已完成功能。
- 以最小可行、可編譯、可測試的垂直切片實作新功能。**SDK 預留空白骨架、擴充目錄及其 `.gitkeep` 必須保留**，不得因尚未接線而刪除；其他類型的占位內容，僅在確認用途、引用與授權後才可另行清理。
- 新 SDK 或 feature 在加入前確認上游官方文件、API 版本、帳號地區資格與安全修補版本。依賴已在 `go.mod` 預先宣告，但未在 runtime 接線的不算已啟用；更新後需執行 `go mod tidy` 並提交真實 `go.sum`，不得藉此刪除下述受保護依賴。
- 使用已修補的 Go 1.27.3 工具鏈；CI／Docker 版本須一致，部署前再固定 image digest。

## 依賴方向

`DESIGN.md` 是唯一架構標靶，含抽象目錄樹；不代表現況已遷移。下述既有目錄名稱以真實 consumer 為準，不因標靶使用 `adapters/` 等名稱就直接搬移。

- `internal/domain` 不得 import `application`、`adapter`、`infrastructure` 或 LINE／Google／AI SDK。
- `internal/application` 可 import `domain`，但不能依賴 `adapter` 與 `infrastructure`。
- `internal/adapter` 將外部 API／HTTP 資料轉為 application 自有的 DTO、port。
- `internal/infrastructure` 負責 SDK client、認證、transport、timeout、observability。
- `internal/bootstrap` 專責依賴組裝；`cmd/` 只負責啟動程序。
- 不預先為所有構想建立空介面、空 `.go` 檔或用無用途的 abstraction 填滿架構。
- 既有 Value Objects 的 `value-objects/` 目錄，其 Go package 識別字仍是 `valueobjects`；不得寫成 `package value-objects`。本工具治理工作不變更既有 import path；新增模型先確認 Bounded Context，目標佈局遵守 `DESIGN.md`。

## DDD／六邊形架構重構前的必行規劃

- **先規劃再修改**：跨 Domain、Application、Adapter、Infrastructure 的改造，必須先列出目標業務不變量、擬新增的具體檔案、刪除／合併的舊檔案、呼叫鏈、相容性界線與可驗證的完成條件。
- **以目標模型主導收斂**：Domain 是業務狀態與決策的單一來源；不能只新增 Entity／Aggregate／Value Object，卻保留 Application／Adapter 的平行規則。檢查並移除舊 Policy／Specification／Factory、無引用的接口、重複驗證。
- **命名與實作不可分離**：既有命名遷移須依 `DESIGN.md`、consumer 與相容性測試分切片處理；Entity、Aggregate、Policy、Specification、Factory、Repository、Events 僅在具體用途及測試存在時才實作，禁止空殼填滿樹。
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
- `.codex/plugins.lock.json` 是受控安裝器的來源鎖定，不取代根 skills-lock，也不授予執行下載內容的權限。第三方 Skill 的 credential fallback 不得自動選用其他帳號、執行 OAuth 或把 key 放進 URL／日誌。
- `cloudbuild.deploy.yaml` 僅允許經授權的 staging；在完善可靠性、IAM、Secret Manager 與人工核准之前，不得執行 production 部署。

## SDK 自動辨識與開發路由

- 新增或修改任何 `go.mod` 直接依賴 SDK 時，先讀 [SDK Catalog](docs/sdk/catalog.toml) 和 [SDK 整合架構](docs/architecture/sdk-integration.md)，再按需套用 [sdk-integration Skill](.agents/skills/sdk-integration/SKILL.md)。只讀相關模組，勿預載全部 SDK。
- `go.mod` 是版本唯一來源；Catalog 用來定位 Provider／Package／Factory／Adapter 與整合狀態，不能當成完整 API 清單或實際授權證明。
- **保留規則：所有已核准直接 SDK／工具依賴（包含尚未使用的 `garr`）、`go.sum`、`internal/infrastructure/dependencies/modules.go`、SDK Catalog 與預留 SDK 空白骨架不得因未使用而刪除；`cmd/codexcheck` 在 CI 中檢查受保護項目。**
- 優先使用已鎖定模組的 Go 原始碼、`go list`、`go doc` 和 gopls 驗證真實 Symbol，再對照官方文件查核認證、配額、冪等性及限制；不可猜測 API。
- 修改依賴、Factory 或 Adapter 時，連同 Catalog、架構文件、相關能力矩陣和測試一併同步；執行 `go run ./cmd/codexcheck` 與 `make check`。
- Application 與 Domain 使用 Go 標準庫及專案自有 Port，不直接 import 外部 SDK；不建立萬用動態 SDK 呼叫器。已宣告、已整合、已經 staging 驗證是不同狀態。
- SDK Runtime 使用明確的 Factory 與 Bootstrap 所擁有的 HTTP Transport；不要在共享 LINE Client 上直接呼叫會修改 Receiver 的 `WithContext`。錯誤需去敏且保留可供程式辨識的型別，不要預設自動重試。

## Codex Agents、MCP 與受控安裝

### 工作契約與角色路由

- `.codex/config.toml` 管連線、角色註冊、啟用與 timeout；角色 TOML 管職責、輸入、可改範圍與交付；本檔管共用路由、架構不變量、授權與 fallback。角色文字不是沙盒或權限升級。
- 複雜跨領域工作由唯讀 `delivery_orchestrator` 定義有序工作契約；小型單一邊界直接交實作者。主 Agent 負責整合與最後驗證；最多五個子 thread，不為湊數啟動。
- L1 研究與 Domain 契約，L2 Go／服務整合／局部重構，必要時 L3 跨 package 語意傳播，L4 獨立審查，L5 文件與交付閘門。不是每次工作都要走完整角色鏈。
- 需求與驗收條件由主 Agent／編排者確認；Domain 不變量交 `domain_architect`。架構對齊交 `architecture_alignment_engineer`，先讀最新版 `DESIGN.md`，建立目標／現況／consumer／遷移矩陣；不重建已刪除的第二份架構文件。
- 函式／package 內行為保持簡化交 `go_code_refactor`；跨 package rename、DTO／Port／interface 演進、搬遷及批量 references 交 `gopls_go_refactor`。後者只傳播已確認契約，不重新決定架構。兩者交接時先停止重疊寫入。
- Firestore、Cloud Tasks、資料遷移與遙測分別交 `firestore_integrator`、`cloud_tasks_integrator`、`data_migration_engineer`、`observability_engineer`；`cloud_platform` 保留 IAM／平台責任。
- `line_integrator` 負責 LINE v8 的 linebot、webhook、messaging_api、liff、channel_access_token、insight、manage_audience、module、module_attach、shop、httphandler。`google_identity_workspace_integrator` 負責 oauth2、oauth2/google、idtoken、oauth2/v2、option、calendar/v3、tasks/v1、forms/v1 及相關 Workspace Adapter。
- 跨 LINE／Google 帳號連結先確定 Domain 自有 DTO／Port，再分派整合角色。查詢 Port 按能力用 Reader／Finder／Inspector，改變外部狀態用 Manager／Command；不新增 Provider、萬用 Client／Call 或動態路由層。
- 正確性、架構、安全、測試、效能、API 契約、CI 分別交對應 reviewer／test engineer；`incident_responder` 預設正式事故唯讀。實作者不可自我批准；同一檔案同時只能有一個寫入 owner。
- `code_cleanup_release_validation` 僅做驗收後必要清理與交付檢查，不反向改架構。清理若改動程式，相關舊驗收即失效，必須重跑測試／審查。

### 自主選用 MCP

所有新舊角色使用 [agent-tool-preflight](.agents/skills/agent-tool-preflight/SKILL.md)，只查本次任務需要的工具，不把工具健康掃描當成所有工作的前置成本。

| 任務 | 優先 MCP | 不可用時 |
| --- | --- | --- |
| Go symbols、型別、references、diagnostics、重構 | `gopls` | 鎖定 module 原始碼、go list／go doc、編譯器與測試，明示降級 |
| 函式庫、SDK、CLI 版本與用法 | `context7` | 對應版本的第一方文件／原始碼 |
| GitHub repo、branch、PR、Issue、workflow／CI | `github` | 已授權 Git CLI；無遠端證據不得推論狀態 |
| OpenAI、Codex、Agents、Skills、MCP、Plugins | `openaiDeveloperDocs` | OpenAI 官方網站／原始碼，Context7 補充 |
| Google Cloud／Workspace API 與認證規格 | `google_developer_knowledge` | Google 官方文件，Context7 補充，再以本地 Go Symbol 核對 |

- 已有同版本且足夠的證據可重用；重要不確定事項或版本差異先查證，不使用「99.99% 把握」等不可驗收門檻。純文件排版不強制 gopls；工具可用時不得以記憶代替必要語意或契約查證。
- `.codex/config.toml` 的 declared／enabled 不代表 Host 已載入、認證或 verified。實際工具清單與最小唯讀呼叫才是可用性證據；錯誤回應、空結果、缺憑證需明示，不無限重試。
- Google Developer Knowledge 使用 Host 安全提供的 `DEVELOPERKNOWLEDGE_API_KEY` 環境映射，不寫入 repo。此文件 MCP 不代替 Calendar／Firestore／Cloud Run 服務的實際授權。
- 停用的服務 MCP 不因文件查詢而啟用。Firestore Toolbox 僅在明確授權的隔離 staging 與最小 ADC 使用，不接觸 production。

### 缺少技能或工具時

- 主 Agent 可先執行 `bash scripts/check-codex-plugins.sh`。預設不安裝、不認證；缺少工具只回報狀態，不捏造成功。
- 在當次工作已授權準備開發工具、來源符合 `.codex/plugins.lock.json`、Host／Sandbox 允許時，主 Agent 可自行執行 `bash scripts/setup-codex-plugins.sh`，不需要使用者重述安裝命令。唯讀角色只交接缺失，不直接安裝。
- 安裝器只取得固定 commit 的技能檔案，不執行上游腳本、不覆寫既有目錄。原生 Plugin 語法／固定版本支援不明或重複 MCP 時，使用有明確標記的 skills-only fallback，不稱為原生 Plugin 已安裝。
- Google marketplace 可用 setup 的 `--register-google-marketplace` 選項註冊；已加入 marketplace 不等於已安裝 Plugin。原生 Google Plugin 的 MCP 重複與安裝認證須另行處理，禁止暗中開啟。
- 安裝不含建立 token、OAuth 登入、啟用付費 API、改 IAM、放寬 sandbox 或改個人模型／approval 控制。既有不同版本、修改過的內容、認證或網路錯誤必須停止該項安裝並回報，不自行覆寫／追蹤浮動分支更新。
- gopls 安裝由已核准的 `scripts/setup-codex-cloud.sh` 處理；使用與 config 一致的 `$HOME/.local/bin/gopls`。MCP 連線另於新工作階段驗證。

### Git 交付閘門

`git_release` 僅於使用者明確授權 commit／push／PR／merge 時操作；先驗證工作區、範圍、品質閘門與 secrets，再使用 fast-forward 或 expected-head 保護更新指定分支。可寫 MCP 不等於操作授權；不自行 force push、merge 或 deploy。推送後驗證遠端 SHA、branch ref 與實際 CI；push／PR／CI 成功不是 Cloud Run／staging／production 部署證明。

參考：[角色矩陣](docs/ai/agents.md)、[MCP／Plugins](docs/ai/mcp-and-plugins.md)、[Codex Cloud](docs/ai/codex-cloud.md)。

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
