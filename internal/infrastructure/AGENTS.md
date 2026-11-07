# Infrastructure Agent Rules

- SDK 版本依官方文件查證，新增後在 `docs/versions.md` 更新採用版本。
- 先用 ADC / Workload Identity／Secret Manager，避免提交長期憑證。
- 限制各 client timeout、重試範圍、連線池與 shutdown；不得重試不具冪等保護的發送。
- Google Cloud 服務優先使用 `google-cloud-go`；Google Workspace 類 API 使用 `google-api-go-client`。
- 日誌、trace、metrics 禁止攜帶 token、原始使用者內容或多餘個資。
- 重用 `httpclient.Managed` 的獨立 Transport 與顯式 Close，SDK Factory 接收注入的原生 Client；需要不同認證的 Provider 不強迫共用單一 Client 或 Token。
- LINE SDK v8.23.0 的 `WithContext` 會修改 Receiver；不可對共享 Client 呼叫它，必須在 Adapter 中隔離每次請求 Context，SDK 升級時重新驗證。
- Firestore TTL 只負責非同步清理，不得當成即時授權或有效期判斷；位置等敏感資料必須由未來業務能力明確定義用途、權限與保存期限後才能持久化。
