# GitHub Workflow Agent Rules

- Go HTTP 工程與 CI 已建立；每次 SDK 變更需核對 `go mod tidy` 的 `go.mod`／`go.sum`，並經 Actions build/test 實際驗證，不能以 YAML 存在聲稱 CI 通過。
- Actions 固定已驗證的完整 commit SHA、限制 GITHUB_TOKEN 權限。
- 使用 GitHub OIDC 與 Google Workload Identity Federation，不提交服務帳號 JSON key。
- 正式環境部署、合併 PR、修改 repo 安全設定，應有明確授權與審核。
- CI 不可直接呼叫正式 LINE Messaging API 或付費 AI／Google Cloud。
