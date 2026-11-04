# 隔離 Staging 驗收：LINE + Google Cloud Run + Firestore + Cloud Tasks

**狀態：尚未執行。** 此頁是可重現的人工驗收契約，不建立資源、不預設已有 Google Cloud 專案／LINE Sandbox 身分。本儲存庫未獲得對真實 GCP/LINE 的授權，CI Emulator 通過不等於本頁通過。

## 前置與准入

- 由擁有者指定單獨計費預算／IAM 的 Staging GCP Project、區域、Firestore database、Cloud Tasks Queue 與 LINE **測試** Channel。不得複用 Production token 或 Secret。
- 使用已核准的 image digest，僅部署到明確 Staging Service；在實際索引/TTL/IAM 完成前保持入口停用。
- Webhook 公開 HTTPS 路由僅接收 `/webhook/line`，使用 `LINE_EVENT_MODE=durable`、`LINE_CHANNEL_SECRET`；**不得**給它 `LINE_CHANNEL_ACCESS_TOKEN`、Cloud Tasks 批量管理權或 Worker 的憑證。
- Worker 為 private IAM-authenticated Cloud Run Service，持有 LINE Channel Access Token、Firestore 最小權限、必要的 Cloud Tasks 發派權限。Cloud Tasks 和 Scheduler 若使用同一 Worker handler 的驗證器，需使用設定中同一受信任 Invoker Email 和正確 Audience，分離時應先更新雙 Actor 授權模型。
- Cloud Tasks enqueue 身分、Cloud Run Invoker、Secret Manager 存取身分各有最小權限；不可使用常駐 JSON service-account key。
- Firestore 必須先建立 `(state,reply_until)`、`(state,lease_until)`、`(state,next_dispatch_at)`、`(state,created_at)`、`(state,updated_at)` 所需複合索引，並建立 Inbox/Outbox `expires_at` TTL。確認 Index 為 Ready。
- **資料遷移**：既有 Pending 文件若缺 `next_dispatch_at`，以可重啟且不覆蓋更新的方式回填；以權威來源數據驗證總量與缺欄位數都正確後，才能切換新 Reconcile。不要在毫無 backfill 的現有資料庫直接切換。

## 驗收用例與必要證據

| Gate | 測試與失敗注入 | 成功證據（附 staging 資源與時間） |
|---|---|---|
| A IAM | 匿名呼叫 `/tasks/line-reply`、`/tasks/dispatch`；無授權 OIDC audience／email；合法 OIDC | 前三種拒絕，合法身分才通過，Cloud Run IAM 狀態與審核紀錄 |
| B 憑證隔離 | 檢查兩服務實際 Revision runtime env/Secret references（安全遮蔽）；以錯誤 `LINE_CHANNEL_ACCESS_TOKEN` 注入 Public dry-deploy | Webhook **無**發送 Token；錯誤配置 fail closed；Private Worker 能使用 Staging Token |
| C Firestore | 對 12+ Workers 競爭同 EventKey；插入斷線及交錯 Transaction | 一個 Inbox、一個 Outbox、最多一個 Sending Claim；拒絕重開 Sent／Unknown |
| D Redelivery | 同一 webhookEventId 的簽章合法重送（LINE 測試 Channel）；首次模擬入庫後 CreateTask 失敗 | Pending 可恢復且不覆寫原本的 Message；Sending/Sent/Unknown 不會第二次 LINE Reply |
| E Reply 時效 | 收到後 45 秒以上延遲；原始時間距今超過 20 分鐘的重送 | 不嘗試已過期 Reply；以原始 Event Timestamp 約束期限，結果標為 Expired |
| F Reconcile | 模擬 115+ Pending、過期 Sending、Queue CreateTask 暫時失敗、Worker Crash | 不會因固定前 100 筆飢餓；Unknown 不會自行重送；冷卻後可補派 Pending |
| G IAM/TTL/Index | 實際測試複合查詢並觀測 Index Ready，查詢 TTL policy、啟用 Scheduler | 查詢無 Missing Index，TTL 僅非同步移除；Scheduler 對私有 Worker 認證成功 |
| H 觀測／告警 | 隔離 Staging 執行 `ops events --check`、Inject Unknown 或長 Pending；停用一次 Scheduler | Job 非零 exit；Cloud Monitoring 分辨失敗和「未執行」，告警收件者收到並確認 |
| I Rollback | 在隔離測試環境以固定 image digest 回退已核准版本，保留資料結構向前相容 | 混合新舊 Worker 無重複發送，回復遺漏/延遲工作可解釋，簽核紀錄 |

## 觀測與安全判準

- Webhook 過期 Reply **不能變成無使用者授權的 Push**。一次 Claim 之後即使崩潰，仍只能形成 Unknown，不得自行重送；這是有意的「寧可少回覆、不重複回覆」取捨，而非 end-to-end exactly-once。
- Cloud Scheduler 的分鐘級間隔不能保證 45 秒 Reply 期限；需記錄 Expired 比率與原因，告警處理靠獨立授權的後續業務。
- Secret、Reply Token、完整 webhook body、LINE user ID、原始訊息與 OIDC bearer token 不得出現在操作輸出、Artifacts 或 Logging。
- 可觀測性訊號應有固定欄位、明確容量上限；不能以高 cardinality EventKey 作為 metric label。
- Firestore TTL 最終清理不可替代同步隱私 tombstone 或正式的使用者資料刪除流程。

## 執行後證據模板

```text
Staging Project / Region:
LINE 測試 Channel（僅代號，不提供 token）:
Webhook / Worker / Job image digests:
IAM reviewer and approval record:
Firestore index Ready evidence:
TTL policy evidence:
Cloud Scheduler OIDC path verified:
Event redelivery / queue retry / unknown fault-injection result:
Operator alert proof and escalation owner:
Rollback run/revision and data compatibility evidence:
Observed at (UTC) and run URL:
Unverified cases / blockers:
Approved for production? (default: NO):
```

**完成判準**：每個 Gate 有真實可定位的 Staging 證據、所有安全測試通過、已審核與簽署，才能變更本頁狀態。此文件不提供 Terraform apply／CreateTask／LINE Send 的自行授權。

參考：
- https://docs.cloud.google.com/run/docs/authenticating/service-to-service
- https://developers.line.biz/en/docs/messaging-api/receiving-messages/
- https://developers.line.biz/en/docs/messaging-api/retrying-api-request/
- https://docs.cloud.google.com/firestore/native/docs/ttl
