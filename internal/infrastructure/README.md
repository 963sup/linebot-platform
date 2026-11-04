# Infrastructure Layer

基礎設施程式碼已併入 `main`（PR #8，2026-11-03）。此處負責官方 SDK Client Factory、認證邊界、HTTP Transport、錯誤分類、觀測資料與資源生命週期；**已實作不等於已部署或取得服務權限**。

- **LINE**：`internal/infrastructure/line` 具備可注入 `*http.Client` 的原生 Messaging Client Factory，`httpclient` 提供獨立 Transport、Timeout、Close，`observability` 僅記錄去敏運作資訊，`resilience` 提供安全錯誤分類。Webhook 與 Push CLI 已由 Bootstrap 管理資源；LINE SDK `WithContext` 的共享可變狀態由 Adapter 的逐請求副本隔離。
- **Google Cloud**：Firestore、Cloud Tasks、Secret Manager、Storage 的 Typed Client Factories 已存在。Firestore Inbox/Outbox Adapter 及 Cloud Tasks Queue Adapter、私有 OIDC Worker／Reconcile 在 **`LINE_EVENT_MODE=durable`** 明確設定後才會啟用；不代表雲端帳號或 IAM 已配置。
- **Google Workspace**：Google Tasks、Calendar、Sheets、Drive 都使用 `google.golang.org/api` 的原生 Factory；使用者 OAuth 同意、Scope、Token 保管、正式業務 Adapter 均尚未完成。
- **AI**：OpenAI、Gemini Developer API、Vertex AI 的原生 Factory 已建立；未接入 Webhook 業務，也不會在啟動時自動呼叫付費 API。
- **保留依賴**：`go.linecorp.com/garr` 為已核准、尚未啟用的工具模組；依賴版本由 `go.mod`／`go.sum` 保存，路由與接線狀態由 SDK Catalog 保存，可選模組 import 由 `internal/infrastructure/dependencies/modules.go` 保存。空白 SDK 目錄不是依賴保留機制，也不代表功能已實作。

驗證：`go run ./cmd/codexcheck` 與 `make check` 由 CI 執行，保護依賴、架構、型別、Race Test 與建置。PR #8 的 Firestore Emulator 整合測試曾通過，但一次性驗收檔已於合併前移除；**真實 GCP IAM／OIDC／Firestore TTL／Cloud Scheduler／LINE 測試 Channel／使用者 OAuth 仍待隔離 Staging 驗收**。

詳細狀態見 [SDK Integration](../../docs/architecture/sdk-integration.md)、[main 基礎設施狀態](../../docs/architecture/infrastructure-status.md)。
