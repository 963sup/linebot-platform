# Codex 零先驗開發驗收（需要全新工作階段）

此頁是**驗收程序**，不是已完成的 Codex Cloud 實測報告。CI 只檢查情境清單格式與路由是否存在，不能證明 Codex 在新工作階段真的會使用對應工具。

## 受控情境

以 [zero-context-scenarios.toml](zero-context-scenarios.toml) 的六個自然語言任務作為唯一情境來源：LINE Rich Menu、Google Tasks、Google Calendar、AI 回覆、Webhook 重送可靠性，以及 Gemini SDK 升級。每個情境包含對應 Go Module、現有檔案與至少三個必須驗證的判準。

## 每個情境的執行流程

1. 從 PR #8 的**同一個 Commit SHA** 建立乾淨的測試工作樹；確認未攜帶個人憑證、不沿用先前對話內容或隱藏上下文。
2. 在新開啟的 Codex Cloud／CLI 工作階段，只提交該情境的 `prompt`。不提供「應使用的 SDK、要 import 哪個 package、要讀哪份文件」等額外提示。
3. 允許 Codex 依專案 `AGENTS.md`、SDK Catalog、Skill、`go.mod`、gopls 及官方文件自行發現 API；每個情境在隔離測試分支執行，不直接修改正在審查的 PR #8。
4. 確認它沒有猜測不存在的 Method、沒有在 Application／Domain 引入外部 SDK 型別，且能提供最小必要的程式碼與測試。
5. 執行 `go run ./cmd/codexcheck` 和 `make check`；測試通過才評估情境內的全部判準。
6. 將每個情境的 Session ID、Commit、Go SDK 版本、所使用的來源及測試摘要保存在**受保護的驗收紀錄**，不包含 Token、訊息原文或私人資料。六個任務都須由審查者確認。

如果 Codex Cloud 沒有 Published 環境、MCP 無法使用或沒有對應權限，應標記 `blocked`，不能以 CI 輸出代替真正的 Codex 工作階段。

## 驗收標準

- **Provider 選擇**：能正確區分 Google Tasks（待辦）、Sheets（試算表）、Calendar（日曆）和 Cloud Tasks（佇列）。
- **版本與 API 證據**：從 `go.mod` 與鎖定 SDK 取得真實方法簽名，不依賴記憶猜測。
- **架構**：只在必要時新增 Application Port；Provider SDK 留在 Adapter／Infrastructure。
- **權限與可靠性**：不擅自取用使用者 OAuth、啟用雲端資源、重送未知的 LINE 發送或記錄秘密。
- **驗證與同步**：執行實際測試並同步影響到的 Catalog、文件、CI 與部署安全門檻。

六個情境的外部執行結果應記錄在 Stage-5 Reviewer Evidence 的 `codex_*` 檢查項，並附可審查的證據連結；**不得預先標為 passed**。

參見 [Staging 驗收與人工證據門檻](../operations/stage5-acceptance.md)。
