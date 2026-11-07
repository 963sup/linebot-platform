# main 基礎設施狀態與歷史殘留稽核

**核對日期：2026-11-03。** 依據已合併的 [PR #8](https://github.com/963sup/linebot-platform/pull/8)、`main` 合併 Commit `901096a7`、[合併後 CI #41568775857](https://github.com/963sup/linebot-platform/actions/runs/41568775857) 及目前原始碼。

## 結論與判定標準

- **SDK 基礎設施（程式碼）已建立且合併**：`go.mod` 鎖定 12 個直接 Module；`go.sum` 提供完整性紀錄，SDK Catalog 完整對應。Client Factories、LINE Runtime、可選的 Firestore Inbox/Outbox 與 Cloud Tasks Worker 已有 Go 原始碼並通過 Go CI。
- **不是 12 項服務都已啟用**：`declared` = 保留依賴，`client_ready` = 可建立原生 SDK Client，`integrated` = 具正式 Go 程式碼的 Adapter／用途；以上三種均**不代表**已取得真實帳號、IAM／OAuth 或完成部署驗收。
- **正式雲端環境未驗證**：儲存庫沒有已驗證的 Firestore TTL、Cloud Scheduler、Cloud Run IAM／OIDC、LINE 測試 Channel 或使用者 OAuth 端到端證據；不能稱為 production-ready。

## SDK 與程式碼現況（go.mod 為版本權威）

| 模組 | Catalog 狀態 | main 程式碼／用途 | 尚需驗證 |
|---|---|---|---|
| LINE SDK v8 | `integrated` | Webhook 驗章、Reply／Push、Flex、受管理 HTTP Client、Context 隔離 | 真實 LINE 測試 Channel／配額 |
| Firestore | `integrated` | 原生 Factory、Inbox/Outbox 原子 Transaction、單次 Claim、Recover | 真實專案 IAM、TTL、擴容測試 |
| Google Cloud Tasks | `integrated` | 原生 Factory、Queue／OIDC Task、私有 Worker、排程恢復入口 | 真實 Queue、Cloud Run OIDC、Scheduler |
| Google Secret Manager | deployment-only | Cloud Run secret references 注入環境變數；無應用 SDK factory | 需部署 IAM 驗證 |
| OpenAI Go SDK v3 | `client_ready` | 原生 Factory；Responses API 型別可查證 | 業務 Adapter、憑證與授權測試 |
| Google Gen AI SDK | `client_ready` | Gemini API 與 Vertex AI 原生 Factory | 業務 Adapter、ADC／API Key 驗證 |
| Google API Client | `integrated` | Cloud client options、iterator、Worker OIDC | 不包含 Workspace 功能 |
| UUID | `integrated` | CLI Push Retry Key（UUID v4） | 已有離線測試 |
| TOML | `integrated` | `cmd/codexcheck` 解析 Codex／Catalog 設定 | 已有 CI 檢查 |
| gRPC | `integrated` | Firestore／Cloud Tasks 狀態碼與錯誤判斷 | 已有 Go 型別及測試 |


## 基礎設施交付證據

- `internal/infrastructure`：LINE、OpenAI、GenAI、MCP、Google Cloud 的 Typed Client Factory；`httpclient`、`resilience`、`observability`。
- `internal/adapter/outbound/google/cloud`：Firestore EventStore、Cloud Tasks Queue；`internal/application/usecases/messaging-delivery` 與 `cmd/worker`：持久化流程與私有 Worker。
- `go run ./cmd/codexcheck`：SDK Catalog、Codex Agents、保留 SDK 模組與 build-tag manifest 驗證；不驗證空 scaffold。合併後 `main` 的 [CI #41568775857](https://github.com/963sup/linebot-platform/actions/runs/41568775857) 通過模組鎖定、格式、Vet、Race Test、Build。
- Firestore Emulator 的交易／併發整合測試最初在 [PR #8 的 CI #41555545761](https://github.com/963sup/linebot-platform/actions/runs/41555545761) 通過；現行 `main` CI 已恢復獨立 Emulator job，使用 Java 25 啟動 Emulator 並執行 `make integration-test`。這是本機交易回歸證據，仍不能替代真實專案的 IAM、TTL、索引、OIDC 或 Staging 驗收。

## 歷史殘留稽核與處置

下表保留 2026-11-03 稽核背景；現行治理已改為按 capability 建立 package，不再將空 SDK 目錄視為受保護資產：

| 歷史內容 | 檢查結果／處置 |
|---|---|
| `CHANGELOG.md`、`internal/infrastructure/README.md` | 從「只有宣告 SDK、尚無 Worker」更新為現有 Factory、Adapter、Worker 的正確狀態 |
| `docs/versions.md`、`docs/sources.md` | 修正 Firestore／Cloud Tasks 狀態、補入缺漏的 TOML Module、移除「只有 LINE SDK 接線」與「依賴尚未驗證」等敘述 |
| `docs/roadmap.md`、`docs/ai/codex-cloud.md` | 將 PR #8 已合併的事項標示完成；Codex Cloud Publish／MCP 實測仍未確認 |
| `docs/architecture/overview.md`、`docs/architecture/event-processing.md` | 區分已存在的程式碼與僅是規劃中的雲端部署，保留歷史 Emulator 驗證證據 |
| `docs/line/feature-matrix.md`、`docs/operations/local-development.md` | 修正「全部未實作」、預設同步與可選 Durable 模式的去重能力混淆 |
| `deploy/README.md`、`deploy/firestore/README.md` | 修正「沒有可用建置／部署設定」及「This PR」的歷史時態；不誤稱雲端資源已建立 |
| 固定文字／Flex 範例回覆 | 仍是目前 MVP 既有行為；不能當作完整對話或正式 AI 整合，也不在純文件更新中擅自移除 |
| 一次性 Stage-5 模擬／假證據／Emulator 驗收檔 | PR #8 清理後未保留在 `main`；Git 歷史保留先前驗證證據，非執行期殘留 |

## 程式碼與文件引用掃描

- 核對 `main` 的 **38 個非測試 Go 原始碼檔**：未找到 `TODO`、`FIXME`、`XXX`、`HACK` 等明確的臨時待辦標記。仍有刻意保留的 MVP／Demo Reply 與必要 SDK 兼容限制註解；不將其誤判為可直接刪除的廢碼。
- 以本次更新分支的 Git Tree 檢查 **47 份 Markdown 的相對連結路徑**：未發現指向不存在的儲存庫檔案／目錄。此檢查不含外部 HTTP 網址的即時可用性。
- 2026-11-03 的 `main` 曾有 **130 個目錄**；此數量只是歷史快照，不是現行 manifest 或驗收門檻。
- **未進行外部機密掃描、真實服務登入或雲端權限測試**；不應把靜態文字掃描視為安全稽核的完整替代品。

## 尚待完成，但不能以刪除 TODO／文件掩蓋

真實 Staging IAM／OIDC／Cloud Tasks／Firestore TTL／Cloud Scheduler／LINE 測試帳號；必要時恢復持久性整合回歸測試；高吞吐的 Recover 游標／索引與 Unknown 告警；尚未建立的進階 LINE／AI 業務 Adapter。這些是功能及營運驗收工作，不是已完成的正式部署。

相關來源：[SDK Catalog](../sdk/catalog.toml) · [版本與依賴](../versions.md) · [事件流程](event-processing.md) · [部署要求](../operations/durable-worker.md) · [Roadmap](../roadmap.md)。
