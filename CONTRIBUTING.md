# Contributing

本專案目前為設計與目錄骨架階段。

1. 先閱讀根目錄 `AGENTS.md` 與修改路徑的 `AGENTS.md`。
2. 從 `main` 建立工作分支，以範圍明確的 PR 提交修改。
3. 每次 PR 優先交付可驗證的小型功能，不一次製作大量空 `.go` 檔案。
4. 真正建立程式檔後，可移除對應的 `.gitkeep`。
5. PR 需寫出實際執行的檢查與未驗證項目，不能虛報 CI／部署成功。
6. 不提交 API 金鑰、Channel Secret、Access Token、真實 webhook payload 或含個資的測試資料。

正式的 Go 品質門檻與 CI 將在可編譯的第一個垂直切片加入。
