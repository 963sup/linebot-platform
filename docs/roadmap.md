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
