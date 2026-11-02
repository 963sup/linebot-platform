# Roadmap

## P0 — Repository scaffold（本次 PR）

- [x] 建立完整目錄骨架及 `.gitkeep`。
- [x] 根 README、AGENTS、分層規範、技術來源與 Codex skills。
- [ ] PR 經審核合併到 main。

## P1 — 可運行 Go 最小切片

- [ ] 根據 SDK 最新安全修補要求確認 Go 版本，建立 `go.mod`。
- [ ] `cmd/webhook` 提供 `GET /healthz`、PORT、graceful shutdown。
- [ ] 單元測試、Makefile、Dockerfile、GitHub Actions CI。
- [ ] 架構邊界測試。

## P2 — LINE webhook 與持久工作

- [ ] LINE Channel Secret 配置／raw body signature verification。
- [ ] 事件型別轉換、去重、Inbox/Outbox、Cloud Tasks／worker。
- [ ] Reply／Push、error classification、發送結果未知與去重。
- [ ] emulator／fake 驗證。

## P3 — 全公開 API、AI 與雲端

- [ ] 依官方 reference 建立可驗證的 LINE endpoint inventory。
- [ ] Message types、Rich Menu、Audience、Insights、Coupon、Account link。
- [ ] OpenAI / Gemini 可替換 adapter；Google Cloud／Workspace。
- [ ] 批准後部署 staging，驗證後再考慮 production。
