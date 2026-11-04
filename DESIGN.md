# DESIGN.md — Go DDD + Hexagonal Architecture 設計契約

> 適用範圍：本儲存庫所有 Go 程式、SDK 整合、CI 與未來擴充。這是架構規範，不代表目前所有功能已完成。維持 Pragmatic DDD、Lightweight Hexagonal Architecture 與 Idiomatic Go；不為形式製造額外 Wrapper 或 Package。

## Codex 必讀：本檔為架構標靶文件與 gopls MCP 工作契約

**開始任何 Go 開發、SDK 整合、DDD / Hexagonal 調整、跨 Package Rename 或 Review 前，先讀本檔，再檢查實際程式與 `.codex/config.toml`。本檔是架構決策的標靶（source of truth），不是自動載入 MCP 的設定檔。**

本儲存庫已在 `.codex/config.toml` 宣告：

```toml
[mcp_servers.gopls]
command = "bash"
args = ["-c", 'exec "$HOME/.local/bin/gopls" mcp']
enabled = true
required = false
startup_timeout_sec = 30
tool_timeout_sec = 60
```

**Codex 操作規則（優先使用 `mcp_servers.gopls`，依變更風險選擇工具）：**

| 任務類型 | gopls 語意檢查 | 必要驗證 |
| --- | --- | --- |
| 純 Markdown / 文件 | 不強制使用 gopls | 連結、章節引用、術語與圖示一致性 |
| 單一函式實作 | 需要時查 definition，修改後 diagnostics | 受影響 Package 測試 |
| Interface / Factory / Adapter 簽名修改 | references、implementations、diagnostics | 相關 Package 測試、build、架構依賴 |
| 跨 Package Rename / 搬移 | 先查全部 references、implementations；可用時使用語意 Rename | 全部相關引用、build、架構測試 |
| `go.mod` / SDK 升級 | `go list`、`go mod tidy`、官方 API 文件；gopls 輔助 | `go.mod` / `go.sum` diff、build、相關測試 |
| OAuth / Firestore / 安全不變量 | gopls 查受影響呼叫鏈及 diagnostics | 安全、相容性、交易與整合測試（環境可用時） |

**工具 Gate**：使用前確認本工作階段 gopls MCP 實際可用、工作區正確；`enabled = true` 並不代表已連線。工具名稱以實際 MCP 提供者為準，不假設每項功能必然存在。若不可用，明確回報 `gopls MCP unavailable`，使用 `go list`、`go doc`、`go test`、`go vet`、`gofmt` 等替代；標註未完成的語意引用驗證，**不得聲稱已使用 gopls**。

**修改 Gate**：跨 Package Rename / Interface 變更先確認受影響引用、相容性與第 3 節依賴方向；優先使用可用的語意重構能力。僅做最小必要修改，不自行改動全域工具設定、未授權部署或合併。

**完成 Gate**：報告實際使用的工具、修改的 Symbols / Packages、依賴方向、測試結果及未驗證事項。新增編譯錯誤、循環依賴或破壞安全不變量時不得宣稱完成；無法安全修復則停止並回報阻礙。

**文件分工**：`DESIGN.md` 定義架構與命名契約；`AGENTS.md` 負責 Codex 角色路由；`.codex/config.toml` 負責 MCP 連線。若指示衝突，先辨識是架構、路由還是工具連線問題，回報衝突並尋求必要確認，不得靜默覆蓋其他檔案。僅修改本文件不會強制 Codex 自動讀取它；任務入口應明確引用 `DESIGN.md`，或由既有 AGENTS 路由引導。

## 1. 不可退化原則

