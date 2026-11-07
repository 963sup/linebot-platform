# 執行期設定契約（自動產生）

本文件由 `internal/bootstrap/config_contract.go` 產生，**禁止直接編輯表格**。請修改 Go 設定契約後執行 `go run ./cmd/configcheck --write-doc`，再以 `make config-check` 驗證。這份清單涵蓋現有 webhook、worker、miniapp 與 operator tools 使用的專案設定；Google SDK / ADC 自身的環境變數不在此表。

| 環境變數 | 使用者 | Secret | 預設 | 條件 | 說明 |
|---|---|---|---|---|---|
| `APP_ENV` | webhook,worker,miniapp | 否 | local | Cloud Run 必須明確設定 staging/production | local、staging、production |
| `PORT` | webhook,worker,miniapp | 否 | 8080 | 選用 | HTTP 監聽埠，1–65535 |
| `LINE_EVENT_MODE` | webhook,worker | 否 | webhook=sync; worker=durable | 正式 Webhook 必須 durable；Worker 僅允許 durable | 本機 Webhook 可 sync；Worker 固定 durable |
| `LINE_API_TIMEOUT` | webhook,worker,miniapp | 否 | 8s | 選用 | 1s–12s，單次 LINE HTTP timeout |
| `LOG_LEVEL` | webhook,worker,miniapp | 否 | info | 選用 | debug、info、warn、error |
| `LOG_FORMAT` | webhook,worker,miniapp | 否 | local=text; others=json | 選用 | staging/production 強制 json |
| `TELEMETRY_MODE` | webhook,worker,miniapp | 否 | off | 選用 | off 或 gcp-otlp；gcp-otlp 固定送往 127.0.0.1:4317 collector |
| `OTEL_TRACE_SAMPLE_RATIO` | webhook,worker,miniapp | 否 | 0.05 | TELEMETRY_MODE=gcp-otlp 時套用 | parent-based trace sample ratio，0–1 |
| `OTEL_METRIC_INTERVAL` | webhook,worker,miniapp | 否 | 60s | TELEMETRY_MODE=gcp-otlp 時套用 | metrics 匯出週期，10s–5m |
| `LINE_MESSAGING_API_CHANNEL_SECRET` | webhook | 是 | — | durable 必需；sync 與 Token 成對 | 僅驗證原始 Webhook 簽章，不應傳給 Worker |
| `LINE_MESSAGING_API_CHANNEL_ACCESS_TOKEN` | worker,local-sync,ops-push,ops-profile | 是 | — | worker 必需；本機 sync 成對 | durable 公開 Webhook 禁止持有 |
| `LINE_MESSAGING_API_BOT_USER_ID` | webhook-durable | 否 | — | durable Webhook 必需 | LINE Bot Info API 的 userId；不是 Console 的 Your user ID |
| `GCP_PROJECT_ID` | webhook-durable,worker | 否 | — | durable 必需 | 最小 IAM 的指定 Google Cloud Project；Mini App 由 ADC 自動偵測 |
| `TASK_QUEUE_NAME` | webhook-durable,worker | 否 | — | durable 必需 | 六段 Queue 資源名稱；Project 必須等於 GCP_PROJECT_ID；Queue ID 僅限 1–100 字元英數或連字號 |
| `TASK_WORKER_URL` | webhook-durable,worker | 否 | — | durable 必需 | HTTPS Worker /tasks/v2/line-reply |
| `TASK_OIDC_AUDIENCE` | webhook-durable,worker | 否 | — | durable 必需 | 必須符合 Worker HTTPS Origin |
| `TASK_INVOKER_EMAIL` | webhook-durable,worker | 否 | — | durable 必需 | 允許呼叫 Worker 的 Service Account Email |
| `K_SERVICE` | webhook,worker,miniapp | 否 | 平台提供 | Cloud Run 自動提供 | 防止在 Cloud Run 以 local profile 啟動 |
| `LINEBOT_STAGING_PROJECT` | ops-events | 否 | — | events inspect/check 必需 | 明確指定只讀操作允許的 Staging Project |
| `LINE_MINI_APP_CHANNEL_ID` | miniapp | 否 | — | 必需 | LINE MINI App Channel ID；ID token audience |
| `LINE_MINI_APP_LIFF_URL` | webhook,miniapp | 否 | — | 正式 LINE MINI App 必需 | Console Web app settings 提供的完整 LIFF URL；Bot 入口與前端 LIFF ID 的單一來源 |
| `LINE_MINI_APP_ENDPOINT_URL` | miniapp | 否 | — | 必需 | Console Web app settings 的完整 Endpoint URL；同源驗證的單一來源 |
| `FIRESTORE_DATABASE` | miniapp | 否 | (default) | 選用 | MINI App Firestore client 使用的明確 database ID；Project 由 ADC 偵測 |

## 使用規則

- **local**：未指定 `APP_ENV` 時的安全本機預設，可使用 `LINE_EVENT_MODE=sync`，但本機 demo 不具持久去重。不提供 LINE 憑證時只開放 `/healthz`。Mini App 只允許 HTTP loopback origin。
- **staging / production**：必須明確設定 `APP_ENV`，Webhook 必須 `LINE_EVENT_MODE=durable` 且不含 LINE 發送 Token；Worker 必須持有發送 Token但不得取得簽章用 Secret；Mini App 必須使用沒有 explicit port 的 HTTPS exact origin，且不得持有 Messaging Secret。即使誤設 `APP_ENV=local`，Cloud Run 亦會拒絕啟動。
- **Credential 分離**：上述 Secret 僅由環境（正式環境建議 Secret Manager references）注入。普通設定可以透過部署的非敏感環境變數傳入。應用不讀取 YAML、JSON 設定檔或自動 `source` `.env`。
- **可觀測性**：local 預設文字日志、staging/production 預設 JSON；Telemetry 預設 `off`；`gcp-otlp` 只連接固定 loopback collector，不接受任意 endpoint 或 exporter credential。日誌、metric 與 trace 永不包含 Token、Channel Secret、LINE user ID、EventKey 或原始 webhook body。
- **本機啟動**：`make dev` 啟動不需憑證的 Health MVP；需要測試 LINE 時在**目前 shell**安全提供兩項測試憑證。Worker 與 Mini App 為獨立入口。
- **無配置副作用**：`go run ./cmd/configcheck --role webhook --validate`、`--role worker --validate` 或 `--role miniapp --validate` 只解析並檢查設定，不建立 SDK Client、不連接 Google Cloud、OTLP collector 或 LINE，也不輸出 Secret 值。
