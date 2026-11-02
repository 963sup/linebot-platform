# Deployment（尚未實作）

預計使用 Cloud Run（公開 webhook＋私有 worker）、Cloud Tasks、Firestore、Secret Manager、Workload Identity Federation 及 Terraform。

目前沒有可執行的部署腳本或雲端資源。目前已準備根目錄的 `cloudbuild.yaml`（只建置／推送映像）與 `cloudbuild.deploy.yaml`（手動 staging-only，預設禁止部署），但仍沒有已執行的建置、部署腳本或雲端資源。使用前必須閱讀 [部署操作文件](../docs/operations/deployment.md)。正式啟用須先具備可用程式、測試、IAM 設計及維護者明確授權。
