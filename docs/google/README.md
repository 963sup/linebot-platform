# Google Calendar v3 與 Google Tasks v1 API 基礎設施

本目錄固定 `google.golang.org/api v0.310.0` 的官方 Discovery 快照：Calendar v3 38 個方法、Tasks v1 14 個方法，共 52 項。

- `calendar-endpoints.json`、`tasks-endpoints.json`：原始碼對照、操作危險性、Scope、整合狀態；`integrated_offline` 表示 SDK Adapter + Use Case + 受控 CLI 已接線及離線 CI 通過，不代表真實帳號授權。
- `client_ready`：僅已具備 Client Factory；`integrated_offline`：實際 SDK Adapter + Application Port／Use Case + CLI 路由及契約測試；`staging_verified`：另外取得隔離測試帳號證據。
- 僅使用既有 `google.golang.org/api`，不改寫官方產生的 `calendar-gen.go`、`tasks-gen.go`。
- 讀取／操作 Google 使用者私有資源需要該使用者 OAuth 及所需 Scope；Cloud Run ADC 不能視為已授權個人 Calendar 或 Tasks。
- Calendar 支援 `watch` 和 `syncToken`，變更通知不包含完整變更資料；Tasks 沒有同等 API，只能以分頁／`updatedMin` 加上完整核對。
- Tasks `due` 只有日期，無法由 API 排定精確時刻；不能假裝 Google Tasks API 支援自訂重複規則。
- `calendars.clear`、`calendars.transferOwnership`、刪除及 ACL 修改等為高風險操作，必須明確確認且具適當權限；執行時不得自動重試未知結果。
- 不在公共 LINE webhook 暴露 Workspace 管理操作；不使用真實帳號做自動化測試。

來源：
- https://github.com/googleapis/google-api-go-client/blob/v0.310.0/calendar/v3/calendar-api.json
- https://github.com/googleapis/google-api-go-client/blob/v0.310.0/calendar/v3/calendar-gen.go
- https://github.com/googleapis/google-api-go-client/blob/v0.310.0/tasks/v1/tasks-api.json
- https://github.com/googleapis/google-api-go-client/blob/v0.310.0/tasks/v1/tasks-gen.go

## 離線整合進度（PR #25 已合併）

- Calendar v3：38/38；Tasks v1：14/14。合計 52/52 項 `integrated_offline`，以 `google.golang.org/api v0.310.0` 為準。
- `freebusy.query` 雖使用 HTTP POST，但**不修改資料**，Operator 應以 `--read` 操作；其他寫入與 Watch 註冊需明確 `--apply --confirm`。
- 這不等同完整個人 OAuth：目前僅提供本機操作者載入短期 Access Token 的封閉入口，未建立 Authorization Code／PKCE 回呼、Token Repository 或真實 Google 帳號驗收。

## Forms v1（離線整合候選，非真實帳號驗證）

- Forms 4、Responses 2、Watches 4，共 10 個原生 Typed SDK 方法，使用 `google.golang.org/api/forms/v1`。
- Forms 呼叫沿用 `WorkspaceRequest`／`cmd/ops workspace --api forms` 的本機測試帳號與操作確認規則。新增 `docs/google/forms-endpoints.json` 作為 Discovery v0.310.0 方法清單。
- 真正的 Forms Watches 通知需要額外 Pub/Sub Topic、通知授權、續約與去重；僅 SDK 離線整合不表示通知服務已部署。
- 本機 CLI 仍使用外部短期 Access Token；OAuth／Token Repository 的離線程式碼已在 `main`，但尚未掛載可信的 LINE 使用者登入與 Google 真實授權／撤銷流程。

## 使用者自助連結規格（整合自 PR #26）

- [LINE 使用者自行綁定 Google 帳號](user-self-service-linking.md)：明確規範連結轉傳防護、瀏覽器重新驗證 LINE 身分、Channel／使用者／Google sub 隔離與取消連結時停止同步。
- OAuth 核心、Firestore、TokenSource 及 Handler 仍是 opt-in 基礎元件；尚未透過已驗證 LINE Login／LIFF 接上公開路由，不能宣稱已能讓每位 LINE 使用者自行綁定。
- [OAuth 基礎實作](oauth-foundation.md) · [隔離 Staging 驗收](staging-acceptance.md)。
