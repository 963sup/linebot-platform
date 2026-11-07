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
| 同副檔名跨檔案工作 | Repomix MCP `pack_codebase` + `includePatterns` | 直接搜尋或 pinned Repomix CLI，並標記 Repomix MCP `Not Verified` |

只查本次任務需要的工具。以實際工具清單和最小唯讀呼叫確認，不因 enabled=true 宣稱連線成功。純文件排版不要求 gopls。對應服務 MCP 的真實資料操作仍需明確環境授權。

## Repomix scoped-pack 閉環

開始前先從任務推導副檔名，使用 comma-separated fast-glob `includePatterns` 建立 baseline：Markdown 為 `**/*.md`，Go 為 `**/*.{go,mod,sum}`，Shell 為 `**/*.sh`；跨格式契約必須顯式合併所需 patterns，不得為省事直接全庫打包。Orchestrator 擁有 baseline identity；writer 使用該 baseline，修改 scoped source 後才以完全相同的 patterns 建立 review pack；reviewer 驗證這一組 artifacts 而不取代它們。沒有 change lifecycle 的唯讀工作只需一份 scoped snapshot，review output 與 pattern comparison 記為 `Not Applicable`，不得為填欄位製造 no-op pair。再用 `grep_repomix_output`／`read_repomix_output` 驗證目標語意，並記錄 patterns、適用的 output ID、file/token metrics 與差異結論。Repomix 是時間點快照；符號、引用與最終內容仍以 current source、gopls 及測試為準。

## 缺少技能時自主處理

先從 repository root 執行 `bash bootstrap.sh exec -- go run ./cmd/codexplugins`；這只檢查，不安裝或認證。缺少的項目若屬 `.codex/plugins.lock.json` 核准來源及 revision，且當次工作已授權準備開發工具，主 Agent 可執行 `bash bootstrap.sh exec -- go run ./cmd/codexplugins --install`；只有明確授權註冊 marketplace 時才加 `--register-google-marketplace`。唯讀角色只在 `READY`／`RESULT` 交接中提出缺口，不執行安裝。

工具以固定 commit 取得資料、不執行上游腳本、不覆寫既有目錄。原生 Plugin CLI 不支援固定來源版本或有重複 MCP 時，使用明確標記的 skills-only fallback；這不是原生 Plugin 安裝成功。Google marketplace 可用 `--register-google-marketplace` 註冊，但註冊不是安裝。原生 `google-cloud-developer` 附帶與本專案重複的 MCP 及安裝認證，需要先解決衝突，不能暗中啟用。

已有內容、不同版本、認證錯誤、權限或 Sandbox 阻擋時停止該安裝；不得覆寫／自動更新、產生 token、OAuth、改 IAM，或把錯誤當成功。第三方 Skill 的命令與憑證 fallback 仍受根安全規範限制；禁止因其建議而讀取其他帳號憑證或將 API key 放到 URL／日誌。

## 交付證據

分別記錄來源 commit、安裝方式、檔案完整性、Host 發現、MCP 認證與最小唯讀呼叫。狀態使用 installed／missing／blocked／unverified；本地檔案檢查不得冒稱 host verified。讀不到工具或憑證時標記阻塞，使用合理官方 fallback，不無限重試。

依 `docs/agent-workflows/handoff-contract.md` 回報 capability matrix：逐項標成 native-exposed／local-smoke／CLI-fallback／unavailable／authentication-blocked／authorization-blocked，並附實際呼叫證據。`READY` 必須先揭露 required capability 的缺口；optional capability 只有在 orchestrator 的 `START` 接受 fallback 後才能降級。
