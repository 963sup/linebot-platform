# 本機開發

## 環境

安裝 Go 1.27.3、Git、make。確認版本後：

```bash
go version
make bootstrap # 首次解析依賴並產生 go.sum
make check
make run-webhook
```

`curl http://localhost:8080/healthz` 應回傳 `ok`。

## LINE 憑證

本機 `LINE_EVENT_MODE=sync` 需要同時透過安全環境提供 `LINE_CHANNEL_SECRET` 與 `LINE_CHANNEL_ACCESS_TOKEN`；未設定時只開放 health，webhook 回 503。**Cloud Run 公開服務必須使用 `LINE_EVENT_MODE=durable`，只提供簽章用的 `LINE_CHANNEL_SECRET`，不得注入發送 Token；僅私有 Worker 持有 Token**。不要把值寫進 git、PR、公共 log 或公開 issue。

對外 webhook 在本機測試時須透過 HTTPS 端點，並以測試 LINE Channel 驗證。這個專案**沒有自動建立 LINE Channel 或 GCP 資源**。

預設 `LINE_EVENT_MODE=sync` 只支援文字事件，輸入 `flex` 或 `卡片` 時回 Flex bubble，其餘文字回固定說明。此預設模式已驗簽但**沒有持久去重**；另有需完整設定才能啟用的 `durable` 模式。詳見 [Durable Worker Staging 契約](durable-worker.md)。

## LINE SDK Runtime 與 Timeout

同步 Webhook 使用受管理的 LINE SDK HTTP Client，預設 Timeout 為 `8s`。選用的 Durable Webhook 改為 Firestore/Cloud Tasks 非同步處理，不在 HTTP 回應期間直接呼叫 LINE Reply。可以在啟動服務前設定：

```bash
export LINE_API_TIMEOUT=6s
make run-webhook
```

允許 `1s` 至 `12s`，不合法的值會在啟動時拒絕；CLI Push 維持預設 `8s`。HTTP Client 使用獨立的連線池，完成 Shutdown 後關閉其閒置連線。SDK 出站 Debug Log 只有 Provider、Status、Duration、錯誤旗標，不應含 URL、Token 或原始訊息。LINE Context 已改為逐次呼叫隔離。**預設 sync Webhook 不提供持久事件去重**；只有明確設定且取得隔離 GCP 權限的 durable 模式具備 Firestore Inbox/Outbox 與單次 Claim，不代表已經完成正式環境驗證。

## Push CLI

```bash
# 缺 --send 時不會對外發送。
go run ./cmd/ops push --to "<recipient>" --text "hello"
# 經明確授權，且已有 LINE_CHANNEL_ACCESS_TOKEN 時可加 --send。
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

Google Workspace（Tasks、Calendar、Sheets、Drive）、Google Cloud（Firestore、Cloud Tasks、Secret Manager、Storage）及 OpenAI／Gen AI 已建立原生 SDK Client Factory。**Firestore 與 Cloud Tasks** 已接入可選的 Durable Webhook／私有 Worker 程式碼；其他服務仍以 `client_ready` 為主，尚未使用真實 OAuth 或正式雲端憑證。`garr` 仍為選用依賴。Catalog 區分 `declared`、`client_ready`、`integrated`；Google Tasks 使用者 OAuth 和 Calendar scopes 見 [Workspace OAuth 說明](../sdk/workspace-oauth.md)。
