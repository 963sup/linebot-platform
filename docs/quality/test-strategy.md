# 測試策略

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
- Bootstrap Config：預設同步模式維持相容，Durable/Worker 的 GCP、LINE、OIDC 欄位缺少時 fail closed。
- Firestore Transaction、併發 Claim 與恢復已在 GitHub Actions 的本機 Firestore Emulator 實際驗證；Cloud Tasks CreateTask、IAM、OIDC、TTL 政策與排程仍需經授權的真實 Staging 驗收。

## CI

GitHub Actions 須先下載依賴，再執行：
```bash
make check
```
包含 gofmt、go vet、go test -race、go build；另外由 `go run ./cmd/codexcheck` 進行 Codex 設定與 SDK Catalog 的離線驗證。基礎 SDK 版本透過 `go.mod`／`go.sum` 及固定 Go toolchain 管理。CI 狀態只有 GitHub run 成功後才可宣稱通過。

## Stage-5 驗收工具

- `go run ./cmd/codexcheck` 會驗證六項 Codex 自然語言情境的 Module、來源路徑與評分標準。這是**情境資料驗證**，不是已執行 Codex Cloud。
- `cmd/acceptance` 的單元測試會驗證 production／錯誤 SHA／過時、缺少或不安全的證據一律 fail closed；最終 Gate 只能透過明確審核後的真實 Staging 證據通過。
- `tests/integration/firestore_emulator_test.go` 以 `//go:build integration` 隔離；GitHub Actions 自動啟動官方本機 Emulator 並執行實際 Firestore Transaction/Claim/Recovery 測試，已於 [#41554849393](https://github.com/963sup/linebot-platform/actions/runs/41554849393) 通過。手動重現可使用 `make acceptance-emulator`。未提供有效 Emulator Host 時整合測試明確失敗。
- CI 驗證 gofmt、vet、race tests、build、情境清單與 Stage-5 Example 的預設拒絕行為；不在一般 PR 中呼叫真實雲端或個人 OAuth。

## Integration / E2E

Firestore Emulator SDK 整合測試已在 GitHub Actions 的本機容器**實際執行並通過**。Cloud Tasks、Cloud Run OIDC、LINE Sandbox 和 Workspace 的真實跨服務 E2E 仍需要隔離憑證、授權和人工審查。CI 通過不得視為這些外部測試通過。
