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

## CI

GitHub Actions 須先下載依賴，再執行：
```bash
make check
```
包含 gofmt、go vet、go test -race、go build；另外由 `go run ./cmd/codexcheck` 進行 Codex 設定與 SDK Catalog 的離線驗證。基礎 SDK 版本透過 `go.mod`／`go.sum` 及固定 Go toolchain 管理。CI 狀態只有 GitHub run 成功後才可宣稱通過。

## Integration / E2E

尚未實作。Firestore Emulator、Cloud Tasks、LINE sandbox／測試帳號與 Google Workspace 都必須有隔離憑證、授權和測試用的最小權限，不能在 PR workflow 直接對正式服務操作。
