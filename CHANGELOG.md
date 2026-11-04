# Changelog

## Unreleased

### 統一設定系統（Draft PR，尚未合併）

- 將 Runtime 設定、角色與環境驗證集中於 Bootstrap，區分非敏感 Config 與獨立 SecretConfig；加入 local/staging/production 與 Cloud Run fail-closed 原則。
- 新增 Go 原生設定契約、生成文件與 `.env.example` 一致性檢查；`make dev` 簡化安全本機啟動，不載入秘密檔。
- 統一 LOG_LEVEL/LOG_FORMAT、JSON staging 日誌與去敏 inbound HTTP metadata；未部署或啟用任何雲端資源。

### Value Object 目錄命名調整（待合併）

- 將 `internal/domain/valueobjects/` 調整為 `internal/domain/value-objects/`，僅移動預留 `.gitkeep`；Go package 名稱保留 `valueobjects`。
- 同步根與 Domain AGENTS、README、架構文件、目錄清單及自動化架構契約測試。此變更不修改 Runtime、SDK、資料模型、發送或部署行為；下方歷史記錄保持原狀。

### 已合併至 main（PR #8，2026-11-03）

- 建立 Go **1.27.3**、DDD／Hexagonal Architecture、Webhook、Reply／Push／Flex、健康檢查、CLI 與驗證環境。
- 保留 **12 個直接 Go Module 依賴**（`go.mod` 是版本唯一來源）、`go.sum` 與已核准 SDK 空白骨架；`garr` 仍屬未啟用的保留工具。
- 加入 SDK Catalog、Codex SDK Skill、**六個**專用 Agents（含 Code Cleanup & Release Validation）、`codexcheck` 依賴／骨架保護與架構驗證。
- 實作 LINE 受管理 HTTP Client、Timeout／Context 隔離、去敏觀測與可分類錯誤；已建立 OpenAI、Gemini／Vertex、Google Workspace（Tasks／Calendar／Sheets／Drive）與 Google Cloud（Firestore／Cloud Tasks／Storage／Secret Manager）原生 Client Factories。
- 以 `LINE_EVENT_MODE=durable` 提供可選的 Firestore 原子 Inbox/Outbox、Cloud Tasks Queue、私有 OIDC Worker、單次 Claim 及異常狀態 Reconcile；**預設同步開發模式未變**。
- 保留 Unit／Contract／Architecture／Race Tests、CI 模組鎖定檢查及 SDK 保留保護。Firestore Emulator 整合測試曾於合併前通過，專用一次性驗收程式在清理階段移除。
- **尚未進行正式部署**：Google Cloud IAM／OIDC／TTL／Cloud Scheduler、LINE 測試帳號及 Google 使用者 OAuth，仍需經授權的隔離 Staging 驗收。CI 通過不代表正式環境可用。

### 歷史骨架工作

- 初版建立 README、DDD／Hexagonal Architecture 目錄、`.gitkeep`、開發規範與技術來源。
- 目錄命名將 Google Gen AI 統一為 `genai`，Value Objects 使用 `valueobjects`。