1. **DDD 邊界**：Domain 管理業務不變條件、Aggregate、Entity、Value Object、Policy、Specification、Factory、Repository 契約及 Domain Event；不依賴外部 SDK。
2. **依賴方向**：Source-code imports 朝向 Domain / Application 契約；具體 SDK 與資料庫實作位於外側。
3. **SDK 優先重用**：官方 SDK 已提供的 Client、Constructor、Options、HTTP Transport、API Request/Response、Webhook 驗證與 OAuth 元件，直接重用；只有跨越業務邊界、必要的安全控制或重複初始化才新增程式。
4. **Go Idioms**：單一 Go Module、明確建構、組合優於繼承、小型消費者導向介面、context 傳遞、顯式錯誤；避免全域 Service Locator、DI Framework 與沒有行為的 Wrapper。
5. **最小必要變更**：保留既有加密資料、Firestore 文件識別、Token Refresh CAS、OAuth 安全規則與公開 API 契約。重構必須以測試證明相容。
6. **保留 Identity DDD 分類**：不得為了減少 Package 數而擅自合併下述八個目錄。

## 2. Module、Package 與 SDK：三個不同概念

- 根目錄 `go.mod` 定義 **一個 Go Module**（`github.com/963sup/linebot-platform`）及版本依賴；`go.sum` 保存模組校驗資訊。
- `import` 使用 **Package 路徑**，例如 `google.golang.org/api/calendar/v3`；多個 Package 可以來自同一個 Module `google.golang.org/api`。
- SDK Package 是第三方已存在的程式庫，**不是**專案需要另外建立的中介層。
- `internal/` 包含專案私有 Package；`cmd/` 是可執行入口。不要為每個 Domain 或 SDK 建立第二個 `go.mod`，除非確實需要獨立發布與版本生命週期。
- `go mod tidy`、`go list -m all`、`go list -deps ./...` 用於核對依賴。直接／間接依賴的分類依 Go Module Graph 與實際 Package imports 判定；不可把未使用的 SDK 宣稱為正在執行的能力。
- `docs/sdk/catalog.toml` 可記錄「使用中／已核准預留／規劃中」，與 `go.mod` 的現有直接依賴分開理解。現有 retention gate 的預留策略在另行授權修改前必須維持；未來若調整，應連同 CI 與 Catalog 一起修改，不可暗中刪除依賴。

### 已使用或預留的 Module 與 Package 對應

| Go Module | 官方 Package 範例 | Factory / 使用位置 |
| --- | --- | --- |
| `github.com/line/line-bot-sdk-go/v8` | `linebot/messaging_api`, `liff`, `channel_access_token`, `insight`, `manage_audience`, `module`, `module_attach`, `shop` | `internal/infrastructure/line/client_factory.go` 及必要的授權元件 |
| 同上 | `linebot/webhook`, `linebot/httphandler` | Inbound HTTP Adapter；不是 Outbound Client Factory |
| 同上 | `linebot` | 依實際 SDK 功能直接 import；不為根 Package 創造空 Wrapper |
| `golang.org/x/oauth2` | `oauth2`, `oauth2/google` | OAuth Flow / TokenSource，使用者授權 HTTP Client |
| `google.golang.org/api` | `idtoken`, `oauth2/v2`, `option`, `calendar/v3`, `tasks/v1`, `forms/v1` | Identity Verifier / Google Client Factory |
| `cloud.google.com/go/firestore` | Firestore SDK | Firestore Adapter / Client 建立 |
| `cloud.google.com/go/cloudtasks` | Cloud Tasks SDK | Cloud Tasks Adapter / Client 建立 |

同一 Provider 的不同 API Family 可能需要不同 Constructor、認證選項與權限；**統一入口不等於強迫所有 API 具有相同簽章**。

### 2.1 從 go.mod 到 client_factory.go：最短依賴路徑（Context7 / Go 官方查證）

**結論：go.mod 不會直接呼叫 Factory。** `go.mod` 只記錄 Module 與版本；Factory 的 `import` 才是對官方 SDK Package 的編譯期依賴。不要再建立專案自有的「SDK Package Layer」。

```text
go.mod: require github.com/line/line-bot-sdk-go/v8 <go.mod 中的實際版本>
    └── infrastructure/line/client_factory.go
          import "github.com/line/line-bot-sdk-go/v8/linebot/messaging_api"
          return messaging_api.NewMessagingApiAPI(token, options...)

go.mod: require google.golang.org/api <go.mod 中的實際版本>
    └── infrastructure/google/api/client_factory.go
          import "google.golang.org/api/calendar/v3"
          return calendar.NewService(ctx, options...)

go.mod: require golang.org/x/oauth2 <go.mod 中的實際版本>
    └── OAuth 授權元件
          import "golang.org/x/oauth2"
          建立使用者專屬 TokenSource / 已授權 HTTP Client
```

