# Contributing

本專案目前是有 LINE Webhook／Reply／Push 的開發用 MVP，尚非正式環境可用的服務。

1. 先閱讀根目錄 `AGENTS.md` 與修改路徑的 `AGENTS.md`。
2. 從 `main` 建立工作分支，以範圍明確的 PR 提交修改。
3. 每次 PR 優先交付可驗證的小型功能，不一次製作大量空 `.go` 檔案。
4. 只在有真實 capability、consumer 與驗收測試時建立 Package；不預建 `.gitkeep` scaffold。
5. PR 需寫出實際執行的檢查與未驗證項目，不能虛報 CI／部署成功。
6. SDK 功能修改先參閱 `docs/sdk/catalog.toml`、`.agents/skills/sdk-integration/SKILL.md`；`go.mod` 版本、Catalog 狀態及相關文件同步更新。
7. 每次提交至少執行 `go run ./cmd/codexcheck` 和 `make check`，在 PR 中標示實際結果。
8. 不提交 API 金鑰、Channel Secret、Access Token、真實 webhook payload 或含個資的測試資料。

Go 品質門檻和 GitHub Actions CI 已存在，另包含 Codex 設定與 SDK Catalog 的機械化檢查。
