# 技術來源與採用界線

以下為本專案的**官方／上游技術參考**。`go.mod` 已鎖定 12 個直接依賴，GitHub Actions 已下載並驗證版本鎖定；目前 LINE SDK 已用於 Webhook／Reply／Push，Firestore 與 Cloud Tasks SDK 已用於可選的 Durable Inbox/Outbox／Worker，其他 Provider 已有原生 Client Factory。**程式碼已接線不代表真實帳號、OAuth、IAM 或正式部署已啟用。**

| 來源（README） | Go module 或參考方式 | 本專案預定用途 |
|---|---|---|
| [google-cloud-go](https://github.com/googleapis/google-cloud-go/blob/main/README.md) | `cloud.google.com/go/<service>` | Firestore、Cloud Tasks、Storage、Secret Manager 等雲端服務。 |
| [google-api-go-client](https://github.com/googleapis/google-api-go-client/blob/main/README.md) | `google.golang.org/api` | Sheets、Drive、Calendar、Google Tasks 等 Discovery/Workspace API；個人資料需使用者 OAuth，且與 Cloud Tasks 不同。 |
| [line-bot-sdk-go](https://github.com/line/line-bot-sdk-go/blob/master/README.md) | `github.com/line/line-bot-sdk-go/v8/linebot` | Messaging API、Webhook、Audience、Insight、Channel token 等。 |
| [line-developers-docs-source](https://github.com/line/line-developers-docs-source) | LINE Developers 文件原始碼 | LINE 平台規格、事件 payload、Messaging API、LIFF、LINE Login 與 Account Linking 的文件查證；Go SDK 型別仍以已鎖定 module 原始碼為準。 |
| [garr](https://github.com/line/garr/blob/main/README.md) | `go.linecorp.com/garr` | 選配 retry、circuit breaker、程序內韌性；不能代替 Cloud Tasks 的持久處理。 |
| [line-developers-docs-source](https://github.com/line/line-developers-docs-source/blob/main/README.md) | 官方文件來源（不當 runtime dependency） | 核對事件與 endpoints。上游標示 read-only，請勿直接複製無授權內容。 |
| [openai-go](https://github.com/openai/openai-go/blob/main/README.md) | `github.com/openai/openai-go/v3` | OpenAI Responses API 等 AI 功能。 |
| [go-genai](https://github.com/googleapis/go-genai/blob/main/README.md) | `google.golang.org/genai` | Google Gen AI SDK（Gemini API 等）；adapter 放 `internal/adapter/outbound/genai/`、client 初始化放 `internal/infrastructure/google/genai/`。 |

Google Tasks、Calendar 的 OAuth 範圍與 Factory 使用方式參閱 [Workspace OAuth 與 API](sdk/workspace-oauth.md)。

Firestore／Cloud Tasks 的持久事件與 OIDC Worker 接線流程參考 [事件處理](architecture/event-processing.md)，隔離 IAM／TTL／排程要求參考 [Durable Worker](operations/durable-worker.md)。

所有 `go.mod` 直接依賴的用途及預定接線路徑可查閱 [SDK Catalog](sdk/catalog.toml)。Catalog 不鎖定版本、不代表 API 已啟用，API 參數與權限仍應以實際模組程式碼和官方文件查證。Codex 工作流程請參考 [SDK Integration 設計](architecture/sdk-integration.md)。

## 採用原則

1. **Google Cloud 服務優先使用 google-cloud-go**，不要為同一目的無差別混用 API client。
2. LINE 與 OpenAI 官方 README 目前都要求至少 Go 1.26，OpenAI README 還要求採用已含 HTTP/2 安全修補的 patch；建立 `go.mod` 時重新查證並使用已修補的工具鏈。
3. 所有客戶端實作隔離於 `adapter/outbound` 與 `infrastructure`；domain 不依賴外部 SDK。
4. 已核准的直接依賴（含暫未使用的 `garr`）由 `go.mod`／`go.sum`、SDK Catalog 與 build-tag dependency manifest 保留；不以空 SDK 目錄保留依賴。**額外新增**依賴時按實際功能、版本查證及測試審核變更 `go.mod`。
5. LINE「全功能」以可用公開 API reference 為追蹤範圍，資格或地區受限的 endpoints 要有 feature gate。
6. `line-developers-docs-source` 是文件來源，不是 Go SDK，不能加入 `go.mod`。
7. `go.linecorp.com/garr` 保留在預先宣告的選用模組中，不能用其程序內 queue 代替 Cloud Tasks 的持久排程。