上述版本占位符必須以當前 `go.mod` 為準；文件不複製可能過期的版本號。這裡是**版本宣告 → import → 官方 Constructor** 的關係，不是 Runtime 呼叫堆疊。

### Factory 選擇規則

**Factory 的唯一權威決策程序見第 5.1 節。** 本節只說明 Module → Package → Constructor 的直接依賴，不另行定義第二套判斷標準。

### 官方 SDK Constructor 直接重用範例

LINE（同一個 Provider Factory 內可提供 Messaging、Blob、Insight、Audience 等**必要**的建立函式）：

```go
// infrastructure/line/client_factory.go
func NewMessagingClient(token string, httpClient *http.Client) (*messaging_api.MessagingApiAPI, error) {
    if strings.TrimSpace(token) == "" || httpClient == nil {
        return nil, errors.New("LINE token and HTTP client required")
    }
    return messaging_api.NewMessagingApiAPI(token, messaging_api.WithHTTPClient(httpClient))
}
```

Google（以**已授權**的 HTTP Client 作為輸入；其他服務同理）：

```go
// infrastructure/google/api/client_factory.go
func NewCalendar(ctx context.Context, authorizedHTTP *http.Client) (*calendar.Service, error) {
    if ctx == nil || authorizedHTTP == nil {
        return nil, errors.New("context and authorized HTTP client required")
    }
    if err := ctx.Err(); err != nil {
        return nil, err
    }
    return calendar.NewService(ctx, option.WithHTTPClient(authorizedHTTP))
}
```

上面範例只展示必要函式本體，實際檔案仍需正常 import。Google 的 `authorizedHTTP` 必須由該使用者的 OAuth TokenSource 建立（例如 `oauth2.NewClient(ctx, source)`）；不可用未授權的普通 HTTP Client 取代。**同一個已授權 HTTP Client 可以供該使用者的多個 Google Service 使用，不得跨使用者共用認證狀態。**

### 依賴方向的精確解讀

依賴關係的**唯一規範圖、箭頭語意、import 矩陣與建立／執行流程**統一參考第 3 節，不再在其他章節維護第二份可能衝突的架構圖。

### 針對本 PR 的最小重構邊界

- 保留 `internal/domain/identity/` 的八個 DDD 分類，不以這次 Factory 收斂為理由更動。
- 現有 `internal/infrastructure/line/client_factory.go` 已有 Messaging / Blob Constructor；可評估把 `audience_client.go`、`insight_client.go` 等重複初始化**整合到同一 Package / Factory 入口**，但不強制合併檔案。
- 現有 `internal/infrastructure/google/api/client_factory.go` 已有 Calendar、Tasks、Forms、Drive、Sheets Constructor；**優先直接重用**，而不是再建立一個泛用 `ClientFactory` Struct 包住同一批函式。
- 不要同時維護 `NewCalendar`、`Factory.Calendar`、`NewWorkspaceCalendarClient` 三種功能等價的入口。
- `bootstrap/workspace.go` 仍可負責組裝；若要減碼，先量測並消除重複建立與配置，不是把組裝責任搬到 Domain 或 Factory。
- 本文件只建立標準；是否達到 40% 減碼率，需以重構前後有效程式行數驗證。

**查證依據（Context7）：** Go 官方 `go.dev/doc/code`、`go.dev/doc/modules/layout`、`go.dev/doc/modules/managing-dependencies` 說明 Module 與 Package 的關係及 `go mod tidy`；LINE 官方 `line-bot-sdk-go` 文件展示 `NewMessagingApiAPI` 與 `WithHTTPClient`。Google Factory 實例則依本儲存庫現有的官方 SDK `NewService` 呼叫整理，未聲稱 Context7 找到完整 Google Go Client 條目。

## 3. 依賴方向：唯一規範圖與實作契約

