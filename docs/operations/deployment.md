# Google Cloud Build / Cloud Run（配置雛形）

目前僅提交 Cloud Build YAML、忽略規則和使用指引。**未設定 Google Cloud Project、Artifact Registry、IAM、Trigger 或 Cloud Run 服務，也未執行任何雲端建置或部署。** 本專案仍是開發用 MVP；持久 Inbox/Outbox、Cloud Tasks Worker 和重送去重的**可選程式碼**已加入，但尚未部署，未建立實際 IAM／Scheduler／Firestore TTL 及隔離 Staging 驗收。

## Stage 4 Worker 建置與安全要求

- `Dockerfile.worker` 和 `cloudbuild.worker.yaml` 可建置**獨立的私有 Worker 映像**，不建立 GCP 資源、部署 Cloud Run、開啟公開路由或觸發付費 API。
- Webhook 預設為 `LINE_EVENT_MODE=sync`；要使用 `durable` 必須事先準備可用 Firestore、Cloud Tasks Queue、私有 Cloud Run Worker、具 OIDC 的派送身分及 Cloud Scheduler。
- Worker **必須要求 IAM 認證、限制 Ingress、驗證 OIDC Audience 和發送服務帳號**。Firestore `expires_at` TTL 與可持續修復 Pending Outbox 的 Scheduler 應一起審核。
- 依 [Durable Worker 隔離部署契約](durable-worker.md) 與 [Firestore 資料保留](../../deploy/firestore/README.md) 實際準備 Staging 前，不可切換公開 Webhook 為 durable。
- 既有 `cloudbuild.deploy.yaml` 只對**既有私有 staging** 服務部署有效，且需要另外人工核准；本次不開啟任何 production 路徑。

## 兩份配置的分工

| 檔案 | 功能 | 行為 |
|---|---|---|
| `cloudbuild.yaml` | Go 編譯／測試 + 建立 webhook 映像 | 執行時會將映像推送至**已存在**的 Artifact Registry；**不部署 Cloud Run** |
| `cloudbuild.deploy.yaml` | 只部署既有 staging 服務 | 預設阻擋；需明確開啟、提供 `@sha256:` digest，且目標必須是現有 `*-staging` 服務。保留私有存取 |

## 上線前必須先準備

1. 建立獨立的 Google Cloud staging project，啟用 Cloud Build、Artifact Registry、Cloud Run API（費用與權限由擁有者管理）。
2. 預先建立 `asia-east1` 的 Docker Artifact Registry repository（預設 `linebot-platform`）。
3. 為建置與部署分別設定最小權限 service account／Cloud Build Trigger，使用 Cloud Logging logs；部署身分應限制到 staging 並設定審核。
4. 預先建立 staging Cloud Run service 與專用 runtime service account；任何 LINE 憑證只能透過 Secret Manager 管理，不能寫入 build substitutions 或 GitHub。
5. `Dockerfile` 與兩份 Cloud Build YAML 的 builder images 尚未固定 digest，必須在正式執行部署前完成供應鏈審查、映像 digest pin 與建置來源驗證。
6. 本範本部署的 staging 服務維持 **private**，LINE 平台不會自動有 webhook 呼叫權限。需要開放公開 webhook 時，要先完成驗證、持久化處理、IAM／流量設計，並獲得另外授權。

## 建置並發布映像（只有經授權才執行）

```bash
# 需已設定 gcloud 登入、專案及自訂最小權限建置帳號。
# 下列指令會產生可計費的 Cloud Build 執行和 Artifact Registry 儲存。
gcloud builds submit . \
  --project=YOUR_STAGING_PROJECT \
  --config=cloudbuild.yaml \
  --substitutions=_REGION=asia-east1,_AR_REPOSITORY=linebot-platform,_IMAGE_NAME=webhook
```

映像標籤使用 Cloud Build 內建的 `$BUILD_ID`（每次執行唯一），**不要使用 mutable tag 當作部署依據**。在 Artifact Registry 查詢該映像對應的 sha256 digest，再由審核人批准 deployment。

## 僅允許手動 staging 部署

```bash
# 注意：範例中的 DIGEST 必須更換成真實 sha256 摘要。
# 需要預先存在的私有 Cloud Run 服務和具有權限的 deployment SA。
gcloud builds submit . \
  --project=YOUR_STAGING_PROJECT \
  --config=cloudbuild.deploy.yaml \
  --substitutions="_APPROVE_STAGING=true,_REGION=asia-east1,_SERVICE=linebot-webhook-staging,_IMAGE_URI=asia-east1-docker.pkg.dev/YOUR_STAGING_PROJECT/linebot-platform/webhook@sha256:DIGEST"
```

`_APPROVE_STAGING=true` **只是防誤觸開關**，不等於 Google Cloud 的正式核准程序。請在 Cloud Build trigger 設定 Approval，並依最小權限 IAM 控管誰能啟動部署。這個檔案沒有 production 路徑，也不會建立 Cloud Run 服務或修改服務的 Secrets 設定。

## gcloudignore 與本機驗證

- `.gcloudignore` 引入 `.gitignore`，排除 `.git`、認證與本機檔案、Codex 配置、文件等非必要上傳內容。
- **不排除** `cmd/`、`internal/`、`tests/`、`go.mod`、`go.sum`、`Dockerfile`。
- 具備 gcloud CLI 後可執行 `gcloud meta list-files-for-upload` 核對將要上傳的來源，這個指令本身不會部署。

官方參考：
- [Cloud Build build config schema](https://docs.cloud.google.com/build/docs/build-config-file-schema)
- [Build container images](https://docs.cloud.google.com/build/docs/building/build-containers)
- [Cloud Run deployment](https://docs.cloud.google.com/run/docs/deploying)
- [Cloud Build service accounts](https://docs.cloud.google.com/build/docs/securing-builds/configure-user-specified-service-accounts)
- [gcloudignore](https://docs.cloud.google.com/sdk/gcloud/reference/topic/gcloudignore)
