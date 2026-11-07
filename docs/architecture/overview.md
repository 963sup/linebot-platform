# 架構概覽（Production baseline 已建立；外部環境尚未驗證）

採 Go modular monolith、DDD 與 Hexagonal Architecture。**預設同步 LINE Webhook／CLI 已接線；可選的 Firestore Inbox/Outbox 與 Cloud Tasks Worker 程式碼已接線，但 Google Cloud 資源、IAM／OIDC、LINE 測試 Channel 與正式部署尚未啟用或驗證，因此本儲存庫不能宣稱 production-ready。**

Repository-specific 的目錄、compile-time dependency、runtime call、interface implementation、事件與驗收語意見唯一的 [Go DDD + Hexagonal Architecture Blueprint](blueprint.md)。

```text
LINE Platform
  -> cmd/webhook: LINE SDK raw-body signature verification
      |-> 預設 LINE_EVENT_MODE=sync -> Text/Flex usecase -> LINE Messaging API
      |-> 選用 LINE_EVENT_MODE=durable
           -> Firestore Inbox + Outbox (atomic transaction)
           -> Cloud Tasks OIDC queue
           -> cmd/worker (private OIDC-authenticated handler)
           -> LINE Reply via outbound Messaging port

LINE MINI App browser
  -> public page/assets/config
  -> explicit Web Geolocation permission
  -> verify Bearer LIFF ID token from liff.getIDToken()
  -> verify the LINE ID token through the Go HTTPS API
  -> request one Web Geolocation reading after an explicit user gesture
  -> keep the reading client-side until a future business use case owns it

OpenAI、GenAI、MCP client 保留原生 factory，但尚未接入 LINE 訊息業務。Workspace 與 Cloud Storage factory 已移除，日後依實際需求整合。
Cloud Run／Firestore／Cloud Tasks 實際資源與 IAM 均未由儲存庫自動建立。
```

MINI App 的 Geolocation 是 browser mechanism。Go API 驗證 LINE ID token，但不公開 provider subject；LocationReading 目前不送往 Firestore，也不宣稱本人實體在場。公開 origin、LIFF／MINI App、瀏覽器權限與外部 token 驗證在 Staging／Production 均為 `Not Verified`。

其他 SDK 的 Client Factory、Codex SDK Catalog 與逐步啟用計畫見 [SDK Integration 設計](sdk-integration.md)。Catalog 僅為開發路由資料，非執行期 Registry。

## 設定契約與啟動邊界

`internal/bootstrap/config_contract.go` 是 Go Runtime 設定的唯一契約；Webhook、Worker 共用同一組解析與驗證邏輯。各角色僅取得需要的 Secret，Cloud Run 必須明確選擇 staging/production Profile 且 Webhook 不得持有 LINE 發送 Token。正式文件由 `go run ./cmd/configcheck --write-doc` 產生，`make check` 檢查設定與文件一致。詳見 [Runtime Configuration](../operations/configuration.md)。

可靠性與依賴規則由架構藍圖定義；durable runtime 的操作細節見 [事件處理](event-processing.md)。目前除了同步 LINE Webhook／Push 示範，已新增需明確設定 `LINE_EVENT_MODE=durable` 才會啟用的 Firestore 原子 Inbox/Outbox、Cloud Tasks OIDC Worker 與定期 Reconcile 程式碼。這些屬於程式與測試層級的 production baseline；**尚未部署任何 Google Cloud 資源**，也未實測真實 IAM、OIDC、LINE redelivery 或高吞吐，因此不是 Staging／Production 驗收證據。預設同步模式仍不具持久保證。
