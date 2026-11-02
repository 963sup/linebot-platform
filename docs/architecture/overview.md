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

## 邊界

- **Domain**：業務模型與不變量，不接觸 SDK。
- **Application**：執行流程、命令、查詢、ports；只有協調，不含 HTTP 細節。
- **Adapter**：HTTP/CLI 入站，SDK / Firestore / Cloud Tasks 等出站。
- **Infrastructure**：外部 client、憑證、transport、觀測與執行設定。
- **Bootstrap**：依賴組裝；cmd 只啟動服務。

## 可靠性

- LINE webhook 驗章與 idempotent inbox。
- Inbox + outbox 在相同持久交易中保存，再由 dispatcher 交給 Cloud Tasks。
- Cloud Tasks 可能重複執行，worker 要使用 lease、防重入與修復流程。
- Reply token 有時效，長時間 AI 工作要明確決定 push 策略。
- 發送 HTTP timeout 可能造成結果不確定，需追蹤相同 retry key 和 request payload。
- Unsend／edit 會影響記憶、儲存與尚未完成的 AI 回覆。
- 保留對資料存取、AI 訊息資料、隱私與帳號限制的審核。

這些都是目標規範；目前**尚未有任何運行中的服務**。
