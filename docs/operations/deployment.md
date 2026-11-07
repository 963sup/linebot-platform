# Google Cloud Build / Cloud Run 部署

**Not Verified / 尚未部署驗證：** 本頁是預期的 production 部署契約與人工操作範本，不表示下列資源、IAM、Revision 或外部服務已存在、可用或通過 Staging／Production 驗收。執行任何會建立或更新付費資源的命令，仍須取得環境擁有者的明確授權並保留驗證證據。

此專案預計使用 Google Cloud Run。經授權部署時，目標名稱固定如下：

| 資源 | 值 |
| --- | --- |
| Google Cloud project | `bubbly-area-513305-s0`（Line-Bot） |
| Region | `asia-east1` |
| Artifact Registry | `line-bot` |
| Public webhook service | `line-bot-public` |
| Private worker service | `line-bot-private` |
| Public MINI App service | `line-miniapp` |
| Firestore database | `(default)` |
| Cloud Tasks queue | `line-bot-events` |

## 建置與部署身分

Cloud Build 預期使用 `line-bot-build@bubbly-area-513305-s0.iam.gserviceaccount.com`。部署前必須驗證它只具備限定於建置來源 bucket 的讀取權、`line-bot` Artifact Registry repository 的寫入權，以及 Cloud Build、Cloud Logging、Cloud Run 與 runtime service account 代管所需的最小權限。

Cloud Run 預期分別使用 `line-bot-public@bubbly-area-513305-s0.iam.gserviceaccount.com`、`line-bot-worker@bubbly-area-513305-s0.iam.gserviceaccount.com` 與 `line-miniapp@bubbly-area-513305-s0.iam.gserviceaccount.com`；不得共用廣權限 runtime 身分。Public 只能讀取 Channel Secret、寫入必要 Firestore 文件並向指定 Queue enqueue；Worker 只能讀取 Channel Access Token、執行必要 Firestore 狀態轉移與 reconcile enqueue；MINI App 只取得 Attendance 所需 Firestore 權限，不得讀取任何 Secret Manager secret 或使用 Cloud Tasks。Cloud Tasks／Scheduler 使用 `line-bot-invoker@bubbly-area-513305-s0.iam.gserviceaccount.com` 呼叫 Worker。部署前還須驗證 Cloud Tasks service agent 能為 invoker 建立 OIDC token，且 private worker 只授予該 invoker `roles/run.invoker`。這些 IAM 敘述是目標狀態，不是目前環境已驗證的事實。

## Runtime 設定與 Secret 分離

環境變數名稱、角色與驗證規則的唯一來源是 [Runtime Configuration](configuration.md) 與 `internal/bootstrap/config_contract.go`。`.env.example` 包含完整非敏感範本；程式不會自動讀取該檔案。

| 設定位置 | 項目 | 用途 |
| --- | --- | --- |
| Codex Cloud secure environment | `GITHUB_MCP_TOKEN` | GitHub MCP（依使用者授權可讀寫），僅可送至 `api.githubcopilot.com`；不屬於 Cloud Run runtime。 |
| Google Secret Manager | `LINE_CHANNEL_SECRET` | 僅注入 `line-bot-public`。 |
| Google Secret Manager | `LINE_CHANNEL_ACCESS_TOKEN` | 僅注入 `line-bot-private`。 |
| MINI App Cloud Run | `LINE_MINI_APP_CHANNEL_ID`、`LINE_MINI_APP_LIFF_URL`、`LINE_MINI_APP_ENDPOINT_URL`、`FIRESTORE_DATABASE` | LINE MINI App audience、LIFF、同源與 Attendance Firestore 契約；Project ID 由 ADC 自動發現。 |
| Cloud Run deployment | `APP_ENV`、`LINE_EVENT_MODE`、`LINE_API_TIMEOUT`、`LOG_LEVEL`、`LOG_FORMAT`、`LINE_BOT_DESTINATION`、`GCP_PROJECT_ID`、`TASK_QUEUE_NAME`、`TASK_WORKER_URL`、`TASK_OIDC_AUDIENCE`、`TASK_INVOKER_EMAIL` | 非敏感 production 設定；由 `cloudbuild.deploy.yaml` 寫入。 |
| Cloud Run platform | `PORT`、`K_SERVICE` | 由平台注入，不手動設定。 |
| 操作人員 shell | `LINEBOT_STAGING_PROJECT` | 只有隔離 staging 的 `ops events` 使用；不注入任一服務。 |

