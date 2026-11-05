# Codex Cloud：設定、安裝與實測分離

此文件是儲存庫環境契約，不代表 Codex Cloud 已 Publish。工具設定只在受信任專案且 Host 支援時載入；CLI、IDE、Cloud 需各自驗證，不互相推論。

## Setup

在 Codex Cloud 的 Install script 使用：

```bash
bash scripts/codex-cloud/install.sh
```

腳本依 go.mod 的 Go 1.27.3 要求，下載／驗證已鎖定 module、執行 codexcheck、安裝既定 gopls v0.24.0 至 `/workspace/.linebot-tools/bin`。Task phase 的 `/home/agent` 可能唯讀，因此可變工具與快取只寫入 `/workspace`；`$HOME/.agents/skills` 僅供探索驗證。所有階段共同載入 `scripts/codex-cloud/lib/paths.sh`，不依賴 install phase 的 export 延續。不以 go mod tidy 修改依賴、不部署、不執行 OAuth／gcloud、不建立 key、不修改個人模型／approval。

新工作階段依序執行：

```bash
bash scripts/codex-cloud/start.sh
```

`start.sh` 會依環境契約強制 fetch、切換 main、reset 至 `origin/main` 並清除未追蹤與 ignored 檔案，然後執行 `verify.sh`。

核准的可選設定：

| 變數 | 預設 | 行為 |
| --- | --- | --- |
| CODEX_CLOUD_INSTALL_SKILLS | 0 | 設 1 安裝 .codex/plugins.lock.json 固定來源的 skills-only 檔案 |
| CODEX_CLOUD_REGISTER_GOOGLE_MARKETPLACE | 0 | 設 1 且 INSTALL_SKILLS=1，依 CLI 能力註冊固定 Google marketplace；不是安裝 Plugin |
| CODEX_CLOUD_WARM_TESTS | 0 | 設 1 額外執行 go test ./... |

未允許安裝時只執行技能檢查；缺少項目會標記，不因可選技能缺少而把整個 Go 工具鏈設置說成失敗。明確要求安裝卻被阻塞時腳本失敗，不吞掉錯誤。CLI 不支援、版本不同或已有非受控目錄時不覆寫。

## 認證與網路

公開 Go modules／gopls 下載需要 Go proxy/checksum 網路。Skills 安裝需要核准 GitHub HTTPS 來源。Agent phase 的網路權限與 setup 不一定相同；由有權限的使用者設定允許網域並 Publish／Republish 後，另開工作階段驗證。

| 能力 | Host 前置條件 |
| --- | --- |
| gopls | `$HOME/.local/bin/gopls`；POSIX／WSL／Cloud Bash |
| Context7 | mcp.context7.com 網路；匿名額度可能受限 |
| OpenAI Docs | developers.openai.com 網路；不需要 OpenAI API key |
| GitHub | 安全提供 GITHUB_MCP_TOKEN，限制 repository／操作權限 |
| Google Developer Knowledge | 安全提供 DEVELOPERKNOWLEDGE_API_KEY；映射 header，不放 repo／URL |
| 其餘服務 MCP | 預設停用；隔離環境、最小權限與明確任務授權 |

不要新增 production LINE token、服務帳號 JSON 或廣權限 PAT 只為驗收工具。角色文字「唯讀」不是 IAM 或 token 的技術隔離。

## 驗證

```bash
go run ./cmd/codexcheck
bash -n scripts/codex-cloud/install.sh
bash -n scripts/codex-cloud/start.sh
bash scripts/codex-cloud/verify.sh
go mod verify
make check
"$HOME/.local/bin/gopls" version
# 只有 Host 確實提供 Codex CLI 才執行：
codex --version
codex mcp list
```

gopls version、codex mcp list 及安裝收據不等於成功的 MCP handshake／工具呼叫。新工作階段按 [驗收表](agents-mcp-plugins-acceptance.md) 執行最小唯讀任務，確認角色／模型、Skills 發現、gopls references、Context7、GitHub、OpenAI Docs、Google 文件查詢。

[Agents](agents.md)、[MCP／Plugins](mcp-and-plugins.md) 及根 AGENTS 是操作規範；CI 僅驗證儲存庫可測的部分，不代表雲端環境已發布、Google 已授權或 Plugin 已在你的主機啟用。
