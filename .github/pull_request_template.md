## 目的

請說明這次 PR 的具體行為與影響範圍。

## 變更

- 

## 驗證（只勾選實際執行項目）

- [ ] gofmt
- [ ] go vet ./...
- [ ] go test -race ./...
- [ ] go build ./...
- [ ] LINE contract tests（fake／fixture）
- [ ] SDK／CI 相容性檢查
- [ ] `go run ./cmd/codexcheck`（含 SDK Catalog 與直接依賴一致性）
- [ ] 涉及 SDK 時同步 Catalog、README、版本與相關架構文件

## 風險與未驗證

記錄外部副作用、可能重送、不確定發送結果，以及尚未驗證的項目。

**不得在 PR 中貼出 token、channel secret 或私人訊息。**
