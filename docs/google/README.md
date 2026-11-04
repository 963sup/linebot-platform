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

## 後續：使用者自行綁定 Google 帳號

本專案的 OAuth P2～P4 目標是**每位經驗證的 LINE 使用者自行連結自己的 Google 帳號**，不是管理員代綁或共用一組 Workspace 憑證。Calendar／Tasks 52 項離線 Adapter 不代表已具備使用者連結流程。完整安全與驗收規格參見 [User Self-Service Linking](user-self-service-linking.md)。
