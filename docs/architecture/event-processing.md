# LINE 事件處理：同步 MVP 與選用的持久工作模式

目前只有本機／離線契約測試；**沒有建立或部署任何 GCP、Firestore、Cloud Tasks、Cloud Scheduler、Cloud Run 或正式 LINE 資源**。新增的持久模式不代表可以直接上 production。

## 模式與入口

- 預設 `LINE_EVENT_MODE=sync`：保留現有 LINE 簽章驗證與文字／Flex 同步 Reply 開發模式。
- 設定 `LINE_EVENT_MODE=durable`：Firestore 原子存入 Inbox/Outbox，Cloud Tasks 以固定任務名稱和 OIDC 派送至私有 `cmd/worker`。設定不足時直接拒絕啟動。
- Durable Webhook 在處理前以原始 Body 驗章，驗證 bot destination，只保存文字訊息所導出的固定回覆，不儲存原始使用者訊息。

## 資料流程

1. **驗章 + 去重**：以 SHA-256(channel + separator + webhookEventId) 產生固定、不洩漏來源的事件 Key。
2. **Firestore 交易**：在同一 Transaction 建立 `line_inbox_v1` 和 `line_outbox_v1`，重送不覆蓋原本的 Reply Token。
3. **Cloud Tasks**：將唯一 opaque Key 送往 HTTPS `/tasks/line-reply`，Task ID 固定；`AlreadyExists` 視為成功。其他派送錯誤回 HTTP 503，保留可修復的 Outbox。
4. **Worker 安全驗證**：必須通過 Google 簽章 OIDC Token、正確 Audience 和呼叫者服務帳號 Email。Cloud Run 另外必須啟用私有 Ingress 和 IAM Authentication。
5. **Single-use Claim**：Worker 使用 Firestore Transaction 將 `pending` 改為 `sending`，再嘗試 LINE Reply；第二個 Worker 不會再次傳送。
6. **終態**：`sent`、`expired`、`unknown`。網路逾時／結果不確定時，不能盲目重試外部發送。
7. **Reconcile**：以明確設定的 Cloud Scheduler 定期呼叫具 OIDC 保護的 `POST /tasks/dispatch`，重排 Pending、將過期的 Sending 移到 Unknown；不會重新發送 Unknown。

## Reply Token、隱私和 TTL

- 保守使用收到事件後 **45 秒** 的 Reply 視窗。過期回覆不會擅自變成 Push，也不使用使用者 ID 做回覆。
- Inbox 以 Fingerprint 記錄 7 日；Outbox（暫存短期 Reply Token、衍生回覆及 Lease）1 小時。正式 staging 必須分別設定 Firestore `expires_at` TTL，並了解 TTL 清除非即時；終態會清空暫存的 Reply Token。
- Cloud Tasks Payload 只包含 SHA-256 Key，並不包含 Reply Token、原始訊息或使用者 ID。日誌不輸出個資及 SDK 原始錯誤。
- Firestore Transaction 可保證單筆去重與原子持久化，但不能宣稱對 LINE 的端到端 exactly-once delivery。若 Worker 在 Claim 後中斷，可能需要人工處理 Unknown。

## 上線前限制

- Webhook 目前只支援文字事件；unsend、edit、媒體、AI 長任務仍未處理。
- Reconcile 目前是有上限的狀態查詢，正式高吞吐需要 Cursor、Lease 到期索引、監控與人工修復工具。
- 未設定 Cloud Scheduler、GCP IAM、Firestore TTL、部署映像或驗證真實 LINE 帳號。不可用 CI 通過代替 Staging 身分、權限及網路驗證。

部署準備與權限參見 [持久模式操作要求](../operations/durable-worker.md) 和 [Firestore 資料保留](../../deploy/firestore/README.md)。
