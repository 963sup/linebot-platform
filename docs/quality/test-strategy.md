# 測試策略

## Configuration contract and observability

- `go run ./cmd/configcheck --check` compares the generated [Runtime Configuration](../operations/configuration.md) with Go metadata, rejects unknown/duplicate keys and non-empty Secret values, **and parses `.env.example` with the same webhook configuration validator**. Invalid Port, Timeout, logging or deployment profile fails CI without external requests.
- Bootstrap role/profile tests cover local safe defaults; staging/production durable-only; Cloud Run `K_SERVICE` fail-closed; Worker vs Webhook secret isolation; explicit Worker `LINE_EVENT_MODE=sync` rejection; invalid LOG_LEVEL/LOG_FORMAT, Port and Timeout.
- Logger/HTTP telemetry tests ensure no raw URL/path, query, body, access token, Channel Secret or LINE ID in logs; route **and client-controlled HTTP method labels** are bounded, while `ResponseController` capabilities remain accessible. Flush commits the effective HTTP status before later `WriteHeader` calls; unit tests verify that telemetry matches the committed response and unsupported Flush returns `ErrNotSupported`.
- Runtime configuration validation does not build GCP/LINE SDK clients or perform external I/O, and cannot be used as evidence of Cloud Run IAM, Secret Manager injection or deployed services.

## Unit

- Config：PORT 範圍、憑證配對。
- Messaging usecase：Reply token、Push retry key、Message validate、Text／Flex 選擇。
- Health endpoint：GET 200、其他方法 405。
- LINE webhook：方法限制、簽章驗證、body 大小、文字轉換、Flex 轉換（使用 fake Sender，不接真實 LINE）。
- CLI：沒有 `--send` 不得發送。
- Go AST：Domain／Application 禁止反向依賴與任何外部套件 import，SDK 型別不進入業務核心。
- SDK Catalog：`go.mod` 直接依賴完整覆蓋、狀態與欄位驗證、import 路徑及整合程式存在性；不需對外 API 權限。

## SDK Runtime Foundation (Stage 2)

- HTTP Client: invalid timeout, independent transport ownership, idempotent Close, decorator errors.
- Observability: no secrets, user messages, request URL or headers in logs; preserve HTTP response.
- LINE factory: reject invalid credentials and nil HTTP Client without network calls.
- LINE outbound SDK: local httptest Reply/Push paths and retry keys, failure classification and sanitized errors.
- Context: cancel in-flight LINE request and execute parallel LINE calls with `go test -race`; do not invoke mutable SDK `WithContext` on the shared Client.
- Bootstrap: construct the LINE runtime only when configured, verify health endpoint and resource cleanup; `LINE_API_TIMEOUT` defaults and invalid bounds.

These are **offline contract tests** against the pinned Go SDK and local HTTP servers. They do not validate real LINE account permissions or provider availability.

## Provider Client Factories (Stage 3)

- **Google Workspace**: local HTTP fake exercises the Google Tasks `Tasklists.List` and Calendar `Events.List` native API, a simulated OAuth Bearer transport, and native Sheets/Drive construction. Factories reject nil/cancelled inputs. No real user account or consent is involved.
- **OpenAI / Gemini**: verify explicit API Key requirements, backend differences (Gemini Developer API vs Vertex AI), and native SDK access without paid API calls.
- **Google Cloud**: type-checked Firestore, Cloud Tasks, Secret Manager and Storage constructors reject nil/cancelled contexts and missing Firestore project. GCP IAM and external services are not tested.
- Google Tasks, Google Calendar and Google Cloud Tasks are distinct products with different credentials, scopes and API models.
- Catalog `client_ready` requires actual native Factory source but does not mean a service is enabled, an Adapter exists or a live API permission was verified.

## Durable Event Processing (Stage 4)

- Application Fake Store／Queue：事件 SHA-256 Fingerprint、單次 Record、重送時重新派送但不覆寫 Reply Token。
- Worker：Pending 單次 Claim／送出、模糊網路失敗記錄 Unknown、同一 Key 的第二次交付不再呼叫 LINE。
- HTTP：原始 LINE Webhook 驗章、bot destination、持久化後才回 200、派送失敗時回 503；Cloud Tasks／Scheduler Callback 必須經 OIDC Verifier 驗證。
- Cloud Tasks Queue：事件 Key + 10 秒派送時窗決定 Task ID，驗證同一時窗去重、跨時窗可恢復，OIDC email/audience、只包含 opaque Key 的 JSON Body、AlreadyExists 與其他失敗區分。
- Bootstrap Config：預設同步模式維持相容，Durable/Worker 的 GCP、LINE、OIDC 欄位缺少時 fail closed；`configcheck --validate` 與 Cloud Tasks Adapter 共用 Queue 資源名稱格式檢查，拒絕多餘路徑段、空 Location/Queue、非法資源 ID 字元、Queue ID 超過 100 字元與跨 Project（無須連接 Google Cloud）。
- Firestore Emulator integration tests 保留供手動隔離驗證，不再是 GitHub Actions gate。Cloud Run 使用真實 Firestore database 時，Cloud Tasks CreateTask、IAM、OIDC、TTL 政策與排程必須以已授權的部署驗收確認。

## CI

GitHub Actions 須先下載依賴，再執行：
```bash
make check
```
包含 gofmt、go vet、go test -race、go build；另外由 `go run ./cmd/codexcheck` 進行 Codex 設定與 SDK Catalog 的離線驗證。基礎 SDK 版本透過 `go.mod`／`go.sum` 及固定 Go toolchain 管理。CI 狀態只有 GitHub run 成功後才可宣稱通過。

## CI 效能（保留完整驗證）

- 無快取的 [Actions #41558587334](https://github.com/963sup/linebot-platform/actions/runs/41558587334) 中，`go vet ./...` 約 91 秒，`go test -race -count=1 ./...` 約 144 秒，下載解析依賴約 9 秒，`go build ./...` 約 3 秒。瓶頸是大型 Go SDK 在全新 Runner 上的編譯與 Race Instrumentation。
- 使用既有 pinned `actions/setup-go`，啟用其內建 module/build cache，並以 `go.sum` 作快取失效依據；首次冷快取可能同樣慢，往後重複 PR 工作可觀察 Cache Hit 與縮短的編譯時間。不會改動 SDK 版本、移除預先保留的依賴或刪除空白 SDK 骨架。
- **不跳過任何品質 Gate**：`go mod tidy`、`go.mod/go.sum` diff、`codexcheck`（含 SDK retention）、`gofmt`、`go vet`、`go test -race -count=1` 及 `go build` 全數保留。觀察最新 HEAD 的實際結果後，才能宣稱快取效益。

## Integration / E2E

GitHub Actions 僅執行 `make check` 與 Codex 設定驗證；Firestore Emulator integration tests 是手動、隔離的選用檢查。實際 Firestore、複合索引、TTL、Cloud Tasks、Cloud Run OIDC、LINE Sandbox 與 Workspace 驗收由已授權的雲端部署流程負責。
