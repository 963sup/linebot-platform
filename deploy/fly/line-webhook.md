# LINE Webhook 設定（Fly.io Singapore Staging）

本文件對應 `cmd/webhook` / `POST /webhook/line`，與 Worker 的
`/tasks/line-reply`、`/tasks/dispatch` 不同。首次設定只使用
**獨立 LINE 測試 Channel**，避免直接修改既有正式 LINE 官方帳號。

## Fly.io Staging

Webhook 與 Worker 設為 `primary_region = "sin"`（新加坡），
各維持一台常駐 Machine；公開 HTTPS URL 會是：

```text
https://WEBHOOK_APP.fly.dev/webhook/line
```

`WEBHOOK_APP` 是**實際建立並已部署**的 Fly App 名稱；
`linebot-webhook-staging` 只是範例，不能視為已存在或可用的網址。
使用自訂網域時，替換主機名稱，不改 `/webhook/line` 路徑。

在輸入 LINE Developers Console 之前，先完成
[Fly staging 指南](README.md) 的 GCP IAM、Firestore、Cloud Tasks、
Google ADC、Secrets 及 Worker OIDC 驗證。僅 `GET /healthz` 回 200
**不足以證明 Webhook 能正常處理或回覆 LINE 訊息**。

## LINE Developers Console（Messaging API Channel）

1. 登入 [LINE Developers Console](https://developers.line.biz/console/)，
   選擇**測試 Provider / Messaging API Channel**。
2. 在 **Basic settings** 取得 `Channel secret`；
   以 Fly **Webhook App** 專用 Secret 注入 `LINE_CHANNEL_SECRET`。
3. 在 **Messaging API** 頁面確認 **Bot basic ID / Bot user ID**（由
   `destination` 表示，形如 `U...`），並設定兩個 Fly App 的
   `LINE_BOT_DESTINATION`。**不可混用 Channel ID、OA Basic ID
   （形如 `@...`）、測試使用者的 User ID 或 Access Token**。
4. 取得專供測試 Channel 使用的 `Channel access token`，僅注入
   **Worker App** 的 `LINE_CHANNEL_ACCESS_TOKEN`。
   Webhook App 不得持有這個 Token；Worker 不得持有 Channel Secret。
5. 在 **Messaging API → Webhook settings → Webhook URL**，輸入
   `https://WEBHOOK_APP.fly.dev/webhook/line`，開啟 **Use webhook**，
   按 **Verify**。測試通過後可在獨立 Staging Channel 開啟
   **Webhook redelivery**，驗證重送去重行為。
6. 在 LINE Official Account Manager 的 **Response settings**
   確認 Webhook 已啟用；若由 Bot 負責回答文字，避免自動回覆及
   歡迎訊息與 Bot 重複。設定可能因帳號方案與 UI 語言而異。
7. 以**測試好友帳號**傳送文字。確認 Webhook 驗章、Firestore
   Inbox/Outbox、Cloud Tasks OIDC Worker、LINE Reply 全流程。

### 對照環境變數

| 設定 | Webhook App | Worker App |
| --- | --- | --- |
| `APP_ENV` | `staging` | `staging` |
| `LINE_EVENT_MODE` | `durable` | `durable` |
| `LINE_CHANNEL_SECRET` | 必要 Secret | **禁止** |
| `LINE_CHANNEL_ACCESS_TOKEN` | **禁止** | 必要 Secret |
| `LINE_BOT_DESTINATION` | 測試 Bot User ID | 同一個測試 Bot User ID |
| `GCP_PROJECT_ID` | Staging GCP Project | 同一個 Staging GCP Project |
| `TASK_QUEUE_NAME` | 實際 Cloud Tasks Queue 完整名稱 | 同一佇列 |
| `TASK_WORKER_URL` | `https://WORKER_APP.fly.dev/tasks/line-reply` | 同值 |
| `TASK_OIDC_AUDIENCE` | `https://WORKER_APP.fly.dev` | 同值 |
| `TASK_INVOKER_EMAIL` | 已核准的 OIDC Invoker SA | 同值 |

Secrets 必須透過各 App 的 Fly Secret Store 安全注入；不提交
`.env`、Service Account JSON 或 Token。上述 GCP 設定仍必須
通過現有 `internal/bootstrap/config_contract.go` 的驗證。

## 驗證與故障排查

部署並配置完成後，可先用無敏感資料的健康檢查確認進程：

```sh
curl -fsS "https://WEBHOOK_APP.fly.dev/healthz"
# expected: ok
```

**Webhook 不可用一般 curl POST 模擬合法 LINE 事件**：
`/webhook/line` 會透過 LINE SDK 驗證 `x-line-signature`。
使用 LINE Developers 的 **Verify**；LINE 會發送空 `events`
的 HTTPS POST，現有 Durable Handler 在簽章與 Destination 正確、
且依賴服務已初始化時回應 HTTP 200。

常見結果：

- `404`：網址或路徑錯誤，或不是 Webhook App。
- `401`：`Channel secret` 與來源 Channel 不符，或 Destination 不符。
- `503`：Webhook 未初始化、Firestore／Cloud Tasks 異常，或缺少
  必要設定；檢查 Fly 與 GCP 的**已遮蔽敏感資訊**的日誌。
- `5xx`／逾時：請先確認 Machine 常駐、GCP IAM、Google ADC、
  Cloud Tasks Queue、Worker OIDC／HTTPS 路由與跨區延遲。

LINE Verify 成功不代表 Worker 可以成功傳送訊息。現有 Durable
Handler 對非文字事件不會建立 Reply 工作；若需要其他訊息型別，
必須另做產品功能與測試。接受重送時依
`webhookEventId` 去重，但不保證端到端 exactly-once。

## 延遲檢測

從台灣的實際用戶網路執行：

```sh
bash deploy/fly/latency-check.sh WEBHOOK_APP
```

這只量測對 Webhook App **公開 Health** 端點的連線時間；無法
代表 LINE 平台所在地至 Fly.io、跨區 Firestore、Cloud Tasks
或 Worker 對 LINE Reply 的完整延遲。正式驗收應記錄各階段的
分位數（P50／P95）與逾時／錯誤比例，不紀錄原始訊息、Token、
LINE User ID 或敏感 payload。

參考：[LINE 接收 Webhook](https://developers.line.biz/en/docs/messaging-api/receiving-messages/)、
[LINE 驗證 Webhook URL](https://developers.line.biz/en/docs/messaging-api/verify-webhook-url/)、
[LINE 驗證簽章](https://developers.line.biz/en/docs/messaging-api/verify-webhook-signature/)。
