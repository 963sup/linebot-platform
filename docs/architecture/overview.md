# 架構概覽（計畫）

採 Go modular monolith + DDD + Hexagonal Architecture，先以最小可測的 LINE 訊息流程驗證。

```text
LINE Platform
  -> Cloud Run webhook (public, raw-body signature verification)
  -> Firestore inbox + outbox
  -> Cloud Tasks (OIDC)
  -> Cloud Run worker (private)
  -> application usecase -> outbound port
  -> LINE Messaging API / OpenAI / Gemini / Google Cloud
```

其他 SDK 的 Client Factory、Codex SDK Catalog 與逐步啟用計畫見 [SDK Integration 設計](sdk-integration.md)。Catalog 僅為開發路由資料，非執行期 Registry。

## 邊界

- **Domain**：業務模型與不變量，不接觸 SDK；Value Objects 預留於 `internal/domain/valueobjects/`。
- **Application**：執行流程、命令、查詢、ports；只有協調，不含 HTTP 細節。
- **Adapter**：HTTP/CLI 入站，SDK / Firestore / Cloud Tasks 等出站；Gemini provider 使用 `internal/adapter/outbound/genai/`。
- **Infrastructure**：外部 client、憑證、transport、觀測與執行設定；Google Gen AI SDK client 預留於 `internal/infrastructure/google/genai/`。
- **Bootstrap**：依賴組裝；cmd 只啟動服務。

## 可靠性

- LINE webhook 驗章與 idempotent inbox。
- Inbox + outbox 在相同持久交易中保存，再由 dispatcher 交給 Cloud Tasks。
- Cloud Tasks 可能重複執行，worker 要使用 lease、防重入與修復流程。
- Reply token 有時效，長時間 AI 工作要明確決定 push 策略。
- 發送 HTTP timeout 可能造成結果不確定，需追蹤相同 retry key 和 request payload。
- Unsend／edit 會影響記憶、儲存與尚未完成的 AI 回覆。
- 保留對資料存取、AI 訊息資料、隱私與帳號限制的審核。

目前已加入可在本機啟動的 Go webhook service，具備 health check、簽章驗證及同步 Text/Flex Reply；CLI 支援需明確授權的 Push。這些**不是**具持久保證的正式 webhook：沒有 inbox/outbox、Cloud Tasks worker、去重或已部署的雲端服務。
