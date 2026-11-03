# 架構概覽（程式碼已建立，部署尚待驗證）

採 Go modular monolith、DDD 與 Hexagonal Architecture。**預設同步 LINE Webhook／CLI 已接線；可選的 Firestore Inbox/Outbox 與 Cloud Tasks Worker 程式碼已接線，但 Google Cloud 資源、IAM／OIDC 與正式部署尚未啟用或驗證。**

```text
LINE Platform
  -> cmd/webhook: LINE SDK raw-body signature verification
      |-> 預設 LINE_EVENT_MODE=sync -> Text/Flex usecase -> LINE Messaging API
      |-> 選用 LINE_EVENT_MODE=durable
           -> Firestore Inbox + Outbox (atomic transaction)
           -> Cloud Tasks OIDC queue
           -> cmd/worker (private OIDC-authenticated handler)
           -> LINE Reply via outbound Messaging port

預留但尚未接入業務：OpenAI、Gemini、Workspace、Storage、Secret Manager。
Cloud Run／Firestore／Cloud Tasks 實際資源與 IAM 均未由儲存庫自動建立。
```

其他 SDK 的 Client Factory、Codex SDK Catalog 與逐步啟用計畫見 [SDK Integration 設計](sdk-integration.md)。Catalog 僅為開發路由資料，非執行期 Registry。

## 設定契約與啟動邊界

`internal/bootstrap/config_contract.go` 是 Go Runtime 設定的唯一契約；Webhook、Worker 共用同一組解析與驗證邏輯。各角色僅取得需要的 Secret，Cloud Run 必須明確選擇 staging/production Profile 且 Webhook 不得持有 LINE 發送 Token。正式文件由 `go run ./cmd/configcheck --write-doc` 產生，`make check` 檢查設定與文件一致。詳見 [Runtime Configuration](../operations/configuration.md)。

## 邊界

- **Domain**：業務模型與不變量，不接觸 SDK；Value Objects 預留於 `internal/domain/value-objects/`（Go package 使用 `valueobjects`）。
- **Application**：執行流程、命令、查詢、ports；只有協調，不含 HTTP 細節。
- **Adapter**：HTTP/CLI 入站，SDK / Firestore / Cloud Tasks 等出站；Gemini provider 使用 `internal/adapter/outbound/genai/`。
- **Infrastructure**：外部 client、憑證、transport、觀測與執行設定；Google Gen AI SDK client 預留於 `internal/infrastructure/google/genai/`。
- **Bootstrap**：依賴組裝；cmd 只啟動服務。

## 可靠性

- LINE webhook 驗章與 idempotent inbox。
- Inbox + outbox 在相同持久交易中保存，再由 dispatcher 交給 Cloud Tasks。
- Cloud Tasks 可能重複執行，worker 要使用 lease、防重入與修復流程。
- Reply token 有時效，長時間 AI 工作要明確決定 push 策略。
- 發送 HTTP timeout 可能造成結果不確定，需追蹤相同 retry key 和 request payload。
- Unsend／edit 會影響記憶、儲存與尚未完成的 AI 回覆。
- 保留對資料存取、AI 訊息資料、隱私與帳號限制的審核。

目前除了同步 LINE Webhook／Push 示範，已新增需明確設定 `LINE_EVENT_MODE=durable` 才會啟用的 Firestore 原子 Inbox/Outbox、Cloud Tasks OIDC Worker 與定期 Reconcile 程式碼。**尚未部署任何 Google Cloud 資源**，也未實測真實 IAM、OIDC、LINE redelivery 或高吞吐；預設同步模式仍不具持久保證。參閱 [事件處理](event-processing.md)。
