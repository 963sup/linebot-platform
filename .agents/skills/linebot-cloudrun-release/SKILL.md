---
name: linebot-cloudrun-release
description: Prepare and validate an authorized Cloud Run release
---

# linebot-cloudrun-release

使用情境：本專案修改相關領域時，由 Codex 按需啟用。本技能為流程文件，不自動授予外部權限。

## 步驟

1. 先確認使用者明確授權的目標環境與變更範圍；未授權只能提供計畫。
2. 核實 commit、固定 image digest、建置測試、CI 與 provenance。
3. 核實 WIF、OIDC、Secret Manager、service-to-service IAM、Cloud Tasks。
4. 先 staging smoke tests，再由使用者核准 production promotion。
5. 記錄實際執行與 rollback 方式，不可自行宣稱完成部署。

## 完成條件

回報受影響檔案、驗證指令、成功／失敗結果與尚未完成事項。不能將規劃中的空目錄視為已通過檢查。
