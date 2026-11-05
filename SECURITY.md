# Security Policy

此儲存庫目前是具 Webhook、CLI、SDK adapters 與可選 durable event pipeline 程式碼的開發用 MVP。Repository 已具可測試的 production baseline，但尚未完成具授權的 Staging／Production 部署、IAM、OAuth、TTL、排程或真實 LINE Channel 驗收，因此不是正式運行服務。

## 安全議題回報

請勿在公開 Issue、PR 或討論中貼出 token、憑證、個資或可利用的漏洞細節。請先以私人管道聯絡儲存庫維護者；如儲存庫已啟用 GitHub 私密漏洞回報，則優先使用該功能。

## 開發原則

- 外部 SDK 與 GitHub Actions 使用經審查版本。
- Cloud Run 採最小權限與工作負載身分，不把長期金鑰提交到 GitHub。
- LINE webhook 必須驗證簽章、限制 body 大小、執行去重與重送保護。
- AI provider 與使用者訊息資料應有資料保留、遮蔽及刪除政策。
- 任何正式環境或付費操作須經維護者明確授權。
