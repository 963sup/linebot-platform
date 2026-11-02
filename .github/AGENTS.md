# GitHub Workflow Agent Rules

- 只有可編譯且可測試的 Go 專案建立後才加入會執行 Go build/test 的 CI，避免誤導性的綠燈。
- Actions 固定已驗證的完整 commit SHA、限制 GITHUB_TOKEN 權限。
- 使用 GitHub OIDC 與 Google Workload Identity Federation，不提交服務帳號 JSON key。
- 正式環境部署、合併 PR、修改 repo 安全設定，應有明確授權與審核。
- CI 不可直接呼叫正式 LINE Messaging API 或付費 AI／Google Cloud。
