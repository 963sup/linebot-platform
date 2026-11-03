# Google Workspace: Tasks, Calendar, Sheets and Drive

## Distinct services

- **Google Tasks**（Google 工作）是使用者的待辦清單與任務。原生 Go package：`google.golang.org/api/tasks/v1`。
- **Google Calendar**（Google 日曆）是行事曆與事件：`google.golang.org/api/calendar/v3`。
- **Google Sheets**（Google 試算表）是試算表、工作表與儲存格：`google.golang.org/api/sheets/v4`。
- **Google Drive** 是檔案與權限管理：`google.golang.org/api/drive/v3`。
- **Google Cloud Tasks** 是雲端背景工作佇列：`cloud.google.com/go/cloudtasks/apiv2`，並不是 Google Tasks。

上述四個 Workspace package 全部由現有 `google.golang.org/api` Go Module 提供。新增 Google Tasks **不需要額外 `go get`**。

## OAuth 與最小權限

`internal/infrastructure/google/api/client_factory.go` 的四個 Factory 接收**已完成使用者 OAuth 授權**的 `*http.Client`。它們不代替使用者同意、不保存或刷新 Token，也不會檢查輸入的 HTTP Client 是否真的擁有所需 Scope。

| 服務 | 唯讀 OAuth Scope | 寫入 Scope（按需） |
|---|---|---|
| Google Tasks | `https://www.googleapis.com/auth/tasks.readonly` | `https://www.googleapis.com/auth/tasks` |
| Google Calendar Events | `https://www.googleapis.com/auth/calendar.events.readonly` | `https://www.googleapis.com/auth/calendar.events` |
| Google Sheets | `https://www.googleapis.com/auth/spreadsheets.readonly` | `https://www.googleapis.com/auth/spreadsheets` |
| Google Drive | `https://www.googleapis.com/auth/drive.metadata.readonly`（只讀 Metadata） | `https://www.googleapis.com/auth/drive.file`（適用的檔案範圍） |

需依業務要求請求最小 Scope。使用者 OAuth 與一般 GCP ADC/Workload Identity 不是同一種授權；不能直接假設 Cloud Run 服務帳號能讀取個人的待辦事項與日曆。

當外部流程已合法取得特定使用者的 `oauth2.TokenSource` 時，整合方式例如：

```go
// source 來自已授權且儲存安全的使用者 OAuth 流程。
authorizedHTTP := oauth2.NewClient(ctx, source)
tasksClient, err := api.NewTasks(ctx, authorizedHTTP)
// 先檢查 err，再使用 tasksClient.Tasklists.List().Do() 等原生 API。
calendarClient, err := api.NewCalendar(ctx, authorizedHTTP)
// 先檢查 err，再使用 calendarClient.Events.List("primary").Do()。
```

這是使用示意，不是可直接執行的完整程式。`source` 的取得、同意流程、Token 加密儲存與輪替、使用者隔離等仍待業務功能實作；不可將 Token 加入版控、Log 或 CI。

## Client Factory 與驗證狀態

- `api.NewTasks`、`api.NewCalendar`、`api.NewSheets`、`api.NewDrive` 會回傳各自的官方原生 `*Service`，完整官方 API 保留可呼叫。
- 測試使用 `httptest.Server` 與模擬的 OAuth Bearer Token，驗證 Tasks `Tasklists.List` 和 Calendar `Events.List`，不接觸真實使用者資料。
- Catalog 的 `client_ready` 表示工廠可編譯、有離線測試，**不代表** 已完成使用者 OAuth、Webhook 業務接線或 Staging 驗證。

官方來源：
- https://developers.google.com/workspace/tasks/quickstart/go
- https://developers.google.com/workspace/tasks/auth
- https://developers.google.com/calendar/api/auth
- https://pkg.go.dev/google.golang.org/api/tasks/v1
- https://pkg.go.dev/google.golang.org/api/calendar/v3
