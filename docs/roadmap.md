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

- [x] 建立直接依賴 Catalog，對齊本階段全部 12 個 `go.mod` 直接依賴（新增的直接分類為既有 gRPC 模組）。
- [x] 建立通用 `sdk-integration` Skill 與查證參考，沿用原有五個 Codex Agents。
- [x] 擴充 `cmd/codexcheck` 驗證 Catalog、架構邊界及測試，並由既有 CI 執行。
- [x] 同步 README、Skills、架構、版本與開發文件。
- [ ] 新分支 PR 審查與 CI 通過後合併 `main`。
- [ ] 在新的 Codex Cloud 工作階段實測 SDK 查證流程與 MCP 可用性。

## SDK Runtime — 分階段實施

- [x] LINE Typed Client Factory 接受受管理 HTTP Client，加入 Timeout、獨立 Transport、生命週期關閉、出站安全 Telemetry、錯誤分類、Context 隔離及離線契約測試。
- [x] 依實際 SDK 特性建立原生 Typed Client Factory，區分 HTTP／gRPC、API Key／Vertex ADC／Workspace 使用者 OAuth。
- [x] Firestore、Cloud Tasks（背景工作）、Secret Manager、Storage 的 Typed Client Factory 與配置驗證。
- [x] Firestore Inbox/Outbox 交易 Adapter、Cloud Tasks OIDC 派送 Adapter、可選 Durable Webhook、私有 Worker／定期 Reconcile 程式碼。
- [ ] Google Cloud Staging IAM／Firestore TTL／Cloud Scheduler／高吞吐與實際 LINE 回覆驗證，Storage 業務 Adapter 另行實作。
- [x] OpenAI、Gemini／Vertex、Drive、Sheets、Calendar、Google Tasks（待辦）的原生 Client Factory 與離線契約測試。
- [ ] 各 Provider 業務 Adapter、使用者 OAuth 同意與 Token 隔離；目前只有 LINE 實際接入 Webhook／CLI。
- [x] 原子 Inbox/Outbox、事件 Key 去重、Worker 單次 Claim、unknown/expired 終態和 Reconcile 程式碼。
- [ ] 實際部署、定期 Cloud Scheduler、Firestore Emulator 與獨立 Staging 故障恢復驗收。
- [ ] 經授權的 staging SDK 整合測試及 Codex 零先驗需求驗收。

## Stage 5 — Zero-context / Emulator / Staging acceptance

- [x] 新增六項 Codex 零先驗任務清單、模組／路徑驗證及審查準則。
- [x] 新增需明確設定本機端點的 Firestore Emulator Transaction／去重／併發 Claim／終態／TTL 整合測試與執行入口。
- [x] 新增綁定 PR HEAD 的 fail-closed Staging Reviewer Evidence Gate、預設 blocked 的範例與 CI 防誤報驗證。
- [x] 同步驗收文件、Git 忽略規則、Makefile、CI 與安全規範。
- [ ] 在本機 Emulator 環境**實際執行**並保留 Transaction／併發驗收證據（CI 的單元測試不能取代）。
- [ ] 在經授權隔離 Staging 完成 Firestore IAM/TTL、Cloud Tasks OIDC、Cloud Scheduler、LINE Sandbox 的端到端驗證。
- [ ] 在六個**全新 Codex 工作階段**逐一驗收自然語言功能開發與來源／SDK／Method／測試證據。
- [ ] Stage-5 Reviewer Evidence Gate／安全審查與人工核准全部通過後，才可考慮合併或部署。

## P2 — LINE webhook 與持久工作

- [x] LINE Channel Secret 配置／使用官方 SDK 驗章（同步 MVP）。
- [x] 文字事件轉換、跨執行個體去重、Inbox/Outbox、Cloud Tasks／Worker 可選程式碼。
- [ ] Unsend、Edit、其他事件類型與高吞吐回復策略。
- [x] 基礎 Reply／Push／Flex Message adapter 及可測試 usecase。
- [x] 發送前持久化單次 Claim，結果未知標示 unknown，不進行盲目重試；保留跨執行個體事件 Fingerprint 去重。
- [x] 應用 Fake Store／Queue、Webhook 驗章與重送、OIDC Worker Handler、Cloud Tasks Payload 離線測試程式碼。
- [ ] Firestore Emulator／GCP Staging 的實際跨服務驗證。

## P3 — 全公開 API、AI 與雲端

- [ ] 依官方 reference 建立可驗證的 LINE endpoint inventory。
- [ ] Message types、Rich Menu、Audience、Insights、Coupon、Account link。
- [ ] OpenAI / Gemini 可替換 adapter；Google Cloud／Workspace。
- [ ] 批准後部署 staging，驗證後再考慮 production。
