# Roadmap

## P0 — Repository scaffold（本次 PR）

- [x] 建立完整目錄骨架及 `.gitkeep`。
- [x] 根 README、AGENTS、分層規範、技術來源與 Codex skills。
- [ ] PR 經審核合併到 main。

## P1 — 可運行 Go 最小切片（此 PR）

- [x] 根據上游 SDK 的最低需求選擇 Go 1.27.3，建立 `go.mod`，宣告 SDK 版本。
- [x] `cmd/webhook` 提供 `GET /healthz`、PORT、graceful shutdown。
- [x] 建立 unit/contract tests、Makefile、Dockerfile、GitHub Actions CI 的實作草案。
- [x] Go AST 架構邊界檢查。
- [x] GitHub runner 取得真實的 `go.sum` 並提交。
- [x] 在 GitHub Actions 已成功執行 gofmt、go vet、race tests 與 Go build；後續 PR 仍須檢查自身 CI。
- [ ] PR 經審查合併。

## SDK-Aware Codex Foundation — 開發知識與驗證

- [x] 建立直接依賴 Catalog，對齊全部 11 個 `go.mod` 直接依賴。
- [x] 建立通用 `sdk-integration` Skill 與查證參考，沿用原有五個 Codex Agents。
- [x] 擴充 `cmd/codexcheck` 驗證 Catalog、架構邊界及測試，並由既有 CI 執行。
- [x] 同步 README、Skills、架構、版本與開發文件。
- [ ] 新分支 PR 審查與 CI 通過後合併 `main`。
- [ ] 在新的 Codex Cloud 工作階段實測 SDK 查證流程與 MCP 可用性。

## SDK Runtime — 後續小型 PR

- [ ] Typed Client Factory / Transport / Observability / Lifecycle，先以 LINE 驗證。
- [ ] Firestore、Cloud Tasks、Secret Manager、Storage 的 Client 與 Adapter（僅啟用有實際需求者）。
- [ ] OpenAI、Gemini、Drive、Sheets、Calendar 的受管理 Client 和契約測試。
- [ ] 持久事件、去重、Inbox/Outbox、Worker 與發送結果未知處理。
- [ ] 經授權的 staging SDK 整合測試及 Codex 零先驗需求驗收。

## P2 — LINE webhook 與持久工作

- [x] LINE Channel Secret 配置／使用官方 SDK 驗章（同步 MVP）。
- [ ] 事件型別轉換、去重、Inbox/Outbox、Cloud Tasks／worker。
- [x] 基礎 Reply／Push／Flex Message adapter 及可測試 usecase。
- [ ] 錯誤分類、發送結果未知的持久追蹤與跨執行個體去重。
- [ ] emulator／fake 驗證。

## P3 — 全公開 API、AI 與雲端

- [ ] 依官方 reference 建立可驗證的 LINE endpoint inventory。
- [ ] Message types、Rich Menu、Audience、Insights、Coupon、Account link。
- [ ] OpenAI / Gemini 可替換 adapter；Google Cloud／Workspace。
- [ ] 批准後部署 staging，驗證後再考慮 production。