下圖是本專案的**概念關係圖**，沿用既定的節點與標籤；它同時呈現版本管理、原始碼 import、物件建立與 Port 實作關係，因此**不可把所有箭頭當成同一種 import 或執行期呼叫**。

```mermaid
flowchart TB
    M["go.mod<br/>管理 Module 版本"]
    SDK["官方 LINE / Google SDK"]
    F["client_factory.go<br/>建立官方 Client"]
    AD["Outbound Adapter<br/>必要的業務轉換"]
    P["Application Port<br/>小型 Interface"]
    UC["Application Use Case"]
    D["DDD Domain"]
    CMD["cmd / Composition Root"]

    M -.->|"提供依賴版本"| SDK
    F -->|"import"| SDK
    CMD -->|"建立"| F
    CMD -->|"組裝"| AD
    CMD -->|"注入"| UC
    AD -.->|"按需建立（可選）"| F
    AD -.->|"實作"| P
    UC -->|"依賴"| P
    UC -->|"依賴"| D

    classDef core fill:#dceafe,stroke:#7398d4,color:#183b65
    class P,UC,D core
```

### 3.1 圖中每一種箭頭的精確語意

| 圖中關係 | 實際語意 | Go 程式碼依賴方向 |
| --- | --- | --- |
| `go.mod → SDK` | Module 版本解析，不是 runtime 呼叫 | `go.mod` 的 `require` 宣告 |
| `Factory → SDK` | Factory 直接 import 官方 Package 並呼叫 Constructor | `infrastructure → SDK` |
| `cmd → Factory` | Composition Root 建立 Client | `cmd → infrastructure` |
| `cmd → Adapter / Use Case` | 手動 Constructor Injection | `cmd → adapter / application` |
| `Adapter → Factory` | **可選**：Adapter 需要延遲建立使用者專屬 Client 時呼叫 Factory | `adapter → infrastructure`；一般情況由 `cmd` 建好 SDK Client 再注入 Adapter |
| `Adapter → Port` | Adapter 實作 Port 的契約（圖中的虛線不是 runtime 呼叫） | Adapter 可 import Port / Domain 型別；Port **不得** import Adapter |
| `Use Case → Port` | Use Case 呼叫注入的 Port | `application/usecase → application/port` |
| `Use Case → Domain` | Use Case 執行領域模型與規則 | `application → domain` |

**重要：** 上圖的 `AD ⇢ F` 是可選關係，不是強制多一層。若 Factory 只在啟動時使用，Adapter 只持有已注入的官方 SDK Client。若 Factory 用於每位使用者 OAuth Client 的按需建立，Adapter 可以持有一個具體 Factory 或所需的最小建構能力；但 Domain / Application 不能 import Factory。

### 3.2 強制 import 依賴矩陣

```text
cmd/{webhook,worker,ops} [Composition Root]
    ├── application/usecases
    ├── adapter/inbound
    ├── adapter/outbound
    └── infrastructure/{line,google,...}

adapter/inbound ───────────> application/usecases / inbound ports
application/usecases ──────> domain + application outbound ports
adapter/outbound ──────────> application ports + domain (必要時) + SDK / infrastructure
infrastructure ────────────> 官方 SDK + Go 標準函式庫 + 技術元件
domain/identity/* ────────> domain/identity/* + Go 標準函式庫
```

- **禁止** Domain import Application、Adapter、Infrastructure、LINE / Google / Firestore SDK。
- **禁止** Application Use Case import 具體 Outbound Adapter、Client Factory、Infrastructure SDK。
- Port 是由 Application 或 Domain 擁有的契約；Outbound Adapter 實作 Port，Port 不反向 import Adapter。
- Inbound Adapter 驗證外部輸入並呼叫 Use Case；不可繞過 Application 執行業務授權或直接改寫 Domain 持久化。
- Infrastructure Factory 只處理官方 SDK 建立與技術設定；不查詢 Domain Policy、不持久化 OAuth Token、不做使用者授權決策。
- `cmd/` 是 Composition Root；`internal/bootstrap/` 僅在多個入口確實共用組裝時保留。兩者都不得承擔業務政策。
- Domain 八個子 Package 需避免循環 import；保留目錄分類不代表每個流程必須穿過所有分類。
- **Identity Package DAG**：`value-objects` 僅依賴標準函式庫；`entities` 可依賴 `value-objects`；`aggregates` 可依賴 `entities`、`value-objects`、`events`；`policies`、`specifications` 可依賴 `aggregates` 與基礎模型，但 Aggregate 不得反向 import 它們；`factories` 可依賴 `aggregates`、`policies`、`specifications`；`repositories` 以 Aggregate / Value Object 定義契約，不得依賴具體 Adapter；`events` 只依賴必要的 Value Objects。任何例外先驗證 DAG 無環並記錄理由。
- Adapter 使用 SDK 的 Request / Response 型別時可以直接 import 官方 SDK；不為了「單一 import 位置」製造一比一 Facade。

