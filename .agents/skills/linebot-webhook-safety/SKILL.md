---
name: linebot-webhook-safety
description: Review LINE webhook security and reliability
---

# linebot-webhook-safety

使用情境：本專案修改相關領域時，由 Codex 按需啟用。本技能為流程文件，不自動授予外部權限。

## 步驟

1. 檢查 raw body 驗章、body size、channel isolation 與所有 webhook event parser。
2. 驗證去重 key、inbox/outbox transaction、dispatcher 補償、Cloud Tasks retry。
3. 驗證 reply token 時效、send unknown 結果、unsend／edit 與過期 AI 回覆。
4. 測試重送、併發、失敗、事件亂序；只在有實作時回報實測結果。
5. 不記錄 token、原始 webhook 或私人訊息。

## 完成條件

回報受影響檔案、驗證指令、成功／失敗結果與尚未完成事項。不能將規劃中的空目錄視為已通過檢查。
