# linebot-platform

以 **Go + Domain-Driven Design (DDD) + Hexagonal Architecture** 為核心的 LINE Messaging API 平台專案，規劃部署到 Google Cloud Run，並整合 OpenAI、Gemini 與 Google Cloud / Google Workspace API。

> **狀態：初始化中。** 此儲存庫目前尚未提供可運行的 LINE Bot、雲端部署或正式 API 整合。預定的目錄骨架、架構規範及技術文件會透過 PR 逐步加入。

## 設計方向

- LINE Messaging API：Webhook、訊息發送、媒體、Rich Menu、Audience、Insights、Coupon 與其他依帳號資格開放的 API。
- Go 單一儲存庫、模組化單體；`domain`、`application`、`adapter`、`infrastructure` 分層。
- Cloud Run webhook 與 background worker 分離；以持久化事件處理降低重送造成的重複副作用。
- 外部 SDK 限制在 adapters / infrastructure；核心業務邏輯保持技術中立。
- Codex 的 `AGENTS.md`、skills、MCP / plugins 依開發需要逐步納入。

## 後續工作

1. 建立目錄骨架與 `.gitkeep`、分層規範與 SDK 參考文件。
2. 加入可編譯的 Go module、健康檢查與 CI。
3. 以測試驗證的方式逐步加入 LINE webhook、持久化處理、訊息發送。
4. 在核准後建置 staging Cloud Run 環境。

請勿將 LINE Channel Secret、Access Token、AI API Key 或 Google Cloud 憑證提交到儲存庫。
