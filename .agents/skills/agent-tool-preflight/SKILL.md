---
name: agent-tool-preflight
description: Select the task-relevant MCP and inspect or install the project's approved pinned skills without changing authentication or production permissions.
---

# Agent 工具前置檢查

先讀根 AGENTS.md 與 `.codex/config.toml`。本 Skill 不取代角色契約，也不授予外部寫入權限。

## 路由

| 任務 | 優先工具 | 降級 |
| --- | --- | --- |
| Go symbols／型別／重構 | `gopls` | 鎖定 module 原始碼、go list、go doc、編譯器與測試 |
| 函式庫、SDK、版本 | `context7` | 第一方版本文件；不得猜 API |
| GitHub repo、PR、CI | `github` | 授權 Git CLI；遠端狀態不可由本機推論 |
| Codex、Agents、Plugins | `openaiDeveloperDocs` | OpenAI 官方網站／原始碼；Context7 補充 |
| Google API／Cloud 文件 | `google_developer_knowledge` | Google 官方文件；Context7 補充，再查本地 Go symbols |

只查本次任務需要的工具。以實際工具清單和最小唯讀呼叫確認，不因 enabled=true 宣稱連線成功。純文件排版不要求 gopls。對應服務 MCP 的真實資料操作仍需明確環境授權。

## 缺少技能時自主處理

先執行 `bash scripts/check-codex-plugins.sh`；這只檢查，不安裝或認證。缺少的項目若屬 `.codex/plugins.lock.json` 核准來源及 revision，且當次工作已授權準備開發工具，主 Agent 可直接執行 `bash scripts/setup-codex-plugins.sh`；不需要使用者再次提供安裝命令。唯讀角色只提出交接，不執行安裝。

工具以固定 commit 取得資料、不執行上游腳本、不覆寫既有目錄。原生 Plugin CLI 不支援固定來源版本或有重複 MCP 時，使用明確標記的 skills-only fallback；這不是原生 Plugin 安裝成功。Google marketplace 可用 `--register-google-marketplace` 註冊，但註冊不是安裝。原生 `google-cloud-developer` 附帶與本專案重複的 MCP 及安裝認證，需要先解決衝突，不能暗中啟用。

已有內容、不同版本、認證錯誤、權限或 Sandbox 阻擋時停止該安裝；不得覆寫／自動更新、產生 token、OAuth、改 IAM，或把錯誤當成功。第三方 Skill 的命令與憑證 fallback 仍受根安全規範限制；禁止因其建議而讀取其他帳號憑證或將 API key 放到 URL／日誌。

## 交付證據

分別記錄來源 commit、安裝方式、檔案完整性、Host 發現、MCP 認證與最小唯讀呼叫。狀態使用 installed／missing／blocked／unverified；本地檔案檢查不得冒稱 host verified。讀不到工具或憑證時標記阻塞，使用合理官方 fallback，不無限重試。
