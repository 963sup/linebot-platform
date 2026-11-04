# DESIGN.md — Go DDD + Hexagonal Architecture 設計契約

本文件是本專案唯一的架構規範，包含目錄樹、分層責任、依賴、效能、命名與驗收規則。避免以另一份文件重複維護相同契約。

本設計以 Go 1.27 的簡潔、明確、低 allocation 與可觀測性為優先，並將 Firestore、Cloud Tasks、Storage、Secret Manager、Google APIs/GenAI、LINE、OpenAI、MCP、HTTP、gRPC 與 OpenTelemetry 視為外部技術，隔離在 adapters/infrastructure。

## 0. 標準目錄樹

```text
project/
├── cmd/
│   ├── api/main.go                         # HTTP/gRPC composition root
│   ├── worker/main.go                      # async task / event consumer root
│   └── migrate/main.go                     # schema/migration entry point
├── internal/
│   ├── domain/
│   │   ├── shared/                         # immutable primitives only
│   │   └── <bounded-context>/
│   │       ├── aggregate.go
│   │       ├── entity.go
│   │       ├── value_object.go
│   │       ├── domain_service.go
│   │       ├── policy.go
│   │       ├── specification.go
│   │       ├── factory.go
│   │       ├── event.go
│   │       ├── repository.go               # domain repository contract, if needed
│   │       └── errors.go
│   ├── application/
│   │   └── <bounded-context>/
│   │       ├── commands/                   # write use cases
│   │       ├── queries/                    # read use cases
│   │       ├── dto.go
│   │       ├── ports.go                    # consumer-owned minimal interfaces
│   │       └── service.go                  # orchestration / transaction boundary
│   ├── ports/
│   │   ├── inbound/                        # external-to-application contracts
│   │   └── outbound/                       # app/domain-required capabilities
│   ├── adapters/
│   │   ├── inbound/{http,grpc,webhook,cli,messaging}/
│   │   └── outbound/
│   │       ├── {firestore,cloud_tasks,cloud_storage,secret_manager}/
│   │       ├── {google_api,google_genai,line,openai,mcp}/
│   │       ├── messaging/
│   │       └── in_memory/                  # tests and local development
│   ├── infrastructure/
│   │   ├── {config,http,grpc,clients,auth}/
│   │   ├── observability/                  # OpenTelemetry, logs, metrics
│   │   ├── concurrency/                    # bounded workers and shutdown
│   │   └── lifecycle/
│   └── bootstrap/{api,worker}.go           # dependency assembly only
├── api/{openapi,proto}/
├── migrations/
├── test/{contract,integration,performance,e2e}/
├── go.mod
├── go.sum
└── README.md
```

此目錄樹是抽象藍圖，不代表所有服務必須建立每一個目錄。只有存在實際責任時才新增 Package；避免空 Package、無行為 Wrapper 與預先建立的抽象。

## 1. 分層責任

### Domain

`internal/domain/<bounded-context>/` 擁有 Aggregate、Entity、Value Object、Domain Service、Policy、Specification、Factory、Domain Event、業務錯誤與必要的 repository contract。

Domain 必須是純 Go：不可 import Application、Ports 的具體實作、Adapter、Infrastructure、ORM、資料庫或任何第三方 SDK。Aggregate 自己維護不變條件；Value Object 建立時驗證並保持不可變。

### Application

`internal/application/<bounded-context>/` 以 commands/queries 組織 use cases，負責交易邊界、流程編排、DTO 與 application-level error mapping。它只能依賴 Domain 與最小的 consumer-owned ports。

Application 不得 import Firestore、gRPC、Google、LINE、OpenAI、MCP 或其他外部 SDK，也不得直接依賴具體 Adapter。

### Ports

`internal/ports/inbound/` 定義外部進入 Application 的契約；`internal/ports/outbound/` 定義 Application/Domain 所需的 repository、clock、publisher、task queue、identity 或 client capability。

Port 必須小型且由消費者需求定義，不 mirror 整份 SDK。若單一 use case 只需要一項能力，可將其 port 放在 `application/<context>/ports.go`，避免不必要的全域抽象。

### Adapters

Inbound adapters 負責 HTTP、gRPC、webhook、CLI、message protocol 的解析、驗證、mapping 與呼叫 use case；不得繞過 Application 修改 Domain 或資料庫。

Outbound adapters 實作 ports，負責 SDK request/response、序列化、錯誤映射、retry、timeout、pagination、batch 與 transaction。Persistence model 與 Domain model 分離，mapping 屬於 adapter。

### Infrastructure / Bootstrap

Infrastructure 管理 config、server、client lifecycle、OAuth/token source、gRPC keepalive/interceptors、OpenTelemetry、worker concurrency、readiness 與 graceful shutdown。它不承載業務政策。

`cmd/*/main.go` 與 `internal/bootstrap/` 只負責 composition root：建立長壽命 client、具體 adapter、use case 與依賴注入。

## 2. 唯一依賴方向

