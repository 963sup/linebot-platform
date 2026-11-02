# Infrastructure Agent Rules

- SDK 版本依官方文件查證，新增後在 `docs/versions.md` 更新採用版本。
- 先用 ADC / Workload Identity／Secret Manager，避免提交長期憑證。
- 限制各 client timeout、重試範圍、連線池與 shutdown；不得重試不具冪等保護的發送。
- Google Cloud 服務優先使用 `google-cloud-go`；Google Workspace 類 API 使用 `google-api-go-client`。
- 日誌、trace、metrics 禁止攜帶 token、原始使用者內容或多餘個資。
