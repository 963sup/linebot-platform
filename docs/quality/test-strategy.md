# 測試策略

## Unit

- Config：PORT 範圍、憑證配對。
- Messaging usecase：Reply token、Push retry key、Message validate、Text／Flex 選擇。
- Health endpoint：GET 200、其他方法 405。
- LINE webhook：方法限制、簽章驗證、body 大小、文字轉換、Flex 轉換（使用 fake Sender，不接真實 LINE）。
- CLI：沒有 `--send` 不得發送。
- Go AST：Domain／Application 禁止反向依賴。

## CI

GitHub Actions 須先下載依賴，再執行：
```bash
make check
```
包含 gofmt、go vet、go test -race、go build。基礎 SDK 版本透過 `go.mod`／`go.sum` 及固定 Go toolchain 管理。CI 狀態只有 GitHub run 成功後才可宣稱通過。

## Integration / E2E

尚未實作。Firestore Emulator、Cloud Tasks、LINE sandbox／測試帳號與 Google Workspace 都必須有隔離憑證、授權和測試用的最小權限，不能在 PR workflow 直接對正式服務操作。