### 3.3 建立期與執行期分離

```text
版本解析：go.mod ──require──> 官方 SDK Modules
原始碼：Factory ──import──> 官方 SDK Packages

建立期：cmd / 可選 bootstrap ──> Factory / 官方 Constructor
                              └──> Adapter ──注入──> Use Case

執行期：Webhook / HTTP → Inbound Adapter → Use Case
        → Domain 規則 + Port → 已注入的 Outbound Adapter → SDK Client → 外部 API
```

Port 在執行期是介面分派的契約，**不是**一個獨立的中繼服務。Factory 通常不在執行期請求路徑中；使用者專屬 Google Client 需要延遲建立時才例外。

## 4. Domain / Identity：固定目錄與命名

```text
internal/domain/identity/
├── aggregates/account_link.go
├── entities/google_connection.go
├── value-objects/
│   ├── line_user_id.go
│   ├── google_account_id.go
│   ├── connection_id.go
│   ├── oauth_scope.go
│   └── connection_status.go
├── policies/
│   ├── linking_policy.go
│   └── unlinking_policy.go
├── specifications/connection_eligibility.go
├── factories/connection_factory.go
├── repositories/connection_repository.go
└── events/
    ├── account_linked.go
    └── account_unlinked.go
```

- 目錄名稱保留 `value-objects/`；Go package 識別字為 `valueobjects`，import 時可用別名。
- Aggregate Root `AccountLink` 控制一致性與狀態轉換；`GoogleConnection` 是 Aggregate 內的 Entity，不得形成平行的授權真相。
- Value Object 驗證資料、封裝不變性；不得依賴 OAuth SDK。
- Policy / Specification 封裝可測試的純業務規則；Factory 建立／還原有效模型。**分類目錄必須保留，但禁止製造重複的業務判斷或無意義轉接**。
- Repository Interface 定義 Domain 所需能力，具體 Firestore 實作在 Adapter；可另用 Application consumer-owned 小介面處理 Token Store 等技術需求。
- Domain Event 在狀態變更時記錄；只有持久化成功後才能對外發布。若需要可靠交付，必須另行設計 Outbox。
- LINE User ID 必須來自已驗證身分；Google Account ID 應使用已驗證的 Google `sub`，不是 Email。

## 5. Client Factory 標準：直接重用官方 SDK

### 5.1 Factory 唯一決策程序與命名

**依序判定，不允許為同一能力建立平行入口：**

1. **既有 Provider Factory 已提供 Client？** 直接重用；不可新增等價 `NewX`、`Factory.X` 或 Wrapper。
2. **官方 Constructor 足夠且只使用一次、無共用設定？** 在 Composition Root 直接呼叫官方 Constructor，不建立空 Factory。
3. **同 Provider 存在多處重複的 Transport / Timeout / Options / Token 檢查或 Client 建立策略？** 在現有 Provider Package 的 `client_factory.go` 提供具體 `NewX` 函式；不強制建立 `ClientFactory` Struct。
4. **使用者專屬 Google OAuth？** Application / Domain 先完成授權判斷；OAuth 元件提供該使用者 TokenSource；Factory 按需建立已授權 Client，不能跨使用者快取認證狀態。
5. **業務需要隔離 SDK？** Application 定義最小消費者導向 Port，Adapter 使用 SDK Client 實作；不鏡像整份 SDK DTO / API。
6. **都不符合？** 不增加 Factory、Port、Wrapper 或 Mapper。

