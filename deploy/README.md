# Deployment（設定範本已建立；尚未部署）

平台程式碼已具備 `cmd/webhook`、`cmd/worker`、獨立 `cmd/miniapp`、Firestore Inbox/Outbox、Cloud Tasks Queue 與 OIDC 驗證流程。根目錄提供單一 `Dockerfile`、`cloudbuild.yaml` 與受人工限制的 `cloudbuild.deploy.yaml`，供建置三個映像及經核准的隔離 Staging 部署。MINI App runtime 不持有 Messaging Secret、Token 或 Cloud Tasks 權限；Attendance 只使用 Firestore 與專用位置加密 key 的 Secret-level access。

**上述是可執行程式碼與建置／部署範本，不表示已建立或執行 Google Cloud Project、Artifact Registry、IAM、Cloud Tasks、Firestore TTL、Cloud Scheduler 或 Cloud Run 資源。** Terraform 仍只有預留骨架，沒有已驗證的基礎設施即程式碼（IaC）部署。

正式部署須先配置最小權限 IAM／OIDC、Secrets、Firestore TTL、背景排程、回滾及 Staging 驗收，再依 [部署操作](../docs/operations/deployment.md)、[Durable Worker](../docs/operations/durable-worker.md) 與 [main 基礎設施稽核](../docs/architecture/infrastructure-status.md) 審核。任何部署或付費操作都需明確授權。
