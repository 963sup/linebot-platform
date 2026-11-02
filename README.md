# linebot-platform

使用 Go、DDD（Domain-Driven Design）與 Hexagonal Architecture 的 LINE Messaging API 平台；規劃部署在 Google Cloud Run，並保留 Google Cloud、Workspace、OpenAI 與 Google Gen AI 整合介面。

> **狀態：開發用 MVP，非 production-ready。** 已加入 HTTP 健康檢查、LINE SDK 驗簽 webhook、文字 Reply、簡單 Flex Message Reply，以及命令列 Push。尚無 Firestore inbox/outbox、Cloud Tasks 或跨執行個體去重；沒有部署任何雲端資源，也沒有真實 LINE 帳號測試。

## 已實作的功能

| 能力 | 實作位置 | 狀態 |
|---|---|---|
| `GET /healthz` | `cmd/webhook`、`adapter/inbound/http/health` | 可在沒有憑證下運行 |
| `POST /webhook/line` | `adapter/inbound/http/line` | 使用 LINE SDK `webhook.ParseRequest` 驗證簽章並同步處理文字訊息 |
| Reply Text | `application/usecases/conversation` 與 LINE adapter | 使用 LINE Messaging API SDK |
| Reply Flex | 同上 | 傳入 `flex` 或 `卡片` 時回覆簡單 bubble |
| Push Text／Flex | `cmd/ops` | 僅 CLI `push --send` 會實際呼叫 API，且需要接收者與 retry key |
| Firestore、Storage、Cloud Tasks、Secret Manager | `go.mod` 依賴與預留目錄 | SDK 版本已聲明；尚未連接 |
| Sheets、Drive、Calendar | `go.mod` 依賴與預留目錄 | SDK 已聲明；未啟用 |
| OpenAI、Google Gen AI、garr | `go.mod` 依賴與預留目錄 | SDK 已聲明；未啟用 |
| LINE 官方文件來源 | [line-developers-docs-source](https://github.com/line/line-developers-docs-source) | 僅供查閱，不是 SDK |

請閱讀 [架構](docs/architecture/overview.md)、[技術與版本](docs/versions.md)、[功能矩陣](docs/line/feature-matrix.md)。

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

- `domain/`：業務不變量、aggregates、valueobjects；本階段保留擴充空間。
- `application/`：Messaging Port、驗證、Reply／Push、基本文字回覆 usecase。
- `adapter/`：LINE webhook 和 outbound Messaging API SDK；健康檢查。
- `infrastructure/`：LINE Client Factory；其餘 SDK 待按需接線。
- `bootstrap/`：設定、組裝、啟動與 shutdown。
- `cmd/webhook`、`cmd/ops`：實際可執行程式；`cmd/worker` 尚待實作。

**重要：目前同步回覆並未持久化 incoming event。** 在 LINE webhook 重送、批次處理部分成功、容器重啟時，仍可能造成重複發送；未完成 Firestore inbox/outbox + Cloud Tasks + 冪等策略前，不能稱為正式環境可靠 Bot。

## 開發與安全## Cloud Build（未啟用）

根目錄新增 `cloudbuild.yaml`（Go 驗證、Docker build、Artifact Registry push）與 `cloudbuild.deploy.yaml`（僅供明確核准的**既有 staging** 服務部署）。兩份配置**不會因為提交 GitHub 而自動執行**；目前未建立 Cloud Build Trigger、Cloud Run 服務，也未授權付費雲端操作。`.gcloudignore` 排除本機／敏感上傳內容，`.gitattributes` 固定文字換行規則。操作前先閱讀 [部署配置與限制](docs/operations/deployment.md)。

`skills-lock.json` 自 [line-bot-go](https://github.com/963sup/line-bot-go/blob/main/skills-lock.json) 精選 13 個與 Go／Cloud Run／安全／CI 有關的技能；**尚未安裝外部 skills**，詳見 [Codex Skills](docs/ai/skills.md)。



[AGENTS.md](AGENTS.md) 定義分層與開發規範；[CONTRIBUTING.md](CONTRIBUTING.md) 說明 PR 與驗證方式；[SECURITY.md](SECURITY.md) 說明安全通報。不要提交 API Key、Channel Secret、token、真實訊息或服務帳號金鑰。

完整 [Roadmap](docs/roadmap.md) 按風險先後處理可靠事件流程、Cloud Tasks／Firestore、再擴充 LINE API 與 AI 功能。
