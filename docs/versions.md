# 版本與依賴策略

Go module 已加入，`go.mod`／`go.sum` 由 Go 工具鏈與 CI 管理；此表僅為說明用途。**版本唯一真實來源為 `go.mod`**，`docs/sdk/catalog.toml` 僅記錄 SDK 路由及目前接線狀態，不另行維護版本。模組已解析不代表外部服務已啟用。

實作時請先查閱各專案當下 README／release：
- LINE Go SDK：`github.com/line/line-bot-sdk-go/v8/linebot`，其 README 要求 Go 1.26+。
- OpenAI Go SDK：`github.com/openai/openai-go/v3`，其 README 要求 Go 1.26+，並特別列出 HTTP/2 安全修補的最低 patch。
- Google Gen AI Go SDK（Gemini 等）：`google.golang.org/genai`；實作路徑 `internal/adapter/outbound/genai/` 與 `internal/infrastructure/google/genai/`。
- Google Cloud：依服務採 `cloud.google.com/go/<service>`。
- Google Workspace／Discovery：`google.golang.org/api`。
- 程序內韌性選配：`go.linecorp.com/garr`。

正式採用前需固定實際版本、CI Go 工具鏈、建置映像與 Actions SHA；不得以浮動 `latest` 作為可重現 build 依據。

## 開發工具依賴

- `github.com/BurntSushi/toml v1.6.0`：Go 的 TOML 解析套件，用於 `cmd/codexcheck` 驗證專案 Codex 與 Agents 設定；不會被 LINE Bot 的執行時程式直接 import。

## main 的直接依賴版本（go.mod）

| SDK module | 宣告版本 | 功能狀態 |
|---|---|---|
| `github.com/BurntSushi/toml` | v1.6.0 | `cmd/codexcheck` 的 TOML／Agents／SDK Catalog 驗證工具 |
| `github.com/google/uuid` | v1.6.0 | 直接依賴；Push CLI 使用 UUID v4 作為 LINE retry key |
| `github.com/line/line-bot-sdk-go/v8` | v8.23.0 | Webhook、Reply、Push、Flex 由 SDK 接線 |
| `github.com/openai/openai-go/v3` | v3.91.1 | 原生 Client Factory 已建，尚未接入業務 |
| `google.golang.org/genai` | v1.77.0 | Gemini／Vertex 分別建立 Factory，未授權啟用 |
| `cloud.google.com/go/firestore` | v1.29.2 | 原生 Client Factory 與 Durable Inbox/Outbox Adapter 已接線；僅在明確啟用持久模式後使用，尚未 Staging 驗收 |
| `cloud.google.com/go/storage` | v1.70.0 | Client Factory 已建，業務 Adapter 未接線 |
| `cloud.google.com/go/cloudtasks` | v1.21.1 | 原生 Client Factory、背景任務 Queue Adapter、OIDC Worker 與 Reconcile 程式碼已實作；尚未正式部署 |
| `cloud.google.com/go/secretmanager` | v1.23.1 | Client Factory 已建，尚未使用正式憑證 |
| `google.golang.org/api` | v0.310.0 | Drive、Sheets、Calendar、Google Tasks 原生 Factory 已建；尚無使用者 OAuth Consent／Token 管理／業務 Adapter |
| `go.linecorp.com/garr` | v0.1.1 | Worker pool / retry / circuit breaker 選用、未啟用 |
| `google.golang.org/grpc` | v1.84.0 | Firestore／Cloud Tasks 狀態碼與 gRPC 錯誤處理的直接依賴（原間接依賴升級分類，未新增下載） |

Google Tasks API 使用現有模組的 `google.golang.org/api/tasks/v1`，**沒有新增 Go Module**；Google Cloud Tasks 則使用獨立的 `cloud.google.com/go/cloudtasks`。

PR #8 實作的 Firestore／Cloud Tasks Adapter 直接使用 `google.golang.org/grpc/codes`／`status`，因此 `go mod tidy` 將既有 gRPC module 轉為直接依賴；`go.sum` 不新增未經驗證的 checksum。

**上述 12 個直接依賴及 SDK 預留空白骨架是經核准的開發基礎，不得因目前 runtime 未啟用就移除。** `integrated` 表示程式碼已接線，不表示 IAM／OAuth／部署完成。詳見 [main 基礎設施狀態](architecture/infrastructure-status.md) 與 [SDK Catalog](sdk/catalog.toml)。

Go 版本鎖定為 **1.27.3**。若 `go mod tidy` 因間接依賴調整版本，應以實際可重現建置結果重新審查本表。Google Cloud Client Libraries 的多 module 架構不等於只安裝一個 `google-cloud-go` root module。
