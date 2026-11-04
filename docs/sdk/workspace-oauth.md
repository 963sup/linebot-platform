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

## Calendar／Tasks API 靜態整合（Draft PR #25）

- `docs/google/calendar-endpoints.json` 釘選 Calendar v3 38 項，`tasks-endpoints.json` 釘選 Tasks v1 14 項，SDK `v0.310.0`。
- `internal/adapter/outbound/google/api/{calendar,tasks}` 以明確 `switch` 呼叫原生 Go SDK，保留資源／方法、必要參數、常見查詢參數及 JSON body；不使用任意 HTTP Endpoint 或反射。
- `internal/application/usecases/workspace` 強制唯讀／寫入／危險操作的明確授權；不允許公有 Webhook 呼叫管理功能。
- `cmd/ops workspace` 僅限明確 `APP_ENV=local` 且無 `K_SERVICE` 的操作，從 owner-only `0600` 檔案載入**外部已核發的短期使用者 Access Token**，結果寫入新的 `0600` 私有檔案；這不是完整 OAuth Consent 或長期 Token 更新方案。
- OAuth Authorization Code + PKCE + 安全保管 Refresh Token、帳號驗證、多使用者隔離與撤銷流程仍未完成，不得直接部署為公共連結介面。
- 不需要執行真實 API 才能驗證 Adapter 的 HTTP Route；先執行 `make check`，再以獨立授權測試帳號驗證。