**固定位置與責任：**

- `internal/infrastructure/line/client_factory.go`：僅集中 LINE Messaging、Blob、LIFF、Insight、Audience、Module、ModuleAttach、Shop 等**實際需要**的 Client 建立；Token 管理由授權元件負責。
- `internal/infrastructure/google/api/client_factory.go`：集中 Google UserInfo、Calendar、Tasks、Forms 及實際使用的 Drive / Sheets 官方 `NewService`。
- Firestore、Cloud Tasks 等只有共用初始化或生命週期需求時才引入 Factory；否則直接使用官方 Constructor。
- Factory 回傳官方 Client 具體型別及 error，不建立 SDK 一比一的 Manager、Provider、Builder、Facade；不同 SDK Constructor 的必要差異必須保留。
- Factory 不執行 Domain Policy、不查使用者授權、不持久化 Token；不要使用反射、Service Locator、泛型大工廠或字串驅動 Client 選擇。
- `client_factory.go` 是**檔名慣例**，不是必須宣告 Factory Struct 或讓 Adapter 一律依賴 Factory。第 3 節圖中的 Adapter ⇢ Factory 只代表延遲建構的可選依賴。

### 5.2 LINE 依賴例

```text
go.mod require github.com/line/line-bot-sdk-go/v8
infrastructure/line/client_factory.go --import--> linebot/messaging_api
cmd --建立 Client 並注入--> adapter/outbound/line
adapter/outbound/line --實作--> application outbound port
```

Webhook / httphandler：`SDK → adapter/inbound/http/line → Application`；不得硬塞入 Outbound Factory。

### 5.3 Google 依賴例與使用者隔離

```text
go.mod require golang.org/x/oauth2 + google.golang.org/api
已驗證 LINE principal / Google OAuth state、PKCE、nonce → 使用者專屬 TokenSource
infrastructure/google/api/client_factory.go --import--> calendar/v3、option
Factory 使用已授權 HTTP Client 呼叫 calendar.NewService(...)
cmd / 授權後的 Adapter --建立與注入--> calendar outbound adapter
calendar outbound adapter --實作--> application outbound port
```

Tasks / Forms / UserInfo 使用各自官方 `NewService`；不需另外建立一層「SDK Package」。

**認證安全**：
- Google 使用者的 Calendar / Tasks / Forms 不得使用 Cloud Run Service Account / ADC 取代使用者 OAuth。
- `option.WithHTTPClient(client)` 必須收到**已授權**的 HTTP Client；若使用 `option.WithTokenSource(source)`，不得又傳入會覆蓋認證行為的普通 HTTP Client。
- 每個使用者的 TokenSource / 認證 HTTP Client 必須隔離；可以共用安全的底層網路資源，但不可跨使用者共用授權狀態。
- 授權 scopes、owner/subject、active 狀態由 Application + Domain 驗證；Factory 只接收已授權的建立參數。
- Token Refresh 必須安全持久化、保留 CAS/transaction 保護，避免多個 Cloud Run 實例競爭。
- 不在 Domain、Log、URL 中保存 Access Token / Refresh Token；Firestore 既有加密 JSON、文件 ID、版本欄位與相容性契約不可擅改。

## 6. 組裝入口與命名一致性

- `cmd/<process>/main.go` 是 Composition Root；多個入口確有共用組裝時才保留 `internal/bootstrap/`。兩者都不承擔業務規則。
- **命名優先順序**：現有正確的公開 API → gopls references / implementations → 相容性要求 → 新命名規則；不因純粹改名而破壞使用端。
- SDK Client 建立採第 5.1 節的 `NewX` 與 `client_factory.go` 規則；Domain Factory 固定於 `domain/<context>/factories/`，不可 import SDK。
- Application Port 依業務能力命名（例如 `CalendarEventCreator`），定義在消費端，保持小型 Interface；不直接鏡像 SDK API。
- Adapter 使用 `adapter/outbound/<provider>/<capability>.go` 或既有業務檔名；只負責必要的 SDK 呼叫、錯誤映射與業務轉換。
- Repository 的契約位於 Domain / Application 所需邊界，具體持久化實作位於 Outbound Adapter；技術 Token Store 不應產生第二份業務真相。
- Policy / Specification 只放純業務判斷；`Manager`、`Provider`、`Service` 僅在具有真實獨立職責時使用，不以名稱本身判斷好壞。
- Go Package 名稱簡短、小寫且無底線；檔案使用 `snake_case.go`；型別使用 `MixedCaps`；Context 在每次操作時傳遞，不儲存在長壽命物件。
- 變更公開符號或移動 Package 前先查 references，避免同一能力存在多個 Constructor、Mapper、DTO 或 Wrapper。

