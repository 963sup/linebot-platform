# Stage 5：Firestore Emulator、Staging 與 Codex 零先驗驗收

**Firestore Emulator 已在 GitHub Actions 的隔離本機容器實際驗證通過。** 真實 Staging、使用者 OAuth、Cloud Tasks OIDC、LINE 測試帳號及 Codex 新工作階段仍尚未實測。

## A. 本機離線驗收（不需真實 GCP 憑證）

**CI 自動執行無憑證測試，不需要手動設定環境變數。** GitHub Actions 的獨立 `firestore-emulator` Job 以 Digest 固定的官方容器綁定 Runner 本機 `127.0.0.1:8085`，執行 `go test -tags=integration -race -count=1` 並清理容器。[驗證紀錄 #41554849393](https://github.com/963sup/linebot-platform/actions/runs/41554849393)已通過兩項真實 SDK／Emulator 測試。

下列是**額外的本機手動重現方式**，才需自行準備 gcloud CLI／Firestore Emulator 及 Go 1.27.3。官方文件：[使用 Firestore Emulator](https://docs.cloud.google.com/firestore/native/docs/emulator)。

在第一個終端啟動只監聽本機的模擬器：

```bash
gcloud emulators firestore start --host-port=127.0.0.1:8085
```

另一個終端執行：

```bash
export FIRESTORE_EMULATOR_HOST=127.0.0.1:8085
make acceptance-emulator
```

此命令先執行 Codex Catalog／六情境驗證與 `make check`，再執行 `go test -tags=integration -race -count=1`。未設定或不是本地 Emulator Host 時**直接失敗**，不能將 skipped integration tests 視為成功。測試使用 `demo-linebot-platform-acceptance`、假 Reply Token，並嘗試清理各測試文件，不使用真實使用者資料。

Emulator Test 檢查 Firestore 原子 Record、重送不覆寫 Token、同時 Claim 至多一位勝出、終態清空 Token、Expired/Unknown 恢復，以及 TTL 欄位。Emulator 對並行交易、索引和正式版限制的行為**並不完全一致**，所以通過仍不代表 Staging 交易與 IAM 已驗證。

## B. 真實 Staging 驗證（需另行授權）

只在受控的**獨立 staging 專案**執行，不使用 production 資源或私人資料。依 [Durable Worker 契約](durable-worker.md) 準備最小權限雲端身分、Firestore TTL、私有 Cloud Run Worker 與 Cloud Scheduler。驗收項包括：

- LINE 測試帳號：正確簽章與目的地、重送去重及未授權請求拒絕。
- Firestore：真實交易衝突、同事件雙 Worker 搶占、終態不重送，以及正式環境索引／鎖定限制。
- Cloud Tasks：正確 OIDC 可呼叫私有 Worker，錯誤 Audience／Service Account 必須遭拒；多次交付不應重複傳送 LINE。
- Cloud Scheduler：重排 Pending、處理 Unknown／Expired，確認過期事件不會擅自轉為 Push；不保證每則訊息都能在 45 秒內回覆。
- 服務身分／TTL／隱私：確認 Firestore TTL、IAM 最小權限、私有 Ingress；日志與 Task Body 不含 Token、使用者 ID 或私人訊息。
- Google Tasks、Calendar 等仍是 `client_ready`，不能因 LINE Staging 通過就宣稱已取得使用者 OAuth。

**本 PR 不部署服務、不建立排程或雲端帳號，也不執行真實 LINE 發送**；上述驗收需要獨立的明確許可與隔離身分。

## C. Codex 全新工作階段

依 [六項零先驗驗收情境](../ai/zero-context-acceptance.md) 實際建立六個新工作階段，審查其 SDK 選擇、Method、Port、測試、權限與結果，再保留可審查的外部紀錄。CI **只**驗證情境清單和程式碼，不會代替真實 Codex 工作階段。

## D. 最終 Reviewer Evidence Gate

1. 複製 [範例](../acceptance/staging-evidence.example.json) 為**不提交 Git**的 `staging-evidence.json`。
2. 填寫**當前 PR HEAD** SHA、獨立 `*-staging` 專案名稱、審查者與近期 RFC3339 時間。
3. 所有檢查項必須真實完成且有可審查的 HTTPS 證據連結，才可以人工改成 `passed`；不可包含 Token、簽名 Query String 或憑證。
4. 在同一 Commit 工作樹執行：

   ```bash
   go run ./cmd/acceptance -evidence staging-evidence.json
   ```

   缺少證據、使用 production 專案、時間過舊、Commit 不一致或 URL 不安全，都會明確失敗。這只是**審查者供給證據的 Gate**，不會連線雲端確認真假。

5. 通過驗收 Gate、程式碼審查及明確核准後，才能考慮合併或後續部署。CI 與文件存在本身不構成權限或授權。

**狀態**：本階段已具備驗收工具和範例，但真實外部驗收仍須維持 `blocked`，不得把「測試準備完成」寫成「Staging 驗證通過」。
