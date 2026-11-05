# Outbox 監控、告警與安全處置（規劃中的 Staging 工具）

本文件描述 `cmd/ops events` 的**只讀**稽核工具。PR 中的程式碼尚未部署；所有 Cloud Monitoring、Scheduler、Firestore 權限或 LINE 服務皆需獨立的隔離 Staging 核准。不得將本工具當成已連線 Production 或具有修復發送結果的能力。

## 操作前權限

1. 使用獨立 Staging 專案、最小權限 Firestore **讀取**身分（建議另立 operator/service account）；不提供 LINE Channel Token。
2. 設定 `LINEBOT_STAGING_PROJECT` 作為顯式 allowlist，且所有 `--project` 必須相符。
3. 只有 `--inspect` 或 `--check` 可以開啟查詢；沒有這些旗標時不會讀取 Firestore。
4. `--check` 不輸出可關聯到事件的 ID；`--inspect` 只顯示 12 字元 opaque Key Hint、狀態與更新時間，**不會輸出** Reply Token、使用者內容或 LINE ID。
5. 工具中的 Count 來自不同時間點的獨立 Aggregation Query，屬操作快照，不提供跨狀態強一致全域交易。

## 使用方式（需在已授權環境）

```sh
export LINEBOT_STAGING_PROJECT='<APPROVED_STAGING_PROJECT>'

# 僅讀取 Unknown 的 20 筆去識別 metadata（明確限定 Staging）
go run ./cmd/ops events --project "$LINEBOT_STAGING_PROJECT" --inspect --state unknown --limit 20

# 排程告警使用，標準輸出 JSON 無事件鍵；不符合閾值時退出非零
go run ./cmd/ops events --project "$LINEBOT_STAGING_PROJECT" --check \
  --max-pending-age 2m --max-expired 20
```

`--check` 預設在 **Unknown > 0** 或 **最舊 Pending > 2 分鐘** 回傳失敗；`--max-expired` 預設 -1（停用），因為 Expired 會在保留時間內持續存在，不應無條件觸發每分鐘告警。超過閾值時的非零 Exit Code 可作為**已審核、定期執行的 Cloud Run Job** 結束狀態，由 Cloud Monitoring 監控 `run.googleapis.com/job/completed_execution_count` 的失敗結果。請在實際 Staging 查核該 metric 的 labels 與失敗值；此外須另監控 Job **停止執行**、Scheduler 失敗與 Cloud Tasks Queue depth（單靠「執行失敗」無法偵測監控工作未啟動）。

### 狀態與操作判斷

| 狀態 | 意義 | 允許的安全處置 |
|---|---|---|
| `pending` | 尚未 claim、等待或等候 Reconcile | 檢查 `next_dispatch_at`、Queue/IAM、定期 Scheduler；僅使用既有經驗證的私有 `/tasks/v2/dispatch` 修復。 |
| `sending` | 已發生單次 Claim；LINE 請求結果仍可能不明 | 檢查 Lease 與 Worker 狀態，**不得**人工再發送；到期由 Reconcile 轉 Unknown。 |
| `sent` | 本地記錄 LINE API 已接受 Reply | 不等於使用者實際收到；本工具不需要處理。 |
| `expired` | Reply 無法在有效期限內安全嘗試 | 當作未回覆的操作事件分析；若要 Push，應另建立有使用者授權及新業務身份的發送流程。 |
| `unknown` | 外部 LINE 可能已接受，但回覆／持久結果不確定 | **永不自動重送**；以時間、trace、LINE 官方可查證證據判斷並進入人工例外流程。 |

**禁止提供 `ops events --retry-unknown` 之類操作。** 對不能由 LINE 保證冪等的 Reply，「重送可恢復」的界線只到尚未 Claim 的 Pending／Expired；一旦已 Claim，重送只能讀取現有狀態。

## 建議告警與操作可觀測性

- 每 5 分鐘經隔離 Staging 身分執行一次 `ops events --check`，用 Job failed execution 與 no-execution 雙告警覆蓋卡死、權限失效、查詢失敗。
- Worker /tasks/v2/line-reply 與 /tasks/v2/dispatch 的 5xx、OIDC 401/403（頻率上升）、Cloud Tasks Queue depth 與 dispatch latency 皆需 Cloud Monitoring dashboard。
- 不要將每筆事件 Key、Reply Token、User ID、原始 HTTP Body 加入 metrics label 或 structured logs。
- Alarm 的 threshold 經真實測試先調整，避免一個 Expired 狀態在 TTL 期間造成持續告警；高流量時需看速率／分位，而非只看 Count。
- 定期 `Inspect` 讀取會產生 Firestore query 成本，按實際規模調整頻率；若需要高頻度與高精度度量，再評估安全且 bounded 的 metrics pipeline，不以無界掃描達成。

## Staging / Production 限制

本工具需在 **Firestore composite indexes 已套用、所有 Pending 已 backfill `next_dispatch_at`、TTL 與 IAM 正確配置** 後才可作為 Production gate。CI Firestore Emulator **不會檢查真正的索引啟用狀態**。對報警規則的生效、Cloud Run Job 的確切指標，以及每一項操作，須另有獨立的授權 Staging 實測報告。

延伸：[Staging 驗收契約](staging-acceptance.md)、[Durable Worker](durable-worker.md)、[Firestore Indexes](../../deploy/firestore/README.md)。

資料來源：
- https://docs.cloud.google.com/run/docs/monitoring
- https://docs.cloud.google.com/monitoring/api/metrics_gcp_p_z
- https://pkg.go.dev/cloud.google.com/go/firestore@v1.29.2
