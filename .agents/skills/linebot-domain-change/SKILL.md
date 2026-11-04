---
name: linebot-domain-change
description: DDD domain model change workflow
---

# linebot-domain-change

使用情境：本專案修改相關領域時，由 Codex 按需啟用。本技能為流程文件，不自動授予外部權限。

## 步驟

1. 閱讀根目錄與 `internal/domain/AGENTS.md`、對應 aggregate 與 usecase。
2. 描述業務不變量、Identity／Aggregate 邊界、Value Objects、Entity、Policy、Specification、Repository 契約，以及目前舊 DTO 的重複欄位；先明確列出保留／淘汰清單。
3. 先提出修改計畫與驗收條件，再建立純業務測試並實施最小必要變更；同時移除無引用的舊檔案，避免平行業務模型。
4. 禁止 domain 依賴 LINE／AI／Google SDK。
5. Value Objects 目錄必須為每個 Bounded Context 的 `value-objects/`，Go package 名為 `valueobjects`；不得復原 `valueobjects/` 與 `value_objects/` 目錄。
6. 驗證 `gofmt`、`go vet`、`go test -race ./...`、架構測試與 GitHub Actions；保留 OAuth Token 加密、Firestore CAS、Cloud Tasks 契約，不因目錄收斂改變持久化協議。

## 完成條件

回報受影響檔案、驗證指令、成功／失敗結果與尚未完成事項。不能將規劃中的空目錄視為已通過檢查。
