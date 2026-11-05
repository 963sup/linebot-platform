# linebot-platform

使用 Go、DDD（Domain-Driven Design）與 Hexagonal Architecture 的 LINE Messaging API 平台；規劃部署在 Google Cloud Run，並保留 Google Cloud、Workspace、OpenAI 與 Google Gen AI 整合介面。

> **狀態：SDK 與事件處理基礎設施已合併至 `main`（PR #8），目前仍是未正式部署的開發用 MVP。** 預設為同步 LINE Webhook／Reply／Push；另有明確啟用才使用的 Firestore Inbox/Outbox、Cloud Tasks OIDC Worker 與 Reconcile。**尚無具授權的實際 GCP／LINE Staging、IAM／OAuth／TTL／排程部署驗收，因此不是 production-ready。**

## 已實作的功能

| 能力 | 實作位置 | 狀態 |
|---|---|---|
| `GET /healthz` | `cmd/webhook`、`adapter/inbound/http/health` | 可在沒有憑證下運行 |
| `POST /webhook/line` | `adapter/inbound/http/line` | 使用 LINE SDK `webhook.ParseRequest` 驗證簽章並同步處理文字訊息 |
| Reply Text | `application/usecases/conversation` 與 LINE adapter | 使用 LINE Messaging API SDK |
| Reply Flex | 同上 | 傳入 `flex` 或 `卡片` 時回覆簡單 bubble |
| Push Text／Flex | `cmd/ops` | 僅 CLI `push --send` 會實際呼叫 API，且需要接收者與 retry key |
| Profile（明確查詢） | `cmd/ops`、`application/usecases/contacts`、LINE adapter | `profile --lookup` 只在 CLI 查詢；已有離線契約測試，未完成真實 Channel 驗收 |
| Firestore、Cloud Tasks（背景佇列） | `internal/adapter/outbound/google/cloud/` | 具備選用 Durable Inbox/Outbox 與派送 Adapter，尚未部署 |
| Storage、Secret Manager | `internal/infrastructure/google/cloud/` | 原生 Client Factory，未接入業務 |
| Sheets、Drive、Calendar、Google Tasks（待辦） | `internal/infrastructure/google/api/` | 原生 Client Factory 已建立；個人資料仍需使用者 OAuth 授權 |
| OpenAI、Google Gen AI | `internal/infrastructure/openai/`、`internal/infrastructure/google/genai/` | 原生 Client Factory 已建立；尚未接入業務／啟用 |
| `garr` | `go.mod` | 選用工具，仍未啟用 |
| LINE 官方文件來源 | [line-developers-docs-source](https://github.com/line/line-developers-docs-source) | 僅供查閱，不是 SDK |

請閱讀 [main 基礎設施狀態與歷史殘留稽核](docs/architecture/infrastructure-status.md)、[架構](docs/architecture/overview.md)、[SDK 整合設計](docs/architecture/sdk-integration.md)、[SDK Catalog](docs/sdk/catalog.toml)、[技術與版本](docs/versions.md)、[功能矩陣](docs/line/feature-matrix.md)。

本專案直接使用 [`github.com/google/uuid`](https://github.com/google/uuid) 的 UUID v4 產生器建立 LINE Push retry key，並驗證手動指定的 retry key；不再自行實作 UUID 編碼。

## 統一設定與本機開發

- **唯一契約**：`internal/bootstrap/config_contract.go` 定義設定名稱、適用角色、Secret 分類與預設值，`internal/bootstrap/config.go` 與 `durable_config.go` 負責同一條解析／驗證路徑。
- **Role 與 Profile**：`APP_ENV=local|staging|production`；Webhook／Worker 按角色驗證，Cloud Run 必須明確指定 staging 或 production。Local 預設 sync，只用於開發；staging/production 的 Webhook 必須 durable。
- **Secret 隔離**：公開 durable Webhook 只讀取 Channel Secret，私有 Worker 只取得 LINE 發送 Token。普通環境變數可隨部署傳入；正式機密應透過 Secret Manager reference／最小 IAM 注入。
- **Logger**：`LOG_LEVEL`／`LOG_FORMAT` 在啟動時驗證；local 預設文字，staging/production 使用結構化 JSON。HTTP 觀測僅記錄固定 route label、status、duration，**不記錄原始路徑、查詢、body、token、LINE ID**。
- **單指令啟動**：`make dev` 使用既有 shell 環境與安全預設；不自動載入 `.env`。必要時用 `go run ./cmd/configcheck --validate --role webhook` 預先檢查設定（無外部 I/O），`make config-check` 檢查生成文件與 `.env.example` 是否一致。

設定詳情：[Runtime Configuration](docs/operations/configuration.md) · [Local Development](docs/operations/local-development.md) · [Durable Worker](docs/operations/durable-worker.md)。

## 環境要求

- Go **1.27.3**（已包含 Go 1.27 的安全修補；依 LINE/OpenAI SDK 當前要求）。
- `make`、Git。
- 本機 `sync` 測試才同時需要 LINE Channel Secret 與 Channel Access Token；Cloud Run 公開 Webhook 的 `durable` 模式只需簽章 Secret，發送 Token 僅屬於私有 Worker。
- 外部 Go dependencies 需要網路下載；`go.sum` 已依 GitHub Actions 的 Go module resolution 產生並提交，不可手寫 checksum。

## 本機使用

```sh
make bootstrap             # go mod tidy + download；產生並檢查 go.sum
make check                 # gofmt、vet、race tests、build
make dev                   # 強制清空 LINE Secret/Token 的安全本機 Health MVP
curl http://localhost:8080/healthz
```

未設定本機 `sync` 的 LINE 憑證時只開放 `/healthz`，`/webhook/line` 回 503。只有本機 sync 模式需要同時設定 `LINE_CHANNEL_SECRET` 與 `LINE_CHANNEL_ACCESS_TOKEN`；正式環境公開 Webhook 禁止配置發送 Token。設定契約與環境驗證請看 [設定文件](docs/operations/configuration.md)，不要把真實憑證寫進 GitHub。

此服務沒有公開的 Push HTTP endpoint；Push 僅透過需明確指定 `--send` 的 CLI：

```sh
go run ./cmd/ops push --to "<LINE_USER_ID>" --text "測試通知"
# 上方故意不會送出；需要明確加上 --send 並設定 LINE_CHANNEL_ACCESS_TOKEN。
```

CLI 發送會先輸出可重用的 UUID retry key。若發生 timeout 或結果未知，使用完全相同訊息內容及 `--retry-key` 重新執行，避免因新 key 重複送出；請勿將 key 和收件者資訊貼到公開 issue。

Audience Group 唯讀查詢範例：`go run ./cmd/ops audiences --kind list --page 1 --size 20 --read`。共享 Audience 的實際存取仍取決於 LINE 帳號資格。

Membership/Coupon 查詢僅供已取得所在地區及帳號功能資格的操作人員使用，例如 `go run ./cmd/ops memberships --kind memberships --read`。請勿存放會員 ID 或訂閱資料至公開日誌。

Coupon 已與 Membership 的內部 Port／Usecase 隔離；可使用 `go run ./cmd/ops coupons --read`，或 `go run ./cmd/ops coupons --kind coupon --coupon-id ID --read`。既有 `memberships --kind coupons|coupon` 仍可使用；所有查詢均受帳號資格與授權限制。

Operator CLI 的實作位於 `internal/adapter/inbound/cli/`，`cmd/ops/main.go` 僅作為程序入口。

Insights 唯讀查詢範例：`go run ./cmd/ops insights --kind followers --date 20261101 --read`。Insights SDK 另以相同受管理 HTTP Transport 建立 Client，但不會由公開 Webhook 自動建立。

Contacts/Group/Room 唯讀查詢須明確指定 `--read`（結果可能含使用者資料），例如 `go run ./cmd/ops contacts --kind followers --limit 10 --read`。不得將其輸出存入公開記錄，需遵循 LINE 服務條款及資料保護規範。

Webhook 設定及測試管理 CLI 需要雙重明確確認：`go run ./cmd/ops webhook-admin --action test --endpoint "https://example.com/webhook" --apply --confirm test`。請勿在未授權環境修改已使用的正式 Callback URL；此指令不會自動從公開 Webhook 執行。

訊息內容驗證可使用 `go run ./cmd/ops validate --kind reply --text "測試" --validate` 呼叫官方 SDK validation endpoint，不會對使用者發送訊息。

Broadcast／Multicast 需已授權的測試 Channel，透過 `go run ./cmd/ops campaign --kind multicast --to "U_ID_1,U_ID_2" --text "測試" --send` 明確觸發；Broadcast 將群發給所有符合 LINE 發送條件的好友，可能產生費用。Narrowcast 使用完整 LINE request JSON，例如 `go run ./cmd/ops campaign --kind narrowcast --request ./narrowcast.json --send`；檔案需包含 1–5 個帶有 `type` 的訊息，並可包含官方的 `recipient`、`filter`、`limit`、`notificationDisabled` 欄位。所有 campaign 的 HTTP timeout 都是結果未知，必須以完全相同內容和已輸出的 retry key 重試，不得產生新 key。

Rich Menu 變更指令需同時提供 `--apply --confirm ACTION`，例如僅在獲授權的隔離測試帳號中執行 `go run ./cmd/ops manage-menu --action set-default --menu-id richmenu-xxx --apply --confirm set-default`。程式碼可用不代表可擅自修改真實帳號；模糊或結果未知的發送不自動重試。

Rich Menu 唯讀查詢可執行 `go run ./cmd/ops menus --kind list --read`，其它 `--kind` 由 `internal/application/usecases/experience-presentation` 的 consumer-owned contract 限制。查詢單一選單需搭配 `--id`，所有操作都不會在公開 Webhook 執行。

LINE Channel 唯讀查詢需由操作人員明確指定：
```sh
go run ./cmd/ops inspect --kind bot-info --read
go run ./cmd/ops inspect --kind message-quota --read
go run ./cmd/ops inspect --kind delivery-push --date 20261101 --read
```
其他支援的 kind 參閱 `internal/adapter/inbound/cli/inspect.go` 與 `internal/application/usecases/provider-configuration`；沒有登入保護的公開查詢 HTTP 路由。這些命令需要具授權的測試 Channel Token，未經帳號授權不可呼叫正式服務。

Profile 查詢只供具授權的操作人員手動執行，不會由公開 Webhook 自動讀取使用者資料。先在安全環境提供 `LINE_CHANNEL_ACCESS_TOKEN`，再明確指定使用者及 `--lookup`：
```sh
go run ./cmd/ops profile --user "<LINE_USER_ID>" --lookup
```
Profile 回應包含個人資料，請勿將 stdout、Shell 紀錄或查詢結果提交至 GitHub。官方端點覆蓋及權限待確認項目請見 [LINE Endpoint Inventory](docs/line/endpoint-inventory.yaml)。

完整設定與測試說明：[本機開發](docs/operations/local-development.md)、[測試策略](docs/quality/test-strategy.md)。

## 架構與限制

- `domain/identity/`：PR #28 的變更已合併至 `main`，包含 `AccountLink` Aggregate、`GoogleConnection` Entity、`value-objects/`（Go package `valueobjects`）、Policy、Specification、Factory、Repository 讀取契約及 Domain Events。OAuth 憑證仍保存在 Adapter 的加密持久化機制，並非 Domain 狀態；其他 Bounded Context 須依真實業務不變量逐步建模。
- `application/`：Messaging Port、Reply／Push、文字回覆，以及持久 Ingestion／Worker Use Case。
- `adapter/`：同步與持久 Webhook、具 OIDC 驗證的 Worker、Firestore、Cloud Tasks、LINE 發送。
- `infrastructure/`：LINE 受管理 HTTP Runtime、Firestore／Cloud Tasks 的可選 Durable 業務接線，以及其餘 Provider 的原生 Typed Client Factories（`client_ready`）；SDK 已可建構不代表其真實服務已獲授權。
- `bootstrap/`：設定、組裝、啟動與 shutdown。
- `cmd/webhook`、`cmd/worker`、`cmd/ops`：已具可編譯入口；Worker 尚未部署。

**重要：預設 `LINE_EVENT_MODE=sync` 僅供本地開發，不持久化事件。Cloud Run 拒絕 sync；公開 Webhook 的 durable 模式只持有 `LINE_CHANNEL_SECRET`，發送 Token 僅交由私有 Worker 持有。** Durable 模式目前僅有程式碼與離線測試，缺少真實 Staging、unsend／edit、正式 IAM 與人工故障修復驗證；不能宣稱 production-ready 或端到端 exactly-once。

## Codex SDK 自動辨識（開發基礎）

- [docs/sdk/catalog.toml](docs/sdk/catalog.toml) 將 `go.mod` 中的全部直接依賴對應到 Provider、用途、import package 與預定的 Factory／Adapter；`go.mod` 仍是唯一版本來源。
- [sdk-integration Skill](.agents/skills/sdk-integration/SKILL.md) 指導 Codex 按需查詢 SDK 原始碼、gopls、官方文件，再依 Hexagonal Architecture 實作與測試。
- `go run ./cmd/codexcheck` 驗證 Catalog 與直接依賴一致、程式配置有效，並確保**已核准 SDK 依賴及預留 SDK 空白骨架不因清理而被移除**；CI 已納入這項檢查。這不代表其他 SDK 已初始化或取得 API 權限。
- 發現新的 SDK 功能時，只需按需擴充既有 Adapter／Factory；無須建立萬用動態 SDK Client。完整流程見 [SDK 整合架構](docs/architecture/sdk-integration.md)。

## Managed LINE SDK Runtime（階段 2）

- Webhook 透過 `bootstrap.NewWebhookRuntime` 組裝 LINE 原生 SDK Client，並管理專屬 HTTP Transport 的生命週期；CLI Push 也使用相同的受管理 HTTP 基礎。
- `internal/infrastructure/httpclient` 採用獨立連線池與 Timeout 設定，`observability` 只記錄安全 HTTP metadata，`resilience` 分類失敗但**不做未經審核的自動重試**。
- `LINE_API_TIMEOUT` 預設 `8s`，可調範圍 `1s` 到 `12s`。LINE v8.23.0 的 `WithContext` 會改動 Client，因此 Adapter 以每次呼叫的獨立副本帶入 Context，避免並行請求共用可變狀態。

## Provider Client Factories（階段 3）

- **Google Workspace**：Google Tasks、Calendar、Sheets、Drive 已有原生 `*Service` Factory，無須新增 Go Module。Google Tasks 是待辦清單，不是 Sheets 工作表，也不是 Google Cloud Tasks 佇列。
- **Google Cloud**：Firestore、Cloud Tasks（背景佇列）、Secret Manager、Storage 已有原生 Client Factory；啟用時需提供適當 ADC／Workload Identity、Project、IAM，並關閉擁有的 Client。
- **AI**：OpenAI、Gemini Developer API、Vertex AI 已建立分別驗證認證模式的原生 Client Factory；建構時不自動發送付費 API 請求。
- SDK Catalog 新增 `client_ready`，意為原生 Factory 存在但**不代表**已接入業務 Adapter、取得使用者 OAuth 或完成 Staging／Production 驗證。

Google Tasks、Calendar 的授權詳見 [Workspace OAuth 與 API](docs/sdk/workspace-oauth.md)。

## 持久 Webhook 與私有 Worker（階段 4）

- 預設 `LINE_EVENT_MODE=sync` 保留原開發模式。**只有**設定 `LINE_EVENT_MODE=durable` 並具備完整 GCP／LINE／OIDC 設定，才建立 Firestore 與 Cloud Tasks Client。
- Webhook 驗章及 bot destination，Firestore Transaction 同時建立 Inbox/Outbox 並以 SHA-256 Event Key 去重；Cloud Tasks 以事件 Key＋10 秒時窗產生可恢復的 Task ID，透過 Google OIDC 將不敏感 Key 交給私有 Worker；真正的發送去重由 Firestore Claim 負責。
- `cmd/worker` 的 `/tasks/v2/line-reply`、`/tasks/v2/dispatch` 必須通過 OIDC 驗證。Worker 執行單次 Claim（`pending -> sending -> sent/expired/unknown`），避免模糊發送結果後盲目重試。
- 單一 `Dockerfile` 與 `cloudbuild.yaml` 同時產出 public webhook、private worker 映像，但不自動部署。仍需隔離 Staging 的 Firestore TTL、Cloud Scheduler、IAM、Cloud Tasks、私有 Cloud Run 實際驗證。
- [事件狀態機](docs/architecture/event-processing.md)／[Staging 操作要求](docs/operations/durable-worker.md) 說明可靠性和安全限制。

## Codex Cloud 開發環境

- [`.codex/config.toml`](.codex/config.toml) 配置 gopls／Context7／OpenAI Docs MCP、可在明確授權下寫入的 GitHub MCP，並保留停用中的 Firestore Toolbox。Codex Cloud agent-phase 需允許 `developers.openai.com`，在 Republish 後的新工作階段驗證連線。
- [六個專用 Agents](docs/ai/agents.md) 負責 DDD、LINE SDK、Cloud Platform、Go Review、官方文件查證，以及交付前的 Code Cleanup & Release Validation。
- `bash scripts/codex-cloud/install.sh` 是 Codex Cloud **Install script**；新工作階段以 `bash scripts/codex-cloud/start.sh` 強制同步 main 並驗證環境。
- 專案配置以 `go run ./cmd/codexcheck` 驗證，無須 Python；詳見 [Codex Cloud 建置步驟](docs/ai/codex-cloud.md)，其中的環境發布與 MCP 授權**尚未執行**。

## Cloud Build／Cloud Run

根目錄提供 `cloudbuild.yaml` 與 `cloudbuild.deploy.yaml`：前者驗證一次並建置 public webhook、private worker，後者以不可變映像部署既有 Cloud Run 服務。部署設定固定指向 `bubbly-area-513305-s0` 的 `line-bot-public` 與 `line-bot-private`；所有建立映像均以 sha256 digest 固定。`.gcloudignore` 排除本機／敏感上傳內容，`.gitattributes` 固定文字換行規則。操作前先閱讀 [部署配置與限制](docs/operations/deployment.md)。

`skills-lock.json` 自 [line-bot-go](https://github.com/963sup/line-bot-go/blob/main/skills-lock.json) 精選 13 個與 Go／Cloud Run／安全／CI 有關的技能；**尚未安裝外部 skills**，詳見 [Codex Skills](docs/ai/skills.md)。


## CI 快取與驗證完整性

- GitHub Actions 使用既有 `actions/setup-go` 的內建 **Go Modules／Build Cache**（以 `go.sum` 為快取依據）。先前無快取的 [#41558587334](https://github.com/963sup/linebot-platform/actions/runs/41558587334) 中，`go vet` 約 91 秒、`go test -race` 約 144 秒，是主要耗時來源。
- 快取用於降低重複編譯成本，**不會**跳過 `go mod tidy`／鎖定檔驗證、`gofmt`、`go vet`、`go test -race -count=1` 或 `go build`；首次冷快取仍可能耗時較長。未使用的 SDK 依賴與預留目錄不刪除。

## Outbox 狀態監控與人工處置（規劃中）

本倉庫新增 `cmd/ops events` 的只讀 Staging 稽核入口，透過 Firestore Aggregation Count 與有限筆數 metadata 檢視 Pending、Sending、Sent、Expired、Unknown。檢查模式只輸出安全 JSON summary，Unknown 或過久 Pending 可以使定期 Cloud Run Job 非零退出，以便在**經核准部署後**使用 Cloud Monitoring 失敗執行告警。此工具不會建立 Task、重發 LINE 或解除 Unknown；所有 Staging 資源、告警與身分仍待實測。

```sh
# 只在已核准、已具最小讀取 IAM 的隔離 Staging 執行
export LINEBOT_STAGING_PROJECT='<APPROVED_STAGING_PROJECT>'
go run ./cmd/ops events --project "$LINEBOT_STAGING_PROJECT" --inspect --state unknown --limit 20
go run ./cmd/ops events --project "$LINEBOT_STAGING_PROJECT" --check --max-pending-age 2m
```

操作與修復界線參閱 [Outbox Operations](docs/operations/outbox-operations.md)；真正的 Cloud Run IAM／OIDC／TTL／Scheduler／LINE Redelivery 驗收契約參閱 [Staging Acceptance](docs/operations/staging-acceptance.md)。

## 開發與安全

[AGENTS.md](AGENTS.md) 定義分層與開發規範；[CONTRIBUTING.md](CONTRIBUTING.md) 說明 PR 與驗證方式；[SECURITY.md](SECURITY.md) 說明安全通報。不要提交 API Key、Channel Secret、token、真實訊息或服務帳號金鑰。

完整 [Roadmap](docs/roadmap.md) 按風險先後處理可靠事件流程、Cloud Tasks／Firestore、再擴充 LINE API 與 AI 功能。
