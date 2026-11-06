# Codex Cloud：設定、安裝與實測分離

此文件是儲存庫環境契約，不代表 Codex Cloud 已 Publish。工具設定只在受信任專案且 Host 支援時載入；CLI、IDE、Cloud 需各自驗證，不互相推論。

## Setup

在 Codex Cloud 的 Install script 使用：

```bash
bash scripts/codex-cloud/install.sh
```

腳本先依 `config/toolchain.lock` 下載並驗證 mise v2026.11.3，再由 repo `mise.toml` 安裝 Go 1.27.3、Node.js 24.22.1 與 gopls 0.24.0；同一檔案提供不含 Secret 的 local bootstrap 預設。mise 的 data、state、config、cache 與 shims 全部位於 `/workspace`。接著下載／驗證已鎖定 module、執行 codexcheck，並依 `tools/mcp-runtime/package-lock.json` 以 npm 安裝固定的本地 MCP runtime 至 `/workspace/.linebot-tools/mcp-runtime`。Install phase 不寫入 Host-owned `/home/agent/.codex/config.toml`，也不自行建立替代 `CODEX_HOME`；專案信任與 repo `.codex/config.toml` 載入由 Cloud Environment Host 負責，並在 Publish／Republish 後的新工作階段驗證。repo `.codex/config.toml` 是 MCP 設定、工具白名單與 approval policy 的單一來源。安裝後會驗證 lock hash、receipt、package versions 與四個 binary；這不等於 Host 已載入 MCP、MCP 已初始化或工具呼叫成功。Task phase 的 `/home/agent` 可能唯讀，因此可變工具、mise、Go、Node、gopls 與 cache 只寫入 `/workspace`；`$HOME/.agents/skills` 僅供探索驗證。所有階段共同載入 `scripts/codex-cloud/lib/paths.sh`，不依賴 install phase 的 export 延續。不以 go mod tidy 修改依賴、不部署、不執行 OAuth／gcloud、不建立 key、不修改個人模型／approval。

每個入口在任何寫入前執行 `preflight.sh`：確認 PATH 前綴、必要系統命令、`/workspace` 可寫，以及所有可變工具與 cache 路徑都位於 `/workspace`。preflight 本身不建立目錄、不安裝工具、不修改 repository。

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

Install 完成後會驗證 repository-owned `.codex/config.toml`、固定 MCP runtime 的 lock hash、receipt、package versions 與四個 binary。Host 是否載入全部 10 個啟用 MCP，以及 `firestore_toolbox` 是否維持停用，留在 Publish／Republish 後的新工作階段由 `verify.sh` 與最小唯讀工具呼叫驗證。缺少 pinned runtime 時安裝失敗，不吞掉錯誤；Host registration、authentication、authorization 與實際 handshake 分開回報。未允許安裝時只執行技能檢查；缺少項目會標記，不因可選技能缺少而把整個 Go 工具鏈設置說成失敗。CLI 不支援、版本不同或已有非受控目錄時不覆寫。

## 認證與網路

公開 Go modules／gopls 下載需要 Go proxy/checksum 網路。Skills 安裝需要核准 GitHub HTTPS 來源。Agent phase 的網路權限與 setup 不一定相同；由有權限的使用者設定允許網域並 Publish／Republish 後，另開工作階段驗證。

| 能力 | Host 前置條件 |
| --- | --- |
| Go／Node／gopls | `mise.toml` 固定版本；執行檔由 `/workspace/.linebot-tools/mise/shims` 提供 |
| 本地 npm MCP runtime | Node.js 22+、npm、jq、sha256sum；依 lock 預安裝到 `/workspace/.linebot-tools/mcp-runtime` |
| Repomix | 預安裝固定 1.18.1 sandbox，只讀取 linebot-platform |
| Context7 | mcp.context7.com 網路；匿名額度可能受限 |
| OpenAI Docs | developers.openai.com 網路；不需要 OpenAI API key |
| GitHub | 安全提供 GITHUB_MCP_TOKEN，限制 repository／操作權限 |
| Google Developer Knowledge | 安全提供 DEVELOPERKNOWLEDGE_API_KEY；映射 header，不放 repo／URL |
| 其餘服務 MCP | 依 repository config 啟用或停用；仍需隔離環境、最小權限與明確任務授權 |
| LINE MCP | 安全提供 CHANNEL_ACCESS_TOKEN；僅使用已列入 allowlist 的唯讀工具 |

所有已設定 Agent 繼承 repository MCP 設定，但這不代表 Host 已啟動／初始化伺服器，也不代表已認證、已授權或已實測。`google_cloud_cli` 啟用 `run_gcloud_command` 並對每次呼叫要求 prompt approval；repository 將用途限於唯讀 Observability 查詢，但這是 policy-only 規範，不是通用 gcloud command 工具的技術唯讀保證。實際可執行範圍仍受 approval、身分與 IAM 控制。不要新增 production LINE token、服務帳號 JSON 或廣權限 PAT 只為驗收工具。角色文字「唯讀」不是 IAM 或 token 的技術隔離。

## 驗證

```bash
go run ./cmd/codexcheck
bash -n scripts/codex-cloud/install.sh
bash scripts/codex-cloud/preflight.sh
bash -n scripts/codex-cloud/start.sh
bash scripts/codex-cloud/verify.sh
go mod verify
make check
gopls version
jq . /workspace/.linebot-tools/mcp-runtime/receipt.json
/workspace/.linebot-tools/mcp-runtime/node_modules/.bin/repomix --version
# 只有 Host 確實提供 Codex CLI 才執行：
codex --version
codex mcp list
```

gopls version、Repomix version、codex mcp list 及安裝收據不等於成功的 MCP handshake／工具呼叫。新工作階段按 [驗收表](agents-mcp-plugins-acceptance.md) 執行最小唯讀任務，確認角色／模型、Skills 發現、Repomix sandbox、gopls references、Context7、GitHub、OpenAI Docs、Google 文件查詢。

[Agents](agents.md)、[MCP／Plugins](mcp-and-plugins.md) 及根 AGENTS 是操作規範；CI 僅驗證儲存庫可測的部分，不代表雲端環境已發布、Google 已授權或 Plugin 已在你的主機啟用。
