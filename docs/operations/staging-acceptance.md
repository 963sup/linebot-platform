# 隔離 Staging 驗收：LINE + Google Cloud Run + Firestore + Cloud Tasks

**狀態：尚未執行。** 此頁是可重現的人工驗收契約，不建立資源、不預設已有 Google Cloud 專案／LINE Sandbox 身分。本儲存庫未獲得對真實 GCP/LINE 的授權，CI Emulator 通過不等於本頁通過。

## 前置與准入

- 由擁有者指定單獨計費預算／IAM 的 Staging GCP Project、區域、Firestore database、Cloud Tasks Queue 與 LINE **測試** Channel。不得複用 Production token 或 Secret。
- 使用已核准的 image digest，僅部署到明確 Staging Service；在實際索引/TTL/IAM 完成前保持入口停用。
- Webhook 公開 HTTPS 路由僅接收 `/webhook/line`，使用 `LINE_EVENT_MODE=durable`、`LINE_CHANNEL_SECRET`；**不得**給它 `LINE_CHANNEL_ACCESS_TOKEN`、Cloud Tasks 批量管理權或 Worker 的憑證。
- Worker 為 private IAM-authenticated Cloud Run Service，持有 LINE Channel Access Token、Firestore 最小權限、必要的 Cloud Tasks 發派權限。Cloud Tasks 和 Scheduler 若使用同一 Worker handler 的驗證器，需使用設定中同一受信任 Invoker Email 和正確 Audience，分離時應先更新雙 Actor 授權模型。
- MINI App 為獨立公開 Cloud Run Service；它不得持有 `LINE_CHANNEL_SECRET`、`LINE_CHANNEL_ACCESS_TOKEN`、Secret Manager 或 Cloud Tasks 權限，只取得 Attendance Firestore 存取。Developing、Review 與 Published internal channel 各自使用對應的 LIFF ID；本輪只能使用 Staging／Developing 設定。
- Cloud Tasks enqueue 身分、Cloud Run Invoker、Secret Manager 存取身分各有最小權限；不可使用常駐 JSON service-account key。
- Firestore 必須先建立 LINE `line_outbox_v1` 的 `(state,reply_until)`、`(state,lease_until)`、`(state,next_dispatch_at)`、`(state,created_at)`、`(state,updated_at)` 複合索引，以及 Identity `identity_outbox_v1` 的 `(state,lease_until)`、`(state,next_dispatch_at)` 複合索引。為 `line_inbox_v1`、`line_outbox_v1` 與 `identity_outbox_v1` 建立經 retention/privacy 核准的 `expires_at` TTL，並確認所有 Index 為 Ready。
- **資料遷移**：既有 Pending 文件若缺 `next_dispatch_at`，以可重啟且不覆蓋更新的方式回填；以權威來源數據驗證總量與缺欄位數都正確後，才能切換新 Reconcile。不要在毫無 backfill 的現有資料庫直接切換。

## 驗收用例與必要證據

