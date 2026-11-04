# Codex MCP 與 Plugins（正式專案設定）

MCP 是**開發期工具**，由 [`.codex/config.toml`](../../.codex/config.toml) 管理；LINE Bot 的 Cloud Run Runtime 仍獨立使用 Go SDK。Codex SDK 任務先讀 [SDK Catalog](../sdk/catalog.toml)，再按 [SDK Integration Skill](../../.agents/skills/sdk-integration/SKILL.md) 使用本地模組原始碼／gopls／官方文件；不需要為每個 SDK 加裝一個 MCP。根 `AGENTS.md` 僅描述何時使用工具，**不能安裝、登入或代管 MCP 憑證**。

## 7 個 MCP 與安全預設

| MCP 名稱 | 官方實作 | 啟用狀態 | 何時使用 |
|---|---|---|---|
| `gopls` | [gopls MCP](https://go.dev/gopls/features/mcp)；`gopls mcp` | 開啟（需安裝） | Go symbols、definitions、references、型別／錯誤診斷、重構前分析 |
| `context7` | [upstash/context7](https://github.com/upstash/context7)；`https://mcp.context7.com/mcp` | 開啟（遠端） | 第三方 SDK 版本、安裝和程式 API 用法，仍需核對官方資料 |
| `openaiDeveloperDocs` | [OpenAI Docs MCP](https://developers.openai.com/learn/docs-mcp)；`https://developers.openai.com/mcp` | 開啟（遠端、唯讀） | 官方 OpenAI API、SDK、Responses API、Codex、Skills、MCP 和 Plugins 文件 |
| `github_readonly` | [github/github-mcp-server](https://github.com/github/github-mcp-server)；`https://api.githubcopilot.com/mcp/` | 啟用（需環境 token） | 依授權讀取 PR、Issue、commit、workflow 等，不寫入 |
| `firestore_toolbox` | [googleapis/mcp-toolbox](https://github.com/googleapis/mcp-toolbox)；Firestore prebuilt | 停用（未配置 staging ADC） | 在**隔離測試專案**審查 Firestore collections、documents、rules |
| `google_cloud_run` | Google Cloud Run remote MCP；`https://run.googleapis.com/mcp` | 停用（尚未配置 Google OAuth／最小 IAM） | 讀取 service、revision、部署與診斷資料 |
| `google_firestore` | Google Firestore remote MCP；`https://firestore.googleapis.com/mcp` | 停用（尚未配置 Google OAuth／最小 IAM） | 讀取隔離 staging 的 Firestore 資料與設定 |

`gopls` detached MCP 只能看到已儲存到磁碟的內容；它可能在分析 package 時下載 Go modules，不應誤認為只是一個純離線讀檔工具。

`context7` 公開文件查詢不需專案預設寫入 token；匿名存取可能有速率限制。若要使用私有文件或較高額度，應在使用者自己的 Codex 設定／認證流程中加入憑證，**不要在這個 repo 的 TOML 硬編碼 Header**。

## GitHub MCP：唯讀與權限

- `github_readonly` 只在環境安全提供 `GITHUB_MCP_TOKEN` 時連線；不要提交 token 或為了讓它能用而在 CI 自動登入。
- 以 `X-MCP-Readonly: true` 限制 MCP 提供的工具，`X-MCP-Toolsets` 收斂至 `repos,issues,pull_requests`；但 token 自身權限仍須單獨限制。

## Google MCP Toolbox：不是 Google Workspace API MCP

`mcp-toolbox` 是資料庫工具伺服器，提供 Firestore、Cloud SQL、BigQuery 等預建工具；**不取代** Google Calendar、Sheets、Drive 的正式 Go API Client。

## Google 管理的遠端 MCP

Google Cloud Run 與 Firestore 都提供 Google 管理的遠端 MCP server，分別使用 `https://run.googleapis.com/mcp` 與 `https://firestore.googleapis.com/mcp`。本專案先將它們保留為停用設定：啟用前必須在 Codex Cloud 取得 Google OAuth 身分，並在隔離 staging 專案授予最小的唯讀 IAM。不要在 TOML 放入 ADC 檔案路徑、access token 或 OAuth client secret。

啟用後先以官方 server 提供的最小診斷 toolset 做唯讀驗證；確認 Cloud Run service/revision 與 Firestore 查詢都只能存取預期的 staging 專案後，才考慮增加任何寫入工具。Cloud Run runtime 本身不使用這些 MCP。

本專案的預留設定使用 `npx -y @toolbox-sdk/server@1.14.0 --prebuilt=firestore --stdio`，預設 `enabled = false` 且以 `enabled_tools` 只暴露 `get_documents`、`query_collection`、`list_collections`、`get_rules`、`validate_rules`。

**重要：**Toolbox 上游 Firestore 工具集本來就含新增／更新／刪除資料操作；Codex 工具 allowlist 不是 GCP IAM 的安全邊界。啟用前須使用獨立 staging 專案、最小權限憑證並確認無 production 資料；不可直接用具廣泛寫入權限的 production ADC。

## 設定與驗證

1. Cloud setup 在非互動情境安裝 `gopls v0.24.0` 至 `$HOME/.local/bin/gopls`，專案 config 用絕對 HOME 路徑執行 `gopls mcp`。
2. 本地執行 `go run ./cmd/codexcheck` 確認七個 MCP、八個 Agents 與 setup shell 語法。
3. 若可使用 Codex CLI，在可信 repo 內執行 `codex mcp list`，檢查已啟用 server 與認證狀態。
4. 在 Codex Cloud 設定網路允許必要主機（包括 `developers.openai.com`），Publish／Republish 後於**新工作階段**再驗證 MCP 可用性；本專案尚未代你發布雲端環境。

官方索引：[Codex Config Reference](https://learn.chatgpt.com/docs/config-file/config-reference)、[Codex Cloud Environments](https://learn.chatgpt.com/docs/environments/cloud-environments)、[GitHub MCP Read-only](https://github.com/github/github-mcp-server/blob/main/docs/server-configuration.md)、[Firestore Toolbox](https://mcp-toolbox.dev/integrations/firestore/prebuilt-configs/firestore/)。

## Plugin 關係

此專案目前不強制安裝 Codex Plugins，並避免重複裝載同一個外部 MCP。外部 `skills-lock.json` 仍是來源／hash 清單，不表示已安裝技能。Plugin 安裝、連線與 OAuth 需由使用者操作和授權；Cloud Run runtime 不使用這些 Plugin。

OpenAI Docs MCP 不需要專案 API Key，也不負責執行 OpenAI API 請求。CI 的 TOML 檢查不等於遠端連線成功；Republish 後需在新 Codex Cloud 工作階段實際搜尋官方文件。
