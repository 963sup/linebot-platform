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
| Firestore database | `(default)` |
| Cloud Tasks queue | `line-bot-events` |

## 建置與部署身分

Cloud Build 預期使用 `line-bot-build@bubbly-area-513305-s0.iam.gserviceaccount.com`。部署前必須驗證它只具備限定於建置來源 bucket 的讀取權、`line-bot` Artifact Registry repository 的寫入權，以及 Cloud Build、Cloud Logging、Cloud Run 與 runtime service account 代管所需的最小權限。

Cloud Run 預期使用 `line-bot-runtime@bubbly-area-513305-s0.iam.gserviceaccount.com`。部署前必須驗證它只能讀寫指定 Firestore database、將事件寫入 Cloud Tasks，並以 `line-bot-scheduler@bubbly-area-513305-s0.iam.gserviceaccount.com` 建立 OIDC Task。另須驗證 Cloud Tasks service agent 能為此 invoker service account 建立 OIDC token，且 private worker 只授予該 invoker `roles/run.invoker`。這些 IAM 敘述是目標狀態，不是目前環境已驗證的事實。

## Runtime 設定與 Secret 分離

環境變數名稱、角色與驗證規則的唯一來源是 [Runtime Configuration](configuration.md) 與 `internal/bootstrap/config_contract.go`。`.env.example` 包含完整非敏感範本；程式不會自動讀取該檔案。

| 設定位置 | 項目 | 用途 |
| --- | --- | --- |
| Codex Cloud secure environment | `GITHUB_MCP_TOKEN` | GitHub MCP（依使用者授權可讀寫），僅可送至 `api.githubcopilot.com`；不屬於 Cloud Run runtime。 |
| Google Secret Manager | `LINE_CHANNEL_SECRET` | 僅注入 `line-bot-public`。 |
| Google Secret Manager | `LINE_CHANNEL_ACCESS_TOKEN` | 僅注入 `line-bot-private`。 |
| Cloud Run deployment | `APP_ENV`、`LINE_EVENT_MODE`、`LINE_API_TIMEOUT`、`LOG_LEVEL`、`LOG_FORMAT`、`LINE_BOT_DESTINATION`、`GCP_PROJECT_ID`、`TASK_QUEUE_NAME`、`TASK_WORKER_URL`、`TASK_OIDC_AUDIENCE`、`TASK_INVOKER_EMAIL` | 非敏感 production 設定；由 `cloudbuild.deploy.yaml` 寫入。 |
| Cloud Run platform | `PORT`、`K_SERVICE` | 由平台注入，不手動設定。 |
| 操作人員 shell | `LINEBOT_STAGING_PROJECT` | 只有隔離 staging 的 `ops events` 使用；不注入任一服務。 |

- 經授權部署的 `line-bot-public` 必須使用 `APP_ENV=production`、`LINE_EVENT_MODE=durable` 與 JSON logs；它只可取得 `LINE_CHANNEL_SECRET` 的 Secret Manager reference，不得取得 Access Token。
- 經授權部署的 `line-bot-private` 必須使用相同 durable、Firestore、Cloud Tasks 與 OIDC 設定；它只可取得 `LINE_CHANNEL_ACCESS_TOKEN` 的 Secret Manager reference，不得取得 Channel Secret。
- `line-bot-private` 的目標 Cloud Run concurrency 為 16，與應用程式的有界 request admission cap 一致。超過容量的 task 應收到 `503 Service Unavailable` 與 `Retry-After`，交由 Cloud Tasks 稍後重試，不在程序內建立無界 backlog；實際平台設定與行為仍須依 Staging capacity gate 驗證。
- `K_SERVICE` 由 Cloud Run 自動注入，程式以此拒絕套用 local profile。
- 既有服務的 `GOOGLE_CLOUD_PROJECT`、`FIRESTORE_DATABASE`、`OPENAI_MODEL`、`AUTO_TASKS`、`AUTO_CALENDAR`、`CALENDAR_SERVICE_ACCOUNT`、`WORKER_MODE` 及錯置 Secret 會在部署時移除。

目標設定中，`line-bot-public` 以 Cloud Run 的 `--no-invoker-iam-check` 接受未驗證的 LINE webhook 請求；簽章由應用程式使用原始 request body 驗證。`line-bot-private` 必須維持 IAM 驗證，並驗證 Cloud Tasks OIDC audience 和 invoker service account。兩者的 live 行為均尚未由本儲存庫驗證。

## 固定映像來源

單一 `Dockerfile` 與兩份 Cloud Build configuration 的基礎／builder images 均固定為 sha256 digest。部署流程只接受 Artifact Registry 的 `@sha256:` image reference；不可使用 mutable tag 部署。

## 驗證

本機只驗證設定及程式碼，不會建立雲端資源：

```bash
go run ./cmd/configcheck --check
go run ./cmd/codexcheck
go test ./tests/contract -count=1
```

## 建置 public 與 private 映像

`cloudbuild.yaml` 只執行一次 Go module 驗證、gofmt、vet 與 race test，並在共享的 `/workspace` 產出兩個 binary；成功後兩個 Docker step 只封裝 distroless runtime 並行推送，不重複下載 Go modules 或編譯。CPU 密集的 race build 使用 Cloud Build `E2_HIGHCPU_8` worker。

```bash
gcloud builds submit . \
  --project=bubbly-area-513305-s0 \
  --config=cloudbuild.yaml \
  --substitutions=_REGION=asia-east1,_AR_REPOSITORY=line-bot
```

查詢每個 image 的 sha256 digest 後，才可進行部署。

## 部署目標服務

`cloudbuild.deploy.yaml` 可建立或更新兩個 Cloud Run service，因此只能在明確授權的目標環境執行。它預設拒絕執行，並在部署前確認 project、Queue、Secrets 和兩個不可變映像均存在；這組 preflight 結果 does not imply production readiness，也不驗證 IAM、LINE、Firestore 或 Cloud Tasks 的 live 行為。

```bash
gcloud builds submit . \
  --project=bubbly-area-513305-s0 \
  --config=cloudbuild.deploy.yaml \
  --substitutions="_APPROVE_DEPLOYMENT=true,_LINE_BOT_DESTINATION=YOUR_LINE_BOT_DESTINATION,_PUBLIC_IMAGE_URI=asia-east1-docker.pkg.dev/bubbly-area-513305-s0/line-bot/line-bot-public@sha256:PUBLIC_DIGEST,_PRIVATE_IMAGE_URI=asia-east1-docker.pkg.dev/bubbly-area-513305-s0/line-bot/line-bot-private@sha256:PRIVATE_DIGEST"
```

完成後應檢查兩個 revision 的 Ready condition、public webhook 的 LINE 簽章拒絕行為，以及 private worker 對匿名請求與錯誤 OIDC audience 的拒絕行為。