| Gate | 測試與失敗注入 | 成功證據（附 staging 資源與時間） |
|---|---|---|
| A IAM | 匿名呼叫 `/tasks/v2/line-reply`、`/tasks/v2/dispatch`；無授權 OIDC audience／email；合法 OIDC | 前三種拒絕，合法身分才通過，Cloud Run IAM 狀態與審核紀錄 |
| B 憑證隔離 | 檢查兩服務實際 Revision runtime env/Secret references（安全遮蔽）；以錯誤 `LINE_CHANNEL_ACCESS_TOKEN` 注入 Public dry-deploy | Webhook **無**發送 Token；錯誤配置 fail closed；Private Worker 能使用 Staging Token |
| C Firestore | 對 12+ Workers 競爭同 EventKey；插入斷線及交錯 Transaction | 一個 Inbox、一個 Outbox、最多一個 Sending Claim；拒絕重開 Sent／Unknown |
| C2 Worker capacity | 對 private Worker 發出至少 17 個同步阻塞請求，保持前 16 個 handler in-flight，再送第 17 個；其後釋放阻塞並啟動受控 shutdown | 最多 16 個進入 handler；第 17 個立即收到 `503 Service Unavailable` 與有效 `Retry-After`；Cloud Tasks 依該回應稍後重試而非程序內排隊；shutdown 停止新 admission 並等待既有 handler drain |
| D Redelivery | 同一 webhookEventId 的簽章合法重送（LINE 測試 Channel）；首次模擬入庫後 CreateTask 失敗 | Pending 可恢復且不覆寫原本的 Message；Sending/Sent/Unknown 不會第二次 LINE Reply |
| E Reply 時效 | 收到後 45 秒以上延遲；原始時間距今超過 20 分鐘的重送 | 不嘗試已過期 Reply；以原始 Event Timestamp 約束期限，結果標為 Expired |
| F Reconcile | 模擬 115+ Pending、過期 Sending、Queue CreateTask 暫時失敗、Worker Crash | 不會因固定前 100 筆飢餓；Unknown 不會自行重送；冷卻後可補派 Pending |
| G IAM/TTL/Index | 實際測試複合查詢並觀測 Index Ready，查詢 TTL policy、啟用 Scheduler | 查詢無 Missing Index，TTL 僅非同步移除；Scheduler 對私有 Worker 認證成功 |
| G2 Identity Outbox | 查詢 `identity_outbox_v1` due pending 與 expired publishing lease；競爭 claim、完成 publish 與 lease recovery | 兩個 Identity 複合索引均為 Ready；`expires_at` TTL policy 可定位；published 不會被 recovery 降級，單一 lease 不會重複派送 |
| H 觀測／告警 | 隔離 Staging 執行 `ops events --check`、Inject Unknown 或長 Pending；停用一次 Scheduler | Job 非零 exit；Cloud Monitoring 分辨失敗和「未執行」，告警收件者收到並確認 |
| I Rollback | 在隔離測試環境以固定 image digest 回退已核准版本，保留資料結構向前相容 | 混合新舊 Worker 無重複發送，回復遺漏/延遲工作可解釋，簽核紀錄 |
| M1 MINI App identity/enrollment | 從 LINE 內建瀏覽器與外部瀏覽器開啟；測試有效、過期、wrong-audience與竄改 ID token；同一註冊碼重放與不同 subject 競爭 | 只有官方驗證成功且 audience 正確的 token 可建立 opaque ParticipantID；註冊碼單次消耗；token、subject與註冊碼不出現在 Logging |
| M2 Attendance concurrency | 對同 participant/workday/kind 並發與 response-lost retry；先 clock-out | 只建立一筆 Punch；重送回同結果；clock-out 必須晚於 clock-in，Firestore 暫時失敗不宣稱成功 |
| M3 Geolocation/privacy | iOS／Android、Safari／Chrome測試允許、拒絕、撤銷、position unavailable與timeout；確認 Attendance 專用 database／等效 IAM boundary 與 GPS retention policy | 只在明確 gesture 後單次定位；座標不回應、不進 URL/log/trace/analytics/map；專用存取邊界與 retention/deletion evidence 完整，否則 Production 維持 `BLOCKED` |
| M4 Workday boundary | 在 Asia/Taipei 日期邊界前後打卡並嘗試由 client 指定時間/日期 | Workday 與 recorded time 僅由 server time 決定，client 無法覆寫 |

**C2 目前為 `BLOCKED`。** Repository 只有單元測試用 blocking fake，沒有可部署的 staging-only blocking handler 或 fault-injection image，因此目前不能確定性地保持及釋放 16 個真實 handler。執行 C2 前必須另行審核一個與 production route/image 分離、固定 digest、只允許隔離 Staging IAM 呼叫的測試 fixture；證據還必須包含實際 Cloud Run Revision 的 `containerConcurrency=16`。不得把測試 blocking route 編入 production image，也不得用真實 LINE 傳送來模擬阻塞。

## 觀測與安全判準

- Webhook 過期 Reply **不能變成無使用者授權的 Push**。一次 Claim 之後即使崩潰，仍只能形成 Unknown，不得自行重送；這是有意的「寧可少回覆、不重複回覆」取捨，而非 end-to-end exactly-once。
- Cloud Scheduler 的分鐘級間隔不能保證 45 秒 Reply 期限；需記錄 Expired 比率與原因，告警處理靠獨立授權的後續業務。
- Secret、Reply Token、完整 webhook body、LINE user ID、原始訊息與 OIDC bearer token 不得出現在操作輸出、Artifacts 或 Logging。
- MINI App ID token、LINE subject與經緯度／accuracy同樣不得出現在操作輸出、Artifacts、Logging、trace或metric attributes；瀏覽器座標可被模擬，不能作為到場、資格、反詐或安全證明。
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
Identity outbox index / TTL evidence:
Cloud Scheduler OIDC path verified:
Event redelivery / queue retry / unknown fault-injection result:
Operator alert proof and escalation owner:
Rollback run/revision and data compatibility evidence:
Observed at (UTC) and run URL:
Unverified cases / blockers:
Approved for production? (default: NO):
Evidence schema version: 1
Public / private immutable image digests:
MINI App immutable image digest / authoritative HTTPS origin:
LIFF internal channel class, LIFF ID and Login Channel ID (Developing only; no token):
MINI App enrollment / attendance concurrency / geolocation / workday gate evidence:
Reviewer identity and approval record:
Promotion approval expires at (UTC):
Evidence object URI and SHA-256:
```

**完成判準**：每個 Gate 有真實可定位的 Staging 證據、所有安全測試通過、已審核與簽署，才能變更本頁狀態。簽署 evidence 必須同時綁定三個 immutable image digest、MINI App HTTPS origin、LIFF URL、MINI App Channel ID 與 Firestore database；Production preflight 逐值比對，禁止重用其它 LIFF／origin 的證據。此文件不提供 Terraform apply／CreateTask／LINE Send 的自行授權。

參考：
- https://docs.cloud.google.com/run/docs/authenticating/service-to-service
- https://developers.line.biz/en/docs/messaging-api/receiving-messages/
- https://developers.line.biz/en/docs/messaging-api/retrying-api-request/
- https://docs.cloud.google.com/firestore/native/docs/ttl
