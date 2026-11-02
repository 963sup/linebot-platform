---
name: linebot-messaging-feature
description: Implement one LINE Messaging API feature at a time
---

# linebot-messaging-feature

使用情境：本專案修改相關領域時，由 Codex 按需啟用。本技能為流程文件，不自動授予外部權限。

## 步驟

1. 查核最新 LINE endpoint、帳號及地區限制、所使用 SDK 版本。
2. 更新 `docs/line/feature-matrix.md` 與端點覆蓋狀態。
3. 先定義 application port、DTO 與錯誤語意，後實作 adapter。
4. 建立 mock/fake 與 contract 測試，重送場景使用同一 idempotency key。
5. 無實際授權時不能呼叫正式 LINE Channel 或宣稱功能可用。

## 完成條件

回報受影響檔案、驗證指令、成功／失敗結果與尚未完成事項。不能將規劃中的空目錄視為已通過檢查。