```text
inbound adapters ──▶ inbound ports ──▶ application ──▶ domain
                                          │              ▲
                                          └──▶ outbound ports

outbound adapters ──▶ outbound ports ─────┘
infrastructure ─────▶ adapters / external SDKs
cmd/bootstrap ───────▶ concrete implementations
```

強制規則：

- Domain 不得反向依賴 Application、Adapter、Infrastructure 或 SDK。
- Application 不得 import 具體 outbound adapter、client factory 或 SDK。
- Adapter 以 Go implicit interface 實作 port；port 不得 import adapter。
- Infrastructure 可依賴 SDK，但不可把 SDK 型別穿透到 Domain/Application。
- 跨 bounded context 必須經由明確的 port、DTO 或 domain event，不共享可變 domain model。
- 禁止循環 import；任何例外必須先重新設計邊界。

## 3. DDD 與高內聚

1. 以 bounded context 分隔語言、aggregate 與業務規則；`shared/` 只放真正穩定的 immutable primitives。
2. Repository contract 放在使用它的 Domain/Application 邊界，實作放 outbound adapter；不得同時維護重複 contract。
3. Domain Event 表示已發生的業務事實；可靠發布需透過 transaction + outbox 或等價機制。
4. Policy、Specification、Factory 只在有獨立業務行為時建立，不用分類名稱製造空 Package。
5. 簡單 CRUD 不強行套用 Aggregate、CQRS 或大量 DTO；架構重量應與 domain complexity 成比例。
6. `pkg/` 只有在確定有外部 module 使用時才建立；其餘程式放 `internal/`。

## 4. 技術 adapter 與效能契約

### Client lifecycle

- Firestore、Storage、Secret Manager、Cloud Tasks、Google API、GenAI、LINE、OpenAI 與 MCP client 在啟動期建立並重用；request handler 不重建 client。
- gRPC 重用 `ClientConn`，依服務設定 deadline、keepalive、interceptor 與 stream cancellation；每個 stream 必須由 context 取消或正常讀完以避免資源洩漏。
- 同一個 client 是否可共享，必須以 SDK thread-safety 與認證 scope 判定；帶使用者 OAuth 狀態的 TokenSource/HTTP client 不得跨使用者共享。

### Data access

- Firestore query 必須有明確的 limit、cursor pagination 與必要 index；避免 offset pagination 與 N+1 reads。
- 批次寫入使用 batch；需要條件一致性時使用 transaction，並以 context deadline 終止重試。
- Repository 對 Domain 隱藏 document schema、SDK snapshot 與 transport error；只回傳 domain/application 可理解的結果。

### Concurrency

- Worker 使用 bounded concurrency、backpressure、idempotency key 與 graceful shutdown。
- 不以無界 goroutine、全域 mutable state 或過度快取換取吞吐量。
- 所有外部操作傳遞 context；不將 context 儲存在長壽命 struct。

### Performance discipline

優先量測 database latency、network round trips、serialization、allocation、GC、lock contention、queue depth 與 cache hit rate，再進行優化。分層本身不是效能證明；避免無意義的 wrapper、DTO 轉換與重複 client 建立。

## 5. Go 實作規則

- 單一根 Go module；版本由 `go.mod` 管理，`go mod tidy` 保持依賴最小且可重現。
- Package 名稱小寫、簡短、不使用底線；檔案使用 `snake_case.go`；公開型別使用 `MixedCaps`。
- Constructor 顯式回傳具體型別與 error；不用 Service Locator、反射式 DI 或泛型大工廠。
- 官方 SDK 的 client、constructor、options、request/response 與 webhook 驗證直接重用；只有重複的 lifecycle/configuration 才建立 provider-specific factory。
- Factory 只建立技術 client，不執行 Domain Policy、授權決策或 Token 持久化。

## 6. 測試與驗收

- Domain：純 Go unit tests，驗證 aggregate、value object、不變條件與 domain events。
- Application：fake outbound ports 測試 use case、交易與錯誤流程。
- Adapter：contract tests；Firestore、gRPC、Cloud Tasks、外部 API 使用 integration tests。
- System：`test/e2e` 驗證 inbound adapter 到外部資源的完整流程。
- Performance：`test/performance` 使用 benchmark、load test 與 profiling，不能只由目錄結構推論高效能。

依變更風險執行 `gofmt`、`go test ./...`、`go vet ./...`、`go build ./...`；涉及 race、persistence、OAuth 或 worker concurrency 時，再加入 `go test -race ./...` 與整合測試。

若 gopls MCP 可用，可用於 definition、references、implementations 與 diagnostics；不可用時明確標示 `gopls MCP unavailable`，改用 Go CLI 工具，不得聲稱已使用 gopls。

## 7. 文件與工具設定

`AGENTS.md` 負責工作流程與角色路由；`.codex/config.toml` 只負責 MCP 連線設定，不是架構規範。任何文件更新不等於程式已重構、測試已通過或服務已部署；交付時必須分別回報文件變更、實際程式驗證與未驗證項目。

