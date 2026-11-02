---
name: linebot-api-upgrade
description: Upgrade LINE and other Go SDKs safely
---

# linebot-api-upgrade

使用情境：本專案修改相關領域時，由 Codex 按需啟用。本技能為流程文件，不自動授予外部權限。

## 步驟

1. 比較當前採用版本與上游 release notes／官方 README。
2. 檢查 Go 最低版本、安全 patch、breaking API changes 與地區資格。
3. 更新 `docs/versions.md`、`docs/sources.md`、功能矩陣及 contract fixtures。
4. 依最小範圍修改 adapters，不讓 SDK 型別滲入 domain。
5. 執行測試與確認回滾路徑；無法測試的項目必須明列。

## 完成條件

回報受影響檔案、驗證指令、成功／失敗結果與尚未完成事項。不能將規劃中的空目錄視為已通過檢查。