## 7. Codex 標準化工作流程與 CI 驗收

**連續工作契約**：讀取本檔 → 按任務風險使用 gopls（見開頭表格）→ 確認 `go.mod` 與官方 SDK → 依第 5.1 節決定 Factory → 依第 3、4 節檢查依賴方向 → 最小修改 → 語意診斷與測試 → 回報結果。

| Gate | 完成條件 | 未通過時 |
| --- | --- | --- |
| 入口 | 需求、受影響 Package、MCP 可用性已確認 | MCP 不可用採明示降級，不假稱已檢查 |
| 設計 | 官方 Constructor / 既有 Factory 已查核，無重複入口 | 暫停新增 Wrapper，先釐清權威實作 |
| 邊界 | Domain / Application 不直接 import SDK，Identity DAG 無循環 | 修正設計後再修改程式 |
| 實作 | 最小必要變更、引用與相容性已確認 | 未確認跨 Package 引用時停止危險 Rename |
| 安全 | OAuth state / PKCE / nonce、Webhook 驗簽、Token Refresh CAS、Firestore 相容性未退化 | 沒有證據不得宣稱可部署 |
| 驗證 | gopls diagnostics（適用時）、`gofmt`、相關 `go test` 通過 | 修復或明確列出失敗與阻礙 |
| 交付 | 說明修改、命名、依賴、測試與未驗證項目 | 不宣稱已完成未執行的驗證 |

**風險分級測試：**文件修改只需文件一致性檢查；單一函式至少執行相關 Package 測試；跨 Package / Interface / SDK 變更加跑 `go vet ./...`、`go test ./...`、`go build ./...` 與架構檢查；涉及 concurrency、OAuth、資料持久化時依風險加跑 `go test -race ./...`、安全／整合測試。僅在依賴變更或需確認 Module Graph 時執行 `go mod tidy`，檢查 `go.mod`、`go.sum` diff，遵守既有 SDK retention gate。

**減碼驗證：**若以 40% 為目標，只計算 SDK 初始化與組裝的專案自有有效程式行數；排除 SDK 原始碼、測試、註解、空行與 Domain 業務規則，不能未量測就宣稱達標。

## 8. 文件一致性與決策優先級

**單一規範來源**：第 3 節是依賴方向唯一規範，第 4 節是 Identity 目錄與內部 DAG，第 5.1 節是 Factory 唯一決策規則，第 6 節是命名規範，第 7 節是執行與驗收 Gate。其他章節只提供範例，不建立競爭規則。

**衝突處理**：先判斷架構規則、既有公開 API 相容性與安全不變量；無法同時滿足時回報具體衝突，不得靜默更改 `AGENTS.md`、`.codex/config.toml`、CI 或資料格式。此文件不能保證 Codex 自動讀取或 MCP 自動連線；執行時須確認。

## 9. 參考與驗證界限

- gopls 官方文件：https://go.dev/gopls
- Go 官方 Module Layout：https://go.dev/doc/modules/layout
- Go Module Reference：https://go.dev/ref/mod
- Go Code Review Comments：https://go.dev/wiki/CodeReviewComments
- LINE SDK：https://github.com/line/line-bot-sdk-go
- Google API Go Client：https://github.com/googleapis/google-api-go-client

Go 官方沒有強制規定 DDD／Hexagonal 的唯一目錄結構；本文件是本儲存庫採用的架構契約。文件更新不等於已完成程式重構、Firestore Emulator 驗證、OAuth 外部驗證或 Cloud Run 部署。
