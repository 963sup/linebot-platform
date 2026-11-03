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
| Firestore、Cloud Tasks（背景佇列） | `internal/adapter/outbound/google/cloud/` | 具備選用 Durable Inbox/Outbox 與派送 Adapter，尚未部署 |
| Storage、Secret Manager | `internal/infrastructure/google/cloud/` | 原生 Client Factory，未接入業務 |
| Sheets、Drive、Calendar、Google Tasks（待辦） | `internal/infrastructure/google/api/` | 原生 Client Factory 已建立；個人資料仍需使用者 OAuth 授權 |
| OpenAI、Google Gen AI | `internal/infrastructure/openai/`、`internal/infrastructure/google/genai/` | 原生 Client Factory 已建立；尚未接入業務／啟用 |
| `garr` | `go.mod` | 選用工具，仍未啟用 |
| LINE 官方文件來源 | [line-developers-docs-source](https://github.com/line/line-developers-docs-source) | 僅供查閱，不是 SDK |

請閱讀 [main 基礎設施狀態與歷史殘留稽核](docs/architecture/infrastructure-status.md)、[架構](docs/architecture/overview.md)、[SDK 整合設計](docs/architecture/sdk-integration.md)、[SDK Catalog](docs/sdk/catalog.toml)、[技術與版本](docs/versions.md)、[功能矩陣](docs/line/feature-matrix.md)。

本專案直接使用 [`github.com/google/uuid`](https://github.com/google/uuid) 的 UUID v4 產生器建立 LINE Push retry key，並驗證手動指定的 retry key；不再自行實作 UUID 編碼。

## 環境要求

- Go **1.27.3**（已包含 Go 1.27 的安全修補；依 LINE/OpenAI SDK 當前要求）。
- `make`、Git。
- 有效 LINE Channel Secret + Channel Access Token：只有啟用 /webhook/line 與 CLI 發送時才需要。
- 外部 Go dependencies 需要網路下載；`go.sum` 已依 GitHub Actions 的 Go module resolution 產生並提交，不可手寫 checksum。

## 本機使用

```sh
make bootstrap             # go mod tidy + download；產生並檢查 go.sum
make check                 # gofmt、vet、race tests、build
make run-webhook           # 預設 0.0.0.0:8080
curl http://localhost:8080/healthz
```

沒有 LINE 憑證時服務只開放 `/healthz`，`/webhook/line` 回傳 503。要接收 LINE webhook，請透過安全方式在執行環境設定 `LINE_CHANNEL_SECRET` 與 `LINE_CHANNEL_ACCESS_TOKEN`（兩個必須同時存在）；不要把憑證寫進檔案或提交 GitHub。

此服務沒有公開的 Push HTTP endpoint；Push 僅透過需明確指定 `--send` 的 CLI：

```sh
go run ./cmd/ops push --to "<LINE_USER_ID>" --text "測試通知"
# 上方故意不會送出；需要明確加上 --send 並設定 LINE_CHANNEL_ACCESS_TOKEN。
```

CLI 發送會先輸出可重用的 UUID retry key。若發生 timeout 或結果未知，使用完全相同訊息內容及 `--retry-key` 重新執行，避免因新 key 重複送出；請勿將 key 和收件者資訊貼到公開 issue。

完整設定與測試說明：[本機開發](docs/operations/local-development.md)、[測試策略](docs/quality/test-strategy.md)。

## 架構與限制

- `domain/`：業務不變量、aggregates、`value-objects/`（Go package `valueobjects`）；本階段保留擴充空間。
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
- `cmd/worker` 的 `/tasks/line-reply`、`/tasks/dispatch` 必須通過 OIDC 驗證。Worker 執行單次 Claim（`pending -> sending -> sent/expired/unknown`），避免模糊發送結果後盲目重試。
- `Dockerfile.worker` 與 `cloudbuild.worker.yaml` 僅是建置範本，不自動部署。仍需隔離 Staging 的 Firestore TTL、Cloud Scheduler、IAM、Cloud Tasks、私有 Cloud Run 實際驗證。
- [事件狀態機](docs/architecture/event-processing.md)／[Staging 操作要求](docs/operations/durable-worker.md) 說明可靠性和安全限制。

## Codex Cloud 開發環境

- [`.codex/config.toml`](.codex/config.toml) 配置 gopls／Context7／OpenAI Docs MCP，並保留停用中的唯讀 GitHub MCP／Firestore Toolbox。Codex Cloud agent-phase 需允許 `developers.openai.com`，在 Republish 後的新工作階段驗證連線。
- [六個專用 Agents](docs/ai/agents.md) 負責 DDD、LINE SDK、Cloud Platform、Go Review、官方文件查證，以及交付前的 Code Cleanup & Release Validation。
- `bash scripts/setup-codex-cloud.sh` 是可加入 Codex Cloud **Install script** 的非互動式環境準備腳本（下載 Go modules、安裝固定版 gopls）。
- 專案配置以 `go run ./cmd/codexcheck` 驗證，無須 Python；詳見 [Codex Cloud 建置步驟](docs/ai/codex-cloud.md)，其中的環境發布與 MCP 授權**尚未執行**。

## Cloud Build（未啟用）

根目錄新增 `cloudbuild.yaml`（Go 驗證、Docker build、Artifact Registry push）與 `cloudbuild.deploy.yaml`（僅供明確核准的**既有 staging** 服務部署）。兩份配置**不會因為提交 GitHub 而自動執行**；目前未建立 Cloud Build Trigger、Cloud Run 服務，也未授權付費雲端操作。`.gcloudignore` 排除本機／敏感上傳內容，`.gitattributes` 固定文字換行規則。操作前先閱讀 [部署配置與限制](docs/operations/deployment.md)。

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
