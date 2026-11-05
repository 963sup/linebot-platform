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
| Google Forms | `https://www.googleapis.com/auth/forms.body.readonly`、`https://www.googleapis.com/auth/forms.responses.readonly` | `https://www.googleapis.com/auth/forms.body`（表單編輯／發布） |
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

## Calendar／Tasks API 靜態整合（PR #25 已合併）

- `docs/google/calendar-endpoints.json` 釘選 Calendar v3 38 項，`tasks-endpoints.json` 釘選 Tasks v1 14 項，SDK `v0.310.0`。
- `internal/adapter/outbound/google/api/{calendar,tasks}` 以明確 `switch` 呼叫原生 Go SDK，保留資源／方法、必要參數、常見查詢參數及 JSON body；不使用任意 HTTP Endpoint 或反射。
- `internal/application/usecases/workspace-automation` 強制唯讀／寫入／危險操作的明確授權；不允許公有 Webhook 呼叫管理功能。
- `cmd/ops workspace` 僅限明確 `APP_ENV=local` 且無 `K_SERVICE` 的操作，從 owner-only `0600` 檔案載入**外部已核發的短期使用者 Access Token**，結果寫入新的 `0600` 私有檔案；這不是完整 OAuth Consent 或長期 Token 更新方案。
- Draft PR #28 的 Identity Domain 已與 OAuth Use Case、Firestore 加密 Token Repository 與 TokenSource 接線，並收斂重複的 `OAuthConnection` DTO。一次性 State、PKCE、撤銷與 Token CAS 的離線檢查已納入 CI；**LINE 使用者瀏覽器再驗證、OAuth 路由掛載與真實 Google 授權／IAM 驗收仍未完成**，不得直接部署為公共綁定介面。
- 不需要執行真實 API 才能驗證 Adapter 的 HTTP Route；先執行 `make check`，再以獨立授權測試帳號驗證。

> **本次整合狀態**：Calendar／Tasks 52 項方法已完成離線接線；Drive／Sheets 仍僅具 Client Factory，因此模組共用 Catalog 的 `client_ready` 不能代表四個服務都未整合，也不能代表完整 OAuth 已完成。詳見 `docs/google/README.md`。

## Forms API（新增）

Forms v1 的 10 個官方方法沿用現有 Workspace Typed SDK 及操作確認規則。使用 `NewForms(ctx, authorizedHTTP)`，不複製另一套 OAuth Transport 或直接呼叫 REST HTTP。Watches 只先支援離線 API 契約；真正 Pub/Sub 訂閱、Renew、IAM 及 Staging 另行驗證。

## 自助綁定規格（來源 PR #26，整合至 PR #27）

[Google 自助綁定安全規格](../google/user-self-service-linking.md) 是 P2–P4 的驗收契約，包含可轉傳連結防護、LINE Login／LIFF 再驗證、Channel／LINE 使用者／Google `sub` 隔離、Token 綁定與每位使用者同步。帳號／Firestore Client 已經存在不等於上述跨服務授權已得到驗證。

## Identity 全鏈依賴／Go Module 對照（PR #28）

`cmd/` 與 `bootstrap/` 只有組裝與啟動責任；`internal/application/usecases/identity` 驗證 Domain ID、連結狀態與 Scope，`workspace-automation` 擁有 Workspace 操作授權。Firestore Adapter 實作 Identity 所需 contract，Google OAuth Adapter 與 Infrastructure 提供使用者授權 TokenSource。

| Module（`go.mod`） | 負責的 Import Packages | 現況 |
| --- | --- | --- |
| `golang.org/x/oauth2` | `oauth2`、`oauth2/google` | Authorization Code、PKCE、使用者專屬 TokenSource |
| `google.golang.org/api` | `idtoken`、`oauth2/v2`、`option`、`calendar/v3`、`tasks/v1`、`forms/v1` | 身分驗證／Workspace SDK；`oauth2/v2` 的 UserInfo 業務路由仍為可選能力，不能當作已接入 |
| `cloud.google.com/go/firestore` | `firestore` | 加密 OAuth State／Connection、CAS、交易 |
| `cloud.google.com/go/secretmanager` | `secretmanager/apiv1` | 外部金鑰取得能力；實際 IAM／Staging 尚未驗證 |

此表以 Module 為單位；`go.mod` 不應為上述每個 Package 新增一條獨立的 require。由於目前兩個核心 OAuth Module 仍直接使用，沒有理由刪除或重新選版本。CI 的 `go mod tidy` 與 `git diff --exit-code -- go.mod go.sum` 驗證鎖定檔維持一致。
