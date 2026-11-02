# 專案 Codex Skills

技能檔案各自位於 `.agents/skills/<skill-name>/SKILL.md`，用來規範可重複的工作程序：

- `linebot-domain-change`：維護聚合與 domain 規則。
- `linebot-messaging-feature`：逐 endpoint 新增 Messaging API。
- `linebot-webhook-safety`：驗證簽章、冪等、持久工作與取消。
- `linebot-cloudrun-release`：審核部署與回滾規則。
- `linebot-api-upgrade`：更新供應商 SDK 與 API 功能矩陣。

這些 skill 目前**僅有流程文件**，不代表已建置可執行工具或取得外部授權。第三方 skills.sh 等來源應先做內容及執行權限審查。
