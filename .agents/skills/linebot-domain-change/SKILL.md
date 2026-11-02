---
name: linebot-domain-change
description: DDD domain model change workflow
---

# linebot-domain-change

使用情境：本專案修改相關領域時，由 Codex 按需啟用。本技能為流程文件，不自動授予外部權限。

## 步驟

1. 閱讀根目錄與 `internal/domain/AGENTS.md`、對應 aggregate 與 usecase。
2. 描述業務不變量、identity、aggregate 邊界和 repository 契約。
3. 先建立純業務測試，再做最小必要修改。
4. 禁止 domain 依賴 LINE／AI／Google SDK。
5. 驗證編譯、測試與分層依賴，無 Go 程式時明確標示未執行。

## 完成條件

回報受影響檔案、驗證指令、成功／失敗結果與尚未完成事項。不能將規劃中的空目錄視為已通過檢查。
