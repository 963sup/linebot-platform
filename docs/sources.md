# 技術來源與採用界線

以下為本專案的**官方／上游技術參考**，目前只有規劃，尚未安裝相應 SDK。

| 來源（README） | Go module 或參考方式 | 本專案預定用途 |
|---|---|---|
| [google-cloud-go](https://github.com/googleapis/google-cloud-go/blob/main/README.md) | `cloud.google.com/go/<service>` | Firestore、Cloud Tasks、Storage、Secret Manager 等雲端服務。 |
| [google-api-go-client](https://github.com/googleapis/google-api-go-client/blob/main/README.md) | `google.golang.org/api` | Sheets、Drive、Calendar 等 Discovery/Workspace API；該專案 README 註明處於 maintenance mode。 |
| [line-bot-sdk-go](https://github.com/line/line-bot-sdk-go/blob/master/README.md) | `github.com/line/line-bot-sdk-go/v8/linebot` | Messaging API、Webhook、Audience、Insight、Channel token 等。 |
| [garr](https://github.com/line/garr/blob/main/README.md) | `go.linecorp.com/garr` | 選配 retry、circuit breaker、程序內韌性；不能代替 Cloud Tasks 的持久處理。 |
| [line-developers-docs-source](https://github.com/line/line-developers-docs-source/blob/main/README.md) | 官方文件來源（不當 runtime dependency） | 核對事件與 endpoints。上游標示 read-only，請勿直接複製無授權內容。 |
| [openai-go](https://github.com/openai/openai-go/blob/main/README.md) | `github.com/openai/openai-go/v3` | OpenAI Responses API 等 AI 功能。 |
| [go-genai](https://github.com/googleapis/go-genai/blob/main/README.md) | `google.golang.org/genai` | Gemini API／支援的 Google AI 後端。 |

## 採用原則

1. **Google Cloud 服務優先使用 google-cloud-go**，不要為同一目的無差別混用 API client。
2. LINE 與 OpenAI 官方 README 目前都要求至少 Go 1.26，OpenAI README 還要求採用已含 HTTP/2 安全修補的 patch；建立 `go.mod` 時重新查證並使用已修補的工具鏈。
3. 所有客戶端實作隔離於 `adapter/outbound` 與 `infrastructure`；domain 不依賴外部 SDK。
4. 只有當功能被實際使用、測試與審核後才更新 `go.mod` 並引入 dependency。
5. LINE「全功能」以可用公開 API reference 為追蹤範圍，資格或地區受限的 endpoints 要有 feature gate。
