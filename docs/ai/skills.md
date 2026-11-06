# 專案 Codex Skills

技能檔案各自位於 `.agents/skills/<skill-name>/SKILL.md`，用來規範可重複的工作程序：

- `agent-tool-preflight`：在動工前核對必要工具、MCP、Skill 與安全 fallback。
- `linebot-domain-change`：維護聚合與 domain 規則。
- `linebot-messaging-feature`：逐 endpoint 新增 Messaging API。
- `linebot-webhook-safety`：驗證簽章、冪等、持久工作與取消。
- `linebot-cloudrun-release`：審核部署與回滾規則。
- `linebot-api-upgrade`：更新供應商 SDK 與 API 功能矩陣。
- `repomix-reference`：評估是否值得生成程式碼參考 skill，並管理生成、審查、Codex 目錄整合及來源變更後的更新。
- `sdk-integration`：從 [SDK Catalog](../sdk/catalog.toml) 選擇模組、核對實際 API、按架構接線並提供測試證據。

這些 skill 目前**僅有流程文件**，不代表已建置可執行工具或取得外部授權。第三方 skills.sh 等來源應先做內容及執行權限審查。

## 外部技能鎖定清單

根目錄 `skills-lock.json` 依照 [963sup/line-bot-go 原始 lockfile](https://github.com/963sup/line-bot-go/blob/main/skills-lock.json) 的 schema，**精選 13 個**本專案近期使用的外部技能；來源欄位與 `computedHash` 原樣保留。

- 雲端／CI：`cloud-run-basics`、`google-cloud-recipe-auth`、`github-actions-hardening`。
- Go：`golang-code-style`、`golang-continuous-integration`、`golang-context`、`golang-dependency-management`、`golang-error-handling`、`golang-naming`、`golang-observability`、`golang-project-layout`、`golang-security`、`golang-testing`。

此 lockfile 僅是**外部來源的版本／內容指紋清單**：不代表已下載、安裝、重新計算雜湊或通過安全稽核；來源可能已更新。首次啟用前必須先驗證來源及 `SKILL.md`／可執行腳本、校驗實際 hash、檢查權限與 license，不能自動信任或執行。
本地 `.agents/skills/` 的專案自訂技能是另一套檔案；實際清單以該目錄中的 `SKILL.md` 為準，不應被外部 lock 覆蓋。


## Repomix 生成的參考技能

使用與更新規則集中在 [repomix-reference](../../.agents/skills/repomix-reference/SKILL.md)。有重複跨檔案探索需求才生成；一次性任務維持直接讀取或 scoped pack。以來源內容、檔案集合、治理契約與生成設定判斷過期，不因無關 commit 或固定天數重建。

`repomix_skills.generate_skill` 固定先寫入 `.claude/skills/<candidate>/`，再審查移入 `.agents/skills/<stable-name>/`；若不要求 MCP 呼叫，可用 CLI `--skill-output .agents/skills/<stable-name>` 直接輸出。這個參數是完整 skill 目錄，不是共用父目錄。既有自訂技能不被覆寫。每份生成參考需有 provenance 與來源 hashes；沒有實際重複用途時不預先建立整庫快照。