- 經授權部署的 `line-bot-public` 必須使用 `APP_ENV=production`、`LINE_EVENT_MODE=durable` 與 JSON logs；它只可取得 `LINE_CHANNEL_SECRET` 的 Secret Manager reference，不得取得 Access Token。
- 經授權部署的 `line-bot-private` 必須使用相同 durable、Firestore、Cloud Tasks 與 OIDC 設定；它只可取得 `LINE_CHANNEL_ACCESS_TOKEN` 的 Secret Manager reference，不得取得 Channel Secret。
- `line-bot-private` 的目標 Cloud Run concurrency 為 16，與應用程式的有界 request admission cap 一致。超過容量的 task 應收到 `503 Service Unavailable` 與 `Retry-After`，交由 Cloud Tasks 稍後重試，不在程序內建立無界 backlog；實際平台設定與行為仍須依 Staging capacity gate 驗證。
- `line-miniapp` 公開提供 embedded assets，但 Attendance API 必須驗證 LINE ID token 與員工註冊。它不得取得 Messaging Secret、Secret Manager 或 Cloud Tasks IAM；GPS fix 是私有 Attendance 資料，不得回傳、記錄到 logs 或作為本人實體在場證明。
- `K_SERVICE` 由 Cloud Run 自動注入，程式以此拒絕套用 local profile。
- Public/Worker 上錯置的 `FIRESTORE_DATABASE`，以及各服務不屬於其角色的 `GOOGLE_CLOUD_PROJECT`、`OPENAI_MODEL`、`AUTO_TASKS`、`AUTO_CALENDAR`、`CALENDAR_SERVICE_ACCOUNT`、`WORKER_MODE` 與 Secret，會在部署時移除；MINI App 保留自己的明確 Attendance database 設定。

目標設定中，`line-bot-public` 以 Cloud Run 的 `--no-invoker-iam-check` 接受未驗證的 LINE webhook 請求；簽章由應用程式使用原始 request body 驗證。`line-bot-private` 必須維持 IAM 驗證，並驗證 Cloud Tasks OIDC audience 和 invoker service account。`line-miniapp` 必須允許使用者下載頁面，但 Attendance API 另以 LINE ID token、Origin、註冊與班次資格驗證。三者的 live 行為均尚未由本儲存庫驗證。

## 固定映像來源

單一 `Dockerfile` 與兩份 Cloud Build configuration 的基礎／builder images 均固定為 sha256 digest。部署流程只接受 Artifact Registry 的 `@sha256:` image reference；不可使用 mutable tag 部署。

## 驗證

本機只驗證設定及程式碼，不會建立雲端資源：

```bash
go run ./cmd/configcheck --check
go run ./cmd/codexcheck
go test ./tests/contract -count=1
```

## 建置 webhook、worker 與 MINI App 映像

`cloudbuild.yaml` 只執行一次 Go module 驗證、gofmt、vet 與 race test，並在共享的 `/workspace` 產出三個 binary；成功後三個 Docker step 只封裝 distroless runtime 並行推送，不重複下載 Go modules 或編譯。CPU 密集的 race build 使用 Cloud Build `E2_HIGHCPU_8` worker。

```bash
gcloud builds submit . \
  --project=bubbly-area-513305-s0 \
  --config=cloudbuild.yaml \
  --substitutions=_REGION=asia-east1,_AR_REPOSITORY=line-bot
```

查詢每個 image 的 sha256 digest 後，才可進行部署。

## 部署目標服務

