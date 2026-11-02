# Codex Cloud 最小、可重現環境

**本檔是已提交的環境安裝契約，不是已發布的 Codex Cloud Environment。** 完成 PR 合併後，需由使用者在 Codex Cloud UI 指定這個儲存庫、執行安裝並審核發布。

## 1. 建立或更新 Codex Cloud 環境

1. 在 ChatGPT / Codex Cloud 選 `Work in → Cloud → Select environment → Create environment`，選 `963sup/linebot-platform`。
2. 設定環境的 **Install script** 為：

   ```bash
   bash scripts/setup-codex-cloud.sh
   ```

3. 讓 Codex Cloud 準備並檢查，確認 Go 工具鏈、`go.mod` 依賴與 `gopls v0.24.0` 實際成功後，**由你選擇 Publish**。已建立的環境修改後需 Edit → Republish，新的工作階段才會使用更新的預備狀態。
4. 對本地 CLI／IDE：在可信儲存庫內也可執行同一 script；先確認其安裝操作與權限。

如果雲端環境無法取得 Go 1.27.3，應先設定相容的 Go Toolchain；腳本不會隨意下載並執行網路上的 shell 一鍵安裝器。

## 2. Setup script 做的事情

- 使用 `GOTOOLCHAIN=auto` 讀取並遵守根 `go.mod` 的版本要求。
- 校驗 `.codex/config.toml`、五份 Agent TOML 與 setup 腳本本身。
- 執行 `go mod download`、`go mod verify`；不執行會改動 `go.mod` 的 `go mod tidy`。
- 固定安裝 Go 官方 `gopls v0.24.0` 到 `$HOME/.local/bin`。腳本可重複執行，已安裝符合版本時跳過。
- 預設**不執行**耗時的 race test；可選擇使用環境變數 `CODEX_CLOUD_WARM_TESTS=1` 在安裝時先做普通 `go test ./...`。
- 不安裝 Context7 本地 NPM server、不啟動 Toolbox、不執行 `gcloud`、不認證 GitHub MCP、不修改你個人的 `~/.codex/config.toml`，也不儲存任何密鑰。

## 3. 網路與秘密

| 功能 | 必要服務／變數 | 預設 |
|---|---|---|
| Go module 與 gopls 下載 | Go module proxy 與 checksum database；安裝階段允許套件下載 | 必要 |
| gopls MCP | 本地 `$HOME/.local/bin/gopls` | 啟用，無憑證 |
| Context7 遠端 MCP | `mcp.context7.com` 的 HTTPS 連線 | 選用；公開文件無硬編碼 key |
| GitHub MCP | `api.githubcopilot.com`、可存放於 personal vault 的 `GITHUB_MCP_TOKEN` | 停用，直到有權限 |
| Firestore Toolbox | Node 20+、`FIRESTORE_PROJECT`、測試專案的 ADC，可能使用 `FIRESTORE_DATABASE` | 停用，直到隔離 staging |

Cloud agent phase 的網路政策與 setup 的套件下載權限不一定相同；需要讓 Context7 遠端工具在 agent phase 可連線時，應依 Cloud 環境 UI 設置允許的主機。沒有權限或無法連線時，Agent 必須退回已快取文檔或官方網頁／原始碼，不能宣稱 MCP 成功。

不要為了方便，在 Codex Cloud 環境設定中新增正式 LINE Channel token、OpenAI token、GCP production service-account JSON 或 GitHub 全權限 PAT。若未來需要測試雲端，請先建立隔離 staging 與短期、最小權限身分。

## 4. 驗證步驟

```bash
python3 scripts/validate-codex-config.py
bash -n scripts/setup-codex-cloud.sh
go mod verify
make check
$HOME/.local/bin/gopls version

# 如果 Cloud 環境確實提供 codex CLI，可再執行：
codex mcp list
```

這些指令需在實際 Cloud 環境執行才算雲端驗證通過。GitHub CI 會先檢查 TOML／shell 語法，但**CI 通過不代表 Cloud MCP 已認證、工具可連線或雲端環境已 Publish**。

## 5. 自訂 Agents 的使用

角色位於 [`.codex/agents/`](../../.codex/agents/)，詳細職責見 [agents.md](agents.md)。已由 Codex 官方文件確認這是專案級 Agents TOML 格式，但其在特定 Codex Cloud 工作階段的載入狀態必須在該環境測試。可在任務描述中要求主 Agent 將獨立工作委派給對應專家；一般簡單工作使用預設 worker，不必浪費多 Agent token。
