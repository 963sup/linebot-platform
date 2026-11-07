# 本機開發

## 環境

安裝 mise、Git、make，然後由 repo `mise.toml` 準備 Go、Node.js 與 gopls：

```bash
mise install
eval "$(mise activate bash)"
go version
make bootstrap # 首次解析依賴並產生 go.sum
make check
make dev
```

`curl http://localhost:8080/healthz` 應回傳 `ok`。

## LINE 憑證

Webhook 僅支援 durable；本機未提供憑證時只開放 health/readiness，webhook 回 503。啟用 LINE 時須提供簽章 Secret 與完整 Firestore／Cloud Tasks 設定；Access Token 只提供給獨立 Worker，不能放在 Webhook。設定清單見 [設定契約](configuration.md)。`make dev` 不讀取 .env 檔案；不要將憑證提交至 git 或日誌。

對外 webhook 在本機測試時須透過 HTTPS 端點，並以測試 LINE Channel 驗證。這個專案**沒有自動建立 LINE Channel 或 GCP 資源**。

訊息入口只接受個人與群組文字事件；Room 會安全略過。輸入 `flex` 或 `卡片` 時回 Flex bubble，其餘文字回固定說明。所有回覆均走 Firestore／Cloud Tasks／Worker，詳見 [Durable Worker 契約](durable-worker.md)。

## 設定檢查與本機快速啟動

```bash
# 不需 LINE 或 GCP 帳號，驗證目前 shell 能啟動本機 Webhook
go run ./cmd/configcheck --validate --role webhook

# 單指令：安全的本機 Health MVP，強制清空兩項 LINE 憑證
make dev

# 真正要使用已授權的 LINE 測試 Channel，須在安全的 shell
# 設定簽章 Secret 與完整 durable 設定，再呼叫：make run-webhook
# 發送 Token 僅交給獨立 Worker

# 由 Go contract 自動驗證文件與安全的 .env.example
make config-check
```

`APP_ENV` 預設 local；只有明確設定 staging 或 production 且滿足 durable 所有條件時可使用部署模式。Cloud Run 的 `K_SERVICE` 若存在，明確拒絕 local。正式環境的 `LINE_MESSAGING_API_CHANNEL_SECRET` 與發送 Token 以 Secret Manager references 注入不同 Service，不能將真實值放到可提交的 YAML 或 `.env.example`。

## LINE SDK Runtime 與 Timeout

LINE SDK HTTP Client 只在 Worker 與明確使用 LINE 的 CLI 建立，預設 Timeout 為 `8s`。Webhook 不在 HTTP 回應期間呼叫 LINE Reply。Worker 可在啟動前設定：

```bash
export LINE_API_TIMEOUT=6s
make run-worker
```

允許 `1s` 至 `12s`，不合法的值會在啟動時拒絕；CLI Push 維持預設 `8s`。HTTP Client 使用獨立的連線池，完成 Shutdown 後關閉其閒置連線。SDK 出站 Debug Log 只有 Provider、Status、Duration、錯誤旗標，不應含 URL、Token 或原始訊息。LINE Context 已改為逐次呼叫隔離。Durable workflow 具備 Firestore Inbox/Outbox 與單次 Claim，不代表已經完成正式環境驗證。

## Push CLI

```bash
# 缺 --send 時不會對外發送。
go run ./cmd/ops push --to "<recipient>" --text "hello"
# 經明確授權，且已有 LINE_MESSAGING_API_CHANNEL_ACCESS_TOKEN 時可加 --send。
```

若發送結果不確定，後續重試務必提供同一 `--retry-key`、同一接收者與相同 payload。CLI 會在第一次發送前輸出 UUID；務必妥善保存，不要放在公開日誌或 issue。

## Codex SDK 查證與設定驗證

`docs/sdk/catalog.toml` 記錄所有直接依賴的 SDK 路由與接線狀態；版本以 `go.mod` 為準。新 SDK 功能請使用 `.agents/skills/sdk-integration/SKILL.md`。沒有真實服務憑證也可執行以下檢查：

```bash
go run ./cmd/codexcheck
go list -m -json google.golang.org/genai
go doc google.golang.org/genai.Client
```

後兩行需要 Go Toolchain 與相應 module 可用，不會對真實 Gemini API 發送請求；遇到無法下載時，不能憑模型記憶猜 API。

## SDK 安裝狀態

`go.mod` 只表示模組版本需求。`go.mod`／`go.sum` 已於 GitHub Actions Go 1.27.3 環境解析並提交。後續升級模組時應在有網路的環境執行 `go mod tidy`、`go mod download`，檢查並提交異動；CI 持續保留依賴解析的審查用 artifact。

Firestore 與 Cloud Tasks 已接入 Durable Workflow；OpenAI、GenAI、MCP client 保留原生 SDK factory；Secret Manager 僅由部署平台注入，不使用應用 SDK。Workspace 自動化延後整合，不保留預先建立的 OAuth 或 API 骨架。