`cloudbuild.deploy.yaml` 可建立或更新三個 Cloud Run service，因此只能在明確授權的目標環境執行。它預設拒絕部署與流量 promotion。唯讀 preflight 會核對 project、Queue、Firestore database、分離的服務身分、Messaging Secrets、三個不可變映像，以及經 SHA-256 固定且明確核准 production promotion 的 Staging evidence；evidence 必須逐值綁定 MINI App endpoint、LIFF URL、MINI App Channel ID 與 database。這些檢查仍不取代 live IAM、LINE、Firestore、LIFF 或 Geolocation 驗收。MINI App endpoint 可使用 Cloud Run URL 或已核准的 custom domain；實際綁定由 staging evidence 證明，preflight 不得因 service 尚未建立而阻塞首次 no-traffic candidate。

部署 build 只建立帶 `candidate` tag 的 no-traffic revisions，並輸出本次 `latestCreatedRevisionName`；它不做 smoke test，也不切換流量。Private Worker 顯式使用 internal ingress、IAM authentication、16 concurrency 與獨立 Worker 身分。`TASK_WORKER_URL` 與 OIDC audience 來自必填 `_TASK_WORKER_ORIGIN`，preflight 必須確認它等於該 Worker service 的權威 Cloud Run URL。

候選 revision 必須由獨立、已授權的 Staging／Production smoke 流程驗證。之後才可使用 `deploy/scripts/promote.sh`；每次呼叫只處理**單一 service**，並要求未過期的核准 evidence 精確綁定 project、region、service、candidate/base revision 與單一 `10`、`50` 或 `100` 百分比。腳本不讀取 `latestReady`、不自動等待觀測時間，也不連續跨 service 或多個階段；操作人員須在每次呼叫後檢查告警、Queue、Unknown/Pending 與 rollback 條件，再為下一個 service／階段建立新的 evidence。Public Webhook、Worker、Task Invoker 與 Deployer 的最小權限骨架位於 `deploy/terraform/`；Terraform 不由 CI apply，實際 IAM、Secret-level denial、Monitoring notification 與 state policy 都需環境擁有者另行審核。

```bash
gcloud builds submit . \
  --project=bubbly-area-513305-s0 \
  --config=cloudbuild.deploy.yaml \
  --substitutions="_APPROVE_DEPLOYMENT=true,_STAGING_EVIDENCE_URI=gs://APPROVED_BUCKET/evidence.json,_STAGING_EVIDENCE_SHA256=EVIDENCE_SHA256,_TASK_WORKER_ORIGIN=https://AUTHORIZED_WORKER_ORIGIN,_LINE_BOT_DESTINATION=YOUR_LINE_BOT_DESTINATION,_PUBLIC_IMAGE_URI=asia-east1-docker.pkg.dev/bubbly-area-513305-s0/line-bot/line-bot-public@sha256:PUBLIC_DIGEST,_PRIVATE_IMAGE_URI=asia-east1-docker.pkg.dev/bubbly-area-513305-s0/line-bot/line-bot-private@sha256:PRIVATE_DIGEST,_MINIAPP_IMAGE_URI=asia-east1-docker.pkg.dev/bubbly-area-513305-s0/line-bot/line-miniapp@sha256:MINIAPP_DIGEST,_LINE_MINI_APP_CHANNEL_ID=YOUR_MINI_APP_CHANNEL_ID,_LINE_MINI_APP_LIFF_URL=https://miniapp.line.me/YOUR_LIFF_ID,_LINE_MINI_APP_ENDPOINT_URL=https://YOUR_MINIAPP_ORIGIN/miniapp/"
```

完成後應檢查三個 revision 的 Ready condition、public webhook 的 LINE 簽章拒絕行為、private worker 對匿名請求與錯誤 OIDC audience 的拒絕行為，以及 MINI App 對無效 ID token、跨來源、註冊碼重放、today 工作日邊界、並發／response-lost Punch、Geolocation 拒絕／timeout、座標不出現在 response／URL／log 的處理。真實 LINE MINI App、裝置、Firestore IAM 與 Attendance GPS retention policy 完成前維持 `Not Verified`。
