# Infrastructure Layer

提供 SDK Client 建構、認證、外部 Transport、錯誤分類、觀測與生命週期管理。已實作：`httpclient` 的受管理 HTTP Client（獨立 Transport、Timeout、Close）、`observability` 的去敏出站 Debug Log、`resilience` 的可分類安全錯誤（**不自動重試**），以及可注入 HTTP Client 的 LINE Messaging Client Factory。Webhook 的組裝與資源釋放由 `internal/bootstrap` 管理，CLI Push 亦重用 `httpclient`。SDK Context 不可直接修改共用 LINE Client；使用每次呼叫的 Context 隔離。

Google Cloud、Workspace、OpenAI 和 Google Gen AI SDK 仍只在 `go.mod` 宣告，未建立可供正式執行的 Client。未完成 Firestore Inbox/Outbox、Cloud Tasks Worker、憑證驗證與實際 LINE E2E；不屬於 production-ready。`garr` 選用功能不等於 Cloud Tasks 持久工作。
