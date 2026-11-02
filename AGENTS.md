# AGENTS.md — linebot-platform

本檔是 Codex 與其他程式代理人在此儲存庫的**專案規範**。優先遵循使用者當次指示與更細粒度的目錄 `AGENTS.md`；不可把規劃說成已實作。

## 先確認狀態

- 先讀 `README.md`、`docs/architecture/overview.md`、`docs/roadmap.md` 和相關子目錄規範。
- 現階段僅有 scaffold／文檔與 `.gitkeep`；尚無可工作的 LINE Bot。
- 以最小可行、可編譯、可測試的垂直切片實作新功能。新增實作時可刪除該目錄的 `.gitkeep`；避免保留冗餘 placeholder。
- 新 SDK 或 feature 在加入前確認上游官方文件、API 版本、帳號地區資格與安全修補版本。不要使用浮動依賴版本當作部署契約。

## 依賴方向

- `internal/domain` 不得 import `application`、`adapter`、`infrastructure` 或 LINE／Google／AI SDK。
- `internal/application` 可 import `domain`，但不能依賴 `adapter` 與 `infrastructure`。
- `internal/adapter` 將外部 API／HTTP 資料轉為 application 自有的 DTO、port。
- `internal/infrastructure` 負責 SDK client、認證、transport、timeout、observability。
- `internal/bootstrap` 專責依賴組裝；`cmd/` 只負責啟動程序。
- 不預先為所有構想建立空介面、空 `.go` 檔或用無用途的 abstraction 填滿架構。
- `value-objects` 是使用者指定的目錄名；實際 Go package 請命名為有效的 `valueobjects`。

## LINE webhook 與發送

- 必須使用 LINE raw request body 驗證簽章，驗證前不得更改內容。
- 持久化 inbox 與 outbox 後才可確認事件接收成功；不得在回覆 200 後靠程序內 goroutine 保證處理。
- 以 channel 與 event ID 去重；Cloud Tasks 工作須能冪等、可重試、可補償。
- Reply token 有時效，不得假設任何長時間背景任務完成後仍可 reply。
- 發送超時可能是「結果未知」；不可在未確認冪等策略時直接重送。
- 支援 unsend／edit 時要處理已刪資料、遲到事件與舊 AI 回覆。
- 各 API 需查詢 channel 權限／地區限制；未授權時明確標示 unavailable。

## 安全與隱私

- 禁止提交真實 `.env`、憑證、LINE Channel Secret、Access Token、AI API Key。
- 禁止把完整 webhook、token、使用者私訊或個資輸出到 log。
- 配置使用 ADC／工作負載身分／Secret Manager；權限最小化。
- 外部發送、刪除、付費雲端資源與正式環境部署都須遵循授權；不自行合併 PR 或啟用 production。
- 不將未審核的外部 skills／MCP 當作可信程式自動執行。

## 驗證與交付

- Go 程式存在後，依序檢查格式、`go vet ./...`、`go test ./...`、`go test -race ./...`、`go build ./...`；若環境不支援，明確記錄原因。
- 任何與 SDK 的整合都需 unit／contract tests；實際雲端／LINE E2E 需隔離且明確授權。
- 提交 PR 必須描述變更、實際測試結果、風險、未完成事項；不得聲稱空目錄已完成實作。
- 發布前檢查 `README.md`、`docs/sources.md`、功能矩陣與相容性紀錄。
